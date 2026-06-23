package tools

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type FetchTool struct {
	offlineMode bool
}

func NewFetchTool(offlineMode bool) *FetchTool {
	return &FetchTool{offlineMode: offlineMode}
}

func (t *FetchTool) Name() string { return "web_fetch" }

func (t *FetchTool) Description() string {
	if t.offlineMode {
		return "发送HTTP GET请求获取网页内容。离线模式下仅允许localhost、内网IP、*.local和裸主机名。"
	}
	return "发送HTTP GET请求获取网页内容。建议仅用于CTF靶机、本地服务和局域网地址。"
}

func (t *FetchTool) Parameters() []ParamDef {
	return []ParamDef{
		{Name: "url", Type: "string", Description: "目标URL", Required: true},
	}
}

func (t *FetchTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	url, err := ExtractArgs(args, "url")
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}

	if t.offlineMode {
		if err := ensureOfflineURL(url); err != nil {
			return "", err
		}
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 100*1024))
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	result := fmt.Sprintf("HTTP %d\n%s", resp.StatusCode, string(body))
	if len(result) > 5000 {
		result = result[:5000] + "\n...[output truncated]"
	}
	return result, nil
}

func ensureOfflineURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported url scheme in offline mode: %s", parsed.Scheme)
	}
	if parsed.User != nil {
		return fmt.Errorf("url userinfo is not allowed in offline mode")
	}
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("url host is required")
	}
	if port := parsed.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n <= 0 || n > 65535 {
			return fmt.Errorf("invalid url port: %s", port)
		}
	}
	if isOfflineAllowedHost(host) {
		return nil
	}
	return fmt.Errorf("offline mode blocks non-local target: %s", host)
}

func isOfflineAllowedHost(host string) bool {
	host = strings.Trim(strings.ToLower(host), "[]")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return true
	}
	if !strings.Contains(host, ".") {
		return true
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}
