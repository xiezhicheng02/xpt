// Package middleware 提供 gin 中间件。
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xiezc/xpt/pkg/util"
)

// contextKey 用于在 gin.Context 中传递认证信息。
const (
	ctxUserID  = "auth_user_id"
	ctxIsAdmin = "auth_is_admin"
)

// Auth 是 JWT 认证中间件。
// 从 Authorization: Bearer <token> 解析 token，注入 userID/isAdmin。
//
// TODO(学习): 目前 token 校验失败直接 401；后续可加黑名单/刷新逻辑。
func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := util.ParseToken(secret, tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set(ctxUserID, claims.UserID)
		c.Set(ctxIsAdmin, claims.IsAdmin)
		c.Next()
	}
}

// RequireAdmin 要求当前用户是管理员（需在 Auth 之后使用）。
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isAdmin, ok := c.Get(ctxIsAdmin); !ok || !isAdmin.(bool) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin required"})
			return
		}
		c.Next()
	}
}

// UserID 从上下文中取当前用户 ID（Auth 之后）。
func UserID(c *gin.Context) int64 {
	if v, ok := c.Get(ctxUserID); ok {
		if id, ok := v.(int64); ok {
			return id
		}
	}
	return 0
}

// IsAdmin 从上下文中取当前用户是否管理员。
func IsAdmin(c *gin.Context) bool {
	if v, ok := c.Get(ctxIsAdmin); ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
