// Package wechat 提供企业微信 Bot 集成
package wechat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Bot 企业微信 Bot 客户端
type Bot struct {
	webhookURL string
}

// NewBot 创建企微 Bot
func NewBot(webhookURL string) *Bot {
	return &Bot{webhookURL: webhookURL}
}

// SendMarkdown 发送 Markdown 消息
func (b *Bot) SendMarkdown(ctx context.Context, content string) error {
	if b.webhookURL == "" {
		return nil // 未配置则跳过
	}

	payload := map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": content,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("wechat: send failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wechat: unexpected status %d", resp.StatusCode)
	}
	return nil
}
