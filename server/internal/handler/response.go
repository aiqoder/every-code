package handler

import (
	"net/http"

	"chengzi-agent-code-server/internal/handler/hresp"
)

func writeOK(w http.ResponseWriter, data interface{}) {
	hresp.WriteOK(w, data)
}

func writeOKMsg(w http.ResponseWriter, msg string, data interface{}) {
	hresp.WriteOKMsg(w, msg, data)
}

func writeFail(w http.ResponseWriter, msg string, err error) {
	hresp.WriteFail(w, msg, err)
}

func writeFailMsg(w http.ResponseWriter, msg, sysMsg string) {
	hresp.WriteFailMsg(w, msg, sysMsg)
}
