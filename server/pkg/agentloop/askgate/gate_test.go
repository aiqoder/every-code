package askgate

import (
	"encoding/json"
	"strings"
	"testing"
)

func sampleSpec() Spec {
	return Spec{
		Title: "确认",
		Questions: []Question{{
			ID:     "go",
			Prompt: "继续吗",
			Options: []Option{
				{ID: "yes", Label: "继续"},
				{ID: "no", Label: "停止"},
			},
		}},
	}
}

func TestValidateRejectsShortOptions(t *testing.T) {
	spec := sampleSpec()
	spec.Questions[0].Options = spec.Questions[0].Options[:1]
	if err := Validate(spec); err == nil {
		t.Fatal("expected error")
	}
}

func TestSubmitReturnsFormattedAnswers(t *testing.T) {
	g := New()
	id, err := g.Open(7, 3, "call_1", sampleSpec())
	if err != nil {
		t.Fatal(err)
	}
	toolCallID, raw, err := g.Submit(7, 3, id, []Answer{{QuestionID: "go", Selected: []string{"yes"}}})
	if err != nil {
		t.Fatal(err)
	}
	if toolCallID != "call_1" {
		t.Fatalf("toolCallID=%s", toolCallID)
	}
	if !strings.Contains(raw, `"yes"`) {
		t.Fatalf("raw=%s", raw)
	}
	// 第二次提交：槽位已消费，应失败。
	if _, _, err := g.Submit(7, 3, id, []Answer{{QuestionID: "go", Selected: []string{"no"}}}); err == nil {
		t.Fatal("second submit should fail")
	}
}

func TestSubmitRejectsUnknownOptionWithoutConsuming(t *testing.T) {
	g := New()
	id, err := g.Open(1, 1, "call_1", sampleSpec())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := g.Submit(1, 1, id, []Answer{{QuestionID: "go", Selected: []string{"maybe"}}}); err == nil {
		t.Fatal("expected invalid option")
	}
	// 不合法答案不消费槽位，可以再次提交。
	if _, _, err := g.Submit(1, 1, id, []Answer{{QuestionID: "go", Selected: []string{"no"}}}); err != nil {
		t.Fatal(err)
	}
}

func TestSubmitOtherWithoutText(t *testing.T) {
	g := New()
	id, err := g.Open(1, 1, "call_1", sampleSpec())
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = g.Submit(1, 1, id, []Answer{{QuestionID: "go", Selected: []string{OtherOptionID}}})
	if err == nil || err.Error() != "请填写「其他」的内容" {
		t.Fatalf("err=%v", err)
	}
	raw := `{"answers":[{"questionId":"go","selected":["yes"]}]}`
	var body struct {
		Answers []Answer `json:"answers"`
	}
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatal(err)
	}
	if _, _, err := g.Submit(1, 1, id, body.Answers); err != nil {
		t.Fatal(err)
	}
}

func TestSubmitRejectsOtherUser(t *testing.T) {
	g := New()
	id, err := g.Open(1, 9, "call_1", sampleSpec())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := g.Submit(1, 8, id, []Answer{{QuestionID: "go", Selected: []string{"yes"}}}); err == nil {
		t.Fatal("expected owner mismatch")
	}
}
