package agentloop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"chengzi-agent-code-server/pkg/llmchat"
)

// Config Agent Loop 配置。
type Config struct {
	Chat         llmchat.Chat
	Model        string
	Messages     []llmchat.Message
	Tools        []Tool
	MaxSteps     int
	OnEvent      func(Event)
	Depth        int
	MaxDepth     int
	Shared       map[string]any
	SystemPrompt string
	// InputContextLimit 模型输入窗口（token）。>0 时启用超窗压缩。
	InputContextLimit int64
	// OutputContextLimit 预留给模型输出的 token。0 则按窗口自动预留。
	OutputContextLimit int64
	Reasoning          bool
	ReasoningEffort    string
	// LLMMaxRetries LLM 流式调用失败后的最大重试次数（不含首次）。
	// 0 用默认值 3；<0 禁用重试。
	// 仅对「尚未产出任何内容」的瞬时失败（429、5xx、网络错误）重试；
	// 流中途失败不重试，避免已输出的文本在前端重复渲染。
	LLMMaxRetries int
	// ToolTimeout 单个工具的执行超时。<=0 用默认值 2 分钟。
	// 超时不中断整轮：以错误 JSON 回填，由模型决定后续动作。
	ToolTimeout time.Duration
	// StuckRepeatLimit 同一工具+同一参数连续调用达到该次数则熔断。
	// 0 用默认值 10；<0 禁用。
	StuckRepeatLimit int
	// StuckEmptyLimit 连续「无正文且工具组合无变化」的轮次达到该次数则熔断。
	// 0 用默认值 10；<0 禁用。仅推理、无正文的轮次不计入空轮。
	StuckEmptyLimit int
	// Ask 登记一次向用户的提问，返回 questionId。nil 表示当前运行不能提问。
	Ask func(ctx context.Context, toolCallID, specJSON string) (questionID string, err error)
	// AskWait 阻塞等待用户提交答案。与 Ask 成对注入；超时语义由工具侧处理。
	AskWait func(ctx context.Context, questionID string) (answerJSON string, err error)
}

// Result 循环运行结果。
type Result struct {
	Messages     []llmchat.Message
	FinalContent string
	Steps        int
	Usage        UsageSnapshot
	// StopReason 循环结束原因：completed（模型正常收尾）、max_steps（步数用尽）、
	// length（输出被长度截断）、content_filter（内容过滤）、stuck（重复调用/空轮次熔断）。
	StopReason string
}

