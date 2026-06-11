// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package router

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Rain-kl/Wavelet/internal/apps/admin"
	admin_auth_source "github.com/Rain-kl/Wavelet/internal/apps/admin/auth_source"
	admin_cache "github.com/Rain-kl/Wavelet/internal/apps/admin/cache"
	admin_db_manage "github.com/Rain-kl/Wavelet/internal/apps/admin/db_manage"
	admin_logs "github.com/Rain-kl/Wavelet/internal/apps/admin/logs"
	admin_status "github.com/Rain-kl/Wavelet/internal/apps/admin/status"
	admin_task "github.com/Rain-kl/Wavelet/internal/apps/admin/task"
	admin_template "github.com/Rain-kl/Wavelet/internal/apps/admin/template"
	admin_user "github.com/Rain-kl/Wavelet/internal/apps/admin/user"
	capApp "github.com/Rain-kl/Wavelet/internal/apps/cap"
	publicconfig "github.com/Rain-kl/Wavelet/internal/apps/config"
	"github.com/Rain-kl/Wavelet/internal/apps/health"
	"github.com/Rain-kl/Wavelet/internal/apps/risk_control"
	"github.com/Rain-kl/Wavelet/internal/apps/upload"
	"github.com/Rain-kl/Wavelet/internal/apps/user"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/util"
	capUtil "github.com/Rain-kl/Wavelet/internal/util/cap"

	// Swagger 文档生成
	_ "github.com/Rain-kl/Wavelet/docs"
	"github.com/Rain-kl/Wavelet/internal/apps/admin/system_config"
	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/Rain-kl/Wavelet/internal/config"
	"github.com/Rain-kl/Wavelet/internal/otel_trace"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// Serve 启动 HTTP API 服务
func Serve() {
	// 运行模式
	if config.Config.App.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化 ClickHouse 异步日志写入器
	risk_control.InitLogWriter()

	// 初始化路由
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	cfg := config.Config.Redis
	addrs := cfg.Addrs
	sessionAddr := "localhost:6379"
	if len(addrs) > 0 {
		sessionAddr = addrs[0]
	}

	sessionStore, err := redis.NewStoreWithDB(
		cfg.MinIdleConn,
		"tcp",
		sessionAddr,
		cfg.Username,
		cfg.Password,
		strconv.Itoa(cfg.DB),
		[]byte(config.Config.App.SessionSecret),
	)
	if err != nil {
		log.Fatalf("[API] init session store failed: %v\n", err)
	}

	// 设置 Session Redis Key 前缀
	if cfg.KeyPrefix != "" {
		if err := redis.SetKeyPrefix(sessionStore, cfg.KeyPrefix+"session:"); err != nil {
			log.Printf("[API] set session key prefix failed: %v\n", err)
		}
	}

	sessionStore.Options(util.GetSessionOptions(config.Config.App.SessionAge))

	r.Use(sessions.Sessions(config.Config.App.SessionCookieName, sessionStore))

	// 补充中间件
	r.Use(otelgin.Middleware(config.Config.App.AppName), loggerMiddleware(), risk_control.RiskControlMiddleware())

	registerRoutes(r)

	srv := &http.Server{
		Addr:              config.Config.App.Addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("[API] server starting on %s\n", config.Config.App.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[API] server failed: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Config.App.GracefulShutdownTimeout)*time.Second)

	otel_trace.Shutdown(shutdownCtx)

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[API] server forced to shutdown: %v\n", err)
		cancel()
		os.Exit(1)
	}
	cancel()

	log.Println("[API] server exited")
}

