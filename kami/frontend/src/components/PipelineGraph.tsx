import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  MarkerType,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
} from '@xyflow/react'
import dagre from '@dagrejs/dagre'
import type { KamiEvent } from '../hooks/useSSE'

interface Props {
  events: KamiEvent[]
  nodeDescriptions?: Record<string, string>
}

const EL_COLORS: Record<string, string> = {
  fire: 'var(--el-fire)',
  water: 'var(--el-water)',
  earth: 'var(--el-earth)',
  air: 'var(--el-air)',
  void: 'var(--el-void)',
}

export { EL_COLORS as ELEMENT_COLORS }

const NODE_WIDTH = 160
const NODE_HEIGHT = 50

function layoutGraph(
  nodeIds: string[],
  edgePairs: { source: string; target: string }[],
): Map<string, { x: number; y: number }> {
  const g = new dagre.graphlib.Graph()
  g.setGraph({ rankdir: 'LR', nodesep: 60, ranksep: 100 })
  g.setDefaultEdgeLabel(() => ({}))

  for (const id of nodeIds) {
    g.setNode(id, { width: NODE_WIDTH, height: NODE_HEIGHT })
  }
  for (const { source, target } of edgePairs) {
    g.setEdge(source, target)
  }

  dagre.layout(g)

  const positions = new Map<string, { x: number; y: number }>()
  for (const id of nodeIds) {
    const n = g.node(id)
    if (n) {
      positions.set(id, { x: n.x - NODE_WIDTH / 2, y: n.y - NODE_HEIGHT / 2 })
    }
  }
  return positions
}

function resolveVar(name: string, fallback: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fallback
}

interface MarbleData {
  nodes: { name: string; element?: string }[]
  edges: { id: string; from: string; to: string }[]
  pipeline: string
}

async function fetchMarble(nodeName: string): Promise<MarbleData | null> {
  try {
    const res = await fetch(`/api/marble/${encodeURIComponent(nodeName)}`)
    if (!res.ok) return null
    const def = await res.json()
    return {
      pipeline: def.pipeline || nodeName,
      nodes: (def.nodes || []).map((n: any) => ({ name: n.name, element: n.element })),
      edges: (def.edges || []).map((e: any) => ({ id: e.id, from: e.from, to: e.to })),
    }
  } catch {
    return null
  }
}

