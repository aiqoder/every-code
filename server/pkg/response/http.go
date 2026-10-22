package response

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Envelope 统一 JSON 响应信封。
type Envelope struct {
	Code   int         `json:"code"`
	Msg    string      `json:"msg"`
	Data   interface{} `json:"data"`
	SysMsg string      `json:"sysMsg"`
}

const (
	CodeOK   = 0
	CodeFail = 1
)

func writeHTTPJSON(w http.ResponseWriter, status int, env Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(env)
}

// WriteOK 成功响应（标准库 HTTP）。
func WriteOK(w http.ResponseWriter, data interface{}) {
	writeHTTPJSON(w, http.StatusOK, Envelope{Code: CodeOK, Data: data})
}

// WriteOKMsg 带提示文案的成功响应（标准库 HTTP）。
func WriteOKMsg(w http.ResponseWriter, msg string, data interface{}) {
	writeHTTPJSON(w, http.StatusOK, Envelope{Code: CodeOK, Msg: msg, Data: data})
}

// WriteFail 失败响应（标准库 HTTP）。
func WriteFail(w http.ResponseWriter, msg, sysMsg string) {
	writeHTTPJSON(w, http.StatusOK, Envelope{Code: CodeFail, Msg: msg, SysMsg: sysMsg})
}

// WriteFailErr 失败响应，sysMsg 取自 error（标准库 HTTP）。
func WriteFailErr(w http.ResponseWriter, msg string, err error) {
	sysMsg := ""
	if err != nil {
		sysMsg = err.Error()
	}
	WriteFail(w, msg, sysMsg)
}

// DecodeEnvelope 解析统一信封；HTTP 非 JSON / 非 2xx 时返回可读错误（避免把 404 page not found 当成 JSON）。
func DecodeEnvelope(statusCode int, raw []byte, dest any) error {
	trim := bytes.TrimSpace(raw)
	if statusCode >= 300 || len(trim) == 0 || trim[0] != '{' {
		return fmt.Errorf("服务端返回 HTTP %d: %s", statusCode, clipBody(raw, statusCode))
	}
	var env Envelope
	if err := json.Unmarshal(trim, &env); err != nil {
		return fmt.Errorf("解析响应失败: %w (HTTP %d %s)", err, statusCode, clipBody(raw, statusCode))
	}
	if env.Code != CodeOK {
		msg := env.Msg
		if msg == "" {
			msg = env.SysMsg
		}
		if msg == "" {
			msg = "请求失败"
		}
		if env.SysMsg != "" && env.SysMsg != msg {
			return fmt.Errorf("%s: %s", msg, env.SysMsg)
		}
		return fmt.Errorf("%s", msg)
	}
	if dest == nil {
		return nil
	}
	b, err := json.Marshal(env.Data)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dest)
}

func clipBody(raw []byte, statusCode int) string {
	s := strings.TrimSpace(string(raw))
	if s == "" {
		s = http.StatusText(statusCode)
	}
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 240 {
		return s[:240]
	}
	return s
}
