package main

import (
	"social_app/internal/auth"
	"social_app/internal/config"
	"social_app/internal/db"
	"social_app/internal/handler"
	"social_app/internal/logger"
	"social_app/internal/middleware"
	"social_app/internal/redis"
	"social_app/internal/websocket"

	"github.com/gin-gonic/gin"
)

func main() {
	logger.Init()
	logger.Info("Starting social app backend...")

	cfg := config.Load()
	logger.Info("Configuration loaded")

	err := db.Init(cfg)
	if err != nil {
		logger.Error("Failed to initialize database: %v", err)
		return
	}
	logger.Info("Database initialized")

	err = redis.Init(cfg)
	if err != nil {
		logger.Error("Failed to initialize Redis: %v", err)
		return
	}
	logger.Info("Redis connected successfully")

	wsServer := websocket.NewServer()
	go wsServer.Run()
	logger.Info("WebSocket server started")

	wsHandler := websocket.NewHandler(wsServer, cfg)

	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

	// Static files for uploads
	r.Static("/uploads", "./uploads")

	api := r.Group("/api/v1")
	{
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", func(c *gin.Context) {
				auth.Register(c, cfg)
			})
			authGroup.POST("/login", func(c *gin.Context) {
				auth.Login(c, cfg)
			})
		}
		api.POST("/user/me", func(c *gin.Context) {
			auth.GetCurrentUser(c, cfg)
		})
		api.POST("/user/update", middleware.AuthMiddleware(cfg), auth.UpdateProfile)
		api.POST("/upload/avatar", middleware.AuthMiddleware(cfg), handler.UploadAvatar)
	}

	r.GET("/ws", wsHandler.HandleWebSocket)

	logger.Info("Server starting on port %s", cfg.ServerPort)
	logger.Info("WebSocket endpoint: /ws")
	logger.Fatal("Server stopped: %v", r.Run(":"+cfg.ServerPort))
}
