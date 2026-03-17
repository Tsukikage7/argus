package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Tsukikage7/argus/internal/domain/tool"
)

// ExecCommandTool 在目标机器执行命令（MVP 阶段为模拟执行）
type ExecCommandTool struct {
	dryRun bool // MVP 阶段仅模拟
}

// NewExecCommandTool 创建命令执行工具
func NewExecCommandTool(dryRun bool) *ExecCommandTool {
	return &ExecCommandTool{dryRun: dryRun}
}

func (t *ExecCommandTool) Name() string { return "exec_command" }

func (t *ExecCommandTool) Description() string {
	return "在目标服务器执行运维命令，如重启服务、查看系统状态、清理磁盘等。需要指定目标主机和要执行的命令。"
}

func (t *ExecCommandTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"host": {
				"type": "string",
				"description": "目标主机名或 IP"
			},
			"command": {
				"type": "string",
				"description": "要执行的 shell 命令"
			}
		},
		"required": ["host", "command"]
	}`)
}

func (t *ExecCommandTool) Execute(ctx context.Context, params map[string]any) (*tool.Result, error) {
	host, _ := params["host"].(string)
	command, _ := params["command"].(string)

	if host == "" || command == "" {
		return &tool.Result{Error: "host and command are required"}, nil
	}

	if t.dryRun {
		return t.simulateExec(host, command)
	}

	// TODO: 实际 SSH 执行
	return &tool.Result{Error: "real SSH execution not implemented yet"}, nil
}

func (t *ExecCommandTool) simulateExec(host, command string) (*tool.Result, error) {
	// 模拟常见运维命令的输出
	switch {
	case strings.Contains(command, "restart") || strings.Contains(command, "systemctl restart"):
		svc := extractServiceName(command)
		return &tool.Result{
			Output: fmt.Sprintf("[%s] Service %s restarted successfully. PID: 12345", host, svc),
		}, nil

	case strings.Contains(command, "df -h"):
		return &tool.Result{
			Output: fmt.Sprintf("[%s] Filesystem      Size  Used Avail Use%%\n/dev/sda1       100G   85G   15G  85%%", host),
		}, nil

	case strings.Contains(command, "free -m"):
		return &tool.Result{
			Output: fmt.Sprintf("[%s] Mem:   16384   14500    1884     256    2048   3200", host),
		}, nil

	case strings.Contains(command, "kill") || strings.Contains(command, "pkill"):
		return &tool.Result{
			Output: fmt.Sprintf("[%s] Process terminated successfully", host),
		}, nil

	default:
		return &tool.Result{
			Output: fmt.Sprintf("[%s] [dry-run] Would execute: %s", host, command),
		}, nil
	}
}

func extractServiceName(cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown"
}

var _ tool.Tool = (*ExecCommandTool)(nil)
