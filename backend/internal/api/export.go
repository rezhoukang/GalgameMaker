package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// exportAll 把指定画布导出为单页播放器 + 资源目录。
func (h *Handlers) exportAll(c *gin.Context) {
	var req struct {
		CanvasID uint `json:"canvasId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.CanvasID == 0 {
		fail(c, http.StatusBadRequest, 400, "缺少画布 ID")
		return
	}
	result, err := h.export.Export(req.CanvasID)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == errStorageNotConfigured.Error() {
			status = http.StatusPreconditionFailed
		}
		fail(c, status, 412, err.Error())
		return
	}
	ok(c, result)
}