export function PipelineGraph({ events, nodeDescriptions }: Props) {
  const [expanded, setExpanded] = useState<Map<string, MarbleData>>(new Map())
  const [hoveredNode, setHoveredNode] = useState<string | null>(null)

  const activeNode = useMemo(() => {
    for (let i = events.length - 1; i >= 0; i--) {
      if (events[i].type === 'node_enter' && events[i].node) {
        return events[i].node
      }
    }
    return null
  }, [events])

  const activeElement = useMemo(() => {
    for (let i = events.length - 1; i >= 0; i--) {
      const e = events[i]
      if (e.type === 'node_enter' && e.node && e.data?.element) {
        return { node: e.node, element: String(e.data.element).toLowerCase() }
      }
    }
    return null
  }, [events])

  const visitedNodes = useMemo(() => {
    const set = new Set<string>()
    for (const e of events) {
      if (e.type === 'node_exit' && e.node) set.add(e.node)
    }
    return set
  }, [events])

  const nodeNames = useMemo(() => {
    const set = new Set<string>()
    for (const e of events) {
      if (e.node) set.add(e.node)
    }
    return Array.from(set)
  }, [events])

  const edgePairs = useMemo(() => {
    const pairs: { source: string; target: string }[] = []
    const seen = new Set<string>()
    for (const e of events) {
      if (e.type === 'transition' && e.data) {
        const from = String(e.data['from'] || '')
        const to = e.node || ''
        const key = `${from}->${to}`
        if (from && to && !seen.has(key)) {
          seen.add(key)
          pairs.push({ source: from, target: to })
        }
      }
    }
    return pairs
  }, [events])

  const brandAccent = resolveVar('--brand-accent', '#ee0000')
  const surfaceRaised = resolveVar('--surface-raised', '#f2f2f2')
  const surfaceCanvas = resolveVar('--surface-canvas', '#ffffff')
  const textPrimary = resolveVar('--text-primary', '#151515')
  const borderDefault = resolveVar('--border-default', '#e0e0e0')
  const edgeColor = resolveVar('--el-earth', '#5e40be')
  const unvisitedColor = resolveVar('--surface-sunken', '#4a4a4a')

  const allNodeNames = useMemo(() => {
    const names = [...nodeNames]
    for (const [parent, data] of expanded) {
      for (const n of data.nodes) {
        names.push(`${parent}/${n.name}`)
      }
    }
    return names
  }, [nodeNames, expanded])

  const allEdgePairs = useMemo(() => {
    const pairs = [...edgePairs]
    for (const [parent, data] of expanded) {
      pairs.push({ source: parent, target: `${parent}/${data.nodes[0]?.name}` })
      for (const e of data.edges) {
        pairs.push({ source: `${parent}/${e.from}`, target: `${parent}/${e.to}` })
      }
    }
    return pairs
  }, [edgePairs, expanded])

  const allPositions = useMemo(
    () => layoutGraph(allNodeNames, allEdgePairs),
    [allNodeNames, allEdgePairs],
  )

  function buildNodeStyle(name: string, isActive: boolean, isVisited: boolean, isInner: boolean) {
    const elementColor = isActive && activeElement?.node === name
      ? EL_COLORS[activeElement.element] || brandAccent
      : undefined
    const bgColor = isActive
      ? (elementColor || brandAccent)
      : isVisited ? surfaceRaised : unvisitedColor
    return {
      background: bgColor,
      color: isActive || !isVisited ? '#ffffff' : textPrimary,
      border: `2px ${isInner ? 'dashed' : 'solid'} ${isActive ? bgColor : isVisited ? borderDefault : unvisitedColor}`,
      borderRadius: isInner ? '4px' : '8px',
      padding: isInner ? '6px 12px' : '10px 16px',
      fontWeight: isActive ? 700 : 500,
      animation: isActive ? 'node-pulse 2s ease-in-out infinite' : undefined,
      textTransform: 'capitalize' as const,
      fontSize: isInner ? '11px' : '13px',
      minWidth: isInner ? '120px' : `${NODE_WIDTH}px`,
      opacity: isInner ? 0.85 : 1,
    }
  }

  const initialNodes: Node[] = useMemo(
    () =>
      allNodeNames.map((name) => {
        const pos = allPositions.get(name) || { x: 0, y: 0 }
        const isInner = name.includes('/')
        const displayName = isInner ? name.split('/').pop()! : name
        const isActive = name === activeNode
        const isVisited = visitedNodes.has(name)

        return {
          id: name,
          position: pos,
          data: { label: displayName },
          style: buildNodeStyle(name, isActive, isVisited, isInner),
        }
      }),
    [allNodeNames, activeNode, activeElement, visitedNodes, allPositions, brandAccent, surfaceRaised, textPrimary, borderDefault, unvisitedColor],
  )

  const initialEdges: Edge[] = useMemo(
    () =>
      allEdgePairs.map(({ source, target }) => {
        const isInner = source.includes('/') || target.includes('/')
        return {
          id: `${source}->${target}`,
          source,
          target,
          animated: true,
          style: { stroke: isInner ? borderDefault : edgeColor, strokeWidth: isInner ? 1 : 2, strokeDasharray: isInner ? '4 2' : undefined },
          markerEnd: {
            type: MarkerType.ArrowClosed,
            color: isInner ? borderDefault : edgeColor,
            width: isInner ? 12 : 16,
            height: isInner ? 12 : 16,
          },
        }
      }),
    [allEdgePairs, edgeColor, borderDefault],
  )

  const [nodes, setNodes] = useNodesState(initialNodes)
  const [edges, setEdges] = useEdgesState(initialEdges)

  useEffect(() => { setNodes(initialNodes) }, [initialNodes, setNodes])
  useEffect(() => { setEdges(initialEdges) }, [initialEdges, setEdges])

  const toggleExpand = useCallback(
    async (nodeId: string) => {
      if (expanded.has(nodeId)) {
        setExpanded((prev) => {
          const next = new Map(prev)
          next.delete(nodeId)
          return next
        })
        return
      }
      const data = await fetchMarble(nodeId)
      if (data) {
        setExpanded((prev) => new Map(prev).set(nodeId, data))
      }
    },
    [expanded],
  )

  const tooltip = hoveredNode && nodeDescriptions?.[hoveredNode]

  return (
    <div className="h-full w-full relative" data-kami="component:pipeline-graph" style={{ minHeight: 400 }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        fitView
        onNodeDoubleClick={(_, node) => toggleExpand(node.id)}
        onNodeMouseEnter={(_, node) => setHoveredNode(node.id)}
        onNodeMouseLeave={() => setHoveredNode(null)}
      >
        <Background color={borderDefault} gap={16} />
        <Controls />
        <MiniMap
          nodeColor={(node) =>
            node.id === activeNode ? brandAccent : surfaceRaised
          }
          style={{ background: surfaceCanvas }}
        />
      </ReactFlow>
      {tooltip && (
        <div
          className="absolute bottom-4 left-4 right-4 bg-raised border border-edge
                     rounded-lg px-4 py-2 text-sm text-fg shadow-lg pointer-events-none z-10"
        >
          <span className="font-semibold capitalize">{hoveredNode}</span>
          <span className="mx-2 text-fg-faint">&mdash;</span>
          <span className="text-fg-muted">{tooltip}</span>
        </div>
      )}
    </div>
  )
}
