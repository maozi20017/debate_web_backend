package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"debate_web/internal/api"
	"debate_web/internal/config"
	"debate_web/internal/models"
	"debate_web/internal/repository"
	"debate_web/internal/service"
	"debate_web/internal/storage"
)

func main() {
	// 載入應用程式配置
	// 從配置文件中讀取設置，如數據庫連接信息和服務器地址等
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化資料庫連接
	// 使用配置中的信息建立到 PostgreSQL 數據庫的連接
	db, err := storage.NewPostgresDB(cfg.DB.Host, cfg.DB.User, cfg.DB.Password, cfg.DB.Name, cfg.DB.Port)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	// 確保在程序結束時關閉數據庫連接
	defer db.Close()

	// 自動遷移資料庫結構
	// 根據定義的模型自動創建或更新數據庫表結構
	// 這裡遷移 User 和 Room 兩個模型
	if err := db.AutoMigrate(&models.User{}, &models.Room{}); err != nil {
		log.Fatalf("Failed to auto migrate database: %v", err)
	}

	// 初始化服務
	// 初始化 repositories
	repos := repository.NewRepositories(db)

	// 初始化 services
	services := service.NewServices(repos)

	// 設置 Gin 路由
	// 創建一個默認的 Gin 路由器並設置路由
	r := gin.Default()
	api.SetupRoutes(r, services)

	// 啟動伺服器
	// 使用配置中指定的地址啟動 HTTP 服務器
	if err := r.Run(cfg.Server.Address); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
