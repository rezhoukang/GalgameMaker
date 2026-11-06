package api

import (
	"galgame-maker/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func (h *Handlers) restoreCanvas(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, 400, 400, "参数错误")
		return
	}
	var req struct {
		Expected string             `json:"expected"`
		Snapshot service.CanvasView `json:"snapshot"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, 400, "历史记录无效")
		return
	}
	view, err := h.nodes.Restore(uint(id), req.Expected, req.Snapshot)
	if err != nil {
		fail(c, http.StatusConflict, 409, err.Error())
		return
	}
	ok(c, view)
}
func (h *Handlers) copyProperties(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, 400, 400, "参数错误")
		return
	}
	var req struct {
		SourceID uint               `json:"sourceId"`
		Snapshot service.CanvasView `json:"snapshot"`
	}
	if c.ShouldBindJSON(&req) != nil || req.SourceID == 0 {
		fail(c, 400, 400, "缺少来源节点")
		return
	}
	if err := h.nodes.CopyProperties(uint(id), req.SourceID, req.Snapshot); err != nil {
		fail(c, 400, 400, err.Error())
		return
	}
	ok(c, gin.H{})
}
