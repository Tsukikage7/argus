// Package es 封装 Elasticsearch 客户端
package es

import (
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
