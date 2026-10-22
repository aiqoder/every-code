// Package identity 封装 user-feedback 身份服务客户端。
// 本项目不再自建账号体系，登录与 OAuth2 授权均以 user-feedback 为唯一身份源。
package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Config identity 客户端配置。
type Config struct {
	// BaseURL user-feedback 服务根地址，如 http://127.0.0.1:8001。
	BaseURL string
	// ClientID / ClientSecret OAuth2 客户端凭证。
	ClientID     string
	ClientSecret string
	// RedirectURI OAuth2 授权码回调地址（指向本服务）。
	RedirectURI string
	// AdminRoleCode 管理员角色编码；用户角色包含该编码即视为管理员。
	// 默认 agent_admin（论坛超级管理员）。
	AdminRoleCode string
	// Timeout HTTP 超时；0 时使用默认 10s。
	Timeout time.Duration
}

// DefaultAdminRoleCode 默认管理员角色编码。
const DefaultAdminRoleCode = "agent_admin"

// User 归一化后的用户身份。
type User struct {
	ID       uint64   `json:"id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	// adminRoleCode 由客户端按配置注入，用于管理员判定。
	adminRoleCode string `json:"-"`
}

// Role 返回本项目角色（admin | user）。
func (u *User) Role() string {
	if u.IsAdmin() {
		return "admin"
	}
	return "user"
}

// IsAdmin 用户角色是否包含管理员角色编码（默认 agent_admin）。
func (u *User) IsAdmin() bool {
	if u == nil {
		return false
	}
	code := strings.TrimSpace(u.adminRoleCode)
	if code == "" {
		code = DefaultAdminRoleCode
	}
	for _, r := range u.Roles {
		if r == code {
			return true
		}
	}
	return false
}

// Client user-feedback 身份服务客户端。
type Client struct {
	cfg  Config
	http *http.Client
}

// New 构造 identity 客户端。
func New(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: timeout},
	}
}

// Configured 是否已配置 user-feedback 地址。
func (c *Client) Configured() bool {
	return c != nil && strings.TrimSpace(c.cfg.BaseURL) != ""
}

// base 返回去除尾部斜杠的服务地址。
func (c *Client) base() string {
	return strings.TrimRight(strings.TrimSpace(c.cfg.BaseURL), "/")
}

// Login 调用 user-feedback 账号密码登录，返回访问令牌与用户身份。
func (c *Client) Login(ctx context.Context, username, password string) (string, *User, error) {
	if !c.Configured() {
		return "", nil, fmt.Errorf("未配置 user-feedback 服务地址")
	}
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return "", nil, fmt.Errorf("账号与密码不能为空")
	}

	var resp struct {
		Token    string `json:"token"`
		Username string `json:"username"`
		Error    string `json:"error"`
	}
	if err := c.postJSON(ctx, c.base()+"/api/auth/login", map[string]string{
		"username": username,
		"password": password,
	}, &resp); err != nil {
		return "", nil, err
	}
	if resp.Error != "" {
		return "", nil, fmt.Errorf("%s", resp.Error)
	}
	if resp.Token == "" {
		return "", nil, fmt.Errorf("登录成功但未返回令牌")
	}

	// 登录响应不含用户 ID 与超管标记，统一通过自省拿到完整身份。
	user, err := c.Introspect(ctx, resp.Token)
	if err != nil {
		return "", nil, err
	}
	return resp.Token, user, nil
}

// Introspect 自省令牌（兼容登录 JWT 与 OAuth2 访问令牌），返回用户身份。
func (c *Client) Introspect(ctx context.Context, token string) (*User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("令牌为空")
	}
	var resp struct {
		Active   bool     `json:"active"`
		UserID   uint64   `json:"user_id"`
		Username string   `json:"username"`
		Roles    []string `json:"roles"`
	}
	if err := c.postJSON(ctx, c.base()+"/oauth2/introspect", map[string]string{"token": token}, &resp); err != nil {
		return nil, err
	}
	if !resp.Active || resp.UserID == 0 {
		return nil, fmt.Errorf("未登录或登录已失效")
	}
	return &User{
		ID:            resp.UserID,
		Username:      resp.Username,
		Roles:         nonNil(resp.Roles),
		adminRoleCode: c.cfg.AdminRoleCode,
	}, nil
}

// ExchangeCode 用授权码换取访问令牌（OAuth2 授权码模式）。
func (c *Client) ExchangeCode(ctx context.Context, code string) (string, *User, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", nil, fmt.Errorf("授权码不能为空")
	}
	var resp struct {
		AccessToken      string `json:"access_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	// /oauth2/token 按 RFC 6749 只接受表单编码，发 JSON 会被判 invalid_request。
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", c.cfg.ClientID)
	form.Set("client_secret", c.cfg.ClientSecret)
	form.Set("redirect_uri", c.cfg.RedirectURI)
	if err := c.postForm(ctx, c.base()+"/oauth2/token", form, &resp); err != nil {
		return "", nil, err
	}
	if resp.Error != "" {
		msg := resp.ErrorDescription
		if msg == "" {
			msg = resp.Error
		}
		return "", nil, fmt.Errorf("%s", msg)
	}
	if resp.AccessToken == "" {
		return "", nil, fmt.Errorf("换取令牌失败")
	}
	user, err := c.UserInfo(ctx, resp.AccessToken)
	if err != nil {
		return "", nil, err
	}
	return resp.AccessToken, user, nil
}

