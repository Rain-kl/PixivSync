// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

// Package root registers custom business routes and frontend serving.
package root

import (
	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/Rain-kl/Wavelet/internal/apps/pixez"
	"github.com/gin-gonic/gin"
)

// RegisterCustomRootRoutes registers custom business routes that belong to the root path.
func RegisterCustomRootRoutes(r *gin.Engine) {
	// PixEz companion sync API keeps its historical /api/pixez path while
	// using Wavelet AccessToken/session authentication.
	pixezRouter := r.Group("/api/pixez")
	pixezRouter.Use(oauth.LoginRequired())
	{
		pixezRouter.GET("/ping", pixez.Ping)
		pixezRouter.GET("/dashboard", pixez.GetDashboard)
		pixezRouter.GET("/bookmark-export-runs", pixez.ListBookmarkExportRuns)
		pixezRouter.GET("/bookmarks/illusts", pixez.ListBookmarkIllusts)
		pixezRouter.GET("/bookmarks/illusts/:illust_id/detail", pixez.GetBookmarkIllustDetail)
		pixezRouter.GET("/bookmarks/novels", pixez.ListBookmarkNovels)
		pixezRouter.GET("/bookmarks/novels/:novel_id/detail", pixez.GetBookmarkNovelDetail)
		pixezRouter.GET("/users", pixez.ListUsers)
		pixezRouter.POST("/users", pixez.AddUser)
		pixezRouter.GET("/login-url", pixez.GetLoginURL)
		pixezRouter.POST("/login-callback", pixez.LoginCallback)
		pixezRouter.GET("/users/:pixiv_user_id", pixez.GetUser)
		pixezRouter.GET("/users/:pixiv_user_id/profile", pixez.GetUserProfile)
		pixezRouter.POST("/users/:pixiv_user_id/refresh-token", pixez.RefreshUserToken)
		pixezRouter.PUT("/users/:pixiv_user_id", pixez.UpsertUser)
		pixezRouter.DELETE("/users/:pixiv_user_id", pixez.DeleteUser)
		pixezRouter.GET("/users/:pixiv_user_id/sync-data", pixez.GetUserData)
		pixezRouter.POST("/users/:pixiv_user_id/sync-data", pixez.PostUserData)
		pixezRouter.GET("/users/:pixiv_user_id/sync-data/hashes", pixez.GetUserDataHashes)
		pixezRouter.GET("/users/:pixiv_user_id/bookmarks/illust/removed", pixez.ListRemovedBookmarkIllusts)
		pixezRouter.POST("/illusts/:illust_id/mirror", pixez.MirrorIllust)
		pixezRouter.GET("/illusts/:illust_id/mirror", pixez.CheckIllustMirror)
		pixezRouter.POST("/illusts/mirror/batch", pixez.BatchCheckIllustMirror)
		pixezRouter.POST("/novels/:novel_id/mirror", pixez.MirrorNovel)
		pixezRouter.GET("/novels/:novel_id/mirror", pixez.CheckNovelMirror)
		pixezRouter.POST("/novels/mirror/batch", pixez.BatchCheckNovelMirror)
		pixezRouter.GET("/mirror/illusts", pixez.ListMirroredIllusts)
		pixezRouter.GET("/mirror/illusts/:illust_id/detail", pixez.GetMirroredIllustManagementDetail)
		pixezRouter.GET("/mirror/novels", pixez.ListMirroredNovels)
		pixezRouter.GET("/mirror/novels/:novel_id/detail", pixez.GetMirroredNovelManagementDetail)
		pixezRouter.DELETE("/mirror/illusts/:illust_id", pixez.DeleteMirroredIllust)
		pixezRouter.DELETE("/mirror/novels/:novel_id", pixez.DeleteMirroredNovel)
		pixezRouter.POST("/mirror/batch-delete", pixez.BatchDeleteMirroredItems)
	}

	mirrorRouter := r.Group("/mirror")
	mirrorRouter.Use(oauth.LoginRequired())
	{
		mirrorRouter.GET("/v1/illust/detail", pixez.GetMirroredIllustDetail)
		mirrorRouter.GET("/pximg/*path", pixez.ServeMirroredImage)
		mirrorRouter.GET("/v1/novel/detail", pixez.GetMirroredNovelDetail)
		mirrorRouter.GET("/webview/v2/novel", pixez.GetMirroredNovelText)
	}
}
