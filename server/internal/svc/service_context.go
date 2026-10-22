package svc

import (
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"

	"chengzi-agent-code-server/internal/config"
	"chengzi-agent-code-server/internal/database"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/repository"
	"chengzi-agent-code-server/pkg/agentloop/askgate"
	"chengzi-agent-code-server/pkg/catalogstore"
	"chengzi-agent-code-server/pkg/clienthub"
	"chengzi-agent-code-server/pkg/identity"
	"chengzi-agent-code-server/pkg/oauthcache"

	"github.com/zeromicro/go-zero/rest"
)

// ServiceContext go-zero 服务上下文，集中装配依赖。
type ServiceContext struct {
	Config config.Config
	DB     *gorm.DB

	ApiKeyRepo       repository.ApiKeyRepository
	LlmProviderRepo  repository.LlmProviderRepository
	LlmModelRepo     repository.LlmModelRepository
	ConversationRepo repository.ConversationRepository
	ToolRepo         repository.ToolRepository
	SkillBindingRepo repository.SkillBindingRepository
	TokenUsageRepo   repository.TokenUsageRepository
	AdeRepo          repository.AdeRepository
	// Ade ADE 领域服务，由 main 装配（类型为 *ade.Service），避免 svc 与 logic 循环依赖。
	Ade       interface{}
	ClientHub *clienthub.Hub
	// AskGate 阻塞 AskQuestion，直到用户提交答案。
	AskGate *askgate.Gate
	// Identity user-feedback 身份服务客户端。
	Identity *identity.Client
	// OAuthCodeCache 授权码兑换结果缓存，使一次性 code 的重复回调幂等。
	OAuthCodeCache *oauthcache.Cache

	Auth      rest.Middleware
	AuthAdmin rest.Middleware
}

// NewServiceContext 构造服务上下文。
func NewServiceContext(c config.Config) *ServiceContext {
	if err := c.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	db, err := database.New(&c.Database)
	if err != nil {
		log.Fatalf("init database: %v", err)
	}

	apiKeyRepo := repository.NewApiKeyRepository(db)
	llmProviderRepo := repository.NewLlmProviderRepository(db)
	llmModelRepo := repository.NewLlmModelRepository(db)
	conversationRepo := repository.NewConversationRepository(db)
	toolRepo := repository.NewToolRepository(db)
	skillBindingRepo := repository.NewSkillBindingRepository(db)
	tokenUsageRepo := repository.NewTokenUsageRepository(db)
	adeRepo := repository.NewAdeRepository(db)
	clientHub := clienthub.NewHub(c.ClientAPI.InvokeTimeout, c.ClientAPI.HeartbeatInterval)
	addr := strings.TrimSpace(c.Redis.Addr)
	if addr == "" {
		log.Fatalf("redis.addr is required")
	}
	ttl := c.Redis.CatalogTTL
	if ttl <= 0 {
		ttl = 90 * time.Second
	}
	catalogStore, err := catalogstore.New(addr, c.Redis.Password, c.Redis.DB, ttl)
	if err != nil {
		// 本机开发与单机部署不强制装 Redis：退回进程内存，重启即清空。
		log.Printf("catalog: %v，改用内存存储（重启后需重新上报 catalog）", err)
		catalogStore = catalogstore.NewMemory(ttl)
	}
	clientHub.SetCatalogStore(catalogStore)

	identityClient := identity.New(identity.Config{
		BaseURL:       c.UserFeedback.BaseURL,
		ClientID:      c.UserFeedback.ClientID,
		ClientSecret:  c.UserFeedback.ClientSecret,
		RedirectURI:   c.UserFeedback.RedirectURI,
		AdminRoleCode: c.UserFeedback.AdminRoleCode,
		Timeout:       c.UserFeedback.Timeout,
	})

	svcCtx := &ServiceContext{
		Config:           c,
		DB:               db,
		ApiKeyRepo:       apiKeyRepo,
		LlmProviderRepo:  llmProviderRepo,
		LlmModelRepo:     llmModelRepo,
		ConversationRepo: conversationRepo,
		ToolRepo:         toolRepo,
		SkillBindingRepo: skillBindingRepo,
		TokenUsageRepo:   tokenUsageRepo,
		AdeRepo:          adeRepo,
		ClientHub:        clientHub,
		AskGate:          askgate.New(),
		Identity:         identityClient,
		OAuthCodeCache:   oauthcache.New(oauthcache.DefaultTTL, 512),
	}

	svcCtx.Auth = middleware.NewAuthMiddleware(middleware.AuthDeps{
		ApiKeyRepo: apiKeyRepo,
		Identity:   identityClient,
		DevToken:   c.DevAuth.Token,
		DevUserID:  c.DevAuth.UserID,
		DevName:    c.DevAuth.Username,
	}).Handle
	svcCtx.AuthAdmin = middleware.NewAuthAdminMiddleware().Handle
	return svcCtx
}

// Shutdown 优雅关闭资源。
func (s *ServiceContext) Shutdown() error {
	if s.ClientHub != nil {
		s.ClientHub.CloseAll()
	}
	if s.DB != nil {
		if err := database.Close(s.DB); err != nil {
			return fmt.Errorf("db close: %w", err)
		}
	}
	return nil
}
