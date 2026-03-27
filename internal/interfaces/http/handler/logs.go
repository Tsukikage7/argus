package handler

import (
	"net/http"
	"strconv"

	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/Tsukikage7/argus/internal/infrastructure/es"
	httputil "github.com/Tsukikage7/argus/internal/interfaces/http"
)

// LogsHandler 处理日志查询 API
type LogsHandler struct {
	es *es.Client
}

// NewLogsHandler 创建日志查询 HTTP 处理器
func NewLogsHandler(esClient *es.Client) *LogsHandler {
	return &LogsHandler{es: esClient}
}

// ServeHTTP 路由 GET /api/v1/logs
func (h *LogsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "仅支持 GET")
		return
	}

	p := task.PrincipalFrom(r.Context())
	if p == nil {
		httputil.WriteError(w, http.StatusUnauthorized, httputil.CodeUnauthorized, "未授权")
		return
	}

	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))

	opts := es.LogQueryOpts{
		Namespace: q.Get("namespace"),
		Keyword:   q.Get("keyword"),
		Level:     q.Get("level"),
		TimeRange: q.Get("time_range"),
		Limit:     limit,
	}

	logs, err := h.es.QueryLogs(r.Context(), p.TenantID, opts)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, "日志查询失败")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, logs)
}

// LogFaultsHandler 处理故障日志查询 API
type LogFaultsHandler struct {
	es *es.Client
}

func NewLogFaultsHandler(esClient *es.Client) *LogFaultsHandler {
	return &LogFaultsHandler{es: esClient}
}

func (h *LogFaultsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "仅支持 GET")
		return
	}
	p := task.PrincipalFrom(r.Context())
	if p == nil {
		httputil.WriteError(w, http.StatusUnauthorized, httputil.CodeUnauthorized, "未授权")
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	opts := es.LogQueryOpts{
		Namespace: q.Get("namespace"),
		Service:   q.Get("service"),
		Keyword:   q.Get("keyword"),
		Level:     q.Get("level"),
		TimeRange: q.Get("time_range"),
		Limit:     limit,
	}
	result, err := h.es.QueryFaultLogs(r.Context(), p.TenantID, opts)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, "故障日志查询失败")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, result)
}

// LogContextHandler 处理日志上下文查询 API
type LogContextHandler struct {
	es *es.Client
}

func NewLogContextHandler(esClient *es.Client) *LogContextHandler {
	return &LogContextHandler{es: esClient}
}

func (h *LogContextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "仅支持 GET")
		return
	}
	p := task.PrincipalFrom(r.Context())
	if p == nil {
		httputil.WriteError(w, http.StatusUnauthorized, httputil.CodeUnauthorized, "未授权")
		return
	}
	requestUUID := r.URL.Query().Get("request_uuid")
	if requestUUID == "" {
		httputil.WriteError(w, http.StatusBadRequest, httputil.CodeValidation, "request_uuid 参数必填")
		return
	}
	timeRange := r.URL.Query().Get("time_range")
	result, err := h.es.QueryLogContext(r.Context(), p.TenantID, requestUUID, timeRange)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, "日志上下文查询失败")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, result)
}

// LogFacetsHandler 处理日志分面聚合 API
type LogFacetsHandler struct {
	es *es.Client
}

func NewLogFacetsHandler(esClient *es.Client) *LogFacetsHandler {
	return &LogFacetsHandler{es: esClient}
}

func (h *LogFacetsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "仅支持 GET")
		return
	}
	p := task.PrincipalFrom(r.Context())
	if p == nil {
		httputil.WriteError(w, http.StatusUnauthorized, httputil.CodeUnauthorized, "未授权")
		return
	}
	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "last 1h"
	}
	facets, err := h.es.QueryLogFacets(r.Context(), p.TenantID, timeRange)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, "日志分面查询失败")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, facets)
}

// LogSummaryHandler 处理日志聚合摘要 API
type LogSummaryHandler struct {
	es *es.Client
}

func NewLogSummaryHandler(esClient *es.Client) *LogSummaryHandler {
	return &LogSummaryHandler{es: esClient}
}

func (h *LogSummaryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "仅支持 GET")
		return
	}
	p := task.PrincipalFrom(r.Context())
	if p == nil {
		httputil.WriteError(w, http.StatusUnauthorized, httputil.CodeUnauthorized, "未授权")
		return
	}
	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "last 1h"
	}
	summary, err := h.es.QueryLogSummary(r.Context(), p.TenantID, timeRange)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, "日志摘要查询失败")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, summary)
}
