package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Tsukikage7/argus/internal/application/query"
)

// TaskHandler 处理任务查询 API
type TaskHandler struct {
	taskQuery   *query.TaskStatusHandler
	historyQuery *query.HistoryHandler
}

// NewTaskHandler 创建任务查询 HTTP 处理器
func NewTaskHandler(tq *query.TaskStatusHandler, hq *query.HistoryHandler) *TaskHandler {
	return &TaskHandler{taskQuery: tq, historyQuery: hq}
}

// ServeHTTP 路由 GET /api/v1/tasks/:id 和 GET /api/v1/tasks
func (h *TaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// 通过 path pattern 获取 task ID，回退到手动解析
	taskID := r.PathValue("id")
	if taskID == "" {
		// 列出历史
		h.listHistory(w, r)
		return
	}

	// 查询单个任务
	h.getTask(w, r, taskID)
}

func (h *TaskHandler) getTask(w http.ResponseWriter, r *http.Request, taskID string) {
	t, err := h.taskQuery.Handle(r.Context(), query.TaskStatusQuery{TaskID: taskID})
	if err != nil {
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

func (h *TaskHandler) listHistory(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.historyQuery.Handle(r.Context(), query.HistoryQuery{Limit: 20})
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}
