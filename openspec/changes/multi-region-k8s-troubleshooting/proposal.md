# 多地域 K8s 智能排障平台

## Summary

将 Argus 从通用 AIOps 诊断工具升级为面向 UCloud 多地域 K8s 环境的业务导向智能排障平台。核心能力：4 层网关调用链还原、request_uuid 父子树解析、业务语义自动提取、3 个竞赛演示场景。

## Motivation

- 当前排障需人工登录跳板机 → 切地域 → 切 namespace → 切 pod → 手动搜日志，一次排障 30+ 分钟
- 请求经过 Access → IAM 灰度网关 → 业务灰度网关 → 后端服务 → 下游服务，人工串联调用链效率极低
- 不同层的日志格式不统一（网关 JSON / 文本日志 / 结构化 JSON），人工理解成本高
- 现有 Argus 只有单层网关模型，无法表达真实 4 层网关架构

## Strategy

- **保留**：ReAct Agent 核心、ES 集成、回放引擎、多租户架构、Vue 3 前端
- **升级**：ES 索引策略（region 维度）、mock 拓扑（4 层网关）、语义提取、Agent 推理链路
- **新增**：request_uuid 树构建算法、业务语义规范化字段、3 个竞赛演示场景、调用链可视化

## Requirements

### P0 — 核心排障能力

1. **ES 索引多地域支持**：索引格式 `{prefix}_{region}_{namespace}-{date}`，查询支持跨地域通配
2. **4 层网关 Mock 拓扑**：Access / IAM / 业务网关 / 后端服务，前三层同 namespace 标签区分
3. **request_uuid 树构建**：从扁平日志构建父子调用树，支持 root/child/grandchild 层级
4. **业务语义规范化**：写入时提取 action/user/org_id/resource_id/zone/region/ret_code/downstream_service
5. **Agent 推理增强**：按 L1→L2→L3→L4 固定顺序检查，区分 HTTP status 和业务 ret_code
6. **Tool 增强**：es_query_logs 支持 region/gateway_layer/semantic 过滤；trace_analyze 输出树结构

### P1 — 竞赛演示场景

7. **CreateUHostInstance 全链路排障**：Access→IAM→UHost 网关→UHost→UResource/UBill
8. **IAM 鉴权失败排障**：Access→IAM 返回非零 retcode，不进入后续层
9. **跨服务级联故障**：UBill 异常→UHost 超时→网关 504

### P2 — 前端可视化

10. **调用链瀑布图**：4 层网关 + 后端服务的时间线瀑布图
11. **request_uuid 树形视图**：可折叠/展开的父子调用树
12. **业务语义摘要卡片**：Action/User/RetCode 等关键信息高亮展示
13. **多地域选择器**：全局 Header 地域切换 + 页面级对比

## Scope

- 后端：Go（ES 索引策略 + mock 拓扑 + 语义提取 + Agent 增强 + Tool 增强）
- 前端：Vue 3（调用链可视化 + 树形视图 + 语义面板 + 地域选择器）
- 不含：真实多 ES 集群连接、K8s API 集成、Prometheus 指标

## Constraints

- 统一 ES 集群，通过索引名编码地域（不做多 ES 实例连接）
- 4 层网关前三层同 prj-apigateway namespace，通过 kubernetes_labels_app + gateway_layer 区分
- ES 写入时新增规范化字段（region/gateway_layer/request_uuid_root/semantic.*）
- HTTP 200 不等于业务成功，必须同时检查 ret_code
- 竞赛演示优先，3 个场景必须端到端可演示
