package api

import (
	"github.com/gin-gonic/gin"

	"debate_web/internal/api/handlers"
	"debate_web/internal/middleware"
	"debate_web/internal/service"
)

// SetupRoutes 設置所有的路由
func SetupRoutes(r *gin.Engine, services *service.Services) {
	// 初始化處理器
	authHandler := handlers.NewAuthHandler(services.UserService)
	roomHandler := handlers.NewRoomHandler(services.RoomService)
	wsHandler := handlers.NewWebSocketHandler(services.WebSocketManager, services.RoomService)

	// 公開路由組，不需要認證
	public := r.Group("/api")
	{
		public.POST("/register", authHandler.Register) // 用戶註冊
		public.POST("/login", authHandler.Login)       // 用戶登入
	}

	// 受保護的路由組，需要認證
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware()) // 使用認證中間件
	{
		// 房間相關路由
		protected.POST("/rooms", roomHandler.CreateRoom)                         // 創建新房間
		protected.GET("/rooms/:id", roomHandler.GetRoom)                         // 獲取特定房間信息
		protected.POST("/rooms/:id/join", roomHandler.JoinRoom)                  // 加入特定房間
		protected.POST("/rooms/:id/start", roomHandler.StartDebate)              // 開始辯論
		protected.POST("/rooms/:id/end", roomHandler.EndDebate)                  // 結束辯論
		protected.GET("/rooms/:id/messages", roomHandler.GetDebateMessages)      //取得辯論訊息
		protected.POST("/rooms/:id/next-round", roomHandler.NextRound)           //下一回合
		protected.GET("/rooms/:id/remaining-time", roomHandler.GetRemainingTime) //取得當前回合剩餘時間
		// WebSocket 路由
		protected.GET("/ws", wsHandler.HandleWebSocket) // 處理 WebSocket 連接

	}
}
