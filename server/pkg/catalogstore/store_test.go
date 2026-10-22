package catalogstore

import (
	"context"
	"errors"
	"testing"
	"time"

	"chengzi-agent-code-server/pkg/clienthub/protocol"
)

func TestKey(t *testing.T) {
	if got := Key(7, "desk"); got != "chengzi:catalog:7:desk" {
		t.Fatalf("key = %s", got)
	}
}

func TestNormalizeOsInfo(t *testing.T) {
	got, err := NormalizeOsInfo(" Windows-ARM64 ")
	if err != nil || got != "windows-arm64" {
		t.Fatalf("got %q err=%v", got, err)
	}
	if _, err := NormalizeOsInfo(""); err == nil {
		t.Fatal("empty should fail")
	}
	if _, err := NormalizeOsInfo("windows"); err == nil {
		t.Fatal("missing arch should fail")
	}
	if _, err := NormalizeOsInfo("windows_arm64"); err == nil {
		t.Fatal("underscore should fail")
	}
}

func TestRedisRoundTrip(t *testing.T) {
	s, err := New("127.0.0.1:6379", "", 0, 3*time.Second)
	if err != nil {
		t.Skip(err)
	}
	defer s.Close()

	ctx := context.Background()
	const (
		userID   = uint64(42)
		clientID = "catalog-store-test"
	)
	t.Cleanup(func() { _ = s.DeleteClient(context.Background(), userID, clientID) })

	err = s.Save(ctx, userID, clientID, protocol.CatalogPayload{
		Tools:       []protocol.ToolMeta{{Name: "grep", Description: "搜索"}},
		Skills:      []protocol.SkillMeta{{ID: "pkg:demo:weather", Name: "天气"}},
		PageChannel: "create",
		OsInfo:      "Windows-AMD64",
		Workspace:   "/work",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Load(ctx, userID, clientID)
	if err != nil {
		t.Fatal(err)
	}
	if got.OsInfo != "windows-amd64" || got.Workspace != "/work" || got.PageChannel != "create" || len(got.Tools) != 1 {
		t.Fatalf("catalog = %+v", got)
	}
	if err := s.DeleteClient(ctx, userID, clientID); err != nil {
		t.Fatal(err)
	}
	_, err = s.Load(ctx, userID, clientID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete err = %v", err)
	}
}

func TestMemoryRoundTrip(t *testing.T) {
	s := NewMemory(time.Minute)
	ctx := context.Background()
	cat := protocol.CatalogPayload{
		Tools:     []protocol.ToolMeta{{Name: "read"}},
		OsInfo:    "darwin-arm64",
		Workspace: "/tmp/x",
	}
	if err := s.Save(ctx, 1, "desk", cat); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := s.Load(ctx, 1, "desk")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Workspace != "/tmp/x" || len(got.Tools) != 1 {
		t.Fatalf("got %+v", got)
	}
	if _, err := s.Load(ctx, 1, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing should be ErrNotFound, got %v", err)
	}
	if err := s.Save(ctx, 1, "desk", cat); err != nil {
		t.Fatalf("resave: %v", err)
	}
	if err := s.TouchClient(ctx, 1, "desk"); err != nil {
		t.Fatalf("touch: %v", err)
	}
	if err := s.DeleteClient(ctx, 1, "desk"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Load(ctx, 1, "desk"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete err = %v", err)
	}
}
