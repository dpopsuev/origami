import { useCallback, useMemo, useState } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
} from '@xyflow/react'
import type { KamiEvent } from '../hooks/useSSE'

interface Props {
  events: KamiEvent[]
}

export const ELEMENT_COLORS: Record<string, string> = {
  fire: '#ee0000',
  water: '#37a3a3',
  earth: '#5e40be',
  air: '#9ad8d8',
  void: '#21134d',
}

export function PipelineGraph({ events }: Props) {
  const [, setCollapsed] = useState<Set<string>>(new Set())

  const activeNode = useMemo(() => {
    for (let i = events.length - 1; i >= 0; i--) {
      if (events[i].type === 'node_enter' && events[i].node) {
        return events[i].node
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

  const initialNodes: Node[] = useMemo(
    () =>
      nodeNames.map((name, i) => ({
        id: name,
        position: { x: 200 * i, y: 100 },
        data: { label: name },
        style: {
          background:
            name === activeNode
              ? '#ee0000'
              : visitedNodes.has(name)
                ? '#daf2f2'
                : '#f5f5f5',
          color: name === activeNode ? 'white' : '#292929',
          border: `2px solid ${name === activeNode ? '#a60000' : '#e0e0e0'}`,
          borderRadius: '8px',
          padding: '10px 16px',
          fontWeight: name === activeNode ? 700 : 500,
        },
      })),
    [nodeNames, activeNode, visitedNodes]
  )

  const initialEdges: Edge[] = useMemo(() => {
    const edges: Edge[] = []
    const seen = new Set<string>()
    for (const e of events) {
      if (e.type === 'transition' && e.data) {
        const from = String(e.data['from'] || '')
        const to = e.node || ''
        const key = `${from}->${to}`
        if (from && to && !seen.has(key)) {
          seen.add(key)
          edges.push({
            id: key,
            source: from,
            target: to,
            animated: true,
            style: { stroke: '#5e40be' },
          })
        }
      }
    }
    return edges
  }, [events])

  const [nodes] = useNodesState(initialNodes)
  const [edges] = useEdgesState(initialEdges)

  const toggleCollapse = useCallback(
    (nodeId: string) => {
      setCollapsed((prev) => {
        const next = new Set(prev)
        if (next.has(nodeId)) next.delete(nodeId)
        else next.add(nodeId)
        return next
      })
    },
    []
  )

  return (
    <div className="h-full w-full" data-kami="component:pipeline-graph" style={{ minHeight: 400 }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        fitView
        onNodeDoubleClick={(_, node) => toggleCollapse(node.id)}
      >
        <Background color="#e0e0e0" gap={16} />
        <Controls />
        <MiniMap
          nodeColor={(node) =>
            node.id === activeNode ? '#ee0000' : '#daf2f2'
          }
        />
      </ReactFlow>
    </div>
  )
}
