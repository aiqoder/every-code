package custom

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zeromicro/go-zero/rest/pathvar"
)

func TestDecodeAnswerQuestionWithoutText(t *testing.T) {
	body := `{"answers":[{"questionId":"go","selected":["yes"]}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/conversations/8/questions/qid", strings.NewReader(body))
	req = pathvar.WithVars(req, map[string]string{"id": "8", "questionId": "qid"})

	id, questionID, answers, err := decodeAnswerQuestion(req)
	if err != nil {
		t.Fatal(err)
	}
	if id != 8 || questionID != "qid" || len(answers) != 1 || answers[0].Text != "" {
		t.Fatalf("id=%d q=%s answers=%+v", id, questionID, answers)
	}
	if len(answers[0].Selected) != 1 || answers[0].Selected[0] != "yes" {
		t.Fatalf("selected=%v", answers[0].Selected)
	}
}
