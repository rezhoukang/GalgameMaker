package api

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// errStorageNotConfigured 存储未配置的错误标记。
var errStorageNotConfigured = errors.New("尚未配置本地存储目录")

// getTree 返回完整树结构（文件夹 → 画布）。
func (h *Handlers) getTree(c *gin.Context) {
	tree, err := h.tree.GetTree()
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	ok(c, tree)
}

// createFolder 新建文件夹。
func (h *Handlers) createFolder(c *gin.Context) {
	var req struct {
		ParentID *uint  `json:"parentId"`
		Name     string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 400, "请求参数错误")
		return
	}
	folder, err := h.tree.CreateFolder(req.ParentID, req.Name)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == errStorageNotConfigured.Error() {
			status = http.StatusPreconditionFailed
		}
		fail(c, status, 412, err.Error())
		return
	}
	ok(c, folder)
}

// createCanvas 新建画布（树里的文件）。
func (h *Handlers) createCanvas(c *gin.Context) {
	var req struct {
		FolderID uint   `json:"folderId"`
		Name     string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 400, "请求参数错误")
		return
	}
	canvas, err := h.tree.CreateCanvas(req.FolderID, req.Name)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	// 新画布自动带一个初始节点（含一个「出」端口），分支从它开始发送
	if _, err := h.nodes.CreateInitialScene(canvas.ID); err != nil {
		fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	ok(c, canvas)
}

// checkCanvas 检测画布问题：空端口 / 双角色端口 / 跳转资源缺失。
func (h *Handlers) checkCanvas(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	issues, err := h.nodes.CheckCanvas(uint(id))
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	ok(c, issues)
}

// getCanvas 返回画布完整视图（HTML 框 + 节点 + 连线）。
func (h *Handlers) getCanvas(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	view, err := h.tree.GetCanvasView(uint(id))
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	// 空白画布（含旧数据）打开时自动补一个初始节点，分支从它开始发送
	if len(view.Scenes) == 0 {
		if _, err := h.nodes.CreateInitialScene(uint(id)); err != nil {
			fail(c, http.StatusInternalServerError, 500, err.Error())
			return
		}
		view, err = h.tree.GetCanvasView(uint(id))
		if err != nil {
			fail(c, http.StatusInternalServerError, 500, err.Error())
			return
		}
	}
	ok(c, view)
}

// updateScene 更新 HTML 框（名称/位置）。
func (h *Handlers) updateScene(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	var req struct {
		Name  string  `json:"name"`
		X     float64 `json:"x"`
		Y     float64 `json:"y"`
		Width float64 `json:"width"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 400, "请求参数错误")
		return
	}
	scene, err := h.nodes.UpdateScene(uint(id), req.Name, req.X, req.Y, req.Width)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	ok(c, scene)
}

// setFirstScene 把某 HTML 框设为全局唯一的「开始」区块。
func (h *Handlers) setFirstScene(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	scene, err := h.nodes.SetFirstScene(uint(id))
	if err != nil {
		fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	ok(c, scene)
}

// uploadSceneVideo 上传 HTML 框关联的 mp4 视频。
func (h *Handlers) uploadSceneVideo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "未收到文件")
		return
	}
	defer file.Close()
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".mp4") {
		fail(c, http.StatusBadRequest, 400, "仅支持 mp4 文件")
		return
	}
	scene, err := h.nodes.SaveSceneVideo(uint(id), file)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	ok(c, scene)
}

// uploadSceneHtml 导入/覆盖节点的 HTML 内容（文件名不变，仅内容）。
func (h *Handlers) uploadSceneHtml(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "未收到文件")
		return
	}
	defer file.Close()
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".html") &&
		!strings.HasSuffix(strings.ToLower(header.Filename), ".htm") {
		fail(c, http.StatusBadRequest, 400, "仅支持 html 文件")
		return
	}
	data, err := io.ReadAll(file)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "读取文件失败")
		return
	}
	scene, err := h.nodes.UpdateSceneHTML(uint(id), data)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	ok(c, scene)
}

// deleteScene 删除 HTML 框。
func (h *Handlers) deleteScene(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.nodes.DeleteScene(uint(id)); err != nil {
		fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	ok(c, gin.H{})
}

// renameItem 重命名。
func (h *Handlers) renameItem(c *gin.Context) {
	var req struct {
		Kind    string `json:"kind"` // folder / canvas
		ID      uint   `json:"id"`
		NewName string `json:"newName"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 400, "请求参数错误")
		return
	}
	if err := h.tree.RenameItem(req.Kind, req.ID, req.NewName); err != nil {
		fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	ok(c, gin.H{})
}

// deleteItem 删除。
func (h *Handlers) deleteItem(c *gin.Context) {
	kind := c.Query("kind")
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.tree.DeleteItem(kind, uint(id)); err != nil {
		fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	ok(c, gin.H{})
}

// resetAll 初始化：清空数据库与存储目录，重新开始。
func (h *Handlers) resetAll(c *gin.Context) {
	if err := h.tree.ResetAll(); err != nil {
		fail(c, http.StatusInternalServerError, 500, err.Error())
		return
	}
	ok(c, gin.H{})
}
