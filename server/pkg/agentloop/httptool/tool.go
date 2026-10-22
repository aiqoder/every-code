package httptool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"chengzi-agent-code-server/pkg/agentloop"
)

var placeholderRe = regexp.MustCompile(`\{\{(\w+)\}\}`)

// HTTPExtra HTTP 类型工具的附加配置。
type HTTPExtra struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// Tool 将 HTTP 请求封装为 agentloop.Tool。
type Tool struct {
	name        string
	displayName string
	description string
	parameters  map[string]any
	extra       HTTPExtra
	client      *http.Client
}

// New 从配置创建 HTTP 工具。
func New(name, displayName, description string, parameters map[string]any, extra HTTPExtra) *Tool {
	if parameters == nil {
		parameters = map[string]any{"type": "object", "properties": map[string]any{}}
	}
	return &Tool{
		name:        name,
		displayName: displayName,
		description: description,
		parameters:  parameters,
		extra:       extra,
		client:      &http.Client{Timeout: 30 * time.Second},
	}
}

func (t *Tool) Name() string { return t.name }

func (t *Tool) Description() string {
	if t.description != "" {
		return t.description
	}
	if t.displayName != "" {
		return t.displayName
	}
	return t.name
}

func (t *Tool) Parameters() map[string]any { return t.parameters }

func (t *Tool) Execute(ctx context.Context, argsJSON string, _ *agentloop.Runtime) (string, error) {
	args := map[string]any{}
	if strings.TrimSpace(argsJSON) != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("解析工具参数失败: %w", err)
		}
	}

	method := strings.ToUpper(strings.TrimSpace(t.extra.Method))
	if method == "" {
		method = http.MethodPost
	}
	url := substitute(t.extra.URL, args)
	if url == "" {
		return "", fmt.Errorf("HTTP 工具 URL 未配置")
	}

	var bodyReader io.Reader
	if t.extra.Body != "" {
		body := substitute(t.extra.Body, args)
		bodyReader = bytes.NewBufferString(body)
	} else if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		payload := map[string]any{
			"name":        t.name,
			"displayName": t.displayName,
			"description": t.description,
			"parameters":  t.parameters,
			"arguments":   args,
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return "", fmt.Errorf("构造请求体失败: %w", err)
		}
		bodyReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return "", fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}
	for k, v := range t.extra.Headers {
		req.Header.Set(k, substitute(v, args))
	}
	if bodyReader != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	const maxBody = 1 << 20 // 1MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return "", fmt.Errorf("读取 HTTP 响应失败: %w", err)
	}

	result := map[string]any{
		"status": resp.StatusCode,
		"body":   string(body),
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 512))
	}
	out, err := json.Marshal(result)
	if err != nil {
		return string(body), nil
	}
	return string(out), nil
}

func substitute(template string, args map[string]any) string {
	return placeholderRe.ReplaceAllStringFunc(template, func(match string) string {
		key := match[2 : len(match)-2]
		if v, ok := args[key]; ok {
			return fmt.Sprint(v)
		}
		return match
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// FromModel 从数据库记录构建 HTTP 工具。
func FromModel(name, displayName, description, parametersJSON, extraJSON string) (agentloop.Tool, error) {
	var parameters map[string]any
	if err := json.Unmarshal([]byte(parametersJSON), &parameters); err != nil {
		return nil, fmt.Errorf("解析 parameters JSON 失败: %w", err)
	}
	var extra HTTPExtra
	if err := json.Unmarshal([]byte(extraJSON), &extra); err != nil {
		return nil, fmt.Errorf("解析 extra JSON 失败: %w", err)
	}
	return New(name, displayName, description, parameters, extra), nil
}
