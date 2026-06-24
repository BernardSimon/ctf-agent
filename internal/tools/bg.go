package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ctf-agent/internal/jobs"
)

// BgRunTool 启动后台长任务（端口扫描/爆破/训练等），不阻塞 Agent。
// 任务日志写到 .ctf-agent/jobs/<id>/{stdout,stderr}.log，远端任务写远端 /tmp 再用 job_tail 拉。
type BgRunTool struct {
	store     *jobs.Store
	local     *jobs.LocalRunner
	ssh       *jobs.SSHRunner
	defaultOn string // "local" | "kali_ssh"
}

func NewBgRunTool(store *jobs.Store, defaultOn string, local *jobs.LocalRunner, ssh *jobs.SSHRunner) *BgRunTool {
	return &BgRunTool{store: store, local: local, ssh: ssh, defaultOn: defaultOn}
}

func (t *BgRunTool) Name() string { return "bg_run" }

func (t *BgRunTool) Description() string {
	return "在后台启动长任务（nmap/gobuster/hashcat 等），立即返回 job_id 不阻塞 Agent。日志落盘可用 job_tail 查看，job_kill 终止。"
}

func (t *BgRunTool) Parameters() []ParamDef {
	return []ParamDef{
		{Name: "command", Type: "string", Description: "要后台执行的命令（必填）", Required: true},
		{Name: "run_on", Type: "string", Description: "执行通道：local（本机）或 kali（远程 Kali SSH）。缺省按运行环境推断。", Required: false},
		{Name: "tag", Type: "string", Description: "任务备注（可选），便于多任务区分", Required: false},
	}
}

func (t *BgRunTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	cmd, err := ExtractArgs(args, "command")
	if err != nil {
		return "", err
	}
	runOn, _ := args["run_on"].(string)
	tag, _ := args["tag"].(string)
	if runOn == "" {
		runOn = t.defaultOn
	}
	switch runOn {
	case "local":
		if t.local == nil {
			return "", fmt.Errorf("本机后台任务运行器未初始化")
		}
		j, err := t.local.Start(cmd, tag)
		if err != nil {
			return "", err
		}
		return formatJobLaunch(j, "本机"), nil
	case "kali", "kali_ssh", "ssh":
		if t.ssh == nil {
			return "", fmt.Errorf("远端后台任务不可用：未启用 SSH 或 runtime.mode=kali（应直接用 run_on=local）")
		}
		j, err := t.ssh.Start(cmd, tag)
		if err != nil {
			return "", err
		}
		return formatJobLaunch(j, "Kali SSH"), nil
	default:
		return "", fmt.Errorf("未知 run_on=%q（支持 local|kali）", runOn)
	}
}

