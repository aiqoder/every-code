package model

// 本项目角色常量。管理员由 user-feedback 的角色编码（如 agent_admin）判定。
const (
	// RoleAdmin 管理员（用户角色包含 agent_admin 等管理员角色编码）。
	RoleAdmin = "admin"
	// RoleUser 普通用户。
	RoleUser = "user"
)
