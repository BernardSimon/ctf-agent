package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"
)

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Index    int          `json:"index,omitempty"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolDef struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

type Response struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type Client struct {
	baseURL    string
	apiKey     string
	model      string
	useFC      bool
	httpClient *http.Client
}

func NewClient(baseURL, apiKey, model string, useFC bool) *Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	}
	return &Client{
		baseURL:    normalizeBaseURL(baseURL),
		apiKey:     apiKey,
		model:      model,
		useFC:      useFC,
		httpClient: &http.Client{Transport: transport},
	}
}

func normalizeBaseURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return ""
	}
	if strings.HasSuffix(baseURL, "/v1") {
		return baseURL
	}
	return baseURL + "/v1"
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	Tools    []ToolDef `json:"tools,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
		Delta   struct {
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// Chat 发送非流式请求
func (c *Client) Chat(ctx context.Context, messages []Message, tools []map[string]any) (*Response, error) {
	var toolDefs []ToolDef
	for _, t := range tools {
		fn, _ := t["function"].(map[string]any)
		name, _ := fn["name"].(string)
		desc, _ := fn["description"].(string)
		toolDefs = append(toolDefs, ToolDef{
			Type: "function",
			Function: ToolFunction{
				Name:        name,
				Description: desc,
				Parameters:  fn["parameters"],
			},
		})
	}

	reqBody := chatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   false,
	}
	if c.useFC && len(toolDefs) > 0 {
		reqBody.Tools = toolDefs
	}

	return c.doRequest(ctx, reqBody)
}

// ChatStream 流式请求，通过onChunk回调返回每个文本片段
func (c *Client) ChatStream(ctx context.Context, messages []Message, tools []map[string]any, onChunk func(string)) (*Response, error) {
	var toolDefs []ToolDef
	for _, t := range tools {
		fn, _ := t["function"].(map[string]any)
		name, _ := fn["name"].(string)
		desc, _ := fn["description"].(string)
		toolDefs = append(toolDefs, ToolDef{
			Type: "function",
			Function: ToolFunction{
				Name:        name,
				Description: desc,
				Parameters:  fn["parameters"],
			},
		})
	}

	reqBody := chatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   true,
	}
	if c.useFC && len(toolDefs) > 0 {
		reqBody.Tools = toolDefs
	}

	return c.doStreamRequestWithRetry(ctx, reqBody, onChunk)
}

func (c *Client) doStreamRequestWithRetry(ctx context.Context, reqBody chatRequest, onChunk func(string)) (*Response, error) {
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			if onChunk != nil {
				onChunk(fmt.Sprintf("\n\033[33m[重试 %d/%d]\033[0m\n", attempt, maxRetries-1))
			}
		}

		resp, err := c.doStreamRequest(ctx, reqBody, onChunk)
		if err == nil {
			return resp, nil
		}

		if !shouldRetryError(err) || attempt == maxRetries-1 {
			return nil, err
		}
		lastErr = err
	}
	return nil, lastErr
}

func shouldRetryError(err error) bool {
	if err == nil {
		return false
	}
	// 检查是否为网络错误
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	// 检查是否为 5xx 错误
	if strings.Contains(err.Error(), "LLM API error 5") {
		return true
	}
	// 检查连接重置/EOF
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	return false
}

func (c *Client) doRequest(ctx context.Context, reqBody chatRequest) (*Response, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := c.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM API error %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return &Response{}, nil
	}

	msg := chatResp.Choices[0].Message
	return &Response{
		Content:   msg.Content,
		ToolCalls: msg.ToolCalls,
	}, nil
}

func (c *Client) doStreamRequest(ctx context.Context, reqBody chatRequest, onChunk func(string)) (*Response, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := c.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM API error %d: %s", resp.StatusCode, string(respBody))
	}

	var fullContent strings.Builder
	var toolCalls []ToolCall
	toolCallMap := make(map[int]*ToolCall)

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk chatResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta
		if delta.Content != "" {
			fullContent.WriteString(delta.Content)
			if onChunk != nil {
				onChunk(delta.Content)
			}
		}

		// 处理流式tool_calls
		for _, tc := range delta.ToolCalls {
			idx := tc.Index
			if tc.ID != "" {
				toolCallMap[idx] = &ToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			} else {
				existing, ok := toolCallMap[idx]
				if !ok {
					existing = &ToolCall{Type: "function"}
					toolCallMap[idx] = existing
				}
				existing.Function.Name += tc.Function.Name
				existing.Function.Arguments += tc.Function.Arguments
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read stream: %w", err)
	}

	indexes := make([]int, 0, len(toolCallMap))
	for idx := range toolCallMap {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)
	for _, idx := range indexes {
		tc := toolCallMap[idx]
		if tc.ID == "" {
			tc.ID = fmt.Sprintf("call_%d", idx)
		}
		toolCalls = append(toolCalls, *tc)
	}

	return &Response{
		Content:   fullContent.String(),
		ToolCalls: toolCalls,
	}, nil
}

// CheckConnection 检查LLM服务连接
func (c *Client) CheckConnection(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	url := c.baseURL + "/models"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("LLM服务不可达: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("LLM服务错误: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Model 返回模型名
func (c *Client) Model() string {
	return c.model
}
