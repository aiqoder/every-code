package clienthub

import (
	"context"
	"testing"
	"time"
)

func TestAllowedRPC(t *testing.T) {
	if !AllowedRPC("workspace.list") || !AllowedRPC("link.status") || !AllowedRPC("conversation.bind") {
		t.Fatal("expected allowlisted methods")
	}
	if AllowedRPC("shell_exec") || AllowedRPC("connector.list") || AllowedRPC("package.install") || AllowedRPC("") {
		t.Fatal("unexpected method allowed")
	}
}

func TestPresenceSubscribe(t *testing.T) {
	h := NewHub(time.Second, 0)
	ch, cancel := h.Subscribe(7)
	defer cancel()

	c := NewRemoteClient(7, "desk", "橙子", nil)
	c.Workspace = "/work"
	if err := h.Register(7, c); err != nil {
		t.Fatal(err)
	}
	ev := <-ch
	if ev.Type != PresenceOnline || ev.Client == nil || ev.Client.Workspace != "/work" || ev.Client.Name != "橙子" {
		t.Fatalf("online: %+v", ev)
	}

	if err := h.UpdatePresence(7, "desk", "新名字", "/other", "https://desk.example.com", true, true); err != nil {
		t.Fatal(err)
	}
	ev = <-ch
	if ev.Type != PresenceCatalog || ev.Client == nil || ev.Client.Name != "新名字" || ev.Client.Workspace != "/other" || ev.Client.PublicURL != "https://desk.example.com" {
		t.Fatalf("catalog: %+v", ev)
	}

	h.Unregister(7, "desk")
	ev = <-ch
	if ev.Type != PresenceOffline || ev.ClientID != "desk" {
		t.Fatalf("offline: %+v", ev)
	}
}

func TestInvokeRPCRequiresOnline(t *testing.T) {
	h := NewHub(time.Second, 0)
	c := NewRemoteClient(7, "desk", "橙子", nil)
	if err := h.Register(7, c); err != nil {
		t.Fatal(err)
	}
	if _, err := h.InvokeRPC(context.Background(), 7, "desk", "shell_exec", nil); err == nil || err.Error() != "不支持的方法" {
		t.Fatalf("method: %v", err)
	}
	if _, err := h.InvokeRPC(context.Background(), 7, "desk", "workspace.list", nil); err == nil || err.Error() != "客户端未连接" {
		t.Fatalf("offline: %v", err)
	}
}
