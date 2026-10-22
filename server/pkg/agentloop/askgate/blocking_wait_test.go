package askgate

import (
	"context"
	"testing"
	"time"
)

// 验证阻塞等待语义：Submit 投递 → Wait 收到；超时/取消 → 槽位清理。
func TestGateBlockingWaitFlow(t *testing.T) {
	g := New()
	id, err := g.Open(1, 1, "tc-1", Spec{Questions: []Question{{ID: "q1", Prompt: "p", Options: []Option{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}}}}})
	if err != nil {
		t.Fatal(err)
	}

	// 1) 并发等待 + 提交
	got := make(chan string, 1)
	go func() {
		raw, err := g.Wait(context.Background(), id)
		if err != nil {
			t.Errorf("wait: %v", err)
			return
		}
		got <- raw
	}()
	time.Sleep(50 * time.Millisecond)
	_, raw, err := g.Submit(1, 1, id, []Answer{{QuestionID: "q1", Selected: []string{"a"}}})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case w := <-got:
		if w != raw {
			t.Fatalf("wait=%q submit=%q", w, raw)
		}
	case <-time.After(time.Second):
		t.Fatal("wait did not receive answer")
	}

	// 2) 槽位已被消费：再次 Submit 应失败
	if _, _, err := g.Submit(1, 1, id, []Answer{{QuestionID: "q1", Selected: []string{"a"}}}); err == nil {
		t.Fatal("submit after consumed should fail")
	}

	// 3) 取消等待 → 槽位清理 → 迟到 Submit 失败
	id2, _ := g.Open(1, 1, "tc-2", Spec{Questions: []Question{{ID: "q1", Prompt: "p", Options: []Option{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}}}}})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if _, err := g.Wait(ctx, id2); err == nil {
		t.Fatal("cancelled wait should error")
	}
	time.Sleep(20 * time.Millisecond)
	if _, _, err := g.Submit(1, 1, id2, []Answer{{QuestionID: "q1", Selected: []string{"a"}}}); err == nil {
		t.Fatal("submit after cancel should fail")
	}
}
