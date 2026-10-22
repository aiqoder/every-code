package llmchat

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOpenAIUserMessageIncludesImage(t *testing.T) {
	msg := openAIUserMessage(Message{
		Role:    RoleUser,
		Content: "图片上什么文字",
		Parts: []Part{{
			Kind:     PartImage,
			MimeType: "image/png",
			Data:     "aGVsbG8=",
		}},
	})
	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	raw := string(b)
	if !strings.Contains(raw, `"image_url"`) || !strings.Contains(raw, "data:image/png;base64,aGVsbG8=") {
		t.Fatalf("image part missing: %s", raw)
	}
	if !strings.Contains(raw, "图片上什么文字") {
		t.Fatalf("text missing: %s", raw)
	}
}

func TestAnthropicUserMessageIncludesImage(t *testing.T) {
	msg := anthropicUserMessage(Message{
		Role:    RoleUser,
		Content: "图片上什么文字",
		Parts: []Part{{
			Kind:     PartImage,
			MimeType: "image/png",
			Data:     "aGVsbG8=",
		}},
	})
	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	raw := string(b)
	if !strings.Contains(raw, `"image"`) || !strings.Contains(raw, "aGVsbG8=") {
		t.Fatalf("image block missing: %s", raw)
	}
}