func formatJobLaunch(j *jobs.Job, channel string) string {
	out := map[string]any{
		"job_id":     j.ID,
		"status":     string(j.Status),
		"pid":        j.Pid,
		"channel":    channel,
		"cmd":        j.Cmd,
		"started_at": j.StartedAt.Format("2006-01-02 15:04:05"),
		"hint":       "用 job_status / job_tail / job_kill 跟踪",
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	return string(b)
}

// JobStatusTool 列出/查询任务。
type JobStatusTool struct {
	store *jobs.Store
}

func NewJobStatusTool(store *jobs.Store) *JobStatusTool { return &JobStatusTool{store: store} }

func (t *JobStatusTool) Name() string        { return "job_status" }
func (t *JobStatusTool) Description() string { return "查询后台任务状态。不传 job_id 时列出所有任务。" }
func (t *JobStatusTool) Parameters() []ParamDef {
	return []ParamDef{
		{Name: "job_id", Type: "string", Description: "任务 ID（可选）；不填则列全部任务", Required: false},
	}
}

func (t *JobStatusTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	id, _ := args["job_id"].(string)
	if id != "" {
		j, err := t.store.Get(id)
		if err != nil {
			return "", err
		}
		tail, _ := t.store.Tail(id, 3, "stdout")
		out := map[string]any{
			"id":         j.ID,
			"cmd":        j.Cmd,
			"run_on":     j.RunOn,
			"status":     j.Status,
			"pid":        j.Pid,
			"started_at": j.StartedAt.Format("2006-01-02 15:04:05"),
			"exit_code":  j.ExitCode,
			"tail":       tail,
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		return string(b), nil
	}
	all, err := t.store.List()
	if err != nil {
		return "", err
	}
	if len(all) == 0 {
		return "[]（暂无任务）", nil
	}
	var sb strings.Builder
	for _, j := range all {
		dur := "-"
		if !j.EndedAt.IsZero() {
			dur = j.EndedAt.Sub(j.StartedAt).Truncate(1e9).String()
		}
		sb.WriteString(fmt.Sprintf("%s  [%s]  pid=%d  on=%s  duration=%s\n  cmd: %s\n",
			j.ID, j.Status, j.Pid, j.RunOn, dur, truncate(j.Cmd, 120)))
	}
	return sb.String(), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// JobTailTool 拉取任务日志末尾 N 行。
type JobTailTool struct {
	store *jobs.Store
	ssh   *jobs.SSHRunner // 只用来标识有 SSH 渠道；实际拉远端日志需 fetcher
	rfetch RemoteFetcher
}

// RemoteFetcher 抽象 SSH 远端日志拉取，避免循环依赖。
type RemoteFetcher interface {
	FetchRemoteFile(path string, maxBytes int64) ([]byte, error)
}

func NewJobTailTool(store *jobs.Store, rfetch RemoteFetcher) *JobTailTool {
	return &JobTailTool{store: store, rfetch: rfetch}
}

func (t *JobTailTool) Name() string { return "job_tail" }
func (t *JobTailTool) Description() string {
	return "查看后台任务日志末尾 N 行。远程 Kali 任务会按需从远端拉取最新日志。"
}
func (t *JobTailTool) Parameters() []ParamDef {
	return []ParamDef{
		{Name: "job_id", Type: "string", Description: "任务 ID（必填）", Required: true},
		{Name: "lines", Type: "integer", Description: "返回末尾多少行（默认 80）", Required: false},
		{Name: "stream", Type: "string", Description: "stdout|stderr|both（默认 stdout）", Required: false},
	}
}

func (t *JobTailTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	id, err := ExtractArgs(args, "job_id")
	if err != nil {
		return "", err
	}
	n := 80
	if v, ok := toInt(args["lines"]); ok && v > 0 {
		n = v
	}
	stream, _ := args["stream"].(string)
	if stream == "" {
		stream = "stdout"
	}

	job, err := t.store.Get(id)
	if err != nil {
		return "", err
	}

	// 远端任务：先把 remote 日志 tail 到本地 stdout.log，然后调 store.Tail
	if job.RunOn == "kali_ssh" && t.rfetch != nil {
		remotePath := fmt.Sprintf("/tmp/ctf-agent-%s.log", job.ID)
		data, ferr := t.rfetch.FetchRemoteFile(remotePath, 256*1024)
		if ferr == nil && len(data) > 0 {
			_ = jobs.CopyRemoteToLocal(t.store.JobDir(job.ID), "stdout", strings.NewReader(string(data)))
		}
	}

	return t.store.Tail(id, n, stream)
}

// JobKillTool 终止任务。
type JobKillTool struct {
	store *jobs.Store
	local *jobs.LocalRunner
	ssh   *jobs.SSHRunner
}

func NewJobKillTool(store *jobs.Store, local *jobs.LocalRunner, ssh *jobs.SSHRunner) *JobKillTool {
	return &JobKillTool{store: store, local: local, ssh: ssh}
}

func (t *JobKillTool) Name() string        { return "job_kill" }
func (t *JobKillTool) Description() string { return "终止指定的后台任务。本机走 SIGTERM→SIGKILL，远端走 ssh kill。" }
func (t *JobKillTool) Parameters() []ParamDef {
	return []ParamDef{
		{Name: "job_id", Type: "string", Description: "任务 ID（必填）", Required: true},
	}
}

func (t *JobKillTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	id, err := ExtractArgs(args, "job_id")
	if err != nil {
		return "", err
	}
	job, err := t.store.Get(id)
	if err != nil {
		return "", err
	}
	switch job.RunOn {
	case "local":
		if t.local == nil {
			return "", fmt.Errorf("本机 runner 未初始化")
		}
		if err := t.local.Kill(id); err != nil {
			return "", err
		}
	case "kali_ssh":
		if t.ssh == nil {
			return "", fmt.Errorf("ssh runner 未初始化")
		}
		if err := t.ssh.Kill(id); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("未知 run_on=%q", job.RunOn)
	}
	return fmt.Sprintf("已请求终止 %s（pid=%d on %s）", id, job.Pid, job.RunOn), nil
}
