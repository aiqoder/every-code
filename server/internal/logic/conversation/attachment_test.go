package conversation

import (
	"encoding/base64"
	"path/filepath"
	"testing"

	"chengzi-agent-code-server/internal/config"
	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"
	"chengzi-agent-code-server/pkg/llmchat"
)

func TestDecodeImageAndRejectUnsupportedModel(t *testing.T) {
	png := base64.StdEncoding.EncodeToString([]byte{0x89, 0x50, 0x4e, 0x47})
	files, err := decodeAttachments([]types.ChatAttachment{{
		FileName: "图片.png",
		MimeType: "image/png",
		Data:     png,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Kind != model.InputCapabilityImage {
		t.Fatalf("files=%+v", files)
	}
	if err := ensureAttachmentCaps(&model.LlmModel{Attachment: true, InputCapabilities: "text"}, files); err == nil {
		t.Fatal("text-only model should reject images")
	}
	if err := ensureAttachmentCaps(&model.LlmModel{Attachment: true, InputCapabilities: "image,text"}, files); err != nil {
		t.Fatal(err)
	}
}

func TestAttachmentRoundTrip(t *testing.T) {
	dir := t.TempDir()
	logic := &ChatLogic{svcCtx: &svc.ServiceContext{Config: config.Config{
		Store: config.StoreConfig{StorageDir: filepath.Join(dir, "store")},
	}}}
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a}
	meta, err := logic.writeAttachments(21, 9, []decodedAttachment{{
		FileName: "图片.png",
		MimeType: "image/png",
		Kind:     model.InputCapabilityImage,
		Data:     png,
	}})
	if err != nil {
		t.Fatal(err)
	}
	parts, extra, err := logic.partsFromMetadata(meta)
	if err != nil {
		t.Fatal(err)
	}
	if extra != "" || len(parts) != 1 || parts[0].Kind != llmchat.PartImage {
		t.Fatalf("parts=%+v extra=%q", parts, extra)
	}
	if parts[0].Data != base64.StdEncoding.EncodeToString(png) || parts[0].MimeType != "image/png" {
		t.Fatalf("part=%+v", parts[0])
	}
}
