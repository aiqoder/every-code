package admin

import (
	toollogic "chengzi-agent-code-server/internal/logic/tool"
	"chengzi-agent-code-server/internal/types"
)

// toToolInput 请求转入参。
func toToolInput(req *types.ToolReq) toollogic.ToolInput {
	return toollogic.ToolInput{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Parameters:  req.Parameters,
		Type:        req.Type,
		Extra:       req.Extra,
		Version:     req.Version,
	}
}
