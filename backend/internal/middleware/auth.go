package middleware

import (
	"net/http"
	"social_app/internal/auth"
	"social_app/internal/config"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		uidStr := c.Query("uid")
		if uidStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "uid required"})
			c.Abort()
			return
		}

		uidValue, err := strconv.ParseUint(uidStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid uid"})
			c.Abort()
			return
		}
		uid := uint(uidValue)

		if auth.IsWhitelistUID(uid) {
			user, err := auth.EnsureWhitelistUser(uid)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ensure whitelist user"})
				c.Abort()
				return
			}
			c.Set("user_id", user.UID) // Use UID
			c.Set("username", user.Username)
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			c.Abort()
			return
		}

		claims, err := auth.ParseToken(parts[1], cfg)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		if claims.UserID != uidValue {
			c.JSON(http.StatusForbidden, gin.H{"error": "uid does not match token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
