package ade

import (
	"encoding/json"
	"sync"

	"chengzi-agent-code-server/pkg/agentloop"
)

// eventLogCap 单个任务在内存里保留的最近事件数，供 SSE 重连回放。
const eventLogCap = 2000

// EventBus 进程内运行事件广播。SSE 订阅者按任务订阅，断线不影响服务端继续跑。
type EventBus struct {
	mu   sync.RWMutex
	subs map[uint64]map[chan Event]struct{} // taskId -> subscribers
	log  map[uint64][]Event                 // taskId -> 当前这一轮的事件
}

// Event 对外事件：任务级包装，附带 run 信息。
type Event struct {
	TaskID         uint64 `json:"taskId"`
	RunID          uint64 `json:"runId"`
	ConversationID uint64 `json:"conversationId,omitempty"`
	Kind           string `json:"kind"`
	Seq            int    `json:"seq"`
	// Agent 原始事件（文本增量、工具调用等）。
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	ToolName  string `json:"toolName,omitempty"`
	ToolCall  string `json:"toolCallId,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	Result    string `json:"result,omitempty"`
	Error     string `json:"error,omitempty"`
	Data      any    `json:"data,omitempty"`
}

// NewEventBus 构造事件总线。
func NewEventBus() *EventBus {
	return &EventBus{
		subs: make(map[uint64]map[chan Event]struct{}),
		log:  make(map[uint64][]Event),
	}
}

// Begin 开始新一轮执行，丢掉上一轮留在内存里的事件。
func (b *EventBus) Begin(taskID uint64) {
	b.mu.Lock()
	delete(b.log, taskID)
	b.mu.Unlock()
}

// Subscribe 订阅某任务的事件流，并带回订阅前已经发生的事件。cancel 幂等。
func (b *EventBus) Subscribe(taskID uint64) (<-chan Event, []Event, func()) {
	ch := make(chan Event, 128)
	b.mu.Lock()
	set, ok := b.subs[taskID]
	if !ok {
		set = make(map[chan Event]struct{})
		b.subs[taskID] = set
	}
	set[ch] = struct{}{}
	recent := append([]Event(nil), b.log[taskID]...)
	b.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			b.mu.Lock()
			if s, ok := b.subs[taskID]; ok {
				delete(s, ch)
				if len(s) == 0 {
					delete(b.subs, taskID)
				}
			}
			b.mu.Unlock()
			close(ch)
		})
	}
	return ch, recent, cancel
}

// Publish 非阻塞投递；订阅者过慢时丢弃该帧，避免拖住 Agent 循环。
func (b *EventBus) Publish(ev Event) {
	b.mu.Lock()
	b.log[ev.TaskID] = append(b.log[ev.TaskID], ev)
	if n := len(b.log[ev.TaskID]); n > eventLogCap {
		b.log[ev.TaskID] = append([]Event(nil), b.log[ev.TaskID][n-eventLogCap:]...)
	}
	set := b.subs[ev.TaskID]
	chans := make([]chan Event, 0, len(set))
	for ch := range set {
		chans = append(chans, ch)
	}
	b.mu.Unlock()
	for _, ch := range chans {
		select {
		case ch <- ev:
		default:
		}
	}
}

// fromLoop 把 agentloop 事件转成对外事件。
func fromLoop(ev agentloop.Event) Event {
	raw, _ := json.Marshal(ev.Data)
	out := Event{
		Type:      string(ev.Type),
		Text:      ev.Text,
		ToolName:  ev.ToolName,
		ToolCall:  ev.ToolCallID,
		Arguments: ev.Arguments,
		Result:    ev.Result,
		Error:     ev.Error,
	}
	if len(ev.Data) > 0 {
		var data map[string]any
		if json.Unmarshal(raw, &data) == nil {
			out.Data = data
		}
	}
	return out
}
