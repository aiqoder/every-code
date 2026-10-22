package tool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/repository"
	"chengzi-agent-code-server/pkg/agentloop/builtin"
	"chengzi-agent-code-server/pkg/agentloop/httptool"
)

var englishNameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

var reservedNames = map[string]struct{}{
	builtin.ToolTodoWriter:  {},
	builtin.ToolTask:        {},
	builtin.ToolAskQuestion: {},
	"subagent":              {},
}

// ToolInput 工具创建/更新入参。
type ToolInput struct {
	Name        string
	DisplayName string
	Description string
	Parameters  map[string]any
	Type        string
	Extra       map[string]any
	Version     int64
}

// BuildToolEntity 构建管理员工具实体（userID 恒为 nil），可维护 http/builtin。
func BuildToolEntity(ctx context.Context, repo repository.ToolRepository, in ToolInput, userID *uint64, excludeID uint64) (*model.Tool, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.Description = strings.TrimSpace(in.Description)
	in.Type = strings.TrimSpace(in.Type)

	if in.Name == "" {
		return nil, fmt.Errorf("名称不能为空")
	}
	if !englishNameRe.MatchString(in.Name) {
		return nil, fmt.Errorf("名称只能使用英文字母、数字和下划线，且以字母开头")
	}
	if in.DisplayName == "" {
		return nil, fmt.Errorf("中文名称不能为空")
	}
	if in.Type == "" {
		in.Type = model.ToolTypeHTTP
	}

	isAdmin := userID == nil
	if !isAdmin {
		if _, ok := reservedNames[in.Name]; ok {
			return nil, fmt.Errorf("名称 %s 为系统保留名称", in.Name)
		}
		if in.Type != model.ToolTypeHTTP {
			return nil, fmt.Errorf("暂不支持的工具类型: %s", in.Type)
		}
	}

	paramsJSON, err := marshalJSON(in.Parameters, "{}")
	if err != nil {
		return nil, fmt.Errorf("参数 JSON 格式无效")
	}

	var extraJSON string
	switch in.Type {
	case model.ToolTypeHTTP:
		if isAdmin {
			if _, ok := reservedNames[in.Name]; ok {
				return nil, fmt.Errorf("名称 %s 为系统保留名称", in.Name)
			}
		}
		extraJSON, err = validateHTTPExtra(in.Extra)
		if err != nil {
			return nil, err
		}
	case model.ToolTypeBuiltin:
		if !isAdmin {
			return nil, fmt.Errorf("暂不支持的工具类型: %s", in.Type)
		}
		if err := builtin.ValidateNames([]string{in.Name}); err != nil {
			return nil, err
		}
		extraJSON = "{}"
	default:
		return nil, fmt.Errorf("暂不支持的工具类型: %s", in.Type)
	}

	exists, err := repo.ExistsByName(ctx, in.Name, userID, excludeID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("工具名称 %s 已存在", in.Name)
	}

	return &model.Tool{
		UserID:      userID,
		Name:        in.Name,
		DisplayName: in.DisplayName,
		Description: in.Description,
		Parameters:  paramsJSON,
		Type:        in.Type,
		Extra:       extraJSON,
		Version:     in.Version,
	}, nil
}

func validateHTTPExtra(extra map[string]any) (string, error) {
	if extra == nil {
		return "", fmt.Errorf("HTTP 工具必须配置附加项")
	}
	raw, err := json.Marshal(extra)
	if err != nil {
		return "", fmt.Errorf("附加项 JSON 格式无效")
	}
	var cfg httptool.HTTPExtra
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return "", fmt.Errorf("附加项 JSON 格式无效")
	}
	if strings.TrimSpace(cfg.URL) == "" {
		return "", fmt.Errorf("HTTP 工具必须配置 url")
	}
	if strings.TrimSpace(cfg.Method) == "" {
		cfg.Method = "POST"
	}
	cfg.Headers = nil
	cfg.Body = ""
	normalized, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("附加项 JSON 格式无效")
	}
	return string(normalized), nil
}

func marshalJSON(v map[string]any, fallback string) (string, error) {
	if v == nil {
		return fallback, nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func parseParametersJSON(raw string) map[string]any {
	out := map[string]any{}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func parseExtraJSON(raw string) map[string]any {
	out := map[string]any{}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

// EnsureAdminTool 校验是否为管理员工具。
func EnsureAdminTool(ctx context.Context, repo repository.ToolRepository, id uint64) (*model.Tool, error) {
	t, err := repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("工具不存在")
		}
		return nil, err
	}
	if t.UserID != nil {
		return nil, fmt.Errorf("该工具不属于管理员工具")
	}
	return t, nil
}

// EnsureEditableAdminTool 校验管理员工具是否允许编辑/删除。
func EnsureEditableAdminTool(ctx context.Context, repo repository.ToolRepository, id uint64) (*model.Tool, error) {
	return EnsureAdminTool(ctx, repo, id)
}
