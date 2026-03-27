/**
 * useTopology - AntV X6 服务拓扑图管理 composable
 *
 * 管理 X6 Graph 实例的创建、节点/边渲染、状态更新和销毁。
 * 支持诊断模式、回放模式以及全量监控模式。
 */
import { ref, type Ref } from 'vue'
import { Graph } from '@antv/x6'
import { TOPOLOGY } from '@/store/useTaskStore'
import type { Diagnosis, ImpactReport, TopologyNode, TopologyEdge } from '@/types'

// ── 节点样式常量 ───────────────────────────────────────────────
/** 节点尺寸 */
const NODE_W = 120
const NODE_H = 52

/** 节点状态对应的描边颜色 */
const COLOR = {
  healthy: '#10b981',  // 翠绿
  degraded: '#f59e0b', // 琥珀
  down: '#ef4444',     // 红色
  critical: '#ef4444', // 红色（与 down 一致）
  default: '#6b7280',  // 灰色（无诊断数据时）
} as const

/** 节点状态对应的填充色（低透明度） */
const FILL = {
  healthy: 'rgba(16,185,129,0.07)',
  degraded: 'rgba(245,158,11,0.10)',
  down: 'rgba(239,68,68,0.10)',
  critical: 'rgba(239,68,68,0.10)',
  default: 'rgba(107,114,128,0.06)',
} as const

/** 异常节点外发光滤镜 */
const GLOW_FILTER = {
  down: 'drop-shadow(0 0 6px rgba(239,68,68,0.6))',
  critical: 'drop-shadow(0 0 6px rgba(239,68,68,0.6))',
  degraded: 'drop-shadow(0 0 4px rgba(245,158,11,0.4))',
} as const

// ── 布局常量：手工定义节点坐标 ─────────────────
const LAYOUT: Record<string, { x: number; y: number }> = {
  'prj-apigateway': { x: 160, y: 10 },
  'prj-ubill':      { x: 20,  y: 85 },
  'prj-uresource':  { x: 160, y: 85 },
  'prj-uhost':      { x: 300, y: 85 },
  'prj-unet':       { x: 90,  y: 160 },
  'prj-udb':        { x: 240, y: 160 },
}

// ── 类型 ─────────────────────────────────────────────────────
type NodeStatus = 'healthy' | 'degraded' | 'down' | 'critical' | 'default'

