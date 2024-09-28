package middleware

import (
	"debate_web/pkg/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 是一個 Gin 中間件，用於驗證請求的 JWT token
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 從請求頭中獲取 Authorization 字段
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// 檢查 Authorization 頭的格式
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			c.Abort()
			return
		}

		// 解析 JWT token
		claims, err := utils.ParseToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// 將用戶信息設置到上下文中
		c.Set("userID", claims.UserID) // 修改這裡，從 "user_id" 改為 "userID"
		c.Set("userRole", claims.Role) // 修改這裡，從 "user_role" 改為 "userRole"，如果需要的話
		c.Next()                       // 繼續處理請求
	}
}
