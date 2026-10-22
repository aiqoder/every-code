package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/repository"
)

var remoteModelsHTTPClient = &http.Client{Timeout: 20 * time.Second}

type openAIModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type anthropicModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// FetchRemoteModels 从供应商配置的 OpenAI Compatible / Anthropic 接口拉取模型 id 列表。
func FetchRemoteModels(ctx context.Context, repo repository.LlmProviderRepository, id uint64) ([]string, error) {
	provider, err := getLlmProviderByID(ctx, repo, id)
	if err != nil {
		return nil, err
	}

	switch {
	case provider.OpenAICompatibleBaseURL != "":
		return fetchOpenAICompatibleModels(ctx, provider.OpenAICompatibleBaseURL, provider.APIKey)
	case provider.AnthropicBaseURL != "":
		if openaiBase, ok := openAIBaseFromAnthropicCompatible(provider.AnthropicBaseURL); ok {
			return fetchOpenAICompatibleModels(ctx, openaiBase, provider.APIKey)
		}
		return fetchAnthropicModels(ctx, provider.AnthropicBaseURL, provider.APIKey)
	default:
		return nil, fmt.Errorf("供应商未配置 OpenAI Compatible 或 Anthropic BaseUrl")
	}
}

func getLlmProviderByID(ctx context.Context, repo repository.LlmProviderRepository, id uint64) (*model.LlmProvider, error) {
	provider, err := repo.GetByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "record not found") {
			return nil, fmt.Errorf("llm provider not found")
		}
		return nil, err
	}
	return provider, nil
}

func openAIBaseFromAnthropicCompatible(anthropicBaseURL string) (string, bool) {
	base := strings.TrimRight(strings.TrimSpace(anthropicBaseURL), "/")
	trimmed, ok := strings.CutSuffix(base, "/anthropic")
	if !ok || trimmed == "" {
		return "", false
	}
	return trimmed, true
}

func fetchOpenAICompatibleModels(ctx context.Context, baseURL, apiKey string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, joinModelsURL(baseURL), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	return doFetchModelIDs(req, parseOpenAIModels)
}

func fetchAnthropicModels(ctx context.Context, baseURL, apiKey string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, joinModelsURL(baseURL), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	if apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
	}
	return doFetchModelIDs(req, parseAnthropicModels)
}

func joinModelsURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(base, "/models") {
		return base
	}
	return base + "/models"
}

func doFetchModelIDs(req *http.Request, parse func([]byte) ([]string, error)) ([]string, error) {
	resp, err := remoteModelsHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求供应商模型列表失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("读取供应商响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 300 {
			msg = msg[:300] + "..."
		}
		return nil, fmt.Errorf("供应商返回 HTTP %d: %s", resp.StatusCode, msg)
	}
	ids, err := parse(body)
	if err != nil {
		return nil, err
	}
	sort.Strings(ids)
	return ids, nil
}

func parseOpenAIModels(body []byte) ([]string, error) {
	var payload openAIModelsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("解析 OpenAI 模型列表失败: %w", err)
	}
	ids := make([]string, 0, len(payload.Data))
	for _, item := range payload.Data {
		ids = append(ids, item.ID)
	}
	return uniqueNonEmptyIDs(ids), nil
}

func parseAnthropicModels(body []byte) ([]string, error) {
	var payload anthropicModelsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("解析 Anthropic 模型列表失败: %w", err)
	}
	ids := make([]string, 0, len(payload.Data))
	for _, item := range payload.Data {
		ids = append(ids, item.ID)
	}
	return uniqueNonEmptyIDs(ids), nil
}

func uniqueNonEmptyIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
