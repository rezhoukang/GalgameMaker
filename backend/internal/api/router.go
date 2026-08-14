// Package api 提供 HTTP 路由与请求处理。
package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"galgame-maker/internal/config"
	"galgame-maker/internal/service"
)

// Handlers 聚合所有服务依赖。
type Handlers struct {
	db       *gorm.DB
	settings *service.SettingsService
	tree     *service.TreeService
	nodes    *service.NodeService
	export   *service.ExportService
}

// NewRouter 创建路由并注册全部接口。
func NewRouter(db *gorm.DB, cfg *config.Config) *gin.Engine {
	r := gin.Default()
	r.Use(corsMiddleware())

	h := &Handlers{
		db:       db,
		settings: service.NewSettingsService(db, cfg),
		tree:     service.NewTreeService(db, service.NewSettingsService(db, cfg)),
		nodes:    service.NewNodeService(db),
		export:   service.NewExportService(db, service.NewSettingsService(db, cfg)),
	}

	// 首次启动：若项目为空，自动创建默认的「未分类」文件夹和「新画布」，并给画布补初始节点
	if canvas, err := h.tree.EnsureDefaults(); err != nil {
		log.Printf("初始化默认结构失败: %v", err)
	} else if canvas != nil {
		if _, err := h.nodes.CreateInitialScene(canvas.ID); err != nil {
			log.Printf("创建初始节点失败: %v", err)
		}
	}

	api := r.Group("/api")
	{
		// 设置（路径固定在后端目录下，只读展示）
		api.GET("/settings", h.getSettings)

		// 树结构（文件夹 → 画布）
		api.GET("/tree", h.getTree)
		api.POST("/folders", h.createFolder)
		api.POST("/canvases", h.createCanvas)
		api.PATCH("/items/rename", h.renameItem)
		api.DELETE("/items", h.deleteItem)

		// 画布视图（HTML 框 + 节点 + 连线）
		api.GET("/canvas/:id", h.getCanvas)
		api.GET("/canvas/:id/check", h.checkCanvas)

		// HTML 框
		api.PUT("/scenes/:id", h.updateScene)
		api.DELETE("/scenes/:id", h.deleteScene)
		api.POST("/scenes/:id/video", h.uploadSceneVideo)
		api.POST("/scenes/:id/first", h.setFirstScene)
		api.POST("/scenes/:id/next", h.createNextNode)
		api.POST("/scenes/:id/magic-port", h.createMagicPort)
		api.PUT("/scenes/:id/html", h.uploadSceneHtml)

		// 节点（配对即「哈希相同」，无独立连线表）
		api.PUT("/nodes/:id", h.updateNode)
		api.DELETE("/nodes/:id", h.deleteNode)
		api.DELETE("/nodes/:id/break", h.breakConnection)
		api.PUT("/nodes/:id/redirect", h.redirectNode)

		// 初始化
		api.POST("/reset", h.resetAll)

		// 导出
		api.POST("/export", h.exportAll)
	}

	// 健康检查
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return r
}

// corsMiddleware 允许前端跨域访问（Vite 开发服务器）。
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = "*"
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// ok 统一成功响应。
func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": data})
}

// fail 统一错误响应，code 用于前端区分错误类型。
func fail(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, gin.H{"code": code, "message": message})
}
