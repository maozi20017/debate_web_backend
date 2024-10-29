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
	// 載入配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("載入config失敗: %v", err)
	}

	// 初始化數據庫連接
	db, err := storage.NewPostgresDB(cfg.DB.Host, cfg.DB.User, cfg.DB.Password, cfg.DB.Name, cfg.DB.Port)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// 自動遷移數據庫結構
	if err := db.AutoMigrate(&models.User{}, &models.Room{}, &models.Message{}); err != nil {
		log.Fatalf("Failed to auto migrate database: %v", err)
	}

	// 初始化 repositories
	repos := repository.NewRepositories(db)

	// 初始化 services
	services := service.NewServices(repos)

	// 設置 Gin 路由
	r := gin.Default()

	api.SetupRoutes(r, services)

	// 啟動服務器
	if err := r.Run(cfg.Server.Address); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
