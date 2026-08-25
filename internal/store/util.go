package store

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
)

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

var idCounter int64

// NewID 生成稳定、可读的实体 ID（带前缀与单调序号，避免依赖外部 UUID 库）。
func NewID(prefix string) string {
	n := atomic.AddInt64(&idCounter, 1)
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixNano()/1e6, n)
}

// Digest 计算一组字节的 SHA-256 十六进制摘要，用作不可变快照指纹。
func Digest(parts ...[]byte) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write(p)
	}
	return hex.EncodeToString(h.Sum(nil))
}
