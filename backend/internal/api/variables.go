package api

import (
	"galgame-maker/internal/models"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func (h *Handlers) saveVariables(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	var req struct {
		Variables []models.Variable `json:"variables"`
	}
	if err = c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 400, "变量值必须是整数")
		return
	}
	if err = h.nodes.SaveVariables(uint(id), req.Variables); err != nil {
		fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	ok(c, gin.H{})
}
