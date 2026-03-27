package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/Tsukikage7/argus/internal/domain/tool"
)

// Verifier 在恢复操作后验证系统是否恢复正常
type Verifier struct {
	tools *tool.Registry
}

// NewVerifier 创建验证器
func NewVerifier(tools *tool.Registry) *Verifier {
	return &Verifier{tools: tools}
}

// Verify 执行恢复后验证：查询受影响 namespace 最近日志，确认错误消失
func (v *Verifier) Verify(ctx context.Context, t *task.Task) error {
	if t.Diagnosis == nil || len(t.Diagnosis.AffectedServices) == 0 {
		return nil
	}

	esQueryTool, err := v.tools.Get("es_query_logs")
	if err != nil {
		return fmt.Errorf("verifier: %w", err)
	}

	// 查询受影响 namespace 最近 2 分钟的 ERROR 日志
	// affected_services 存储的是 namespace 名称（如 prj-ubill）
	for _, ns := range t.Diagnosis.AffectedServices {
		result, err := esQueryTool.Execute(ctx, map[string]any{
			"namespace":  ns,
			"keyword":    "ERROR",
			"time_range": "last 2m",
		})
		if err != nil {
			return fmt.Errorf("verifier: query %s failed: %w", ns, err)
		}

		// 如果仍有错误日志，验证失败
		// 注意：es_query_logs 查到日志时结果在 Output 中（"Found X log entries"），
		// Error 仅在查询本身失败时才有值，因此需要检查 Output 是否包含命中结果
		if result.Error != "" {
			return fmt.Errorf("verifier: query %s error: %s", ns, result.Error)
		}
		if strings.Contains(result.Output, "Found") && !strings.Contains(result.Output, "Found 0") {
			if t.Recovery != nil {
				t.Recovery.Status = task.RecoveryFailed
			}
			return fmt.Errorf("verifier: %s still has errors: %s", ns, result.Output)
		}
	}

	if t.Recovery != nil {
		t.Recovery.Status = task.RecoverySuccess
		now := time.Now()
		t.Recovery.VerifiedAt = &now
	}
	return nil
}
