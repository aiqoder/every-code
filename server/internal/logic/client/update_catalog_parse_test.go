package client

import (
	"net/http"
	"strings"
	"testing"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func TestUpdateCatalogParseClientPayload(t *testing.T) {
	body := `{
		"clientId": "desktop-1",
		"pageChannel": "create",
		"skills": [{"id": "s1", "name": "技能", "description": "", "source": "local", "path": ""}],
		"tools": [{"name": "read", "description": "读文件"}, {"name": "page_state", "description": "看当前页"}],
		"osInfo": "darwin-arm64",
		"workspace": "/Users/me/work"
	}`
	req, err := http.NewRequest(http.MethodPut, "/api/clients/catalog", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	var parsed UpdateCatalogReq
	if err := httpx.Parse(req, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.ClientId != "desktop-1" || parsed.PageChannel != "create" {
		t.Fatalf("parsed: %+v", parsed)
	}
	if parsed.OsInfo != "darwin-arm64" || parsed.Workspace != "/Users/me/work" {
		t.Fatalf("unexpected fields: %+v", parsed)
	}
	if len(parsed.Skills) != 1 || len(parsed.Tools) != 2 {
		t.Fatalf("skills/tools: %+v", parsed)
	}
}