// Run 执行一轮完整的 agent loop。
//
// 核心流程（每步 step）：
//  1. 上下文准备（超窗则压缩/截断）→ 上报用量事件
//  2. 流式调用 LLM，实时转发 text / reasoning delta
//  3. 将 assistant 回复写入历史；若无 tool_calls 则结束循环
//  4. 逐个执行工具，结果回填为 tool 消息，进入下一步
//
// 终止条件：模型不再请求工具、达到 MaxSteps、卡住熔断、ctx 取消，或准备/流式调用出错。
// 出错或取消时尽量保留已生成的部分文本，便于上层展示与续聊。
//
// 结束原因（Result.StopReason / run_finished 事件的 data.reason）：
//   - completed：模型正常收尾；
//   - length：最终答复被长度截断（发出 run_warning 提示）；
//   - content_filter：输出被内容过滤拦截（半截工具调用不执行）；
//   - max_steps：步数用尽，额外做一次无工具收尾调用，让模型总结后再结束。
//   - stuck：同一工具反复调用或连续空轮次被熔断，同样做一次无工具收尾。
func Run(ctx context.Context, cfg Config) (*Result, error) {
	// ---------- 参数校验与默认值 ----------
	if cfg.Chat == nil {
		return nil, fmt.Errorf("agentloop: chat client is required")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("agentloop: model is required")
	}
	// MaxSteps：单次 Run 最多「模型调用」次数，防止工具循环失控。
	if cfg.MaxSteps <= 0 {
		cfg.MaxSteps = 12
	}
	// MaxDepth：子 agent（如 Task 工具）嵌套上限，由 Runtime 传给工具侧检查。
	if cfg.MaxDepth <= 0 {
		cfg.MaxDepth = 2
	}
	// LLMMaxRetries：LLM 瞬时失败的重试次数上限；<0 表示显式禁用。
	maxRetries := cfg.LLMMaxRetries
	if maxRetries == 0 {
		maxRetries = 5
	} else if maxRetries < 0 {
		maxRetries = 0
	}
	// ToolTimeout：单工具执行超时，防止卡住的工具拖死整轮。
	toolTimeout := cfg.ToolTimeout
	if toolTimeout <= 0 {
		toolTimeout = 2 * time.Minute
	}
	// 卡住熔断：重复调用 / 无进展空轮次。阈值 0 用默认值，<0 禁用。
	tracker := newStuckTracker(
		resolveStuckLimit(cfg.StuckRepeatLimit, defaultStuckRepeatLimit),
		resolveStuckLimit(cfg.StuckEmptyLimit, defaultStuckEmptyLimit),
	)
	// Shared：跨工具/子循环共享的可变状态（如工作区路径），保证非 nil 便于写入。
	if cfg.Shared == nil {
		cfg.Shared = map[string]any{}
	}
	// emit：对外事件回调；未配置时用空函数，后续无需判空。
	emit := cfg.OnEvent
	if emit == nil {
		emit = func(Event) {}
	}

	// ---------- 组装初始消息列表 ----------
	// 预留少量容量给本轮可能追加的 assistant / tool 消息。
	messages := make([]llmchat.Message, 0, len(cfg.Messages)+4)
	// SystemPrompt 单独前置为 system 消息，与调用方传入的历史区分开。
	if cfg.SystemPrompt != "" {
		messages = append(messages, llmchat.Message{Role: llmchat.RoleSystem, Content: cfg.SystemPrompt})
	}
	messages = append(messages, cfg.Messages...)

	emit(Event{Type: EventRunStarted, Data: map[string]any{"depth": cfg.Depth}})

	// ---------- 工具运行时与上下文管理器 ----------
	// Runtime 注入到 Tool.Execute，使工具能发事件、调模型、感知嵌套深度与窗口限制。
	rt := &Runtime{
		Emit:               emit,
		Depth:              cfg.Depth,
		MaxDepth:           cfg.MaxDepth,
		Chat:               cfg.Chat,
		Model:              cfg.Model,
		Tools:              cfg.Tools,
		Shared:             cfg.Shared,
		InputContextLimit:  cfg.InputContextLimit,
		OutputContextLimit: cfg.OutputContextLimit,
		Reasoning:          cfg.Reasoning,
		ReasoningEffort:    cfg.ReasoningEffort,
		Ask:                cfg.Ask,
		AskWait:            cfg.AskWait,
	}

	// toolDefs：发给模型的工具 schema；与 cfg.Tools 一一对应。
	toolDefs := ToolDefs(cfg.Tools)
	// contextManager：按 InputContextLimit 做用量统计与超窗压缩；
	// summarizer 复用同一 Chat/Model，在需要时用 LLM 摘要旧轮次。
	mgr := newContextManager(ContextPolicy{
		InputLimit:    cfg.InputContextLimit,
		OutputReserve: cfg.OutputContextLimit,
	}, toolDefs, newLLMSummarizer(cfg.Chat, cfg.Model))
	// maxToolTok：单条工具结果写入历史时的 token 上限，避免一次工具输出撑爆窗口。
	maxToolTok := mgr.policy.maxToolResultTokens()

	var finalContent string // 最近一步 assistant 正文（或出错前的部分文本）
	steps := 0              // 实际执行的步数（从 1 计）
	var lastUsage UsageSnapshot
	// stopReason 记录循环结束原因；默认 max_steps（for 条件耗尽而未 break），
	// 正常 break 出口会覆盖为 completed / length / content_filter / stuck。
	stopReason := "max_steps"

	// ---------- 主循环：模型 ↔ 工具 ----------
	for step := 0; step < cfg.MaxSteps; step++ {
		steps = step + 1
		// stepText 累积本步流式 text delta，用于：
		// - 流中途失败时回填已输出内容；
		// - turn.Content 为空时的兜底正文。
		var stepText strings.Builder

		// Prepare：克隆 messages 后做用量统计与超窗压缩，并算出本步 MaxOutput。
		// prepared.Messages 只作为本次 ChatStream 的输入视图；
		// 本地 messages 不被回写，完整历史仍用于后续步追加与 Result 返回。
		prepared, prepErr := mgr.Prepare(ctx, messages)
		if prepErr != nil {
			// 用户取消不发 Error 事件，避免前端把取消当成故障。
			if !errors.Is(prepErr, context.Canceled) {
				emit(Event{Type: EventError, Error: prepErr.Error()})
			}
			// 此时尚未开始流式输出，stepText 通常为空；保留分支以统一错误返回形态。
			partial := stepText.String()
			if partial != "" {
				messages = append(messages, llmchat.Message{Role: llmchat.RoleAssistant, Content: partial})
				finalContent = partial
			}
			return &Result{
				Messages:     messages,
				FinalContent: finalContent,
				Steps:        steps,
				Usage:        lastUsage,
			}, prepErr
		}
		lastUsage = prepared.Usage
		// 向前端上报用量分桶；若本步发生了压缩，一并发出 compacted 事件。
		emitContextEvents(emit, prepared)

		req := &llmchat.ChatRequest{
			Model:           cfg.Model,
			Messages:        prepared.Messages, // 使用压缩/裁剪后的视图，而非原始 messages
			Tools:           toolDefs,
			Reasoning:       cfg.Reasoning,
			ReasoningEffort: cfg.ReasoningEffort,
		}
		// MaxOutput 由窗口策略算出；>0 时限制模型输出长度，给输入侧留足预算。
		if prepared.MaxOutput > 0 {
			req.MaxTokens = prepared.MaxOutput
		}

		// 流式调用：边收边 emit，保证 UI 可实时渲染。
		// emitted 标记本步是否已向外产出内容；一旦产出过 delta，
		// 失败就不再重试（重试会从头生成，导致前端文本重复渲染）。
		emitted := false
		hasReasoning := false
		onStream := func(ev llmchat.StreamEvent) error {
			switch ev.Type {
			case llmchat.StreamEventTextDelta:
				emitted = true
				stepText.WriteString(ev.Text)
				emit(Event{Type: EventTextDelta, Text: ev.Text})
			case llmchat.StreamEventReasoningDelta:
				// 推理内容只转发事件，不写入 messages / finalContent。
				emitted = true
				hasReasoning = true
				emit(Event{Type: EventReasoningDelta, Text: ev.Text})
			}
			return nil
		}
		turn, err := cfg.Chat.ChatStream(ctx, req, onStream)
		// 瞬时失败（429 / 5xx / 网络抖动）按指数退避重试。
		// ctx.Err() != nil（用户取消或整体超时）时不重试，直接退出。
		for attempt := 1; err != nil && attempt <= maxRetries &&
			!emitted && ctx.Err() == nil && llmchat.IsRetryable(err); attempt++ {
			delay := retryDelay(attempt)
			emit(Event{Type: EventLLMRetrying, Error: err.Error(), Data: map[string]any{
				"attempt":      attempt,
				"max_attempts": maxRetries,
				"delay_ms":     delay.Milliseconds(),
			}})
			if sleepErr := retrySleep(ctx, delay); sleepErr != nil {
				err = sleepErr
				break
			}
			turn, err = cfg.Chat.ChatStream(ctx, req, onStream)
		}
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				emit(Event{Type: EventError, Error: err.Error()})
			}
			// 流式中断：把已累积的正文落盘到历史，避免用户已看到的内容丢失。
			partial := stepText.String()
			if partial != "" {
				messages = append(messages, llmchat.Message{Role: llmchat.RoleAssistant, Content: partial})
				finalContent = partial
			}
			return &Result{
				Messages:     messages,
				FinalContent: finalContent,
				Steps:        steps,
				Usage:        lastUsage,
			}, err
		}

		// 优先用完整 turn 的 Content；部分实现可能只在流里出字，此时用 stepText 兜底。
		content := turn.Content
		if content == "" {
			content = stepText.String()
		}
		finalContent = content
		// FinishReason 归一化：length（截断）/ content_filter（过滤）需要显式处理，
		// 避免把截断回复或半截 tool call 当成正常一轮。
		finishReason := normalizeFinishReason(turn.FinishReason)

		// 无论是否带工具调用，先把本步 assistant 消息写入历史（含 ToolCalls 元数据）。
		asst := llmchat.Message{
			Role:      llmchat.RoleAssistant,
			Content:   content,
			ToolCalls: turn.ToolCalls,
		}
		messages = append(messages, asst)

		// 内容过滤：输出（含可能半截的工具调用）不再执行，直接结束本轮。
		if finishReason == "content_filter" {
			// 清掉悬空 ToolCalls：被过滤的调用不会有 tool 结果回填，
			// 留在历史里会让下一轮请求被协议层拒绝（tool_calls 必须有 tool 消息跟随）。
			messages[len(messages)-1].ToolCalls = nil
			stopReason = "content_filter"
			emitFinishWarning(emit, finishReason)
			break
		}

		// 无工具调用 → 本轮任务结束（正常收尾出口）。
		if len(turn.ToolCalls) == 0 {
			if finishReason == "length" {
				// 最终答复被长度截断：保留已有内容，但显式告知用户。
				stopReason = "length"
				emitFinishWarning(emit, finishReason)
			} else {
				stopReason = "completed"
			}
			break
		}
		// 长度截断但仍带工具调用：最后一个调用的参数可能是半截 JSON，
		// 由下方 validToolArgs 校验拦截并走修复轮（错误信息会注明截断可能）。

		// 空轮次熔断：连续多轮无正文（也无推理）且工具组合不变，视为没有新进展。
		// 先于工具执行判定，避免第 N 次再空跑一遍相同工具。
		stuckReason := tracker.noteEmptyTurn(content, turn.ToolCalls, hasReasoning)

		// 按调用顺序逐个执行本步工具，并将结果以 tool 角色回填。
		plans := planToolCalls(turn.ToolCalls, cfg.Tools, tracker, stuckReason, finishReason)
		for i, call := range turn.ToolCalls {
			emit(toolStartEvent(call))
			if plans[i].exec {
				res, _ := runToolCall(ctx, call, plans[i].tool, rt, toolTimeout)
				plans[i].result = res
			}
			if plans[i].stuck != "" {
				stuckReason = plans[i].stuck
			}
			emit(Event{
				Type:       EventToolCallEnd,
				ToolCallID: call.ID,
				ToolName:   call.Name,
				Arguments:  call.Arguments,
				Result:     truncateRunes(plans[i].result, 4000),
			})
			messages = append(messages, llmchat.Message{
				Role:       llmchat.RoleTool,
				Content:    CapToolResult(plans[i].result, maxToolTok),
				ToolCallID: call.ID,
				Name:       call.Name,
			})
		}

		if stuckReason != "" {
			stopReason = "stuck"
			emit(Event{Type: EventRunWarning, Text: fmt.Sprintf(
				"检测到%s，已中止以免空转，正在让模型收尾总结", stuckReason)})
			break
		}
	}

	// ---------- 步数用尽 / 卡住：显式收尾 ----------
	// 循环因 MaxSteps 耗尽或卡住熔断退出时，不能让对话停在工具结果上无声结束：
	// 额外做一次不带工具的收尾调用，让模型基于已有信息给出阶段性总结。
	// 收尾失败不影响已有内容。
	switch stopReason {
	case "max_steps":
		emit(Event{Type: EventRunWarning, Text: fmt.Sprintf(
			"已达最大步数上限（%d），正在让模型收尾总结", cfg.MaxSteps)})
		messages, finalContent = wrapUpForced(ctx, cfg, mgr, messages, emit, finalContent, maxStepsWrapUpPrompt)
	case "stuck":
		messages, finalContent = wrapUpForced(ctx, cfg, mgr, messages, emit, finalContent, stuckWrapUpPrompt)
	}

	// ---------- 正常结束 ----------
	emit(Event{Type: EventRunFinished, Text: finalContent, Data: map[string]any{
		"steps":  steps,
		"limit":  lastUsage.Limit,
		"used":   lastUsage.Used,
		"reason": stopReason,
	}})
	return &Result{
		Messages:     messages,
		FinalContent: finalContent,
		Steps:        steps,
		Usage:        lastUsage,
		StopReason:   stopReason,
	}, nil
}

