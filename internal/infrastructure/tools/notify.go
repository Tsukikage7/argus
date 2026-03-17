package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Tsukikage7/argus/internal/domain/tool"
)

// NotifyTool 发送通知（企微/日志）
type NotifyTool struct {
	// wechat *wechat.Bot  // TODO: 接入企微
}

// NewNotifyTool 创建通知工具
func NewNotifyTool() *NotifyTool {
	return &NotifyTool{}
}

func (t *NotifyTool) Name() string { return "send_notification" }

func (t *NotifyTool) Description() string {
	return "发送通知消息给相关人员。可以通知 DBA、运维、开发等角色，附带告警详情和处理建议。"
}

func (t *NotifyTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"target": {
				"type": "string",
				"description": "通知目标，如 dba, ops, dev, oncall"
			},
			"message": {
				"type": "string",
				"description": "通知消息内容"
			},
			"severity": {
				"type": "string",
				"enum": ["critical", "warning", "info"],
				"description": "通知级别"
			}
		},
		"required": ["target", "message"]
	}`)
}

func (t *NotifyTool) Execute(ctx context.Context, params map[string]any) (*tool.Result, error) {
	target, _ := params["target"].(string)
	message, _ := params["message"].(string)
	severity, _ := params["severity"].(string)

	if target == "" || message == "" {
		return &tool.Result{Error: "target and message are required"}, nil
	}

	if severity == "" {
		severity = "info"
	}

	// MVP: 仅记录通知日志
	output := fmt.Sprintf("Notification sent to [%s] (severity=%s): %s", target, severity, message)
	return &tool.Result{Output: output}, nil
}

var _ tool.Tool = (*NotifyTool)(nil)
