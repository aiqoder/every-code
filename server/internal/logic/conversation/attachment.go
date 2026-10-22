package conversation

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"chengzi-agent-code-server/internal/chatattach"
	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/types"
	"chengzi-agent-code-server/pkg/llmchat"
	"chengzi-agent-code-server/pkg/strutil"
)

const (
	maxAttachments     = 5
	maxAttachmentBytes = 8 << 20
	maxAttachmentB64   = 12_000_000
)

// decodedAttachment 已校验的附件。Text 非空表示正文内联；否则 Data 为原始字节。
type decodedAttachment struct {
	FileName string
	MimeType string
	Kind     string
	Data     []byte
	Text     string
}

type storedAttachment struct {
	FileName string `json:"fileName"`
	MimeType string `json:"mimeType"`
	Kind     string `json:"kind"`
	File     string `json:"file"`
}

type userAttachmentMeta struct {
	Attachments []storedAttachment `json:"attachments"`
}

func decodeAttachments(items []types.ChatAttachment) ([]decodedAttachment, error) {
	if len(items) == 0 {
		return nil, nil
	}
	if len(items) > maxAttachments {
		return nil, fmt.Errorf("最多 %d 个附件", maxAttachments)
	}
	out := make([]decodedAttachment, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.FileName)
		if name == "" || strings.TrimSpace(item.Data) == "" {
			return nil, fmt.Errorf("附件缺少文件名或内容")
		}
		if len(item.Data) > maxAttachmentB64 {
			return nil, fmt.Errorf("%s 超过 8MB", name)
		}
		raw, err := base64.StdEncoding.DecodeString(item.Data)
		if err != nil {
			raw, err = base64.RawStdEncoding.DecodeString(strings.TrimRight(item.Data, "="))
		}
		if err != nil {
			return nil, fmt.Errorf("%s 内容无法解析", name)
		}
		if len(raw) == 0 {
			return nil, fmt.Errorf("%s 内容为空", name)
		}
		if len(raw) > maxAttachmentBytes {
			return nil, fmt.Errorf("%s 超过 8MB", name)
		}
		kind, mime, err := classifyAttachment(name, item.MimeType)
		if err != nil {
			return nil, err
		}
		decoded := decodedAttachment{FileName: filepath.Base(name), MimeType: mime, Kind: kind, Data: raw}
		if kind == model.InputCapabilityText {
			if !utf8.Valid(raw) {
				return nil, fmt.Errorf("%s 不是文本文件", name)
			}
			decoded.Text = string(raw)
			decoded.Data = nil
		}
		out = append(out, decoded)
	}
	return out, nil
}

func classifyAttachment(fileName, mime string) (kind, normalized string, err error) {
	mime = strings.ToLower(strings.TrimSpace(mime))
	name := strings.ToLower(fileName)
	switch mime {
	case "image/jpg", "image/pjpeg":
		mime = "image/jpeg"
	}
	if mime == "" || mime == "application/octet-stream" {
		mime = mimeFromName(name)
	}
	switch {
	case mime == "image/svg+xml" || strings.HasSuffix(name, ".svg"):
		return model.InputCapabilityText, "image/svg+xml", nil
	case strings.HasPrefix(mime, "image/") || imageName(name):
		if mime == "image/bmp" || strings.HasSuffix(name, ".bmp") {
			return "", "", fmt.Errorf("%s 暂不支持，请改用 png 或 jpeg", fileName)
		}
		switch mime {
		case "image/png", "image/jpeg", "image/gif", "image/webp":
			return model.InputCapabilityImage, mime, nil
		default:
			return "", "", fmt.Errorf("%s 不是支持的图片格式", fileName)
		}
	case mime == "application/pdf" || strings.HasSuffix(name, ".pdf"):
		return model.InputCapabilityPDF, "application/pdf", nil
	case strings.HasPrefix(mime, "text/") || mime == "application/json" || textName(name):
		if mime == "" {
			mime = "text/plain"
		}
		return model.InputCapabilityText, mime, nil
	default:
		return "", "", fmt.Errorf("%s 不在支持的附件类型里", fileName)
	}
}

