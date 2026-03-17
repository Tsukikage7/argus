// Package tool 定义 Agent 可调用的工具抽象
package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Tool 是 Agent 可调用的工具接口
type Tool interface {
	Name() string
	Description() string
	Parameters() json.RawMessage // JSON Schema，用于 LLM function calling
	Execute(ctx context.Context, params map[string]any) (*Result, error)
}

// Result 是工具执行结果
type Result struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

// Registry 管理所有已注册的工具
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry 创建工具注册中心
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register 注册一个工具
func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

// Get 按名称获取工具
func (r *Registry) Get(name string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool %q not found", name)
	}
	return t, nil
}

// List 返回所有已注册工具
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		list = append(list, t)
	}
	return list
}
