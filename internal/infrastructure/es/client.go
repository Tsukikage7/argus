// Package es 封装 Elasticsearch 客户端
package es

import (
	"fmt"

	"github.com/Tsukikage7/argus/internal/interfaces/config"
	"github.com/elastic/go-elasticsearch/v8"
)

// Client 封装 ES 操作
type Client struct {
	es     *elasticsearch.Client
	prefix string // index prefix, e.g. "argus"
}

// New 创建 ES 客户端
func New(cfg *config.ESConfig) (*Client, error) {
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
	})
	if err != nil {
		return nil, err
	}
	return &Client{es: es, prefix: cfg.IndexPrefix}, nil
}

// Raw 返回底层 ES 客户端（供高级操作使用）
func (c *Client) Raw() *elasticsearch.Client {
	return c.es
}

// Prefix 返回索引前缀
func (c *Client) Prefix() string {
	return c.prefix
}

// namespaceIndex 返回指定 namespace 的索引通配模式
// 格式：{prefix}_{namespace}-*，如 uae-c1_prj-apigateway-*
func (c *Client) namespaceIndex(namespace string) string {
	return fmt.Sprintf("%s_%s-*", c.prefix, namespace)
}

// isDefaultTenant 判断是否为默认租户（非多租户模式）
func isDefaultTenant(tenantID string) bool {
	return tenantID == "" || tenantID == "default"
}

// TenantIndex 返回租户隔离的索引通配模式
// 默认租户回退到写入侧格式 {prefix}_*，多租户模式使用 {prefix}-{tenantID}-logs-*
func (c *Client) TenantIndex(tenantID string) string {
	if isDefaultTenant(tenantID) {
		return c.allIndex()
	}
	return fmt.Sprintf("%s-%s-logs-*", c.prefix, tenantID)
}

// TenantNamespaceIndex 返回租户 + namespace 的索引通配模式
// 默认租户回退到写入侧格式 {prefix}_{namespace}-*，多租户模式使用 {prefix}-{tenantID}-{namespace}-*
func (c *Client) TenantNamespaceIndex(tenantID, namespace string) string {
	if isDefaultTenant(tenantID) {
		return c.namespaceIndex(namespace)
	}
	return fmt.Sprintf("%s-%s-%s-*", c.prefix, tenantID, namespace)
}

// allIndex 返回跨所有 namespace 的索引通配模式（仅限内部/管理路径使用）
// 格式：{prefix}_*
func (c *Client) allIndex() string {
	return fmt.Sprintf("%s_*", c.prefix)
}