func ensureAttachmentCaps(m *model.LlmModel, files []decodedAttachment) error {
	if len(files) == 0 {
		return nil
	}
	if m == nil || !m.Attachment {
		return fmt.Errorf("当前模型不支持附件")
	}
	allowed := map[string]struct{}{}
	caps := strutil.SplitCSV(m.InputCapabilities)
	if len(caps) == 0 {
		caps = model.InputCapabilityOptions()
	}
	for _, cap := range caps {
		allowed[cap] = struct{}{}
	}
	for _, f := range files {
		if _, ok := allowed[f.Kind]; ok {
			continue
		}
		switch f.Kind {
		case model.InputCapabilityImage:
			return fmt.Errorf("当前模型不能看图片")
		case model.InputCapabilityPDF:
			return fmt.Errorf("当前模型不能读 PDF")
		default:
			return fmt.Errorf("当前模型不能读文本附件")
		}
	}
	return nil
}

func (l *ChatLogic) writeAttachments(conversationID, messageID uint64, files []decodedAttachment) (string, error) {
	dir := filepath.Join(chatattach.Root(l.svcCtx.Config), fmt.Sprintf("%d", conversationID), fmt.Sprintf("%d", messageID))
	if err := os.RemoveAll(dir); err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	stored := make([]storedAttachment, 0, len(files))
	for i, f := range files {
		rel := fmt.Sprintf("%d/%d/%d", conversationID, messageID, i)
		body := f.Data
		if f.Kind == model.InputCapabilityText {
			body = []byte(f.Text)
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%d", i)), body, 0o644); err != nil {
			return "", err
		}
		stored = append(stored, storedAttachment{
			FileName: f.FileName,
			MimeType: f.MimeType,
			Kind:     f.Kind,
			File:     rel,
		})
	}
	b, err := json.Marshal(userAttachmentMeta{Attachments: stored})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (l *ChatLogic) partsFromMetadata(metadata string) ([]llmchat.Part, string, error) {
	metadata = strings.TrimSpace(metadata)
	if metadata == "" {
		return nil, "", nil
	}
	var meta userAttachmentMeta
	if err := json.Unmarshal([]byte(metadata), &meta); err != nil || len(meta.Attachments) == 0 {
		return nil, "", nil
	}
	var text strings.Builder
	var parts []llmchat.Part
	root := chatattach.Root(l.svcCtx.Config)
	for _, item := range meta.Attachments {
		full, err := chatattach.Resolve(root, item.File)
		if err != nil {
			return nil, "", err
		}
		body, err := os.ReadFile(full)
		if err != nil {
			return nil, "", fmt.Errorf("读取附件 %s 失败", item.FileName)
		}
		switch item.Kind {
		case model.InputCapabilityText:
			if text.Len() > 0 {
				text.WriteString("\n\n")
			}
			fmt.Fprintf(&text, "附件 %s：\n%s", item.FileName, string(body))
		case model.InputCapabilityImage, model.InputCapabilityPDF:
			kind := llmchat.PartImage
			if item.Kind == model.InputCapabilityPDF {
				kind = llmchat.PartPDF
			}
			parts = append(parts, llmchat.Part{
				Kind:     kind,
				FileName: item.FileName,
				MimeType: item.MimeType,
				Data:     base64.StdEncoding.EncodeToString(body),
			})
		}
	}
	return parts, text.String(), nil
}

func mimeFromName(name string) string {
	switch {
	case strings.HasSuffix(name, ".png"):
		return "image/png"
	case strings.HasSuffix(name, ".jpg"), strings.HasSuffix(name, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(name, ".gif"):
		return "image/gif"
	case strings.HasSuffix(name, ".webp"):
		return "image/webp"
	case strings.HasSuffix(name, ".bmp"):
		return "image/bmp"
	case strings.HasSuffix(name, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(name, ".pdf"):
		return "application/pdf"
	case strings.HasSuffix(name, ".json"):
		return "application/json"
	case strings.HasSuffix(name, ".txt"), strings.HasSuffix(name, ".md"), strings.HasSuffix(name, ".markdown"),
		strings.HasSuffix(name, ".csv"), strings.HasSuffix(name, ".log"), strings.HasSuffix(name, ".xml"),
		strings.HasSuffix(name, ".yaml"), strings.HasSuffix(name, ".yml"):
		return "text/plain"
	default:
		return ""
	}
}

func imageName(name string) bool {
	return strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg") ||
		strings.HasSuffix(name, ".gif") || strings.HasSuffix(name, ".webp") || strings.HasSuffix(name, ".bmp")
}

func textName(name string) bool {
	return strings.HasSuffix(name, ".txt") || strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".markdown") ||
		strings.HasSuffix(name, ".csv") || strings.HasSuffix(name, ".log") || strings.HasSuffix(name, ".xml") ||
		strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".json")
}
