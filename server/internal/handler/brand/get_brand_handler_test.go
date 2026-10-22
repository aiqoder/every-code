package brand

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"chengzi-agent-code-server/internal/config"
	"chengzi-agent-code-server/internal/svc"
)

func TestGetBrandHandlerDefaults(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/brand", nil)
	GetBrandHandler(&svc.ServiceContext{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var env struct {
		Code int `json:"code"`
		Data struct {
			Name string `json:"name"`
			Logo string `json:"logo"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Code != 0 || env.Data.Name != config.DefaultBrandName || env.Data.Logo != config.DefaultBrandLogo {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestGetBrandHandlerConfigured(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/brand", nil)
	GetBrandHandler(&svc.ServiceContext{
		Config: config.Config{Brand: config.BrandConfig{Name: " 示例 ", Logo: " https://cdn.example/logo.png "}},
	}).ServeHTTP(rec, req)

	var env struct {
		Data struct {
			Name string `json:"name"`
			Logo string `json:"logo"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Data.Name != "示例" || env.Data.Logo != "https://cdn.example/logo.png" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
