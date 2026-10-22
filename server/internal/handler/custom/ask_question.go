package custom

import (
	"encoding/json"
	"net/http"
	"strings"

	"chengzi-agent-code-server/internal/logic/conversation"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/pkg/agentloop/askgate"
	"chengzi-agent-code-server/pkg/response"

	"github.com/zeromicro/go-zero/rest/pathvar"
)

type answerQuestionBody struct {
	Answers []askgate.Answer `json:"answers"`
}

// AnswerQuestionHandler POST /api/conversations/:id/questions/:questionId
func AnswerQuestionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, questionID, answers, err := decodeAnswerQuestion(r)
		if err != nil {
			response.WriteFail(w, err.Error(), "")
			return
		}
		err = conversation.NewChatLogic(r.Context(), svcCtx).AnswerQuestion(
			id, middleware.UserID(r.Context()), questionID, answers,
		)
		if err != nil {
			response.WriteFail(w, err.Error(), "")
			return
		}
		response.WriteOK(w, map[string]bool{"ok": true})
	}
}

// decodeAnswerQuestion 用标准 JSON 解析。text 只在选择「其他」时才有值，不能走 go-zero 的必填校验。
func decodeAnswerQuestion(r *http.Request) (uint64, string, []askgate.Answer, error) {
	vars := pathvar.Vars(r)
	id, err := parsePathID(r)
	if err != nil || id == 0 {
		return 0, "", nil, errString("会话 ID 无效")
	}
	questionID := strings.TrimSpace(vars["questionId"])
	if questionID == "" {
		return 0, "", nil, errString("提问 ID 无效")
	}
	var body answerQuestionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return 0, "", nil, errString("请求格式不正确")
	}
	return id, questionID, body.Answers, nil
}

type errString string

func (e errString) Error() string { return string(e) }
