package service

import (
	"crypto/rand"
	"encoding/binary"
	"strconv"
	"strings"
	"time"
)

// NewHash 生成 12 位 base36 哈希串，作为节点/连线的唯一 ID。
func NewHash() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	ts := uint64(time.Now().UnixNano())
	n := binary.LittleEndian.Uint64(b[:])
	v := ts ^ n
	s := strconv.FormatUint(v, 36)
	if len(s) < 12 {
		s = strings.Repeat("0", 12-len(s)) + s
	}
	return s[:12]
}

// sanitizeFileName 清理文件/目录名中的非法字符，避免与真实文件系统冲突。
func sanitizeFileName(name string) string {
	replacer := strings.NewReplacer(
		"\\", "_", "/", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
	)
	name = replacer.Replace(name)
	name = strings.TrimSpace(name)
	if name == "" {
		return "未命名"
	}
	return name
}
