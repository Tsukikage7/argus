package wechat

import (
	"fmt"
	"strings"

	"github.com/Tsukikage7/argus/internal/domain/task"
)

// FormatDiagnosisCard 将诊断结果格式化为企微 Markdown 卡片
func FormatDiagnosisCard(t *task.Task) string {
	var sb strings.Builder

	sb.WriteString("## 🔍 Argus 诊断报告\n\n")
	sb.WriteString(fmt.Sprintf("**任务ID**: %s\n", t.ID))
	sb.WriteString(fmt.Sprintf("**输入**: %s\n", t.Input))
	sb.WriteString(fmt.Sprintf("**状态**: %s\n\n", t.Status))

	if t.Diagnosis != nil {
		d := t.Diagnosis
		sb.WriteString(fmt.Sprintf("### 根因\n%s\n\n", d.RootCause))
		sb.WriteString(fmt.Sprintf("**置信度**: %.0f%%\n", d.Confidence*100))
		sb.WriteString(fmt.Sprintf("**影响服务**: %s\n", strings.Join(d.AffectedServices, ", ")))
		sb.WriteString(fmt.Sprintf("**影响范围**: %s\n\n", d.Impact))

		if len(d.Suggestions) > 0 {
			sb.WriteString("### 恢复建议\n")
			for i, s := range d.Suggestions {
				sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, s))
			}
		}
	}

	if t.Recovery != nil {
		sb.WriteString(fmt.Sprintf("\n### 恢复状态: %s\n", t.Recovery.Status))
	}

	return sb.String()
}
