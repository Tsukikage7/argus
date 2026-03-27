// Package handler 提供 HTTP 请求处理器
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Tsukikage7/argus/internal/domain/task"
)

// ExportTaskRepository 导出功能需要的任务读取接口
type ExportTaskRepository interface {
	Get(ctx context.Context, tenantID, id string) (*task.Task, error)
}

// ExportHandler 处理诊断报告导出的 API 请求
type ExportHandler struct {
	taskRepo ExportTaskRepository
}

// NewExportHandler 创建导出 HTTP 处理器
func NewExportHandler(repo ExportTaskRepository) *ExportHandler {
	return &ExportHandler{taskRepo: repo}
}

// ServeHTTP 处理 GET /api/v1/tasks/{id}/export
func (h *ExportHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	taskID := r.PathValue("id")
	if taskID == "" {
		http.Error(w, `{"error":"task id is required"}`, http.StatusBadRequest)
		return
	}

	// 查询参数决定导出格式，默认为 markdown
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "markdown"
	}

	t, err := h.taskRepo.Get(r.Context(), task.PrincipalFrom(r.Context()).TenantID, taskID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "task not found"})
		return
	}

	switch format {
	case "json":
		h.exportJSON(w, t)
	default:
		h.exportMarkdown(w, t)
	}
}

// exportMarkdown 将任务以 Markdown 格式输出
func (h *ExportHandler) exportMarkdown(w http.ResponseWriter, t *task.Task) {
	filename := fmt.Sprintf("argus-report-%s.md", t.ID)
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	var sb strings.Builder

	// 基本信息
	sb.WriteString("# Argus 诊断报告\n\n")
	sb.WriteString("## 基本信息\n\n")
	sb.WriteString(fmt.Sprintf("- 任务 ID: %s\n", t.ID))
	sb.WriteString(fmt.Sprintf("- 输入: %s\n", t.Input))
	sb.WriteString(fmt.Sprintf("- 来源: %s\n", t.Source))
	sb.WriteString(fmt.Sprintf("- 状态: %s\n", t.Status))
	sb.WriteString(fmt.Sprintf("- 创建时间: %s\n", t.CreatedAt.Format("2006-01-02 15:04:05")))
	if t.CompletedAt != nil {
		sb.WriteString(fmt.Sprintf("- 完成时间: %s\n", t.CompletedAt.Format("2006-01-02 15:04:05")))
	}
	sb.WriteString("\n")

	// 诊断结论
	if t.Diagnosis != nil {
		d := t.Diagnosis
		sb.WriteString("## 诊断结论\n\n")
		sb.WriteString(fmt.Sprintf("- **根因**: %s\n", d.RootCause))
		sb.WriteString(fmt.Sprintf("- **置信度**: %.0f%%\n", d.Confidence*100))
		sb.WriteString(fmt.Sprintf("- **影响范围**: %s\n", d.Impact))
		sb.WriteString(fmt.Sprintf("- **受影响服务**: %s\n", strings.Join(d.AffectedServices, ", ")))

		if len(d.Suggestions) > 0 {
			sb.WriteString("\n### 恢复建议\n\n")
			for i, s := range d.Suggestions {
				sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, s))
			}
		}
		sb.WriteString("\n")
	}

	// 推理过程
	if len(t.Steps) > 0 {
		sb.WriteString("## 推理过程\n\n")
		for _, step := range t.Steps {
			sb.WriteString(fmt.Sprintf("### Step %d (%s)\n\n", step.Index, step.Timestamp.Format("15:04:05")))

			if step.Think != "" {
				sb.WriteString(fmt.Sprintf("**Think**: %s\n\n", step.Think))
			}

			if step.Action != nil {
				// 将工具参数序列化为简洁 JSON
				paramsBytes, _ := json.Marshal(step.Action.Params)
				sb.WriteString(fmt.Sprintf("**Action**: %s(%s)\n\n", step.Action.Tool, string(paramsBytes)))
			}

			if step.Observe != "" {
				sb.WriteString(fmt.Sprintf("**Observe**: %s\n\n", step.Observe))
			}
		}
	}

	// 恢复记录
	if t.Recovery != nil && len(t.Recovery.Actions) > 0 {
		sb.WriteString("## 恢复记录\n\n")
		for _, action := range t.Recovery.Actions {
			status := "成功"
			if !action.Success {
				status = "失败"
			}
			sb.WriteString(fmt.Sprintf("- %s: %s (%s)\n", action.Description, action.Result, status))
		}
		sb.WriteString("\n")
	}

	fmt.Fprint(w, sb.String())
}

// exportJSON 将任务以格式化 JSON 输出
func (h *ExportHandler) exportJSON(w http.ResponseWriter, t *task.Task) {
	filename := fmt.Sprintf("argus-report-%s.json", t.ID)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(t); err != nil {
		// 头部已发送，只能记录错误无法再返回 HTTP 错误
		_ = fmt.Errorf("export json encode: %w", err)
	}
}
