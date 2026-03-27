# Design — 多地域 K8s 智能排障平台

## 架构决策

### D1: ES 索引策略

**决策**：索引格式 `{prefix}_{region}_{namespace}-{yyyy.MM.dd}`

- 示例：`argus_c1_prj-apigateway-2026.03.20`
- 查询模式：
  - 已知地域：`{prefix}_{region}_*`
  - 已知地域+namespace：`{prefix}_{region}_{namespace}-*`
  - 跨地域搜索：`{prefix}_*_{namespace}-*`
  - 全局搜索：`{prefix}_*`（必须带 time range）
- 影响文件：`internal/infrastructure/es/client.go` 的 index target builder

### D2: 规范化字段

**决策**：ES 写入时新增以下 keyword 字段到 `_source`

| 字段 | 类型 | 来源 |
|------|------|------|
| `region` | keyword | 配置/索引名 |
| `gateway_layer` | keyword | L1_Access / L2_IAM / L3_BizGateway / L4_Backend |
| `request_uuid_root` | keyword | 从 request_uuid 提取根 UUID |
| `request_uuid_full` | keyword | 完整 request_uuid（含 .step.substep） |
| `semantic.action` | keyword | 从 input.Action 提取 |
| `semantic.org_id` | keyword | 从 OrgId/OrganizationId/CompanyId 统一 |
| `semantic.zone` | keyword | 从 Zone/ZoneId 统一 |
| `semantic.region` | keyword | 从 Region/RegionId 统一 |
| `semantic.ret_code` | keyword | 从 RetCode/x-api-retcode 统一 |
| `semantic.downstream_service` | keyword | 从 x-gray-gw-product/real-server 提取 |
| `semantic.user` | keyword | 从 user_email/iam_identity/CompanyId 统一 |
| `semantic.resource_id` | keyword | 从 ResourceId 提取 |

### D3: 4 层网关模型

**决策**：前三层同 `prj-apigateway` namespace，通过标签区分

| 层级 | kubernetes_labels_app | gateway_layer | 日志格式 |
|------|----------------------|---------------|----------|
| L1 Access | gray-gateway-gw | L1_Access | Type A JSON |
| L2 IAM | gray-gateway-gw-iam | L2_IAM | Type A JSON |
| L3 业务网关 | gray-gateway-gw-{product}-backend | L3_BizGateway | Type A JSON |
| L4 后端 | go-{service}-http | L4_Backend | Type B/C |

### D4: request_uuid 树算法

**决策**：O(n log n) 树构建算法

```
1. 从每条日志提取 request_uuid_full
2. 拆出 request_uuid_root 和 path tokens
3. 按 request_uuid_full 分组形成 node，聚合同一节点的多条日志
4. parent_id = 去掉最后一个 path token；缺失父节点时创建 synthetic parent
5. 节点内按时间排序，计算 start/end/duration/namespace/app/level/semantic 汇总
6. 整棵树按 first_seen 排序输出
```

输出结构：
- `tree.nodes[]`: id, parent_id, root_uuid, depth, region, namespace, app, gateway_layer, start_ts, end_ts, status, ret_code, action
- `tree.timeline[]`: node_id, timestamp, level, message
- `tree.edges[]`: from, to, type (request_uuid_parent | service_call | trace_hop)

### D5: Agent System Prompt 增强

**决策**：固定推理顺序 L1→L2→L3→L4

推理规则：
1. 先定位 root request_uuid 和 Action
2. 按 L1 Access → L2 IAM → L3 BizGateway → L4 Backend 固定顺序检查
3. 优先看 request_uuid 树，再用 trace-line 补 hop/函数耗时
4. HTTP status 和业务 ret_code 分开判断
5. IAM 非零 ret_code 且无后续业务子节点 → 鉴权失败
6. IAM 正常、业务网关 ret_code/trace-line 指向某子调用 → 业务故障
7. 上游 5xx 但下游先出现 timeout/OOM/RetCode → 级联故障

