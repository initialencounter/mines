package utils

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// CacheEntry 缓存条目的结构体
type CacheEntry struct {
	Code         string
	CreationTime time.Time
}

// CodeCache 缓存的结构体
type CodeCache struct {
	data  map[string]CacheEntry
	mutex sync.RWMutex
}

// NewCodeCache 创建新的缓存
func NewCodeCache() *CodeCache {
	return &CodeCache{
		data: make(map[string]CacheEntry),
	}
}

// GenerateCode 生成一个6位数的验证码
func GenerateCode() (CacheEntry, error) {
	b := make([]byte, 3) // 3字节可以生成0-999999的6位数
	_, err := rand.Read(b)
	if err != nil {
		return CacheEntry{}, err
	}
	code := int(b[0])<<16 + int(b[1])<<8 + int(b[2])
	return CacheEntry{
		Code:         fmt.Sprintf("%06d", code),
		CreationTime: time.Now()}, nil
}

// Set 将验证码存入缓存
func (c *CodeCache) Set(userName string, code string, creationTime time.Time) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data[userName] = CacheEntry{Code: code, CreationTime: creationTime}
}

// Get 从缓存中获取验证码
func (c *CodeCache) Get(userName string) (CacheEntry, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	entry, found := c.data[userName]
	return entry, found
}

// IsCodeExpired 检查验证码是否过期
func IsCodeExpired(creationTime time.Time) bool {
	return time.Since(creationTime) > 120*time.Minute
}