// UserInfo 通过访问令牌获取用户身份。
func (c *Client) UserInfo(ctx context.Context, accessToken string) (*User, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, fmt.Errorf("访问令牌为空")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base()+"/oauth2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 user-feedback 失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("获取用户信息失败: HTTP %d", resp.StatusCode)
	}

	var payload struct {
		Sub      string   `json:"sub"`
		Username string   `json:"username"`
		Roles    []string `json:"roles"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("解析用户信息失败: %w", err)
	}
	if payload.Sub == "" {
		return nil, fmt.Errorf("用户信息缺失")
	}
	var id uint64
	if _, err := fmt.Sscanf(payload.Sub, "%d", &id); err != nil {
		return nil, fmt.Errorf("用户标识无效: %w", err)
	}
	return &User{
		ID:            id,
		Username:      payload.Username,
		Roles:         nonNil(payload.Roles),
		adminRoleCode: c.cfg.AdminRoleCode,
	}, nil
}

// nonNil 保证角色切片非 nil，便于 JSON 序列化为 []。
func nonNil(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

// AuthorizeURL 构造 OAuth2 授权码模式的授权地址。
func (c *Client) AuthorizeURL(state string) string {
	q := url.Values{}
	q.Set("client_id", c.cfg.ClientID)
	q.Set("redirect_uri", c.cfg.RedirectURI)
	q.Set("response_type", "code")
	if state != "" {
		q.Set("state", state)
	}
	return c.base() + "/oauth2/authorize?" + q.Encode()
}

// RedirectURI 返回配置的回调地址。
func (c *Client) RedirectURI() string {
	return c.cfg.RedirectURI
}

// postJSON 发送 JSON POST 并解析响应体到 dest。
// 适用于 /api/auth/login、/oauth2/introspect 等接受 JSON 的端点。
func (c *Client) postJSON(ctx context.Context, endpoint string, payload any, dest any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	body, err := c.do(req)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("解析 user-feedback 响应失败: %w", err)
	}
	return nil
}

// postForm 发送表单 POST（application/x-www-form-urlencoded）并解析响应体到 dest。
// /oauth2/token 等标准 OAuth2 端点按 RFC 6749 只接受表单编码，发 JSON 会被判为
// invalid_request（EOF）。此类端点的错误信息放在 4xx 响应体的 error/error_description 中，
// 因此响应体照常解析，由调用方结合状态码判定。
func (c *Client) postForm(ctx context.Context, endpoint string, values url.Values, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	body, reqErr := c.do(req)
	// 无论状态码都尽量解析，让调用方能读到 OAuth2 的 error/error_description。
	if len(bytes.TrimSpace(body)) > 0 {
		_ = json.Unmarshal(body, dest)
	}
	return reqErr
}

// do 发送请求并返回响应体；非 2xx 时返回带响应摘要的错误，便于定位 user-feedback 侧原因。
func (c *Client) do(req *http.Request) ([]byte, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 user-feedback 失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return body, fmt.Errorf("user-feedback 返回 HTTP %d: %s", resp.StatusCode, clipBody(body))
	}
	return body, nil
}

// clipBody 压缩响应体摘要，避免错误信息过长。
func clipBody(raw []byte) string {
	s := strings.Join(strings.Fields(string(raw)), " ")
	if len(s) > 240 {
		return s[:240]
	}
	if s == "" {
		return "空响应体"
	}
	return s
}
