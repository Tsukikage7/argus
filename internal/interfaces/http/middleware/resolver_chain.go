package middleware

import (
	"context"

	"github.com/Tsukikage7/argus/internal/domain/task"
)

// ChainResolver 链式 KeyResolver，按顺序尝试多个 resolver
// 第一个返回非 nil Principal 的 resolver 胜出
type ChainResolver struct {
	resolvers []KeyResolver
}

// NewChainResolver 创建链式 resolver
func NewChainResolver(resolvers ...KeyResolver) *ChainResolver {
	return &ChainResolver{resolvers: resolvers}
}

// Resolve 按顺序尝试每个 resolver
func (c *ChainResolver) Resolve(ctx context.Context, rawKey string) (*task.Principal, error) {
	for _, r := range c.resolvers {
		p, err := r.Resolve(ctx, rawKey)
		if err != nil {
			return nil, err
		}
		if p != nil {
			return p, nil
		}
	}
	return nil, nil
}
