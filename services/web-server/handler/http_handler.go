// Package handler 实现 web-server 的 HTTP 处理函数。
package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/xiezc/xpt/pkg/model"
	"github.com/xiezc/xpt/pkg/util"
	"github.com/xiezc/xpt/services/web-server/middleware"
	"github.com/xiezc/xpt/services/web-server/service"
)

// WebHandler 聚合所有 HTTP handler。
type WebHandler struct {
	svc       *service.WebService
	jwtSecret string
	tokenTTL  time.Duration
}

// New 构造 WebHandler。
func New(svc *service.WebService, jwtSecret string) *WebHandler {
	return &WebHandler{
		svc:       svc,
		jwtSecret: jwtSecret,
		tokenTTL:  24 * time.Hour,
	}
}

// ---- 认证 ----

// RegisterRequest 注册请求体。
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email"`
}

// Register 处理 POST /api/auth/register。
func (h *WebHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := h.svc.Register(req.Username, req.Password, req.Email)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": u.ID, "username": u.Username})
}

// LoginRequest 登录请求体。
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 处理 POST /api/auth/login，成功返回 JWT。
// TODO(学习): 密码校验尚未实现，当前返回的 token 仅基于 service 层查询结果。
func (h *WebHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	u, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "bad credentials"})
		return
	}
	token, err := util.SignToken(h.jwtSecret, u.ID, u.IsAdmin, h.tokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sign token failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user": u})
}

// Me 处理 GET /api/auth/me（需登录）。
func (h *WebHandler) Me(c *gin.Context) {
	uid := middleware.UserID(c)
	c.JSON(http.StatusOK, gin.H{"user_id": uid})
}

// ---- 种子 ----

// ListTorrents 处理 GET /api/torrents?page=1&size=20。
func (h *WebHandler) ListTorrents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	list, err := h.svc.ListTorrents(page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"torrents": list})
}

// UploadTorrentRequest 上传种子请求体。
type UploadTorrentRequest struct {
	InfoHash string `json:"info_hash" binding:"required"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
}

// UploadTorrent 处理 POST /api/torrents（需登录）。
// TODO(学习): 完整实现需要解析 .torrent 文件，从中提取 info_hash/name/size。
func (h *WebHandler) UploadTorrent(c *gin.Context) {
	var req UploadTorrentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t := &model.Torrent{
		InfoHash:   req.InfoHash,
		Name:       req.Name,
		Size:       req.Size,
		UploadedBy: middleware.UserID(c),
	}
	if err := h.svc.UploadTorrent(t); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"ok": true})
}

// TorrentPeers 处理 GET /api/torrents/:id/peers（需登录）。
// TODO(学习): 当前仅演示通过 gRPC 调 tracker；infohash 由 ID 查询得到。
func (h *WebHandler) TorrentPeers(c *gin.Context) {
	infoHash := c.Param("info_hash")
	resp, err := h.svc.GetTorrentPeers(c.Request.Context(), infoHash)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "tracker unavailable"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Stats 处理 GET /api/stats（需登录）。
func (h *WebHandler) Stats(c *gin.Context) {
	resp, err := h.svc.GetTrackerStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "tracker unavailable"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// KnownHashes 处理 GET /api/dht/hashes（需登录）。
func (h *WebHandler) KnownHashes(c *gin.Context) {
	resp, err := h.svc.GetKnownInfoHashes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "dht unavailable"})
		return
	}
	c.JSON(http.StatusOK, resp)
}
