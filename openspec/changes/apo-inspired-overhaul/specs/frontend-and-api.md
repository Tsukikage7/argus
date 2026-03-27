# Spec: 前端页面与后端 API 规格

## 1. 前端页面规格

### 1.1 AppLayout 布局
- 侧边栏宽度：折叠 64px / 展开 240px
- 顶部栏高度：56px
- 侧边栏分组：概览（Dashboard）/ 可观测（Topology, Logs, Traces, Alerts）/ 诊断（Diagnose, Replay, Tasks）/ 管理（Settings, Admin）
- 响应式：< 768px 时侧边栏默认折叠

### 1.2 Dashboard 总览页
- 统计卡片：4 张（总服务数、活跃告警、今日诊断、平均诊断耗时）
- 服务健康网格：每行 3 张卡片，显示服务名/状态/错误率/P99延迟
- 告警摘要：按 critical/warning/info 分组计数 + 最近 5 条告警
- 最近诊断：最近 5 条诊断任务（状态/根因摘要/耗时/置信度）
- RED 图表：时间序列折线图，支持 15m/1h/6h/24h 切换

### 1.3 服务拓扑页
- 全屏拓扑图（X6 Dagre 自动布局）
- 节点：服务名 + 健康状态圆点（绿/黄/红）+ 错误率标签
- 边：调用关系，线宽表示流量大小
- 右侧详情面板（点击节点展开）：服务信息/端点列表/最近日志/告警
- 支持缩放、拖拽、全屏

### 1.4 日志探索器
- 左侧分面过滤：namespace/service/level/pod 聚合计数，点击过滤
- 中间日志列表：虚拟滚动，列（时间戳/级别/服务/消息），行展开查看完整内容
- 右侧上下文抽屉：按 request_uuid 展示前后文日志
- 搜索栏：关键词 + 级别多选 + 时间范围（联动全局）

### 1.5 链路追踪页
- Trace 列表：request_uuid/入口服务/状态码/总耗时/时间戳，支持排序
- Trace 详情：水平瀑布图（各服务调用耗时）+ 关联日志列表
- 火焰图 Tab：SVG 渲染（mock 数据）

### 1.6 告警仪表盘
- 告警事件列表：时间/严重度/服务/描述/状态/关联诊断任务
- 严重度分布图：饼图（critical/warning/info）
- 告警详情抽屉：告警信息 + 关联诊断 + 跳转按钮

### 1.7 效果验证页
- 效率对比面板：AI 诊断 vs 人工诊断（耗时/步骤数/准确率）
- 诊断统计图表：成功率趋势、平均耗时趋势、场景覆盖率
- 一键演示按钮：触发完整演示流程

### 1.8 能力沉淀页
- Agent 架构图：ReAct 循环动态展示
- Tool 注册表：列出所有 Tool 及能力描述
- 场景库：已沉淀场景列表 + 详情

## 2. 后端 API 规格

### 2.1 GET /api/v1/dashboard/summary（mock）
```json
{
  "total_services": 6,
  "active_alerts": 3,
  "today_diagnoses": 12,
  "avg_diagnose_time_seconds": 45,
  "service_health": [
    {"name": "gateway", "status": "healthy", "error_rate": 0.01, "p99_latency_ms": 120}
  ],
  "recent_alerts": [
    {"id": "a1", "severity": "critical", "service": "payment-service", "message": "...", "time": "..."}
  ],
  "recent_diagnoses": [
    {"task_id": "t1", "status": "completed", "root_cause": "...", "duration_seconds": 38, "confidence": 0.92}
  ]
}
```

### 2.2 GET /api/v1/topology/graph（real，增强）
```json
{
  "nodes": [
    {"id": "gateway", "label": "gateway", "health": "healthy", "error_rate": 0.01, "alert_count": 0}
  ],
  "edges": [
    {"source": "gateway", "target": "user-service", "weight": 100}
  ]
}
```

### 2.3 GET /api/v1/logs/faults（real）
```json
{
  "total": 156,
  "logs": [
    {"id": "l1", "timestamp": "...", "level": "ERROR", "service": "payment-service", "message": "...", "request_uuid": "..."}
  ]
}
```

### 2.4 GET /api/v1/logs/context?request_uuid=xxx（real）
```json
{
  "request_uuid": "xxx",
  "logs": [
    {"timestamp": "...", "level": "INFO", "service": "gateway", "message": "..."}
  ]
}
```

### 2.5 GET /api/v1/logs/facets（real）
```json
{
  "namespaces": [{"name": "prj-apigateway", "count": 500}],
  "services": [{"name": "gateway", "count": 500}],
  "levels": [{"name": "ERROR", "count": 156}, {"name": "WARN", "count": 89}],
  "pods": [{"name": "gateway-pod-1", "count": 250}]
}
```

### 2.6 GET /api/v1/traces（real）
```json
{
  "total": 89,
  "traces": [
    {"request_uuid": "xxx", "entry_service": "gateway", "status_code": 504, "duration_ms": 5200, "timestamp": "...", "services": ["gateway", "order-service", "payment-service"]}
  ]
}
```

### 2.7 GET /api/v1/traces/{uuid}（real）
```json
{
  "request_uuid": "xxx",
  "spans": [
    {"service": "gateway", "operation": "POST /api/orders", "start_ms": 0, "duration_ms": 5200, "status": "error", "logs": [...]}
  ]
}
```

### 2.8 GET /api/v1/traces/{uuid}/flamegraph（mock）
```json
{
  "root": {"name": "gateway", "value": 5200, "children": [
    {"name": "order-service", "value": 4800, "children": [
      {"name": "payment-service.processPayment", "value": 4500, "children": [
        {"name": "db.query", "value": 4200, "children": []}
      ]}
    ]}
  ]}
}
```

### 2.9 GET /api/v1/alerts/active（mock）
```json
{
  "total": 5,
  "alerts": [
    {"id": "a1", "fingerprint": "fp1", "severity": "critical", "service": "payment-service", "message": "...", "status": "firing", "starts_at": "...", "task_id": "t1"}
  ]
}
```

### 2.10 GET /api/v1/stats/efficiency（mock）
```json
{
  "ai_avg_time_seconds": 45,
  "manual_avg_time_seconds": 1800,
  "ai_avg_steps": 8,
  "manual_avg_steps": 25,
  "ai_accuracy": 0.85,
  "scenarios_covered": 3,
  "total_diagnoses": 47,
  "time_saved_hours": 12.5
}
```
