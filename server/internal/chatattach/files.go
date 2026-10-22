// Package chatattach 保存聊天附件文件，避免把图片字节写进消息表。
package chatattach

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"chengzi-agent-code-server/internal/config"
)

// Root 附件根目录，与商店包目录同级。
func Root(c config.Config) string {
	if dir := strings.TrimSpace(c.Store.StorageDir); dir != "" {
		return filepath.Join(filepath.Dir(dir), "chat-attachments")
	}
	if p := strings.TrimSpace(c.Database.SqlitePath); p != "" {
		return filepath.Join(filepath.Dir(p), "chat-attachments")
	}
	return filepath.Join("data", "chat-attachments")
}

// DiscardConversation 删除某会话已落盘的附件。
func DiscardConversation(c config.Config, conversationID uint64) {
	if conversationID == 0 {
		return
	}
	_ = os.RemoveAll(filepath.Join(Root(c), fmt.Sprintf("%d", conversationID)))
}

// DiscardMessage 删除某条消息已落盘的附件。
func DiscardMessage(c config.Config, conversationID, messageID uint64) {
	if conversationID == 0 || messageID == 0 {
		return
	}
	_ = os.RemoveAll(filepath.Join(Root(c), fmt.Sprintf("%d", conversationID), fmt.Sprintf("%d", messageID)))
}

// Resolve 把元数据里的相对路径解析到根目录内。
func Resolve(root, rel string) (string, error) {
	rel = filepath.Clean(filepath.FromSlash(rel))
	if rel == "." || rel == "" || filepath.IsAbs(rel) || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("附件路径无效")
	}
	root = filepath.Clean(root)
	full := filepath.Join(root, rel)
	if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("附件路径无效")
	}
	return full, nil
}
