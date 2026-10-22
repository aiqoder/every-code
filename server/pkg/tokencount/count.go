// Package tokencount 用 tiktoken cl100k_base 精确计 token。
// 消息开销取 OpenAI chatml 惯例（每条 +3，回复预填充 +3）。
// 工具定义按字段累加，避免 encoding/json 把非 ASCII 编成 \uXXXX。
package tokencount

import (
	"encoding/json"
	"strconv"
	"sync"

	"chengzi-agent-code-server/pkg/llmchat"

	"github.com/tiktoken-go/tokenizer"
)

const (
	Encoding            = "cl100k_base"
	TokensPerMessage    = 3
	TokensPerName       = 1
	TokensPerToolCall   = 3
	ReplyPriming        = 3
	TokensPerToolDef    = 7
	ToolsPromptOverhead = 3
)

var codecPool = sync.Pool{
	New: func() any {
		enc, err := tokenizer.Get(tokenizer.Cl100kBase)
		if err != nil {
			panic(err)
		}
		return enc
	},
}

func withCodec(fn func(tokenizer.Codec) error) {
	c := codecPool.Get().(tokenizer.Codec)
	defer codecPool.Put(c)
	_ = fn(c)
}

// Count 返回 text 的 cl100k_base token 数。
func Count(text string) int {
	if text == "" {
		return 0
	}
	var n int
	withCodec(func(c tokenizer.Codec) error {
		v, err := c.Count(text)
		if err != nil {
			ids, _, e2 := c.Encode(text)
			if e2 != nil {
				return e2
			}
			n = len(ids)
			return nil
		}
		n = v
		return nil
	})
	return n
}

// CountMessage 单条消息（角色、正文、tool_calls），不含回复预填充。
func CountMessage(m llmchat.Message) int {
	n := TokensPerMessage + Count(string(m.Role)) + Count(m.Content)
	if m.Name != "" {
		n += Count(m.Name) + TokensPerName
	}
	n += Count(m.ToolCallID)
	for _, p := range m.Parts {
		switch p.Kind {
		case llmchat.PartImage:
			// 视觉 token 无法从 base64 估算，按一张高清图的量级计入，避免把编码正文算进上下文。
			n += 1100
		case llmchat.PartPDF:
			n += 2000
		default:
			n += Count(p.Data)
		}
	}
	for _, tc := range m.ToolCalls {
		n += TokensPerToolCall + Count(tc.ID) + Count(tc.Name) + Count(tc.Arguments)
	}
	return n
}

// CountMessages 消息列表，含 assistant 回复预填充。
func CountMessages(msgs []llmchat.Message) int {
	n := 0
	for _, m := range msgs {
		n += CountMessage(m)
	}
	if len(msgs) > 0 {
		n += ReplyPriming
	}
	return n
}

// CountTools 工具定义占用（与前端 countTools 对齐）。
func CountTools(tools []llmchat.ToolDef) int {
	if len(tools) == 0 {
		return 0
	}
	n := ToolsPromptOverhead
	for _, t := range tools {
		n += TokensPerToolDef + Count(t.Name) + Count(t.Description)
		params := t.Parameters
		if len(params) == 0 {
			params = llmchat.DefaultToolParameters()
		}
		n += countValue(params)
	}
	return n
}

// CountPrompt 即将送给模型的 prompt：消息 + 工具定义。
func CountPrompt(msgs []llmchat.Message, toolsTok int) int {
	return CountMessages(msgs) + toolsTok
}

// Prefix 按 token 取前缀；截断发生在 token 边界。
func Prefix(s string, max int) string {
	if max <= 0 || s == "" {
		return ""
	}
	out := s
	withCodec(func(c tokenizer.Codec) error {
		ids, _, err := c.Encode(s)
		if err != nil || len(ids) <= max {
			return err
		}
		text, err := c.Decode(ids[:max])
		if err != nil {
			return err
		}
		out = text
		return nil
	})
	return out
}

// Suffix 按 token 取后缀；截断发生在 token 边界。
func Suffix(s string, max int) string {
	if max <= 0 || s == "" {
		return ""
	}
	out := s
	withCodec(func(c tokenizer.Codec) error {
		ids, _, err := c.Encode(s)
		if err != nil || len(ids) <= max {
			return err
		}
		text, err := c.Decode(ids[len(ids)-max:])
		if err != nil {
			return err
		}
		out = text
		return nil
	})
	return out
}

func countValue(v any) int {
	switch x := v.(type) {
	case nil:
		return Count("null")
	case string:
		return Count(x)
	case bool:
		if x {
			return Count("true")
		}
		return Count("false")
	case float64:
		return Count(strconv.FormatFloat(x, 'f', -1, 64))
	case float32:
		return Count(strconv.FormatFloat(float64(x), 'f', -1, 32))
	case int:
		return Count(strconv.Itoa(x))
	case int64:
		return Count(strconv.FormatInt(x, 10))
	case json.Number:
		return Count(string(x))
	case map[string]any:
		n := 0
		for k, val := range x {
			n += Count(k) + countValue(val)
		}
		return n
	case []any:
		n := 0
		for _, val := range x {
			n += countValue(val)
		}
		return n
	case []string:
		n := 0
		for _, s := range x {
			n += Count(s)
		}
		return n
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return 0
		}
		return Count(string(b))
	}
}
