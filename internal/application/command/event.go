package command

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AlertEvent 告警事件（来自企微 Webhook 或监控系统）
type AlertEvent struct {
	AlertName   string `json:"alert_name"`
	Service     string `json:"service"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Source      string `json:"source"`    // prometheus / custom / wechat
	TenantID    string `json:"tenant_id"` // 租户 ID（可选，由中间件注入）
}

// AlertEventHandler 处理告警事件（含去重）
type AlertEventHandler struct {
	diagnoseHandler *DiagnoseHandler
	mu              sync.Mutex
	dedup           map[string]time.Time // key -> 上次触发时间
	dedupTTL        time.Duration
}

// NewAlertEventHandler 创建告警事件处理器
func NewAlertEventHandler(dh *DiagnoseHandler) *AlertEventHandler {
	return &AlertEventHandler{
		diagnoseHandler: dh,
		dedup:           make(map[string]time.Time),
		dedupTTL:        5 * time.Minute,
	}
}

// dedupKey 生成去重键
func dedupKey(event AlertEvent) string {
	return fmt.Sprintf("%s:%s:%s", event.TenantID, event.AlertName, event.Service)
}

// Handle 处理告警事件 → 转化为诊断命令（含去重）
func (h *AlertEventHandler) Handle(ctx context.Context, event AlertEvent) (string, error) {
	// 告警去重：同一租户+告警名+服务在 TTL 内不重复触发
	key := dedupKey(event)
	h.mu.Lock()
	if last, ok := h.dedup[key]; ok && time.Since(last) < h.dedupTTL {
		h.mu.Unlock()
		return "", fmt.Errorf("告警去重: %s 在 %v 内已触发过", key, h.dedupTTL)
	}
	// 清理过期条目
	for k, t := range h.dedup {
		if time.Since(t) > h.dedupTTL {
			delete(h.dedup, k)
		}
	}
	h.mu.Unlock()

	input := event.Description
	if input == "" {
		input = event.AlertName + " on " + event.Service
	}

	t, err := h.diagnoseHandler.Handle(ctx, DiagnoseCommand{
		TenantID: event.TenantID,
		Input:    input,
		Source:   "webhook:" + event.Source,
	})
	if err != nil {
		return "", err
	}

	// 诊断成功后才记录去重时间戳
	h.mu.Lock()
	h.dedup[key] = time.Now()
	h.mu.Unlock()

	return t.ID, nil
}