// maxStepsWrapUpPrompt 步数用尽后收尾调用注入的系统提示。
const maxStepsWrapUpPrompt = "本轮任务的步数预算已用尽，无法再调用任何工具。" +
	"请基于目前已经获得的信息直接给出阶段性总结：已完成什么、哪些尚未完成、用户接下来可以怎么做。"

// wrapUpForced 强制收尾调用：不带工具、注入提示，让模型给出总结。
// 用于步数用尽或卡住熔断。成功时把总结追加进历史（不含 ToolCalls，避免留下悬空调用）；
// 失败或被取消时保留调用前的历史与内容。
func wrapUpForced(
	ctx context.Context,
	cfg Config,
	mgr *contextManager,
	messages []llmchat.Message,
	emit func(Event),
	fallback string,
	prompt string,
) ([]llmchat.Message, string) {
	if prompt == "" {
		prompt = maxStepsWrapUpPrompt
	}
	// Prepare 内部会克隆消息，追加的系统提示只进入本次请求视图，不污染落库历史。
	nudged := append(messages, llmchat.Message{
		Role:    llmchat.RoleSystem,
		Content: prompt,
	})
	prepared, prepErr := mgr.Prepare(ctx, nudged)
	if prepErr != nil {
		if !errors.Is(prepErr, context.Canceled) {
			emit(Event{Type: EventRunWarning, Text: "收尾总结准备失败：" + prepErr.Error()})
		}
		return messages, fallback
	}
	req := &llmchat.ChatRequest{
		Model:           cfg.Model,
		Messages:        prepared.Messages, // 不带 Tools：模型只能产出文本总结
		Reasoning:       cfg.Reasoning,
		ReasoningEffort: cfg.ReasoningEffort,
	}
	if prepared.MaxOutput > 0 {
		req.MaxTokens = prepared.MaxOutput
	}
	var wrapText strings.Builder
	turn, err := cfg.Chat.ChatStream(ctx, req, func(ev llmchat.StreamEvent) error {
		switch ev.Type {
		case llmchat.StreamEventTextDelta:
			wrapText.WriteString(ev.Text)
			emit(Event{Type: EventTextDelta, Text: ev.Text})
		case llmchat.StreamEventReasoningDelta:
			emit(Event{Type: EventReasoningDelta, Text: ev.Text})
		}
		return nil
	})
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			emit(Event{Type: EventRunWarning, Text: "收尾总结调用失败：" + err.Error()})
		}
		return messages, fallback
	}
	content := turn.Content
	if content == "" {
		content = wrapText.String()
	}
	if content == "" {
		return messages, fallback
	}
	messages = append(messages, llmchat.Message{Role: llmchat.RoleAssistant, Content: content})
	if fr := normalizeFinishReason(turn.FinishReason); fr != "" {
		emitFinishWarning(emit, fr)
	}
	return messages, content
}

