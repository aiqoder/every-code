package middleware

import (
	"context"
	"net/http"
	"strings"

	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/repository"
	"chengzi-agent-code-server/pkg/identity"
	"chengzi-agent-code-server/pkg/response"
)

type ctxKey string

const (
	ctxUserID ctxKey = "authUserID"
	ctxRole   ctxKey = "authRole"
	ctxName   ctxKey = "authName"
	ctxRoles  ctxKey = "authRoles"
)

// AuthDeps 鉴权中间件依赖。
type AuthDeps struct {
	ApiKeyRepo repository.ApiKeyRepository
	// Identity user-feedback 身份服务客户端，用于校验透传令牌。
	Identity *identity.Client
	// DevToken 本机联调令牌，非空时直接认作 DevUserID。
	DevToken  string
	DevUserID uint64
	DevName   string
}

// AuthMiddleware user-feedback 令牌（透传）/ API Key 鉴权。
type AuthMiddleware struct {
	deps AuthDeps
}

func NewAuthMiddleware(deps AuthDeps) *AuthMiddleware {
	return &AuthMiddleware{deps: deps}
}

func (m *AuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		token := extractBearerToken(r)
		if token == "" {
			token = strings.TrimSpace(r.URL.Query().Get("token"))
		}
		if token != "" {
			// 本机联调令牌优先，避免每次都要连身份服务。
			if m.deps.DevToken != "" && token == m.deps.DevToken {
				userID := m.deps.DevUserID
				if userID == 0 {
					userID = 1
				}
				ctx = withAuth(ctx, userID, model.RoleAdmin, m.deps.DevName, []string{model.RoleAdmin})
				next(w, r.WithContext(ctx))
				return
			}
			// 令牌由 user-feedback 签发，透传校验其身份与角色。
			if user, uErr := m.deps.Identity.Introspect(ctx, token); uErr == nil {
				ctx = withAuth(ctx, user.ID, user.Role(), user.Username, user.Roles)
				next(w, r.WithContext(ctx))
				return
			}
			if userID, apiErr := common.ResolveAPIKeyUserID(ctx, m.deps.ApiKeyRepo, token); apiErr == nil {
				ctx = withAuth(ctx, userID, model.RoleUser, "", nil)
				next(w, r.WithContext(ctx))
				return
			}
			response.WriteFail(w, "未登录或登录已失效", "invalid token or api key")
			return
		}

		apiKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
		if apiKey == "" {
			apiKey = strings.TrimSpace(r.URL.Query().Get("api_key"))
		}
		if apiKey == "" {
			response.WriteFail(w, "请先登录", "missing Authorization or X-API-Key")
			return
		}
		userID, err := common.ResolveAPIKeyUserID(ctx, m.deps.ApiKeyRepo, apiKey)
		if err != nil {
			response.WriteFail(w, "无效的 API Key", err.Error())
			return
		}
		ctx = withAuth(ctx, userID, model.RoleUser, "", nil)
		next(w, r.WithContext(ctx))
	}
}

// AuthAdminMiddleware 管理员鉴权（需在 Auth 之后）。
type AuthAdminMiddleware struct{}

func NewAuthAdminMiddleware() *AuthAdminMiddleware {
	return &AuthAdminMiddleware{}
}

func (m *AuthAdminMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if Role(r.Context()) != model.RoleAdmin {
			response.WriteFail(w, "需要管理员权限", "forbidden")
			return
		}
		next(w, r)
	}
}

func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}
	return ""
}

func withAuth(ctx context.Context, userID uint64, role, name string, roles []string) context.Context {
	ctx = context.WithValue(ctx, ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxRole, role)
	ctx = context.WithValue(ctx, ctxName, name)
	ctx = context.WithValue(ctx, ctxRoles, roles)
	return ctx
}

// UserID 从上下文取当前用户 ID（user-feedback 用户 ID）。
func UserID(ctx context.Context) uint64 {
	v, _ := ctx.Value(ctxUserID).(uint64)
	return v
}

// ContextWithUser 写入用户 ID，供单测构造已登录上下文。
func ContextWithUser(ctx context.Context, userID uint64) context.Context {
	return withAuth(ctx, userID, "", "", nil)
}

// Role 从上下文取当前用户角色。
func Role(ctx context.Context) string {
	v, _ := ctx.Value(ctxRole).(string)
	return v
}

// Roles 从上下文取当前用户角色编码列表。
func Roles(ctx context.Context) []string {
	v, _ := ctx.Value(ctxRoles).([]string)
	return v
}

// Name 从上下文取当前用户名。
func Name(ctx context.Context) string {
	v, _ := ctx.Value(ctxName).(string)
	return v
}

// IsAdmin 当前用户是否管理员。
func IsAdmin(ctx context.Context) bool {
	return Role(ctx) == model.RoleAdmin
}
