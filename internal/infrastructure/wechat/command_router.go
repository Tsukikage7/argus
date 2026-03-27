// Package wechat 提供企业微信命令路由与消息格式化
package wechat

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// TextMessage 解析后的企微文本消息
type TextMessage struct {
	FromUserID string // 消息发送者 UserID
	Content    string // 消息文本内容
	MsgID      string // 消息 ID
	AgentID    int    // 接收消息的应用 ID
}

// textMessageXML 企微推送文本消息的 XML 结构
type textMessageXML struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	MsgId        string   `xml:"MsgId"`
	AgentID      int      `xml:"AgentID"`
}

// ParseTextMessage 从解密后的 XML 字节流解析文本消息
func ParseTextMessage(xmlData []byte) (*TextMessage, error) {
	var raw textMessageXML
	if err := xml.Unmarshal(xmlData, &raw); err != nil {
		return nil, fmt.Errorf("wechat: 解析文本消息 XML 失败: %w", err)
	}

	if raw.MsgType != "text" {
		return nil, fmt.Errorf("wechat: 非文本消息类型: %s", raw.MsgType)
	}

	return &TextMessage{
		FromUserID: raw.FromUserName,
		Content:    strings.TrimSpace(raw.Content),
		MsgID:      raw.MsgId,
		AgentID:    raw.AgentID,
	}, nil
}

// CommandRouter 解析企微消息并路由到对应处理动作
type CommandRouter struct{}

// NewCommandRouter 创建命令路由器
func NewCommandRouter() *CommandRouter {
	return &CommandRouter{}
}

// Route 路由消息到对应处理动作
// 返回 action（"diagnose" 或 "help"）和 input（诊断输入内容）
//
// 支持的命令格式：
//   - "诊断 <内容>"     → action=diagnose, input=<内容>
//   - "帮助" / "help"  → action=help
//   - 其他任意文本       → action=diagnose, input=原始文本
func (r *CommandRouter) Route(msg *TextMessage) (action string, input string) {
	content := msg.Content

	// 归一化：去除全角空格
	content = strings.ReplaceAll(content, "\u3000", " ")

	lower := strings.ToLower(content)

	switch {
	case lower == "帮助" || lower == "help" || lower == "?" || lower == "？":
		return "help", ""

	case strings.HasPrefix(lower, "诊断 ") || strings.HasPrefix(lower, "诊断\t"):
		// "诊断 <内容>" → 提取内容部分
		idx := strings.IndexAny(content, " \t")
		if idx >= 0 {
			return "diagnose", strings.TrimSpace(content[idx+1:])
		}
		return "diagnose", content

	default:
		// 其他文本直接作为诊断输入
		return "diagnose", content
	}
}

// HelpText 返回帮助信息
func HelpText() string {
	return `**Argus 智能诊断助手**

支持以下命令：
- **诊断 <描述>** — 触发 AI 诊断，例如：诊断 prj-ubill 连接池耗尽
- **帮助 / help** — 显示此帮助信息

也可直接发送故障描述，Argus 会自动触发诊断。`
}

// FormatPendingReply 生成收到消息后的即时回复内容（被动回复用）
func FormatPendingReply(input string) string {
	if len(input) > 50 {
		input = input[:50] + "..."
	}
	return fmt.Sprintf("收到，正在诊断：%s\n\n请稍候，诊断完成后将推送结论卡片。", input)
}
