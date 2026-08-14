package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// updateNode 更新节点（名称/坐标）。
func (h *Handlers) updateNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	var req struct {
		Name          string  `json:"name"`
		X             float64 `json:"x"`
		Y             float64 `json:"y"`
		EndingContent *string `json:"endingContent"`
		TargetKind    *string `json:"targetKind"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 400, "请求参数错误")
		return
	}
	node, err := h.nodes.UpdateNode(uint(id), req.Name, req.X, req.Y, req.EndingContent, req.TargetKind)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	ok(c, node)
}

// deleteNode 删除节点。
func (h *Handlers) deleteNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.nodes.DeleteNode(uint(id)); err != nil {
		fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	ok(c, gin.H{})
}

// breakConnection 断开端口连接：删除当前端口与配对端口（保留节点，不删子树）。
func (h *Handlers) breakConnection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.nodes.BreakConnection(uint(id)); err != nil {
		fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	ok(c, gin.H{})
}

// redirectNode 重定向端口一端到指定节点（按节点哈希）：side=prev 改上一节点 / next 改下一节点。
func (h *Handlers) redirectNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	var req struct {
		Hash string `json:"hash"`
		Side string `json:"side"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Hash == "" || (req.Side != "prev" && req.Side != "next") {
		fail(c, http.StatusBadRequest, 400, "参数错误（需要 hash 与 side=prev/next）")
		return
	}
	if err := h.nodes.RedirectNode(uint(id), req.Hash, req.Side); err != nil {
		fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	ok(c, gin.H{})
}

// createNextNode 从节点的右侧加号创建下一跳：新增出端口 + 右侧空白节点（带入端口），两端共用同一哈希（配对）。
func (h *Handlers) createNextNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	scene, outNode, entryNode, err := h.nodes.CreateNextNode(uint(id))
	if err != nil {
		fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	ok(c, gin.H{"scene": scene, "outNode": outNode, "entryNode": entryNode})
}

// createMagicPort 在节点上新建魔法出端口并连向指定哈希的目标节点（不画线、导出生效）。
func (h *Handlers) createMagicPort(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	var req struct {
		Hash string `json:"hash"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Hash == "" {
		fail(c, http.StatusBadRequest, 400, "缺少目标节点哈希")
		return
	}
	outNode, entryNode, err := h.nodes.CreateMagicPort(uint(id), req.Hash)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	ok(c, gin.H{"outNode": outNode, "entryNode": entryNode})
}
