// Package router 组装 gin 路由。
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/xiezc/xpt/services/web-server/handler"
	"github.com/xiezc/xpt/services/web-server/middleware"
)

// New 创建 gin 引擎并注册全部路由。
//
// 路由规划（学习参考）：
//   - /api/auth/*       公开：注册、登录
//   - /api/torrents/*   需登录：列表、上传、查询 peer
//   - /api/admin/*      需管理员：删除种子、管理用户（TODO）
//   - /api/stats       需登录：全局统计
//   - /api/dht/*       需登录：DHT 发现的 infohash
func New(h *handler.WebHandler, jwtSecret string) *gin.Engine {
	r := gin.Default()

	api := r.Group("/api")

	// 公开路由。
	auth := api.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
	}

	// 需要登录的路由。
	protected := api.Group("")
	protected.Use(middleware.Auth(jwtSecret))
	{
		protected.GET("/auth/me", h.Me)
		protected.GET("/torrents", h.ListTorrents)
		protected.POST("/torrents", h.UploadTorrent)
		protected.GET("/torrents/:info_hash/peers", h.TorrentPeers)
		protected.GET("/stats", h.Stats)
		protected.GET("/dht/hashes", h.KnownHashes)
	}

	// 需要管理员权限（TODO: 实现管理员 handler 后挂载）。
	admin := api.Group("/admin")
	admin.Use(middleware.Auth(jwtSecret), middleware.RequireAdmin())
	{
		// 预留：admin.DELETE("/torrents/:id", h.DeleteTorrent)
	}

	return r
}
