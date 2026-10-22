package hresp

import (
	"net/http"

	"chengzi-agent-code-server/pkg/response"
)

func WriteOK(w http.ResponseWriter, data interface{}) {
	response.WriteOK(w, data)
}

func WriteOKMsg(w http.ResponseWriter, msg string, data interface{}) {
	response.WriteOKMsg(w, msg, data)
}

func WriteFail(w http.ResponseWriter, msg string, err error) {
	response.WriteFailErr(w, msg, err)
}

func WriteFailMsg(w http.ResponseWriter, msg, sysMsg string) {
	response.WriteFail(w, msg, sysMsg)
}
