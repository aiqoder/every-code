package agentloop

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"chengzi-agent-code-server/pkg/llmchat"
)

type fakeChat struct {
	prompts [][]llmchat.Message
	reply   llmchat.AssistantTurn
}

func (f *fakeChat) Protocol() llmchat.Protocol { return llmchat.ProtocolOpenAICompatible }

func (f *fakeChat) ChatStream(_ context.Context, req *llmchat.ChatRequest, onEvent llmchat.EventHandler) (*llmchat.AssistantTurn, error) {
	if isCompactRequest(req) {
		return &llmchat.AssistantTurn{Content: "用户在推进长对话测试。"}, nil
	}
	f.prompts = append(f.prompts, cloneMessages(req.Messages))
	turn := f.reply
	if turn.Content == "" && len(turn.ToolCalls) == 0 {
		turn.Content = "ok"
	}
	if onEvent != nil && turn.Content != "" {
		_ = onEvent(llmchat.StreamEvent{Type: llmchat.StreamEventTextDelta, Text: turn.Content})
	}
	return &turn, nil
}

func TestRunCompactsOversizedHistory(t *testing.T) {
	chat := &fakeChat{}
	history := make([]llmchat.Message, 0, 16)
	for i := 0; i < 8; i++ {
		history = append(history,
			llmchat.Message{Role: llmchat.RoleUser, Content: strings.Repeat("问", 400)},
			llmchat.Message{Role: llmchat.RoleAssistant, Content: strings.Repeat("答", 400)},
		)
	}
	res, err := Run(context.Background(), Config{
		Chat:               chat,
		Model:              "test-model",
		Messages:           history,
		MaxSteps:           1,
		SystemPrompt:       "sys",
		InputContextLimit:  2500,
		OutputContextLimit: 400,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(chat.prompts) != 1 {
		t.Fatalf("prompts=%d", len(chat.prompts))
	}
	sent := EstimateMessages(chat.prompts[0])
	if sent >= 2500 {
		t.Fatalf("sent %d tokens, still over window", sent)
	}
	if res.Usage.Used <= 0 || res.Usage.Used >= 2500 {
		t.Fatalf("usage=%+v", res.Usage)
	}
	// 落库用的完整历史不应被摘要替换
	if len(res.Messages) < len(history) {
		t.Fatalf("canonical history shrank: %d < %d", len(res.Messages), len(history))
	}
}

func TestRunCapsToolResultInTranscript(t *testing.T) {
	chat := &fakeChat{reply: llmchat.AssistantTurn{
		Content: "use tool",
		ToolCalls: []llmchat.ToolCall{{
			ID:        "c1",
			Name:      "boom",
			Arguments: "{}",
		}},
	}}
	huge := strings.Repeat("字", 20000)
	res, err := Run(context.Background(), Config{
		Chat:  chat,
		Model: "test-model",
		Messages: []llmchat.Message{
			{Role: llmchat.RoleUser, Content: "go"},
		},
		Tools:              []Tool{staticTool{name: "boom", result: huge}},
		MaxSteps:           2,
		InputContextLimit:  20_000,
		OutputContextLimit: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range res.Messages {
		if m.Role != llmchat.RoleTool {
			continue
		}
		found = true
		if EstimateTokens(m.Content) > 9000 {
			t.Fatalf("tool result not capped: %d", EstimateTokens(m.Content))
		}
		if !strings.Contains(m.Content, "工具结果已截断") {
			t.Fatalf("missing cap marker")
		}
	}
	if !found {
		t.Fatal("no tool message")
	}
}

type staticTool struct {
	name   string
	result string
}

func (t staticTool) Name() string               { return t.name }
func (t staticTool) Description() string        { return t.name }
func (t staticTool) Parameters() map[string]any { return llmchat.DefaultToolParameters() }
func (t staticTool) Execute(context.Context, string, *Runtime) (string, error) {
	return t.result, nil
}

// ---------- 重试 / 工具失败相关 ----------

// flakyNetErr 模拟瞬时网络错误（实现 net.Error，IsRetryable 判定为可重试）。
type flakyNetErr struct{ msg string }

func (e flakyNetErr) Error() string   { return e.msg }
func (e flakyNetErr) Timeout() bool   { return false }
func (e flakyNetErr) Temporary() bool { return true }

// scriptChat 按调用次序返回预置错误/回复，用于重试与多轮工具场景。
type scriptChat struct {
	calls         int
	errs          []error                 // errs[i] 非 nil 时第 i 次调用直接失败
	turns         []llmchat.AssistantTurn // 第 i 次调用的回复；不足时重复最后一个
	emitBeforeErr bool                    // 失败前先流出对应 turn 的文本（模拟流中途失败）
	reqs          []*llmchat.ChatRequest  // 记录每次主循环调用的请求（压缩请求除外）
}

func (s *scriptChat) Protocol() llmchat.Protocol { return llmchat.ProtocolOpenAICompatible }

func (s *scriptChat) ChatStream(_ context.Context, req *llmchat.ChatRequest, onEvent llmchat.EventHandler) (*llmchat.AssistantTurn, error) {
	if isCompactRequest(req) {
		return &llmchat.AssistantTurn{Content: "摘要"}, nil
	}
	s.reqs = append(s.reqs, req)
	i := s.calls
	s.calls++
	turn := llmchat.AssistantTurn{Content: "ok"}
	if i < len(s.turns) {
		turn = s.turns[i]
	} else if len(s.turns) > 0 {
		turn = s.turns[len(s.turns)-1]
	}
	if i < len(s.errs) && s.errs[i] != nil {
		if s.emitBeforeErr && onEvent != nil && turn.Content != "" {
			_ = onEvent(llmchat.StreamEvent{Type: llmchat.StreamEventTextDelta, Text: turn.Content})
		}
		return nil, s.errs[i]
	}
	if onEvent != nil && turn.Content != "" {
		_ = onEvent(llmchat.StreamEvent{Type: llmchat.StreamEventTextDelta, Text: turn.Content})
	}
	return &turn, nil
}

// stubRetrySleep 把退避睡眠替换成立即返回，避免测试真实等待。
func stubRetrySleep(t *testing.T) {
	t.Helper()
	old := retrySleep
	retrySleep = func(context.Context, time.Duration) error { return nil }
	t.Cleanup(func() { retrySleep = old })
}

// 瞬时错误（未产出内容）应退避重试直至成功，并发出 llm_retrying 事件。
func TestRunRetriesTransientLLMError(t *testing.T) {
	stubRetrySleep(t)
	chat := &scriptChat{
		errs:  []error{flakyNetErr{"connection reset"}, flakyNetErr{"connection reset"}},
		turns: []llmchat.AssistantTurn{{}, {}, {Content: "恢复成功"}},
	}
	retryEvents := 0
	res, err := Run(context.Background(), Config{
		Chat:     chat,
		Model:    "test-model",
		Messages: []llmchat.Message{{Role: llmchat.RoleUser, Content: "hi"}},
		MaxSteps: 1,
		OnEvent: func(ev Event) {
			if ev.Type == EventLLMRetrying {
				retryEvents++
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if chat.calls != 3 {
		t.Fatalf("calls=%d, want 3", chat.calls)
	}
	if retryEvents != 2 {
		t.Fatalf("retry events=%d, want 2", retryEvents)
	}
	if res.FinalContent != "恢复成功" {
		t.Fatalf("content=%q", res.FinalContent)
	}
}

// 非瞬时错误（如 400 参数错误）不重试，直接失败。
func TestRunDoesNotRetryPermanentError(t *testing.T) {
	stubRetrySleep(t)
	chat := &scriptChat{errs: []error{errors.New("400 bad request")}}
	_, err := Run(context.Background(), Config{
		Chat:     chat,
		Model:    "test-model",
		Messages: []llmchat.Message{{Role: llmchat.RoleUser, Content: "hi"}},
		MaxSteps: 1,
	})
	if err == nil {
		t.Fatal("want error")
	}
	if chat.calls != 1 {
		t.Fatalf("calls=%d, want 1 (no retry)", chat.calls)
	}
}

// 重试次数耗尽后返回最后一次错误。
func TestRunRetryExhausted(t *testing.T) {
	stubRetrySleep(t)
	chat := &scriptChat{errs: []error{
		flakyNetErr{"boom"}, flakyNetErr{"boom"}, flakyNetErr{"boom"},
	}}
	_, err := Run(context.Background(), Config{
		Chat:          chat,
		Model:         "test-model",
		Messages:      []llmchat.Message{{Role: llmchat.RoleUser, Content: "hi"}},
		MaxSteps:      1,
		LLMMaxRetries: 2,
	})
	if err == nil {
		t.Fatal("want error after retries exhausted")
	}
	if chat.calls != 3 { // 1 次原始调用 + 2 次重试
		t.Fatalf("calls=%d, want 3", chat.calls)
	}
}

// 流中途失败（已产出部分内容）不重试：保留部分文本并退出，避免前端重复渲染。
func TestRunNoRetryAfterPartialOutput(t *testing.T) {
	stubRetrySleep(t)
	chat := &scriptChat{
		errs:          []error{flakyNetErr{"reset"}},
		turns:         []llmchat.AssistantTurn{{Content: "半截"}, {Content: "第二次"}},
		emitBeforeErr: true,
	}
	res, err := Run(context.Background(), Config{
		Chat:     chat,
		Model:    "test-model",
		Messages: []llmchat.Message{{Role: llmchat.RoleUser, Content: "hi"}},
		MaxSteps: 1,
	})
	if err == nil {
		t.Fatal("want error")
	}
	if chat.calls != 1 {
		t.Fatalf("calls=%d, want 1 (no retry after partial output)", chat.calls)
	}
	if res.FinalContent != "半截" {
		t.Fatalf("partial content=%q", res.FinalContent)
	}
}

// spyTool 记录是否被执行。
type spyTool struct{ executed int }

func (s *spyTool) Name() string               { return "spy" }
func (s *spyTool) Description() string        { return "spy" }
func (s *spyTool) Parameters() map[string]any { return llmchat.DefaultToolParameters() }
func (s *spyTool) Execute(context.Context, string, *Runtime) (string, error) {
	s.executed++
	return "ok", nil
}

// 工具参数 JSON 不合法：不执行工具，错误回填给模型形成修复轮，循环继续。
func TestRunInvalidToolArgsJSON(t *testing.T) {
	spy := &spyTool{}
	chat := &scriptChat{turns: []llmchat.AssistantTurn{
		{ToolCalls: []llmchat.ToolCall{{ID: "c1", Name: "spy", Arguments: "{invalid"}}},
		{Content: "已修正"},
	}}
	res, err := Run(context.Background(), Config{
		Chat:     chat,
		Model:    "test-model",
		Messages: []llmchat.Message{{Role: llmchat.RoleUser, Content: "go"}},
		Tools:    []Tool{spy},
		MaxSteps: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if spy.executed != 0 {
		t.Fatalf("tool executed %d times, want 0", spy.executed)
	}
	found := false
	for _, m := range res.Messages {
		if m.Role == llmchat.RoleTool && strings.Contains(m.Content, "合法 JSON") {
			found = true
		}
	}
	if !found {
		t.Fatal("missing invalid-JSON tool error message")
	}
	if res.FinalContent != "已修正" {
		t.Fatalf("content=%q", res.FinalContent)
	}
}

// blockTool 阻塞直到 ctx 取消，用于验证工具超时。
type blockTool struct{}

func (blockTool) Name() string               { return "block" }
func (blockTool) Description() string        { return "block" }
func (blockTool) Parameters() map[string]any { return llmchat.DefaultToolParameters() }
func (blockTool) Execute(ctx context.Context, _ string, _ *Runtime) (string, error) {
	<-ctx.Done()
	return "", ctx.Err()
}

// 工具执行超时：回填明确的超时错误，整轮不中断，模型继续给出回复。
func TestRunToolTimeout(t *testing.T) {
	chat := &scriptChat{turns: []llmchat.AssistantTurn{
		{ToolCalls: []llmchat.ToolCall{{ID: "c1", Name: "block", Arguments: "{}"}}},
		{Content: "超时后继续"},
	}}
	res, err := Run(context.Background(), Config{
		Chat:        chat,
		Model:       "test-model",
		Messages:    []llmchat.Message{{Role: llmchat.RoleUser, Content: "go"}},
		Tools:       []Tool{blockTool{}},
		MaxSteps:    3,
		ToolTimeout: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range res.Messages {
		if m.Role == llmchat.RoleTool && strings.Contains(m.Content, "超时") {
			found = true
		}
	}
	if !found {
		t.Fatal("missing timeout error in tool message")
	}
	if res.FinalContent != "超时后继续" {
		t.Fatalf("content=%q", res.FinalContent)
	}
}

// ---------- FinishReason / 步数用尽收尾 ----------

// warnRecorder 收集 run_warning 文本与 run_finished 的 Data。
type warnRecorder struct {
	warnings   []string
	finishData map[string]any
}

func (w *warnRecorder) onEvent(ev Event) {
	switch ev.Type {
	case EventRunWarning:
		w.warnings = append(w.warnings, ev.Text)
	case EventRunFinished:
		w.finishData = ev.Data
	}
}

// 输出因长度限制被截断（无工具调用）：保留内容、发出警告、StopReason=length。
func TestRunWarnsOnLengthTruncation(t *testing.T) {
	chat := &scriptChat{turns: []llmchat.AssistantTurn{
		{Content: "半截回复", FinishReason: "length"},
	}}
	rec := &warnRecorder{}
	res, err := Run(context.Background(), Config{
		Chat:     chat,
		Model:    "test-model",
		Messages: []llmchat.Message{{Role: llmchat.RoleUser, Content: "hi"}},
		MaxSteps: 3,
		OnEvent:  rec.onEvent,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.StopReason != "length" {
		t.Fatalf("stopReason=%q, want length", res.StopReason)
	}
	if res.FinalContent != "半截回复" {
		t.Fatalf("content=%q", res.FinalContent)
	}
	if len(rec.warnings) != 1 || !strings.Contains(rec.warnings[0], "截断") {
		t.Fatalf("warnings=%v", rec.warnings)
	}
	if rec.finishData["reason"] != "length" {
		t.Fatalf("run_finished reason=%v", rec.finishData["reason"])
	}
}

// Anthropic 的 max_tokens 结束原因同样归一化为截断处理。
func TestRunWarnsOnAnthropicMaxTokens(t *testing.T) {
	chat := &scriptChat{turns: []llmchat.AssistantTurn{
		{Content: "x", FinishReason: "max_tokens"},
	}}
	rec := &warnRecorder{}
	res, err := Run(context.Background(), Config{
		Chat:     chat,
		Model:    "test-model",
		Messages: []llmchat.Message{{Role: llmchat.RoleUser, Content: "hi"}},
		MaxSteps: 1,
		OnEvent:  rec.onEvent,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.StopReason != "length" {
		t.Fatalf("stopReason=%q, want length", res.StopReason)
	}
	if len(rec.warnings) != 1 {
		t.Fatalf("warnings=%v", rec.warnings)
	}
}

// 内容过滤：不执行工具、清掉悬空 ToolCalls、StopReason=content_filter。
func TestRunContentFilterStops(t *testing.T) {
	spy := &spyTool{}
	chat := &scriptChat{turns: []llmchat.AssistantTurn{{
		Content:      "部分输出",
		FinishReason: "content_filter",
		ToolCalls:    []llmchat.ToolCall{{ID: "c1", Name: "spy", Arguments: "{}"}},
	}}}
	rec := &warnRecorder{}
	res, err := Run(context.Background(), Config{
		Chat:     chat,
		Model:    "test-model",
		Messages: []llmchat.Message{{Role: llmchat.RoleUser, Content: "go"}},
		Tools:    []Tool{spy},
		MaxSteps: 3,
		OnEvent:  rec.onEvent,
	})
	if err != nil {
		t.Fatal(err)
	}
	if spy.executed != 0 {
		t.Fatalf("tool executed %d times, want 0", spy.executed)
	}
	if res.StopReason != "content_filter" {
		t.Fatalf("stopReason=%q", res.StopReason)
	}
	for _, m := range res.Messages {
		if m.Role == llmchat.RoleAssistant && len(m.ToolCalls) > 0 {
			t.Fatal("dangling tool calls left in history")
		}
	}
	if len(rec.warnings) != 1 || !strings.Contains(rec.warnings[0], "拦截") {
		t.Fatalf("warnings=%v", rec.warnings)
	}
	if rec.finishData["reason"] != "content_filter" {
		t.Fatalf("run_finished reason=%v", rec.finishData["reason"])
	}
}

// 步数用尽：发出警告、额外做一次无工具收尾调用、总结写入历史、reason=max_steps。
func TestRunMaxStepsWrapUp(t *testing.T) {
	chat := &scriptChat{turns: []llmchat.AssistantTurn{
		{Content: "第一步", ToolCalls: []llmchat.ToolCall{{ID: "c1", Name: "spy", Arguments: "{}"}}},
		{Content: "第二步", ToolCalls: []llmchat.ToolCall{{ID: "c2", Name: "spy", Arguments: "{}"}}},
		{Content: "阶段性总结"},
	}}
	rec := &warnRecorder{}
	res, err := Run(context.Background(), Config{
		Chat:     chat,
		Model:    "test-model",
		Messages: []llmchat.Message{{Role: llmchat.RoleUser, Content: "go"}},
		Tools:    []Tool{&spyTool{}},
		MaxSteps: 2,
		OnEvent:  rec.onEvent,
	})
	if err != nil {
		t.Fatal(err)
	}
	if chat.calls != 3 { // 2 步主循环 + 1 次收尾
		t.Fatalf("calls=%d, want 3", chat.calls)
	}
	// 收尾调用不带工具，并注入了步数用尽的系统提示。
	wrapReq := chat.reqs[2]
	if len(wrapReq.Tools) != 0 {
		t.Fatalf("wrap-up request should have no tools, got %d", len(wrapReq.Tools))
	}
	nudgeFound := false
	for _, m := range wrapReq.Messages {
		if m.Role == llmchat.RoleSystem && strings.Contains(m.Content, "步数预算已用尽") {
			nudgeFound = true
		}
	}
	if !nudgeFound {
		t.Fatal("wrap-up request missing max-steps nudge")
	}
	if res.StopReason != "max_steps" {
		t.Fatalf("stopReason=%q", res.StopReason)
	}
	if res.FinalContent != "阶段性总结" {
		t.Fatalf("content=%q", res.FinalContent)
	}
	// 总结写入历史且不带悬空 ToolCalls。
	last := res.Messages[len(res.Messages)-1]
	if last.Role != llmchat.RoleAssistant || last.Content != "阶段性总结" || len(last.ToolCalls) > 0 {
		t.Fatalf("last message=%+v", last)
	}
	if len(rec.warnings) == 0 || !strings.Contains(rec.warnings[0], "最大步数") {
		t.Fatalf("warnings=%v", rec.warnings)
	}
	if rec.finishData["reason"] != "max_steps" {
		t.Fatalf("run_finished reason=%v", rec.finishData["reason"])
	}
}

// 收尾调用失败：保留已有内容，reason 仍为 max_steps，并发出收尾失败警告。
func TestRunMaxStepsWrapUpFailure(t *testing.T) {
	chat := &scriptChat{
		errs: []error{nil, nil, errors.New("wrap up boom")},
		turns: []llmchat.AssistantTurn{
			{Content: "第一步", ToolCalls: []llmchat.ToolCall{{ID: "c1", Name: "spy", Arguments: "{}"}}},
			{Content: "第二步", ToolCalls: []llmchat.ToolCall{{ID: "c2", Name: "spy", Arguments: "{}"}}},
		},
	}
	rec := &warnRecorder{}
	res, err := Run(context.Background(), Config{
		Chat:     chat,
		Model:    "test-model",
		Messages: []llmchat.Message{{Role: llmchat.RoleUser, Content: "go"}},
		Tools:    []Tool{&spyTool{}},
		MaxSteps: 2,
		OnEvent:  rec.onEvent,
	})
	if err != nil {
		t.Fatal(err)
	}
	if chat.calls != 3 {
		t.Fatalf("calls=%d, want 3", chat.calls)
	}
	if res.StopReason != "max_steps" {
		t.Fatalf("stopReason=%q", res.StopReason)
	}
	if res.FinalContent != "第二步" {
		t.Fatalf("content=%q", res.FinalContent)
	}
	found := false
	for _, w := range rec.warnings {
		if strings.Contains(w, "收尾总结调用失败") {
			found = true
		}
	}
	if !found {
		t.Fatalf("warnings=%v", rec.warnings)
	}
}

// ---------- 卡住熔断（重复调用 / 空轮次） ----------

type namedSpy struct {
	name     string
	executed int
}

func (s *namedSpy) Name() string               { return s.name }
func (s *namedSpy) Description() string        { return s.name }
func (s *namedSpy) Parameters() map[string]any { return llmchat.DefaultToolParameters() }
func (s *namedSpy) Execute(context.Context, string, *Runtime) (string, error) {
	s.executed++
	return "ok", nil
}

// 同一工具同一参数连续 3 次：前 2 次执行，第 3 次熔断不执行，收尾后 reason=stuck。
func TestRunRepeatToolCallStuck(t *testing.T) {
	spy := &namedSpy{name: "read"}
	chat := &scriptChat{turns: []llmchat.AssistantTurn{
		{Content: "先读", ToolCalls: []llmchat.ToolCall{{ID: "c1", Name: "read", Arguments: `{"path":"a"}`}}},
		{Content: "再读", ToolCalls: []llmchat.ToolCall{{ID: "c2", Name: "read", Arguments: `{"path":"a"}`}}},
		{Content: "又读", ToolCalls: []llmchat.ToolCall{{ID: "c3", Name: "read", Arguments: `{"path":"a"}`}}},
		{Content: "卡住后的总结"},
	}}
	rec := &warnRecorder{}
	res, err := Run(context.Background(), Config{
		Chat:             chat,
		Model:            "test-model",
		Messages:         []llmchat.Message{{Role: llmchat.RoleUser, Content: "go"}},
		Tools:            []Tool{spy},
		MaxSteps:         10,
		StuckRepeatLimit: 3,
		OnEvent:          rec.onEvent,
	})
	if err != nil {
		t.Fatal(err)
	}
	if spy.executed != 2 {
		t.Fatalf("executed=%d, want 2 (third skipped)", spy.executed)
	}
	if chat.calls != 4 { // 3 步主循环 + 1 次收尾
		t.Fatalf("calls=%d, want 4", chat.calls)
	}
	if res.StopReason != "stuck" {
		t.Fatalf("stopReason=%q", res.StopReason)
	}
	if rec.finishData["reason"] != "stuck" {
		t.Fatalf("run_finished reason=%v", rec.finishData["reason"])
	}
	if res.FinalContent != "卡住后的总结" {
		t.Fatalf("content=%q", res.FinalContent)
	}
	found := false
	for _, w := range rec.warnings {
		if strings.Contains(w, "反复调用") {
			found = true
		}
	}
	if !found {
		t.Fatalf("warnings=%v", rec.warnings)
	}
	// 第 3 次调用应回填熔断错误，避免悬空 tool_calls。
	errFilled := false
	for _, m := range res.Messages {
		if m.Role == llmchat.RoleTool && strings.Contains(m.Content, "已中止后续工具执行") {
			errFilled = true
		}
	}
	if !errFilled {
		t.Fatal("missing stuck error tool result")
	}
	wrapReq := chat.reqs[3]
	if len(wrapReq.Tools) != 0 {
		t.Fatalf("wrap-up should have no tools, got %d", len(wrapReq.Tools))
	}
}

// 参数 JSON 键顺序不同仍视为同一调用。
func TestRunRepeatToolCallIgnoresArgKeyOrder(t *testing.T) {
	spy := &namedSpy{name: "read"}
	chat := &scriptChat{turns: []llmchat.AssistantTurn{
		{ToolCalls: []llmchat.ToolCall{{ID: "c1", Name: "read", Arguments: `{"path":"a","q":1}`}}},
		{ToolCalls: []llmchat.ToolCall{{ID: "c2", Name: "read", Arguments: `{"q":1,"path":"a"}`}}},
		{ToolCalls: []llmchat.ToolCall{{ID: "c3", Name: "read", Arguments: `{"path":"a","q":1}`}}},
		{Content: "总结"},
	}}
	res, err := Run(context.Background(), Config{
		Chat:             chat,
		Model:            "test-model",
		Messages:         []llmchat.Message{{Role: llmchat.RoleUser, Content: "go"}},
		Tools:            []Tool{spy},
		MaxSteps:         10,
		StuckRepeatLimit: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if spy.executed != 2 {
		t.Fatalf("executed=%d, want 2", spy.executed)
	}
	if res.StopReason != "stuck" {
		t.Fatalf("stopReason=%q", res.StopReason)
	}
}

// 单步内连续 3 次相同调用：执行 2 次后熔断，不再进入下一步。
func TestRunRepeatToolCallInSameStep(t *testing.T) {
	spy := &namedSpy{name: "read"}
	chat := &scriptChat{turns: []llmchat.AssistantTurn{
		{ToolCalls: []llmchat.ToolCall{
			{ID: "c1", Name: "read", Arguments: `{"path":"a"}`},
			{ID: "c2", Name: "read", Arguments: `{"path":"a"}`},
			{ID: "c3", Name: "read", Arguments: `{"path":"a"}`},
		}},
		{Content: "总结"},
	}}
	res, err := Run(context.Background(), Config{
		Chat:             chat,
		Model:            "test-model",
		Messages:         []llmchat.Message{{Role: llmchat.RoleUser, Content: "go"}},
		Tools:            []Tool{spy},
		MaxSteps:         10,
		StuckRepeatLimit: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if spy.executed != 2 {
		t.Fatalf("executed=%d, want 2", spy.executed)
	}
	if chat.calls != 2 { // 1 步主循环 + 收尾
		t.Fatalf("calls=%d, want 2", chat.calls)
	}
	if res.StopReason != "stuck" {
		t.Fatalf("stopReason=%q", res.StopReason)
	}
}

// 连续 3 轮无正文且工具组合不变（两种不同工具交替，重复熔断不会触发）：空轮次熔断。
func TestRunEmptyTurnStuck(t *testing.T) {
	read := &namedSpy{name: "read"}
	grep := &namedSpy{name: "grep"}
	emptyBoth := llmchat.AssistantTurn{ToolCalls: []llmchat.ToolCall{
		{ID: "r", Name: "read", Arguments: `{"path":"a"}`},
		{ID: "g", Name: "grep", Arguments: `{"q":"x"}`},
	}}
	chat := &scriptChat{turns: []llmchat.AssistantTurn{
		emptyBoth,
		{ToolCalls: []llmchat.ToolCall{ // 顺序对调，组合相同
			{ID: "g2", Name: "grep", Arguments: `{"q":"x"}`},
			{ID: "r2", Name: "read", Arguments: `{"path":"a"}`},
		}},
		emptyBoth,
		{Content: "空轮总结"},
	}}
	rec := &warnRecorder{}
	res, err := Run(context.Background(), Config{
		Chat:            chat,
		Model:           "test-model",
		Messages:        []llmchat.Message{{Role: llmchat.RoleUser, Content: "go"}},
		Tools:           []Tool{read, grep},
		MaxSteps:        10,
		StuckEmptyLimit: 3,
		OnEvent:         rec.onEvent,
	})
	if err != nil {
		t.Fatal(err)
	}
	if read.executed != 2 || grep.executed != 2 {
		t.Fatalf("read=%d grep=%d, want 2/2 (third step skipped)", read.executed, grep.executed)
	}
	if res.StopReason != "stuck" {
		t.Fatalf("stopReason=%q", res.StopReason)
	}
	found := false
	for _, w := range rec.warnings {
		if strings.Contains(w, "无正文") {
			found = true
		}
	}
	if !found {
		t.Fatalf("warnings=%v", rec.warnings)
	}
	if res.FinalContent != "空轮总结" {
		t.Fatalf("content=%q", res.FinalContent)
	}
}

// 空正文但每轮换工具参数：视为在推进，不应熔断。
func TestRunEmptyTurnWithChangingToolsContinues(t *testing.T) {
	spy := &namedSpy{name: "read"}
	chat := &scriptChat{turns: []llmchat.AssistantTurn{
		{ToolCalls: []llmchat.ToolCall{{ID: "c1", Name: "read", Arguments: `{"path":"a"}`}}},
		{ToolCalls: []llmchat.ToolCall{{ID: "c2", Name: "read", Arguments: `{"path":"b"}`}}},
		{ToolCalls: []llmchat.ToolCall{{ID: "c3", Name: "read", Arguments: `{"path":"c"}`}}},
		{Content: "读完了"},
	}}
	res, err := Run(context.Background(), Config{
		Chat:     chat,
		Model:    "test-model",
		Messages: []llmchat.Message{{Role: llmchat.RoleUser, Content: "go"}},
		Tools:    []Tool{spy},
		MaxSteps: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if spy.executed != 3 {
		t.Fatalf("executed=%d, want 3", spy.executed)
	}
	if res.StopReason != "completed" {
		t.Fatalf("stopReason=%q, want completed", res.StopReason)
	}
}

type reasoningChat struct {
	calls int
}

func (r *reasoningChat) Protocol() llmchat.Protocol { return llmchat.ProtocolOpenAICompatible }

func (r *reasoningChat) ChatStream(_ context.Context, req *llmchat.ChatRequest, onEvent llmchat.EventHandler) (*llmchat.AssistantTurn, error) {
	if isCompactRequest(req) {
		return &llmchat.AssistantTurn{Content: "摘要"}, nil
	}
	r.calls++
	if len(req.Tools) == 0 {
		if onEvent != nil {
			_ = onEvent(llmchat.StreamEvent{Type: llmchat.StreamEventTextDelta, Text: "推理后总结"})
		}
		return &llmchat.AssistantTurn{Content: "推理后总结"}, nil
	}
	if onEvent != nil {
		_ = onEvent(llmchat.StreamEvent{Type: llmchat.StreamEventReasoningDelta, Text: "思考中"})
	}
	return &llmchat.AssistantTurn{
		ToolCalls: []llmchat.ToolCall{{ID: "c1", Name: "read", Arguments: `{}`}},
	}, nil
}

// 只有推理、没有正文的工具轮不计入空轮次（思考模型的常见形态）。
func TestRunReasoningOnlyTurnNotEmptyStuck(t *testing.T) {
	spy := &namedSpy{name: "read"}
	chat := &reasoningChat{}
	res, err := Run(context.Background(), Config{
		Chat:             chat,
		Model:            "test-model",
		Messages:         []llmchat.Message{{Role: llmchat.RoleUser, Content: "go"}},
		Tools:            []Tool{spy},
		MaxSteps:         4,
		StuckRepeatLimit: -1, // 本用例只验证空轮次不误伤思考模型
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.StopReason != "max_steps" {
		t.Fatalf("stopReason=%q, want max_steps (not stuck)", res.StopReason)
	}
	if spy.executed != 4 {
		t.Fatalf("executed=%d, want 4", spy.executed)
	}
}

// 显式禁用熔断后，相同调用会一路跑到步数上限。
func TestRunStuckDisabled(t *testing.T) {
	spy := &namedSpy{name: "read"}
	chat := &scriptChat{turns: []llmchat.AssistantTurn{
		{ToolCalls: []llmchat.ToolCall{{ID: "c1", Name: "read", Arguments: `{}`}}},
	}}
	res, err := Run(context.Background(), Config{
		Chat:             chat,
		Model:            "test-model",
		Messages:         []llmchat.Message{{Role: llmchat.RoleUser, Content: "go"}},
		Tools:            []Tool{spy},
		MaxSteps:         4,
		StuckRepeatLimit: -1,
		StuckEmptyLimit:  -1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.StopReason != "max_steps" {
		t.Fatalf("stopReason=%q, want max_steps", res.StopReason)
	}
	if spy.executed != 4 {
		t.Fatalf("executed=%d, want 4", spy.executed)
	}
}
