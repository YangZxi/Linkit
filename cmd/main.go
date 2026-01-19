package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"linkit/internal/config"
	"linkit/internal/db"
	"linkit/internal/middleware"
	"linkit/internal/server"
	"linkit/internal/storage"
	"linkit/internal/task"
)

func main() {
	cfg := config.Load()
	logger := server.NewLogger(cfg.LogLevel)
	shouldRunServer, err := handleCLI(cfg, logger, os.Args[1:])
	if err != nil {
		logger.Error("CLI 执行失败", "err", err)
		os.Exit(1)
	}
	if !shouldRunServer {
		return
	}
	logger.Info("当前项目配置：", "config", cfg)
	gin.SetMode(gin.ReleaseMode)
	store, err := db.NewStore(cfg, logger, true)
	if err != nil {
		logger.Error("初始化数据库失败", "err", err)
		os.Exit(1)
	}
	defer store.Close()

	// 同步数据库配置到内容
	if err := cfg.Sync(context.Background(), store.AppConfig); err != nil {
		logger.Error("同步配置失败", "err", err)
		os.Exit(1)
	}
	logger.Info("当前项目APP配置：", "config", cfg.AppConfig)
	storageReg, err := storage.SetupRegistry(cfg, logger)
	if err != nil {
		logger.Error("初始化存储失败", "err", err)
		os.Exit(1)
	}

	Init(cfg, storageReg)

	r := gin.New()
	// r.Use(middleware.CORS(cfg.FrontendOrigin))
	r.Use(middleware.RequestLogger(logger))
	r.Use(gin.Recovery())
	r.Use(middleware.AuthOptional(store, cfg))

	r.GET("/r/:code", server.DownloadHandler(store, storageReg))

	api := r.Group("/api")
	{
		api.POST("/login", server.LoginHandler(store, cfg))
		api.GET("/share/:code", server.ShareInfoHandler(store))
		api.GET("/upload", server.UploadQueryHandler(&cfg))
		api.POST("/upload", server.UploadHandler(store, &cfg, storageReg))

		apiAuth := api.Group("")
		apiAuth.Use(middleware.AuthRequired(store, cfg))
		apiAuth.GET("/me", server.MeHandler())
		apiAuth.POST("/refresh", server.RefreshHandler(store, cfg))
		apiAuth.POST("/logout", server.LogoutHandler(store, cfg))

		apiAuth.GET("/gallery", server.GalleryHandler(store))
		apiAuth.POST("/gallery/delete", server.GalleryDeleteHandler(store, storageReg))
		apiAuth.POST("/share", server.CreateShareHandler(store))

		apiAdmin := apiAuth.Group("/admin")
		apiAdmin.Use(middleware.AdminRequired(cfg))
		apiAdmin.GET("/stats", server.AdminDashboardStatsHandler(store))
		apiAdmin.GET("/config", server.AdminGetConfigHandler(store, &cfg))
		apiAdmin.POST("/config", server.AdminUpsertConfigHandler(store, &cfg))
		apiAdmin.POST("/password", server.AdminChangePasswordHandler(store, cfg))
	}

	// 静态资源
	r.Static("/assets", "./public/assets")

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/r/") {
			c.JSON(http.StatusNotFound, server.Fail[any]("404", 404))
			return
		}
		filePath := filepath.Join("public", filepath.Clean(path))
		if _, err := os.Stat(filePath); err == nil {
			logger.Info("加载静态资源", "path", filePath)
			c.File(filePath)
			return
		}
		c.File("./public/index.html")
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	logger.Info("服务器启动", "port", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("服务器启动失败", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	logger.Info("服务器已退出")
}

func Init(cfg config.Config, storageReg *storage.Registry) {
	// 启动 S3 备份任务
	task.StartS3DBBackup(cfg, storageReg)
}
