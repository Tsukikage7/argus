package handler

import (
	"net/http"
	"strings"

	"github.com/Tsukikage7/argus/internal/domain/task"
	"github.com/Tsukikage7/argus/internal/infrastructure/es"
	"github.com/Tsukikage7/argus/internal/infrastructure/mock"
	httputil "github.com/Tsukikage7/argus/internal/interfaces/http"
)

// TopologyHandler 返回服务拓扑配置（供前端动态获取）
type TopologyHandler struct{}

// TopologyGraphHandler 返回带健康度的服务拓扑图
type TopologyGraphHandler struct {
	es *es.Client
}

// NewTopologyHandler 创建拓扑 HTTP 处理器
func NewTopologyHandler() *TopologyHandler {
	return &TopologyHandler{}
}

// NewTopologyGraphHandler 创建拓扑图 HTTP 处理器
func NewTopologyGraphHandler(esClient *es.Client) *TopologyGraphHandler {
	return &TopologyGraphHandler{es: esClient}
}

type topologyResponse struct {
	Services []string            `json:"services"`
	Edges    [][2]string         `json:"edges"`
	Chains   map[string][]string `json:"chains"`
}

type topologyGraphResponse struct {
	Nodes []topologyGraphNode `json:"nodes"`
	Edges []topologyGraphEdge `json:"edges"`
}

type topologyGraphNode struct {
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	Health     string  `json:"health"`
	ErrorRate  float64 `json:"error_rate"`
	AlertCount int     `json:"alert_count"`
}

type topologyGraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Weight int    `json:"weight"`
}

// ServeHTTP GET /api/v1/topology
func (h *TopologyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	topo := mock.Topology()
	services := make([]string, len(topo))
	for i, s := range topo {
		services[i] = s.Namespace
	}

	httputil.WriteJSON(w, http.StatusOK, topologyResponse{
		Services: services,
		Edges:    topologyEdges(),
		Chains:   topologyChains(),
	})
}

// ServeHTTP GET /api/v1/topology/graph
func (h *TopologyGraphHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "仅支持 GET")
		return
	}

	p := task.PrincipalFrom(r.Context())
	if p == nil {
		httputil.WriteError(w, http.StatusUnauthorized, httputil.CodeUnauthorized, "未授权")
		return
	}

	metrics := map[string]es.TopologyNodeMetric{}
	if h.es != nil {
		if queried, err := h.es.QueryTopologyNodeMetrics(r.Context(), p.TenantID, "last 1h"); err == nil {
			metrics = queried
		}
	}

	httputil.WriteJSON(w, http.StatusOK, buildTopologyGraphResponse(metrics))
}

func buildTopologyGraphResponse(metrics map[string]es.TopologyNodeMetric) topologyGraphResponse {
	topo := mock.Topology()
	nodes := make([]topologyGraphNode, 0, len(topo))
	for _, svc := range topo {
		metric, ok := metrics[svc.Namespace]
		if !ok {
			metric = es.TopologyNodeMetric{
				Namespace:  svc.Namespace,
				Health:     "healthy",
				ErrorRate:  0,
				AlertCount: 0,
			}
		}

		nodes = append(nodes, topologyGraphNode{
			ID:         svc.Namespace,
			Label:      strings.TrimPrefix(svc.Namespace, "prj-"),
			Health:     metric.Health,
			ErrorRate:  metric.ErrorRate,
			AlertCount: metric.AlertCount,
		})
	}

	edges := make([]topologyGraphEdge, 0, len(topologyEdges()))
	for _, edge := range topologyEdges() {
		edges = append(edges, topologyGraphEdge{
			Source: edge[0],
			Target: edge[1],
			Weight: 100,
		})
	}

	return topologyGraphResponse{
		Nodes: nodes,
		Edges: edges,
	}
}

func topologyEdges() [][2]string {
	return [][2]string{
		{"prj-apigateway", "prj-ubill"},
		{"prj-apigateway", "prj-uresource"},
		{"prj-apigateway", "prj-uhost"},
		{"prj-ubill", "prj-uresource"},
		{"prj-uresource", "prj-unet"},
		{"prj-uhost", "prj-udb"},
	}
}

func topologyChains() map[string][]string {
	return map[string][]string{
		"prj-ubill":      {"prj-apigateway", "prj-ubill"},
		"prj-uresource":  {"prj-apigateway", "prj-uresource"},
		"prj-uhost":      {"prj-apigateway", "prj-uhost"},
		"prj-unet":       {"prj-apigateway", "prj-uresource", "prj-unet"},
		"prj-udb":        {"prj-apigateway", "prj-uhost", "prj-udb"},
		"prj-apigateway": {"prj-apigateway"},
	}
}
