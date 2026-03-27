# Tasks — 多地域 K8s 智能排障平台

## Phase 1: ES 基础设施升级

- [ ] 1.1 修改 `es/client.go` index target builder，支持 `{prefix}_{region}_{namespace}-{date}` 格式，新增 RegionIndex/RegionNamespaceIndex 方法
- [ ] 1.2 修改 `es/model.go` UCloudLog 结构体，新增 Region/GatewayLayer/RequestUUIDRoot/RequestUUIDFull/Semantic 规范化字段
- [ ] 1.3 修改 `es/message_parser.go` 新增 ExtractSemantic 函数，从 3 种日志格式统一提取 action/org_id/zone/region/ret_code/downstream_service/user/resource_id
- [ ] 1.4 修改 `es/message_parser.go` 新增 ExtractRequestUUIDRoot 函数，从 request_uuid_full 提取根 UUID
- [ ] 1.5 修改 `es/traceline.go` ParseTraceLine 支持逗号分隔函数段（如 `FuncA:0.1,FuncB:0.2`）
- [ ] 1.6 新增 `es/uuid_tree.go` 实现 request_uuid 树构建算法：BuildUUIDTree(logs) → Tree{Nodes, Timeline, Edges}
- [ ] 1.7 修改 `es/query.go` 查询方法支持 region/gateway_layer/root_request_uuid/semantic 过滤参数
- [ ] 1.8 修改 `es/query.go` parseUCloudLogResponse 保留 _index 信息，用于提取 region

## Phase 2: Mock 拓扑与场景升级

- [ ] 2.1 修改 `mock/topology.go` 扩展 4 层网关拓扑：L1 Access (gray-gateway-gw) / L2 IAM (gray-gateway-gw-iam) / L3 BizGateway (gray-gateway-gw-{product}-backend)，前三层同 prj-apigateway namespace
- [ ] 2.2 修改 `mock/topology.go` 新增 ServiceByGatewayLayer 查找方法，UCloudService 新增 GatewayLayer 字段
- [ ] 2.3 修改 `mock/scenarios.go` makeGatewayLog/makeTextLog/makeStructuredLog 辅助函数支持 region 和规范化字段写入
- [ ] 2.4 新增场景 CreateUHostInstanceFullChain：Access(root)→IAM(root.1)→UHost 业务网关(root.2)→UHost 服务→UResource(root.3)/UBill(root.24) 全链路，UResource 返回 RetCode!=0 导致创建失败
- [ ] 2.5 新增场景 IAMAuthFailed：Access(root)→IAM(root.1) 返回 x-api-retcode=161（权限不足），不进入业务网关和后端
- [ ] 2.6 新增场景 CascadeUBillFailure：Access→IAM→UHost 网关→UBill 子调用(root.24) DB 连接池耗尽→UHost 超时→网关 504
- [ ] 2.7 修改 `mock/generator.go` GenerateAll 支持多地域数据生成（为每个 region 生成独立索引数据）

## Phase 3: Tool 增强

- [ ] 3.1 修改 `tools/es_query.go` ESQueryLogsTool v2：新增 regions/gateway_layer/root_request_uuid/semantic_filters 参数，返回结构化 hits 含规范化字段
- [ ] 3.2 修改 `tools/trace_analyze.go` TraceAnalyzeTool v2：输入 root_request_uuid，输出 request_uuid 树 + 4 层路径 + trace-line 解析 + 语义汇总
- [ ] 3.3 修改 `tools/es_query.go` 和 `tools/trace_analyze.go` 的 Parameters() JSON Schema 更新

## Phase 4: Agent 推理增强

- [ ] 4.1 修改 `domain/agent/agent.go` systemPrompt：引导 LLM 按 L1→L2→L3→L4 固定顺序检查，区分 HTTP status 和业务 ret_code
- [ ] 4.2 修改 `domain/agent/agent.go` systemPrompt：新增 3 条启发式规则（IAM 失败 / 业务故障 / 级联故障判定）
- [ ] 4.3 修改 `domain/agent/agent.go` systemPrompt：诊断结论必须包含 region + gateway_layer + namespace/service + request_uuid node

## Phase 5: 配置与 API 升级

- [ ] 5.1 修改 `interfaces/config/config.go` ESConfig 新增 Regions []string 字段
- [ ] 5.2 修改 `configs/config.example.yaml` 新增 regions 配置示例（c1/c2/c3/b1）
- [ ] 5.3 新增 API 端点 `GET /api/v1/trace-tree/{request_uuid}` 返回 request_uuid 树 JSON
- [ ] 5.4 新增 API 端点 `GET /api/v1/regions` 返回可用地域列表
- [ ] 5.5 修改现有 API Handler 支持 region 查询参数

## Phase 6: 前端可视化

- [ ] 6.1 新增 `frontend/src/components/CallChainWaterfall.vue` 调用链瀑布图组件（4 层网关 + 后端服务时间线）
- [ ] 6.2 新增 `frontend/src/components/UUIDTree.vue` request_uuid 树形视图组件（可折叠/展开，节点按 RetCode 着色）
- [ ] 6.3 新增 `frontend/src/components/SemanticSummary.vue` 业务语义摘要卡片（Action/User/RetCode/Zone 高亮）
- [ ] 6.4 新增 `frontend/src/components/RegionSelector.vue` 全局地域选择器（Header 右侧下拉，Pinia 持久化）
- [ ] 6.5 新增 `frontend/src/views/TroubleshootView.vue` 一键排障主页面：输入 request_uuid → 自动串联全链路 → SSE 推送 Agent 分析
- [ ] 6.6 修改前端路由和导航，新增"智能排障"入口