// normalizeFinishReason 跨协议归一化结束原因：
// OpenAI 的 length / Anthropic 的 max_tokens → length（输出截断）；
// OpenAI 的 content_filter / Anthropic 的 refusal → content_filter（内容过滤）；
// 其余（stop / end_turn / tool_use / tool_calls / 空等）视为正常，返回 ""。
func normalizeFinishReason(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "length", "max_tokens":
		return "length"
	case "content_filter", "refusal":
		return "content_filter"
	default:
		return ""
	}
}

// emitFinishWarning 针对异常结束原因向用户发出警告事件。
func emitFinishWarning(emit func(Event), reason string) {
	var text string
	switch reason {
	case "length":
		text = "模型输出达到长度上限，内容可能被截断"
	case "content_filter":
		text = "模型输出被内容安全策略拦截，本轮已结束"
	}
	if text != "" {
		emit(Event{Type: EventRunWarning, Text: text})
	}
}

type toolPlan struct {
	result string
	exec   bool
	tool   Tool
	stuck  string
}

// planToolCalls 决定每个调用是执行还是直接回填错误。
func planToolCalls(calls []llmchat.ToolCall, tools []Tool, tracker *stuckTracker, stuckReason, finishReason string) []toolPlan {
	plans := make([]toolPlan, len(calls))
	skip := stuckReason != ""
	for i, call := range calls {
		if skip {
			plans[i].result = stuckToolResult(stuckReason)
			plans[i].stuck = stuckReason
			continue
		}
		if tracker.noteCall(call) {
			stuckReason = tracker.repeatNotice(call)
			skip = true
			plans[i].result = stuckToolResult(stuckReason)
			plans[i].stuck = stuckReason
			continue
		}
		tool := FindTool(tools, call.Name)
		if tool == nil {
			plans[i].result = fmt.Sprintf(`{"error":"unknown tool: %s"}`, call.Name)
			continue
		}
		if !validToolArgs(call.Arguments) {
			hint := ""
			if finishReason == "length" {
				hint = "（本步输出因长度限制被截断，参数可能不完整，请缩短参数内容）"
			}
			plans[i].result = fmt.Sprintf(`{"error":%q}`, fmt.Sprintf(
				"工具 %s 的参数不是合法 JSON%s：%s。请修正参数后重新调用。",
				call.Name, hint, truncateRunes(call.Arguments, 200)))
			continue
		}
		plans[i].exec = true
		plans[i].tool = tool
	}
	return plans
}

