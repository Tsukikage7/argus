package agent

import (
	"context"
	"fmt"
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

// Verify 执行恢复后验证：查询目标服务最近日志，确认错误消失
func (v *Verifier) Verify(ctx context.Context, t *task.Task) error {
	if t.Diagnosis == nil || len(t.Diagnosis.AffectedServices) == 0 {
		return nil
	}

	esQueryTool, err := v.tools.Get("es_query_logs")
	if err != nil {
		return fmt.Errorf("verifier: %w", err)
	}

	// 查询受影响服务最近 2 分钟的 ERROR 日志
	for _, svc := range t.Diagnosis.AffectedServices {
		result, err := esQueryTool.Execute(ctx, map[string]any{
			"service":    svc,
			"severity":   "ERROR",
			"time_range": "last 2m",
		})
		if err != nil {
			return fmt.Errorf("verifier: query %s failed: %w", svc, err)
		}

		// 如果仍有错误日志，验证失败
		if result.Error != "" {
			if t.Recovery != nil {
				t.Recovery.Status = task.RecoveryFailed
			}
			return fmt.Errorf("verifier: %s still has errors: %s", svc, result.Output)
		}
	}

	if t.Recovery != nil {
		t.Recovery.Status = task.RecoverySuccess
		now := time.Now()
		t.Recovery.VerifiedAt = &now
	}
	return nil
}
