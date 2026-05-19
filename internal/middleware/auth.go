package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	myjwt "kama-chat-server/pkg/jwt"
	"kama-chat-server/pkg/zlog"
)

// JWTAuth Gin 中间件：校验 Authorization: Bearer <token> 或 query ?token=<jwt>
// 校验通过后，将 uuid 和 isAdmin 注入 gin.Context
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string

		// 1. 优先从 Authorization header 获取
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenStr = parts[1]
			}
		}

		// 2. 如果 header 中没有，从 query 参数获取（用于 WebSocket）
		if tokenStr == "" {
			tokenStr = c.Query("token")
		}

		if tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "缺少认证token",
			})
			c.Abort()
			return
		}

		claims, err := myjwt.ParseToken(tokenStr)
		if err != nil {
			zlog.Error("JWT解析失败: " + err.Error())

			msg := "token无效"
			switch {
			case errors.Is(err, jwt.ErrTokenExpired):
				msg = "token已过期"
			case errors.Is(err, jwt.ErrTokenNotValidYet):
				msg = "token尚未生效"
			case errors.Is(err, jwt.ErrTokenMalformed):
				msg = "token格式错误"
			case errors.Is(err, jwt.ErrTokenSignatureInvalid):
				msg = "token签名无效"
			}

			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": msg,
			})
			c.Abort()
			return
		}

		c.Set("uuid", claims.Uuid)
		c.Set("isAdmin", claims.IsAdmin)
		c.Next()
	}
}

// AdminOnly Gin 中间件：要求当前用户必须是管理员（isAdmin == 1）
// 必须在 JWTAuth 之后使用
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, exists := c.Get("isAdmin")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "无管理员权限",
			})
			c.Abort()
			return
		}
		isAdminInt, ok := isAdmin.(int8)
		if !ok || isAdminInt != 1 {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "无管理员权限",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