func toolStartEvent(call llmchat.ToolCall) Event {
	return Event{
		Type:       EventToolCallStart,
		ToolCallID: call.ID,
		ToolName:   call.Name,
		Arguments:  call.Arguments,
	}
}

// runToolCall 执行单个工具调用。
// 普通执行失败时仍返回错误 JSON 字符串与 nil error，由模型自行处理。
func runToolCall(ctx context.Context, call llmchat.ToolCall, tool Tool, rt *Runtime, defaultTimeout time.Duration) (string, error) {
	timeout := defaultTimeout
	// 参数感知的超时优先：同一种工具，长命令与短命令的上限可以不同。
	if tf, ok := tool.(interface{ TimeoutFor(argsJSON string) time.Duration }); ok {
		if d := tf.TimeoutFor(call.Arguments); d > 0 {
			timeout = d
		}
	} else if lt, ok := tool.(interface{ Timeout() time.Duration }); ok {
		if d := lt.Timeout(); d > 0 {
			timeout = d
		}
	}
	toolCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	callRT := *rt
	callRT.ToolCallID = call.ID
	out, execErr := tool.Execute(toolCtx, call.Arguments, &callRT)
	if execErr != nil {
		if ctx.Err() == nil && errors.Is(execErr, context.DeadlineExceeded) {
			execErr = fmt.Errorf("工具执行超时（超过 %s）", timeout)
		}
		return fmt.Sprintf(`{"error":%q}`, execErr.Error()), nil
	}
	return out, nil
}

// validToolArgs 校验工具参数是否为合法 JSON。
// 空参数视为合法（无参工具的常见输出），由工具侧按 "{}" 语义处理。
func validToolArgs(args string) bool {
	s := strings.TrimSpace(args)
	if s == "" {
		return true
	}
	return json.Valid([]byte(s))
}

func truncateRunes(s string, n int) string {
	if n <= 0 || s == "" {
		return s
	}
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}
