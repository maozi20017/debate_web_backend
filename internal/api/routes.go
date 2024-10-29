package api

import (
	"debate_web/internal/api/handlers"
	"debate_web/internal/middleware"
	"debate_web/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, services *service.Services) {
	// 初始化 handlers
	authHandler := handlers.NewAuthHandler(services.User)
	roomHandler := handlers.NewRoomHandler(services.Room)
	wsHandler := handlers.NewWebSocketHandler(services.WebSocket, services.Room)

	// API 路由群組
	api := r.Group("/api")

	// 處理 404 錯誤
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "找不到該路徑",
		})
	})

	// 公開路由
	{
		// 用戶認證相關
		api.POST("/register", authHandler.Register)
		api.POST("/login", authHandler.Login)

		// 基本的健康檢查
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "ok",
			})
		})
	}

	// 需要驗證的路由
	authorized := api.Group("/")
	authorized.Use(middleware.AuthMiddleware())
	{
		// 辯論室相關
		rooms := authorized.Group("/rooms")
		{
			// 基本操作
			rooms.GET("", roomHandler.ListRooms)   // 獲取房間列表
			rooms.POST("", roomHandler.CreateRoom) // 創建房間
			rooms.GET("/:id", roomHandler.GetRoom) // 獲取房間信息

			// 房間參與
			rooms.POST("/:id/join", roomHandler.JoinRoom)   // 加入房間
			rooms.POST("/:id/leave", roomHandler.LeaveRoom) // 離開房間

			// WebSocket 連接（移到房間路由下）
			rooms.GET("/:id/ws", wsHandler.HandleWebSocket) // WebSocket 連接點
		}
	}
}
