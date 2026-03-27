package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// StreamToken 表示一个 SSE 流令牌
type StreamToken struct {
	Token    string
	TenantID string
	TaskID   string
	Created  time.Time
	Used     bool
}

// StreamTokenStore 管理 SSE 流令牌（单次使用，TTL=5min，绑定 tenant+task）
type StreamTokenStore struct {
	mu     sync.Mutex
	tokens map[string]*StreamToken
}

// NewStreamTokenStore 创建令牌存储
func NewStreamTokenStore() *StreamTokenStore {
	s := &StreamTokenStore{
		tokens: make(map[string]*StreamToken),
	}
	// 启动后台清理过期令牌
	go s.cleanup()
	return s
}

// Issue 为指定 tenant+task 生成一次性令牌
func (s *StreamTokenStore) Issue(tenantID, taskID string) string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token := fmt.Sprintf("stk_%s", hex.EncodeToString(b))

	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token] = &StreamToken{
		Token:    token,
		TenantID: tenantID,
		TaskID:   taskID,
		Created:  time.Now(),
	}
	return token
}

// Validate 验证并消费令牌（单次使用）
// 返回 tenantID 和 taskID，验证失败返回空字符串
func (s *StreamTokenStore) Validate(token, taskID string) (tenantID string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, exists := s.tokens[token]
	if !exists {
		return "", false
	}

	// 检查过期（TTL=5min）
	if time.Since(st.Created) > 5*time.Minute {
		delete(s.tokens, token)
		return "", false
	}

	// 检查是否已使用
	if st.Used {
		delete(s.tokens, token)
		return "", false
	}

	// 检查 taskID 绑定
	if st.TaskID != taskID {
		return "", false
	}

	// 标记为已使用并删除
	st.Used = true
	delete(s.tokens, token)
	return st.TenantID, true
}

// cleanup 定期清理过期令牌
func (s *StreamTokenStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		for k, v := range s.tokens {
			if time.Since(v.Created) > 5*time.Minute {
				delete(s.tokens, k)
			}
		}
		s.mu.Unlock()
	}
}
