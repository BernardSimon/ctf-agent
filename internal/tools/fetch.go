package tools

import (
	"bytes"
	"context"
	"encoding/json"
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
		return "发送 HTTP 请求获取响应。支持 GET/POST/PUT/DELETE/HEAD 和自定义 headers/body/form/json。离线模式下仅允许 localhost、内网 IP、*.local 和裸主机名。"
	}
	return "发送 HTTP 请求获取响应。支持 GET/POST/PUT/DELETE/HEAD 和自定义 headers/body/form/json。建议仅用于 CTF 靶机和局域网。"
}

func (t *FetchTool) Parameters() []ParamDef {
	return []ParamDef{
		{Name: "url", Type: "string", Description: "目标 URL", Required: true},
		{Name: "method", Type: "string", Description: "HTTP 方法：GET（默认）/POST/PUT/DELETE/HEAD", Required: false},
		{Name: "headers", Type: "object", Description: "自定义请求头（key-value 对象）", Required: false},
		{Name: "body", Type: "string", Description: "请求体原文（与 form/json 互斥）", Required: false},
		{Name: "form", Type: "object", Description: "form-urlencoded 请求体（与 body/json 互斥）", Required: false},
		{Name: "json", Type: "object", Description: "JSON 请求体（与 body/form 互斥；优先级最高）", Required: false},
	}
}

func (t *FetchTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	rawURL, err := ExtractArgs(args, "url")
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "http://" + rawURL
	}

	if t.offlineMode {
		if err := ensureOfflineURL(rawURL); err != nil {
			return "", err
		}
	}

	method := strings.ToUpper(strings.TrimSpace(strFromAny(args["method"])))
	if method == "" {
		method = "GET"
	}
	switch method {
	case "GET", "POST", "PUT", "DELETE", "HEAD", "PATCH", "OPTIONS":
	default:
		return "", fmt.Errorf("unsupported method: %s", method)
	}

	var body io.Reader
	contentType := ""
	warns := []string{}
	jsonObj, hasJSON := args["json"]
	formObj, hasForm := args["form"]
	bodyStr, hasBody := args["body"].(string)
	hasBody = hasBody && bodyStr != ""

	switch {
	case hasJSON:
		if hasForm || hasBody {
			warns = append(warns, "json/form/body 同时传入，按 json 优先")
		}
		buf, err := json.Marshal(jsonObj)
		if err != nil {
			return "", fmt.Errorf("encode json: %w", err)
		}
		body = bytes.NewReader(buf)
		contentType = "application/json"
	case hasForm:
		if hasBody {
			warns = append(warns, "form/body 同时传入，按 form 优先")
		}
		formMap, ok := formObj.(map[string]any)
		if !ok {
			return "", fmt.Errorf("form 必须是 object")
		}
		values := url.Values{}
		for k, v := range formMap {
			values.Set(k, fmt.Sprintf("%v", v))
		}
		body = strings.NewReader(values.Encode())
		contentType = "application/x-www-form-urlencoded"
	case hasBody:
		body = strings.NewReader(bodyStr)
		contentType = "text/plain"
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if hdrs, ok := args["headers"].(map[string]any); ok {
		for k, v := range hdrs {
			req.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 100*1024))
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	var sb strings.Builder
	for _, w := range warns {
		sb.WriteString("[warn] " + w + "\n")
	}
	sb.WriteString(fmt.Sprintf("HTTP %d %s\n", resp.StatusCode, resp.Status))
	for k, v := range resp.Header {
		sb.WriteString(fmt.Sprintf("%s: %s\n", k, strings.Join(v, ", ")))
	}
	sb.WriteString("\n")
	sb.Write(respBody)
	return sb.String(), nil
}

func strFromAny(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
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
