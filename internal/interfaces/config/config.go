// Package config 定义应用配置结构体
package config

import (
	"time"

	"github.com/Tsukikage7/servex/logger"
)

// Config 是应用的根配置
type Config struct {
	App           AppConfig        `yaml:"app"`
	Log           logger.Config    `yaml:"log"`
	Providers     []ProviderConfig `yaml:"providers"`
	Agent         AgentConfig      `yaml:"agent"`
	Elasticsearch ESConfig         `yaml:"elasticsearch"`
	Redis         RedisConfig      `yaml:"redis"`
	Postgres      PostgresConfig   `yaml:"postgres"`
	Wechat        WechatConfig     `yaml:"wechat"`
	Mock          MockConfig       `yaml:"mock"`
	Replay        ReplayConfig     `yaml:"replay"`
	Monitor       MonitorConfig    `yaml:"monitor"`
	Live          LiveConfig       `yaml:"live"`
	MultiTenant   MultiTenantConfig `yaml:"multi_tenant"`
}

// MultiTenantConfig 多租户配置
type MultiTenantConfig struct {
	Enabled            bool     `yaml:"enabled"`              // 是否启用多租户模式
	BootstrapAdminKeys []string `yaml:"bootstrap_admin_keys"` // 引导 AdminKey 列表
	AllowedOrigins     []string `yaml:"allowed_origins"`      // CORS 允许的来源列表
}

// AppConfig 基础服务配置
type AppConfig struct {
	Name     string   `yaml:"name"`
	Addr     string   `yaml:"addr"`
	APIKeys  []string `yaml:"api_keys"`
	AdminKey string   `yaml:"admin_key"`
}

// ProviderConfig AI 提供商配置（兼容 OpenAI 格式）
type ProviderConfig struct {
	Name         string        `yaml:"name"`
	BaseURL      string        `yaml:"base_url"`
	APIKey       string        `yaml:"api_key"`
	DefaultModel string        `yaml:"default_model"`
	Timeout      time.Duration `yaml:"timeout"`
	MaxTokens    int           `yaml:"max_tokens"`
	Models       []string      `yaml:"models"`
}

// AgentConfig Agent 行为配置
type AgentConfig struct {
	MaxSteps             int           `yaml:"max_steps"`
	AutoRecoverThreshold float64       `yaml:"auto_recover_threshold"`
	ConfirmThreshold     float64       `yaml:"confirm_threshold"`
	Timeout              time.Duration `yaml:"timeout"`
}

// ESConfig Elasticsearch 配置
type ESConfig struct {
	Addresses   []string `yaml:"addresses"`
	IndexPrefix string   `yaml:"index_prefix"`
	Username    string   `yaml:"username"`
	Password    string   `yaml:"password"`
	Namespaces  []string `yaml:"namespaces"` // K8s namespace 列表，如 ["prj-apigateway", "prj-ubill"]
}

// RedisConfig Redis 连接配置
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// PostgresConfig PostgreSQL 连接配置
type PostgresConfig struct {
	DSN string `yaml:"dsn"`
}

// WechatConfig 企业微信配置
type WechatConfig struct {
	CorpID         string `yaml:"corp_id"`
	AgentID        int    `yaml:"agent_id"`
	Secret         string `yaml:"secret"`
	WebhookURL     string `yaml:"webhook_url"`      // Bot Webhook 推送地址
	Token          string `yaml:"token"`            // 应用回调 Token（URL 验证 + 消息签名）
	EncodingAESKey string `yaml:"encoding_aes_key"` // 应用回调加解密密钥（43 字符 Base64）
}

// MockConfig Mock 数据生成配置
type MockConfig struct {
	Enabled    bool     `yaml:"enabled"`
	Namespaces []string `yaml:"namespaces"` // 模拟的 K8s namespace 列表（原 services 改为 namespaces）
}

// ReplayConfig 回放功能配置
type ReplayConfig struct {
	Enabled               bool          `yaml:"enabled"`
	DefaultFaultIntensity float64       `yaml:"default_fault_intensity"`
	DefaultTrafficRate    float64       `yaml:"default_traffic_rate"`
	MaxDuration           time.Duration `yaml:"max_duration"`
	AutoDiagnose          bool          `yaml:"auto_diagnose"`
}

// MonitorConfig 日志监控配置
type MonitorConfig struct {
	Enabled   bool          `yaml:"enabled"`
	Interval  time.Duration `yaml:"interval"`  // 扫描间隔，默认 30s
	Cooldown  time.Duration `yaml:"cooldown"`  // 冷却时间，默认 5m
	Threshold int           `yaml:"threshold"` // ERROR 触发阈值，默认 5
}

// LiveConfig 实时日志生成配置
type LiveConfig struct {
	RPS       int     `yaml:"rps"`        // 每秒请求数，默认 5
	FaultRate float64 `yaml:"fault_rate"` // 故障概率，默认 0.1
}
