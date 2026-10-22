package askgate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// OtherOptionID 是界面始终提供的「其他」选项，模型不必写进 options。
const OtherOptionID = "__other__"

// Option 一个可选答案。
type Option struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Question 一道选择题。
type Question struct {
	ID            string   `json:"id"`
	Prompt        string   `json:"prompt"`
	Options       []Option `json:"options"`
	AllowMultiple bool     `json:"allow_multiple"`
}

// Spec 一次提问。
type Spec struct {
	Title     string     `json:"title"`
	Questions []Question `json:"questions"`
}

// Answer 用户对一题的选择。Selected 为选项 id；选「其他」时包含 OtherOptionID，并在 Text 里给出原文。
type Answer struct {
	QuestionID string   `json:"questionId"`
	Selected   []string `json:"selected"`
	Text       string   `json:"text,optional,omitempty"`
}

// Gate 登记进行中的提问，按 (convID, userID, questionID) 校验提交。
// AskQuestion 发问后通过 Wait 阻塞等待；用户 Submit 或 ctx 取消/超时后结束。
type Gate struct {
	mu      sync.Mutex
	pending map[string]*slot
}

type slot struct {
	conv       uint64
	user       uint64
	toolCallID string
	spec       Spec
	// done 缓冲 1：Submit 可先于 Wait 投递；Wait 取消时不再接收。
	done chan string
}

// New 创建空的提问登记表。
func New() *Gate {
	return &Gate{pending: map[string]*slot{}}
}

// Validate 检查题目结构。
func Validate(spec Spec) error {
	_, err := Normalize(spec)
	return err
}

// Normalize 去掉空白并检查题目结构，返回可存储的副本。
func Normalize(spec Spec) (Spec, error) {
	spec.Title = strings.TrimSpace(spec.Title)
	if len(spec.Questions) == 0 {
		return Spec{}, fmt.Errorf("至少需要一道题")
	}
	seenQ := map[string]struct{}{}
	questions := make([]Question, len(spec.Questions))
	for i, q := range spec.Questions {
		q.ID = strings.TrimSpace(q.ID)
		q.Prompt = strings.TrimSpace(q.Prompt)
		if q.ID == "" || q.Prompt == "" {
			return Spec{}, fmt.Errorf("第 %d 题缺少 id 或 prompt", i+1)
		}
		if _, ok := seenQ[q.ID]; ok {
			return Spec{}, fmt.Errorf("题目 id 重复: %s", q.ID)
		}
		seenQ[q.ID] = struct{}{}
		if len(q.Options) < 2 {
			return Spec{}, fmt.Errorf("题目 %s 至少需要两个选项", q.ID)
		}
		seenOpt := map[string]struct{}{}
		options := make([]Option, len(q.Options))
		for j, opt := range q.Options {
			opt.ID = strings.TrimSpace(opt.ID)
			opt.Label = strings.TrimSpace(opt.Label)
			if opt.ID == "" || opt.Label == "" {
				return Spec{}, fmt.Errorf("题目 %s 的选项缺少 id 或 label", q.ID)
			}
			if opt.ID == OtherOptionID {
				return Spec{}, fmt.Errorf("选项 id %s 为系统保留", OtherOptionID)
			}
			if _, ok := seenOpt[opt.ID]; ok {
				return Spec{}, fmt.Errorf("题目 %s 的选项 id 重复: %s", q.ID, opt.ID)
			}
			seenOpt[opt.ID] = struct{}{}
			options[j] = opt
		}
		q.Options = options
		questions[i] = q
	}
	spec.Questions = questions
	return spec, nil
}

// Open 登记一次提问并返回 questionId。toolCallID 由调用方（AskQuestion 工具）写入，
// 后续 Submit 返回时一并带回，便于上层把答案关联回对应的 tool_call。
func (g *Gate) Open(convID, userID uint64, toolCallID string, spec Spec) (string, error) {
	spec, err := Normalize(spec)
	if err != nil {
		return "", err
	}
	if g == nil {
		return "", fmt.Errorf("提问通道未就绪")
	}
	id := uuid.NewString()
	g.mu.Lock()
	g.pending[id] = &slot{
		conv:       convID,
		user:       userID,
		toolCallID: toolCallID,
		spec:       spec,
		done:       make(chan string, 1),
	}
	g.mu.Unlock()
	return id, nil
}

