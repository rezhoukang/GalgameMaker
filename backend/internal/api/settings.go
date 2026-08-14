package api

import (
	"github.com/gin-gonic/gin"
)

// getSettings 获取当前设置（存储/导出目录，路径固定在后端目录下，只读展示）。
func (h *Handlers) getSettings(c *gin.Context) {
	ok(c, h.settings.GetSettings())
}