### D6: Tool 增强方案

**决策**：增强现有 Tool，不新增第三个 Tool

**es_query_logs v2**：
- 新增参数：regions, gateway_layer, root_request_uuid, semantic_filters
- 返回结构化 hits（含规范化字段），不仅是文本摘要
- 支持按 semantic.action/org_id/ret_code/downstream_service 过滤

**trace_analyze v2**：
- 输入 root_request_uuid
- 输出 request_uuid 树 + 4 层路径 + trace-line 解析 + 语义汇总
- 不再只按时间排序输出文本

### D7: 前端可视化方案

**决策**：瀑布图 + 树形视图 + 语义卡片

- 调用链：水平瀑布图（Gantt 风格），展示每层耗时和状态
- UUID 树：可折叠树形组件，节点按 RetCode 着色（红=5xx，黄=4xx，绿=200）
- 语义面板：Sticky 侧边栏，展示 Action/User/RetCode 等关键信息
- 地域选择器：全局 Header 右侧下拉，持久化到 Pinia + LocalStorage

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `internal/infrastructure/es/client.go` | 修改 | index target builder 支持 region 维度 |
| `internal/infrastructure/es/query.go` | 修改 | 查询方法支持 region/gateway_layer/semantic 过滤 |
| `internal/infrastructure/es/model.go` | 修改 | UCloudLog 新增规范化字段 |
| `internal/infrastructure/es/message_parser.go` | 修改 | 新增 SemanticExtractor 统一提取业务字段 |
| `internal/infrastructure/es/traceline.go` | 修改 | 支持逗号分隔函数段 |
| `internal/infrastructure/es/uuid_tree.go` | 新增 | request_uuid 树构建算法 |
| `internal/infrastructure/mock/topology.go` | 修改 | 4 层网关拓扑（Access/IAM/BizGW 多个 labels_app） |
| `internal/infrastructure/mock/scenarios.go` | 修改 | 3 个新竞赛场景 |
| `internal/infrastructure/tools/es_query.go` | 修改 | v2 参数和返回结构 |
| `internal/infrastructure/tools/trace_analyze.go` | 修改 | v2 树输出 |
| `internal/domain/agent/agent.go` | 修改 | system prompt 增强 |
| `internal/interfaces/config/config.go` | 修改 | ESConfig 新增 Regions 字段 |
| `frontend/src/views/TroubleshootView.vue` | 新增 | 排障主页面 |
| `frontend/src/components/CallChainWaterfall.vue` | 新增 | 瀑布图组件 |
| `frontend/src/components/UUIDTree.vue` | 新增 | UUID 树形组件 |
| `frontend/src/components/SemanticSummary.vue` | 新增 | 语义摘要卡片 |
| `frontend/src/components/RegionSelector.vue` | 新增 | 地域选择器 |

## PBT 属性

| 属性 | 不变量 | 伪造策略 |
|------|--------|----------|
| UUID 树完整性 | 所有子节点的 parent_id 指向存在的父节点 | 随机删除日志条目，验证 synthetic parent 创建 |
| UUID 树幂等性 | 相同日志集合构建的树结构一致 | 打乱日志顺序重复构建 |
| 语义提取一致性 | 同一日志的 3 种格式提取结果字段集相同 | 对同一事件生成 Type A/B/C 格式日志 |
| 索引路由正确性 | region=c1 的日志只出现在 *_c1_* 索引中 | 批量写入多地域日志后验证索引分布 |
| Agent 推理顺序 | L1→L2→L3→L4 顺序不跳层 | 注入各层故障，验证 Agent 不跳过前置层 |
| ret_code 优先级 | HTTP 200 + ret_code!=0 判定为业务失败 | 构造 HTTP 200 但 ret_code=27013 的场景 |
| 跨地域搜索 | root_uuid 搜索返回所有地域的相关日志 | 将同一请求的日志分散到多个地域索引 |