// Wait 阻塞直到用户 Submit 该提问，或 ctx 取消/超时。
// 超时或取消时清理槽位，迟到的 Submit 会失败。
func (g *Gate) Wait(ctx context.Context, id string) (string, error) {
	if g == nil {
		return "", fmt.Errorf("提问通道未就绪")
	}
	g.mu.Lock()
	s, ok := g.pending[id]
	if !ok {
		g.mu.Unlock()
		return "", fmt.Errorf("提问不存在或已结束")
	}
	ch := s.done
	g.mu.Unlock()

	select {
	case raw := <-ch:
		return raw, nil
	case <-ctx.Done():
		g.mu.Lock()
		if cur, still := g.pending[id]; still && cur == s {
			delete(g.pending, id)
		}
		g.mu.Unlock()
		// Submit 与取消竞态：答案已进 channel 则仍返回答案。
		select {
		case raw := <-ch:
			return raw, nil
		default:
			return "", ctx.Err()
		}
	}
}

// Submit 校验并消费一次提问，返回对应的 toolCallID 与格式化后的答案 JSON。
// 答案不合法时不消费提问，用户可以修正后重试。
// 合法答案会投递给正在 Wait 的调用方（由 AskQuestion 工具在同一轮 Run 内回填）。
func (g *Gate) Submit(convID, userID uint64, id string, answers []Answer) (string, string, error) {
	if g == nil {
		return "", "", fmt.Errorf("提问通道未就绪")
	}
	g.mu.Lock()
	s, ok := g.pending[id]
	if !ok || s.conv != convID || s.user != userID {
		g.mu.Unlock()
		return "", "", fmt.Errorf("提问不存在或已结束")
	}
	spec := s.spec
	toolCallID := s.toolCallID
	ch := s.done
	g.mu.Unlock()

	raw, err := formatAnswers(spec, answers)
	if err != nil {
		return "", "", err
	}
	// 格式化通过后再删除槽位；并发的另一次 Submit 会因为找不到槽位而失败。
	g.mu.Lock()
	if cur, still := g.pending[id]; !still || cur != s {
		g.mu.Unlock()
		return "", "", fmt.Errorf("提问不存在或已结束")
	}
	delete(g.pending, id)
	g.mu.Unlock()
	ch <- raw
	return toolCallID, raw, nil
}

func formatAnswers(spec Spec, answers []Answer) (string, error) {
	byID := map[string]Answer{}
	for _, a := range answers {
		id := strings.TrimSpace(a.QuestionID)
		if id == "" {
			return "", fmt.Errorf("答案缺少 questionId")
		}
		if _, ok := byID[id]; ok {
			return "", fmt.Errorf("题目 %s 重复作答", id)
		}
		a.QuestionID = id
		a.Text = strings.TrimSpace(a.Text)
		byID[id] = a
	}
	out := make([]Answer, 0, len(spec.Questions))
	for _, q := range spec.Questions {
		a, ok := byID[q.ID]
		if !ok {
			return "", fmt.Errorf("题目 %s 未作答", q.ID)
		}
		delete(byID, q.ID)
		selected, err := normalizeSelected(q, a)
		if err != nil {
			return "", err
		}
		item := Answer{QuestionID: q.ID, Selected: selected}
		if contains(selected, OtherOptionID) {
			if a.Text == "" {
				return "", fmt.Errorf("请填写「其他」的内容")
			}
			item.Text = a.Text
		}
		out = append(out, item)
	}
	if len(byID) > 0 {
		return "", fmt.Errorf("包含未知题目")
	}
	b, err := json.Marshal(map[string]any{"answers": out})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func normalizeSelected(q Question, a Answer) ([]string, error) {
	allowed := map[string]struct{}{OtherOptionID: {}}
	for _, opt := range q.Options {
		allowed[strings.TrimSpace(opt.ID)] = struct{}{}
	}
	seen := map[string]struct{}{}
	selected := make([]string, 0, len(a.Selected))
	for _, id := range a.Selected {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := allowed[id]; !ok {
			return nil, fmt.Errorf("题目 %s 的选项不存在: %s", q.ID, id)
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		selected = append(selected, id)
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("题目 %s 未选择选项", q.ID)
	}
	if !q.AllowMultiple && len(selected) != 1 {
		return nil, fmt.Errorf("题目 %s 只能选择一项", q.ID)
	}
	return selected, nil
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
