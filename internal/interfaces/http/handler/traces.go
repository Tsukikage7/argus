package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/Tsukikage7/argus/internal/infrastructure/es"
	httputil "github.com/Tsukikage7/argus/internal/interfaces/http"
)

// TracesHandler 处理 GET /api/v1/traces（Trace 列表）
type TracesHandler struct {
	es *es.Client
}

func NewTracesHandler(esClient *es.Client) *TracesHandler {
	return &TracesHandler{es: esClient}
}

func (h *TracesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	opts := es.TraceQueryOpts{
		RequestUUID: q.Get("request_uuid"),
		Service:     q.Get("service"),
		TimeRange:   q.Get("time_range"),
		Limit:       limit,
	}

	result, err := h.es.QueryTraces(r.Context(), p.TenantID, opts)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, "链路查询失败")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, result)
}

// TraceDetailHandler 处理 GET /api/v1/traces/{uuid}（Trace 详情）
type TraceDetailHandler struct {
	es *es.Client
}

func NewTraceDetailHandler(esClient *es.Client) *TraceDetailHandler {
	return &TraceDetailHandler{es: esClient}
}

func (h *TraceDetailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "仅支持 GET")
		return
	}
	p := task.PrincipalFrom(r.Context())
	if p == nil {
		httputil.WriteError(w, http.StatusUnauthorized, httputil.CodeUnauthorized, "未授权")
		return
	}

	uuid := r.PathValue("uuid")
	if uuid == "" {
		httputil.WriteError(w, http.StatusBadRequest, httputil.CodeValidation, "uuid 参数必填")
		return
	}

	timeRange := r.URL.Query().Get("time_range")
	detail, err := h.es.QueryTraceDetail(r.Context(), p.TenantID, uuid, timeRange)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			httputil.WriteError(w, http.StatusNotFound, httputil.CodeNotFound, "链路不存在")
			return
		}
		httputil.WriteError(w, http.StatusInternalServerError, httputil.CodeInternal, "链路详情查询失败")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, detail)
}

// FlameNode 火焰图节点
type FlameNode struct {
	Name     string      `json:"name"`
	Value    int         `json:"value"`
	Children []FlameNode `json:"children"`
}

// FlameGraphResponse 火焰图响应
type FlameGraphResponse struct {
	Root FlameNode `json:"root"`
}

// TraceFlameGraphHandler 处理 GET /api/v1/traces/{uuid}/flamegraph（mock）
type TraceFlameGraphHandler struct{}

func NewTraceFlameGraphHandler() *TraceFlameGraphHandler {
	return &TraceFlameGraphHandler{}
}

func (h *TraceFlameGraphHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "仅支持 GET")
		return
	}
	p := task.PrincipalFrom(r.Context())
	if p == nil {
		httputil.WriteError(w, http.StatusUnauthorized, httputil.CodeUnauthorized, "未授权")
		return
	}

	// Mock 火焰图数据
	resp := FlameGraphResponse{
		Root: FlameNode{
			Name:  "gateway",
			Value: 5200,
			Children: []FlameNode{
				{
					Name:  "order-service",
					Value: 4800,
					Children: []FlameNode{
						{
							Name:  "payment-service.processPayment",
							Value: 4500,
							Children: []FlameNode{
								{Name: "db.query", Value: 4200, Children: nil},
							},
						},
					},
				},
				{
					Name:     "user-service.validateToken",
					Value:    200,
					Children: nil,
				},
			},
		},
	}
	httputil.WriteJSON(w, http.StatusOK, resp)
}