func registerRoutes(r *gin.Engine) {
	// Serve files by ID
	r.GET("/f/:id", upload.ServeFileByID)

	// Dynamic robots.txt serving
	r.GET("/robots.txt", publicconfig.GetRobotsTXT)

	apiGroup := r.Group(config.Config.App.APIPrefix)
	{
		if !config.Config.App.IsProduction() {
			// Swagger
			apiGroup.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		}

		// CAPTCHA
		capGroup := apiGroup.Group("/cap")
		{
			capGroup.POST("/challenge", capApp.Challenge)
			capGroup.POST("/redeem", capApp.Redeem)
		}

		// API V1
		apiV1Router := apiGroup.Group("/v1")
		{
			// Health
			apiV1Router.GET("/health", health.Health)

			// OAuth
			apiV1Router.GET("/oauth/sources", oauth.GetLoginSources)
			apiV1Router.GET("/oauth/login", oauth.GetLoginURL)
			apiV1Router.GET("/oauth/:source/authorize", oauth.Authorize)
			apiV1Router.GET("/oauth/logout", oauth.Logout)
			apiV1Router.POST("/oauth/callback", oauth.Callback)
			apiV1Router.GET("/oauth/user-info", oauth.LoginRequired(), oauth.UserInfo)
			apiV1Router.GET("/user-info", oauth.LoginRequired(), oauth.UserInfo)
			apiV1Router.GET("/oauth/external-accounts", oauth.LoginRequired(), oauth.ListExternalAccounts)
			apiV1Router.POST("/oauth/external-accounts/:id/delete", oauth.LoginRequired(), oauth.DeleteExternalAccount)

			// User
			userRouter := apiV1Router.Group("/user")
			{
				userRouter.POST("/login", capApp.VerifyMiddleware(capUtil.GetDefaultManager(), "login", func() bool {
					enabled, err := model.GetBoolByKey(context.Background(), model.ConfigKeyCapLoginEnabled)
					if err != nil {
						return false
					}
					return enabled
				}), user.Login)
				userRouter.POST("/register", user.Register)
				userRouter.POST("/send-email-code", user.SendEmailCode)
				userRouter.GET("/logout", user.Logout)
				userRouter.GET("/self", oauth.LoginRequired(), oauth.UserInfo)
				userRouter.POST("/change-password", oauth.LoginRequired(), user.ChangePassword)
				userRouter.PUT("/profile", oauth.LoginRequired(), user.UpdateProfile)

				// Access Token
				tokenRouter := userRouter.Group("/access-tokens")
				tokenRouter.Use(oauth.LoginRequired())
				{
					tokenRouter.GET("", user.ListAccessTokens)
					tokenRouter.POST("", user.CreateAccessToken)
					tokenRouter.DELETE("/:id", user.DeleteAccessToken)
					tokenRouter.POST("/:id/rotate", user.RotateAccessToken)
				}
			}

			// Upload
			uploadRouter := apiV1Router.Group("/upload")
			uploadRouter.Use(oauth.LoginRequired())
			{
				uploadRouter.POST("", upload.UploadFile)
				uploadRouter.GET("/my", upload.ListMyFiles)
				uploadRouter.DELETE("/:id", upload.DeleteFile)
				uploadRouter.GET("/download/:id", upload.DownloadFile)
				uploadRouter.POST("/download/batch", upload.BatchDownloadFiles)
			}

			// Config (public)
			configRouter := apiV1Router.Group("/config")
			{
				configRouter.GET("/public", publicconfig.GetPublicConfig)
			}

			// Admin
			adminRouter := apiV1Router.Group("/admin")
			adminRouter.Use(oauth.LoginRequired(), admin.LoginAdminRequired())
			{
				// System status
				adminRouter.GET("/status", admin_status.GetSystemStatus)

				// Database info & export
				adminRouter.GET("/db-info", admin_status.GetDatabaseInfo)
				adminRouter.GET("/db-export", admin_status.ExportDatabase)

				// Database management
				adminRouter.GET("/db-manage/overview", admin_db_manage.GetDBOverview)
				adminRouter.GET("/db-manage/tables", admin_db_manage.ListDBTables)
				adminRouter.GET("/db-manage/table-data", admin_db_manage.GetDBTableData)
				adminRouter.POST("/db-manage/query", admin_db_manage.ExecuteSQL)

				// Cache management
				adminRouter.GET("/cache/status", admin_cache.GetCacheStatus)
				adminRouter.POST("/cache/config", admin_cache.UpdateCacheConfig)
				adminRouter.POST("/cache/clear", admin_cache.ClearCache)

				// System logs
				adminRouter.GET("/logs", admin_logs.GetLogs)
				adminRouter.GET("/logs/access", admin_logs.GetAccessLogs)
				adminRouter.GET("/logs/analytics", admin_logs.GetLogsAnalytics)
				adminRouter.GET("/logs/ws", admin_logs.HandleLogWebSocket)

				// Task dispatch
				adminRouter.GET("/tasks/types", admin_task.ListTaskTypes)
				adminRouter.POST("/tasks/dispatch", admin_task.DispatchTask)

				// Task executions
				adminRouter.GET("/tasks/executions", admin_task.ListTaskExecutions)
				adminRouter.GET("/tasks/executions/:id", admin_task.GetTaskExecution)
				adminRouter.POST("/tasks/executions/:id/retry", admin_task.RetryTask)

				// Task schedules
				adminRouter.GET("/tasks/schedules", admin_task.ListSchedules)
				adminRouter.POST("/tasks/schedules", admin_task.CreateSchedule)
				adminRouter.PUT("/tasks/schedules/:id", admin_task.UpdateSchedule)
				adminRouter.DELETE("/tasks/schedules/:id", admin_task.DeleteSchedule)

				// Users
				adminRouter.GET("/users", admin_user.ListUsers)
				adminRouter.POST("/users", admin_user.CreateUser)
				adminRouter.GET("/users/:id", admin_user.GetUser)
				adminRouter.PUT("/users/:id/status", admin_user.UpdateUserStatus)
				adminRouter.DELETE("/users/:id", admin_user.DeleteUser)

				// Uploads
				adminRouter.GET("/uploads/types", upload.GetDistinctUploadTypes)

				// System Config
				adminRouter.POST("/system-configs", system_config.CreateSystemConfig)
				adminRouter.GET("/system-configs", system_config.ListSystemConfigs)
				adminRouter.POST("/system-configs/smtp/test", system_config.TestSMTP)

				systemConfigRouter := adminRouter.Group("/system-configs/:key")
				{
					systemConfigRouter.GET("", system_config.GetSystemConfig)
					systemConfigRouter.PUT("", system_config.UpdateSystemConfig)
				}

				// Templates
				adminRouter.GET("/templates", admin_template.ListTemplates)
				adminRouter.POST("/templates", admin_template.CreateTemplate)

				templateRouter := adminRouter.Group("/templates/:key")
				{
					templateRouter.GET("", admin_template.GetTemplate)
					templateRouter.PUT("", admin_template.UpdateTemplate)
					templateRouter.DELETE("", admin_template.DeleteTemplate)
				}

				// Auth Sources
				adminRouter.GET("/auth-sources", admin_auth_source.ListAuthSources)
				adminRouter.POST("/auth-sources", admin_auth_source.CreateAuthSource)
				adminRouter.PUT("/auth-sources/:id", admin_auth_source.UpdateAuthSource)
				adminRouter.PUT("/auth-sources/:id/toggle", admin_auth_source.ToggleAuthSource)
				adminRouter.DELETE("/auth-sources/:id", admin_auth_source.DeleteAuthSource)
			}

			// Register custom business routes
			registerCustomRoutes(apiV1Router)
		}
	}

	// 注册前端静态路由（当启用 embed_frontend 编译标签时）
	registerFrontend(r)
}