export function useTopology() {
  const graphRef = ref<Graph | null>(null)
  /** 响应式主题状态，通过 MutationObserver 监听 data-theme 变化 */
  const darkMode = ref(document.documentElement.getAttribute('data-theme') === 'dark')
  let themeObserver: MutationObserver | null = null

  // 事件处理回调
  let onNodeClick: ((nodeId: string) => void) | null = null

  // ── 工具函数：取短名称（去掉 "prj-" 前缀） ─────────────────────
  function shortName(svc: string): string {
    return svc.replace(/^prj-/, '')
  }

  // ── 工具函数：获取当前主题的背景色和文字色 ─────────────────────
  function themeColors() {
    return {
      bg: darkMode.value ? '#1d232a' : '#ffffff',
      labelColor: darkMode.value ? 'rgba(255,255,255,0.85)' : 'rgba(0,0,0,0.85)',
      subLabelColor: darkMode.value ? 'rgba(255,255,255,0.45)' : 'rgba(0,0,0,0.45)',
    }
  }

  // ── initGraph：初始化 X6 Graph 实例 ───────────────────────────
  function initGraph(container: HTMLElement, options: { height?: number | string; interacting?: boolean } = {}) {
    if (graphRef.value) {
      graphRef.value.dispose()
    }

    // 监听主题切换，同步更新画布背景色
    themeObserver = new MutationObserver((mutations) => {
      for (const m of mutations) {
        if (m.attributeName === 'data-theme') {
          darkMode.value = document.documentElement.getAttribute('data-theme') === 'dark'
          const graph = graphRef.value
          if (graph) {
            graph.drawBackground({ color: 'transparent' })
          }
        }
      }
    })
    themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] })

    const graph = new Graph({
      container,
      width: container.clientWidth || 430,
      height: typeof options.height === 'number' ? options.height : (container.clientHeight || 250),
      background: { color: 'transparent' },
      interacting: options.interacting ? { nodeMovable: true, edgeMovable: false } : false,
      grid: false,
      panning: options.interacting,
      mousewheel: options.interacting ? { enabled: true, modifiers: 'ctrl' } : false,
    })

    if (options.interacting) {
      graph.on('node:click', ({ node }) => {
        if (onNodeClick) onNodeClick(node.id)
      })
    }

    graphRef.value = graph

    // 渲染默认拓扑
    _renderNodes({}, 'default')
    _renderEdges(new Set(), null)

    return graph
  }

  // ── _renderNodes：渲染/更新所有节点 ─────────────────────────────
  function _renderNodes(
    statusMap: Record<string, NodeStatus>,
    globalStatus: NodeStatus = 'default',
    metricsMap: Record<string, { error_rate?: number; alert_count?: number }> = {}
  ) {
    const graph = graphRef.value
    if (!graph) return

    const { labelColor, subLabelColor } = themeColors()

    for (const svc of TOPOLOGY.value.services) {
      const pos = LAYOUT[svc] || { x: 100, y: 100 }
      const status = statusMap[svc] ?? globalStatus
      const metrics = metricsMap[svc]
      const stroke = COLOR[status]
      const fill = FILL[status]
      
      const errorRateText = metrics?.error_rate !== undefined ? `${(metrics.error_rate * 100).toFixed(1)}%` : ''
      const alertCount = metrics?.alert_count ?? 0

      const nodeAttrs = {
        body: {
          fill,
          stroke,
          strokeWidth: status === 'down' || status === 'critical' || status === 'degraded' ? 2.5 : 1.5,
          rx: 12,
          ry: 12,
          filter: status === 'down' || status === 'critical' ? GLOW_FILTER.down : status === 'degraded' ? GLOW_FILTER.degraded : 'none',
        },
        label: {
          text: shortName(svc).length > 10 ? shortName(svc).slice(0, 9) + '…' : shortName(svc),
          fill: labelColor,
          fontSize: 11,
          fontWeight: 600,
          refY: '35%',
          title: shortName(svc),
        },
        statusDot: {
          fill: stroke,
          refX: 12,
          refY: NODE_H - 12,
          r: 4,
          display: status === 'default' ? 'none' : 'block'
        },
        errorRate: {
          text: errorRateText,
          fill: metrics?.error_rate && metrics.error_rate > 0 ? COLOR.down : subLabelColor,
          fontSize: 10,
          refX: NODE_W - 12,
          refY: NODE_H - 12,
          textAnchor: 'end',
        },
        alertBadgeBg: {
          fill: COLOR.down,
          refX: NODE_W - 12,
          refY: -4,
          width: 16,
          height: 16,
          rx: 8,
          ry: 8,
          display: alertCount > 0 ? 'block' : 'none',
        },
        alertBadge: {
          text: alertCount > 0 ? String(alertCount) : '',
          fill: '#ffffff',
          fontSize: 9,
          fontWeight: 'bold',
          refX: NODE_W - 4,
          refY: 4,
          textAnchor: 'middle',
          display: alertCount > 0 ? 'block' : 'none',
        }
      }

      const existingNode = graph.getCellById(svc)
      if (existingNode && existingNode.isNode()) {
        existingNode.updateAttrs(nodeAttrs)
      } else {
        graph.addNode({
          id: svc,
          shape: 'rect',
          x: pos.x,
          y: pos.y,
          width: NODE_W,
          height: NODE_H,
          attrs: nodeAttrs,
          markup: [
            { tagName: 'rect', selector: 'body' },
            { tagName: 'text', selector: 'label' },
            { tagName: 'circle', selector: 'statusDot' },
            { tagName: 'text', selector: 'errorRate' },
            { tagName: 'rect', selector: 'alertBadgeBg' },
            { tagName: 'text', selector: 'alertBadge' },
          ],
        })
      }
    }
  }

  // ── _renderEdges：渲染/更新所有边 ────────────────────────────────
  function _renderEdges(
    faultEdgeSet: Set<string>,
    _highlightChain: string[] | null,
    weightsMap: Record<string, number> = {}
  ) {
    const graph = graphRef.value
    if (!graph) return

    for (const [src, tgt] of TOPOLOGY.value.edges) {
      const id = `${src}->${tgt}`
      const isFault = faultEdgeSet.has(id)
      const weight = weightsMap[id] ?? 1
      const strokeWidth = isFault ? 3 : Math.min(1 + (weight / 50), 4)

      const edgeAttrs = {
        line: {
          stroke: isFault ? COLOR.down : (darkMode.value ? '#4b5563' : '#9ca3af'),
          strokeWidth,
          strokeDasharray: isFault ? '6 4' : '0',
          targetMarker: {
            name: 'block',
            width: isFault ? 8 : 6,
            height: isFault ? 6 : 5,
            fill: isFault ? COLOR.down : (darkMode.value ? '#4b5563' : '#9ca3af'),
          },
        },
      }

      const existingEdge = graph.getCellById(id)
      if (existingEdge && existingEdge.isEdge()) {
        existingEdge.updateAttrs(edgeAttrs)
        if (isFault) {
          existingEdge.attr('line/strokeDashoffset', 20)
        } else {
          existingEdge.attr('line/strokeDashoffset', 0)
        }
      } else {
        graph.addEdge({
          id,
          source: src,
          target: tgt,
          attrs: edgeAttrs,
        })
      }
    }
  }

  // ── updateNodes：根据诊断结论或影响面报告更新拓扑 ────────────────
  function updateNodes(
    diagnosis: Diagnosis | null,
    impactReport: ImpactReport | null,
  ) {
    const graph = graphRef.value
    if (!graph) return

    if (!diagnosis && !impactReport) {
      _renderNodes({}, 'default')
      _renderEdges(new Set(), null)
      return
    }

    if (impactReport && impactReport.affected_services?.length > 0) {
      const statusMap: Record<string, NodeStatus> = {}
      const metricsMap: Record<string, { error_rate: number }> = {}
      for (const si of impactReport.affected_services) {
        statusMap[si.name] = si.status
        metricsMap[si.name] = { error_rate: si.error_rate }
      }
      for (const svc of TOPOLOGY.value.services) {
        if (!(svc in statusMap)) statusMap[svc] = 'healthy'
      }

      const faultEdgeSet = new Set<string>()
      for (const [src, tgt] of TOPOLOGY.value.edges) {
        const s = statusMap[src] ?? 'healthy'
        const t = statusMap[tgt] ?? 'healthy'
        if ((['down', 'critical', 'degraded'] as string[]).includes(s) && (['down', 'critical', 'degraded'] as string[]).includes(t)) {
          faultEdgeSet.add(`${src}->${tgt}`)
        }
      }

      _renderNodes(statusMap, 'healthy', metricsMap)
      _renderEdges(faultEdgeSet, null)
      return
    }

    if (diagnosis && diagnosis.affected_services?.length > 0) {
      const affectedSet = new Set(diagnosis.affected_services.map(s => s.toLowerCase()))
      const statusMap: Record<string, NodeStatus> = {}
      for (const svc of TOPOLOGY.value.services) {
        statusMap[svc] = affectedSet.has(svc) ? 'down' : 'healthy'
      }

      let chain: string[] | null = null
      for (const svc of diagnosis.affected_services) {
        const c = TOPOLOGY.value.chains[svc]
        if (c && (!chain || c.length > chain.length)) chain = [...c]
      }

      const faultEdgeSet = new Set<string>()
      if (chain && chain.length > 1) {
        for (let i = 0; i < chain.length - 1; i++) {
          const src = chain[i]
          const tgt = chain[i + 1]
          const edgeExists = TOPOLOGY.value.edges.some(([s, t]) => s === src && t === tgt)
          if (edgeExists) faultEdgeSet.add(`${src}->${tgt}`)
        }
      }

      _renderNodes(statusMap, 'healthy')
      _renderEdges(faultEdgeSet, chain)
    }
  }

  // ── updateGraph：全量渲染拓扑图 ────────────────────────────────
  function updateGraph(nodes: TopologyNode[], edges: TopologyEdge[]) {
    const graph = graphRef.value
    if (!graph) return

    TOPOLOGY.value.services = nodes.map(n => n.id)
    TOPOLOGY.value.edges = edges.map(e => [e.source, e.target])

    const statusMap: Record<string, NodeStatus> = {}
    const metricsMap: Record<string, { error_rate: number; alert_count: number }> = {}
    nodes.forEach(n => {
      statusMap[n.id] = n.health
      metricsMap[n.id] = { error_rate: n.error_rate, alert_count: n.alert_count }
    })

    const weightsMap: Record<string, number> = {}
    edges.forEach(e => {
      weightsMap[`${e.source}->${e.target}`] = e.weight
    })

    _renderNodes(statusMap, 'healthy', metricsMap)
    _renderEdges(new Set(), null, weightsMap)
  }

  // ── highlightNode：高亮正在查询的节点 ─────────────────────────
  function highlightNode(namespace: string | null) {
    const graph = graphRef.value
    if (!graph) return

    for (const node of graph.getNodes()) {
      const id = node.id
      const currentStroke = node.getAttrByPath('body/stroke') as string
      if (namespace && id === namespace) {
        node.setAttrByPath('body/strokeWidth', 4)
        node.setAttrByPath('body/stroke', '#818cf8') 
      } else {
        const isDown = currentStroke === COLOR.down
        const isDegraded = currentStroke === COLOR.degraded
        node.setAttrByPath('body/strokeWidth', isDown || isDegraded ? 2.5 : 1.5)
        if (id !== namespace && currentStroke === '#818cf8') {
          node.setAttrByPath('body/stroke', COLOR.healthy)
        }
      }
    }
  }

  function setOnNodeClick(cb: (nodeId: string) => void) {
    onNodeClick = cb
  }

  // ── dispose：销毁图实例 + 清理主题监听 ─────────────────────────
  function dispose() {
    if (themeObserver) {
      themeObserver.disconnect()
      themeObserver = null
    }
    if (graphRef.value) {
      graphRef.value.dispose()
      graphRef.value = null
    }
  }

  // ── resizeGraph：响应容器尺寸变化 ─────────────────────────────
  function resizeGraph(width: number, height?: number | string) {
    if (graphRef.value) {
      const h = typeof height === 'number' ? height : (typeof height === 'string' ? parseInt(height) : 250)
      // X6 resize needs number usually, but let's be safe. If it's 100%, we might need the actual offsetHeight.
      const finalH = isNaN(h) ? 250 : h
      graphRef.value.resize(width, finalH)
    }
  }


  return {
    graphRef: graphRef as Ref<Graph | null>,
    initGraph,
    updateNodes,
    updateGraph,
    highlightNode,
    setOnNodeClick,
    dispose,
    resizeGraph,
  }
}
