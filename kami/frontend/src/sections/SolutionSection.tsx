import { useMemo } from 'react'
import {
  ReactFlow,
  Background,
  MarkerType,
  type Node,
  type Edge,
} from '@xyflow/react'
import dagre from '@dagrejs/dagre'

interface Props {
  nodes: Record<string, string>
}

const NODE_W = 140
const NODE_H = 44

const PALETTE = [
  { bg: 'var(--el-water)', border: 'var(--el-water)' },
  { bg: 'var(--el-earth)', border: 'var(--el-earth)' },
  { bg: 'var(--brand-accent)', border: 'var(--brand-accent)' },
  { bg: 'var(--el-air)', border: 'var(--el-air)' },
]

function layoutLinear(names: string[]) {
  const g = new dagre.graphlib.Graph()
  g.setGraph({ rankdir: 'LR', nodesep: 40, ranksep: 80 })
  g.setDefaultEdgeLabel(() => ({}))
  for (const n of names) {
    g.setNode(n, { width: NODE_W, height: NODE_H })
  }
  for (let i = 0; i < names.length - 1; i++) {
    g.setEdge(names[i], names[i + 1])
  }
  dagre.layout(g)
  return { g, names }
}

export function SolutionSection({ nodes }: Props) {
  const entries = Object.entries(nodes)
  const names = entries.map(([n]) => n)

  const { flowNodes, flowEdges, graphHeight } = useMemo(() => {
    const { g, names: laidOut } = layoutLinear(names)
    let maxY = 0

    const fNodes: Node[] = laidOut.map((name, i) => {
      const pos = g.node(name)
      const y = pos.y - NODE_H / 2
      if (y + NODE_H > maxY) maxY = y + NODE_H
      const colors = PALETTE[i % PALETTE.length]
      return {
        id: name,
        position: { x: pos.x - NODE_W / 2, y },
        data: { label: name },
        style: {
          background: colors.bg,
          color: '#ffffff',
          border: `2px solid ${colors.border}`,
          borderRadius: '8px',
          padding: '6px 12px',
          fontWeight: 600,
          fontSize: '12px',
          textTransform: 'capitalize' as const,
          width: `${NODE_W}px`,
          textAlign: 'center' as const,
        },
        draggable: false,
      }
    })

    const edgeColor = getComputedStyle(document.documentElement)
      .getPropertyValue('--el-earth').trim() || '#5e40be'

    const fEdges: Edge[] = []
    for (let i = 0; i < laidOut.length - 1; i++) {
      fEdges.push({
        id: `${laidOut[i]}->${laidOut[i + 1]}`,
        source: laidOut[i],
        target: laidOut[i + 1],
        style: { stroke: edgeColor, strokeWidth: 2 },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: edgeColor,
          width: 14,
          height: 14,
        },
      })
    }

    return { flowNodes: fNodes, flowEdges: fEdges, graphHeight: Math.max(maxY + 40, 120) }
  }, [names])

  return (
    <section
      id="solution"
      data-kami="section:solution"
      className="section flex items-center justify-center bg-canvas px-8"
      aria-label="The Solution"
    >
      <div className="max-w-5xl w-full text-center">
        <h2 className="text-4xl font-bold text-fg mb-4">The Solution</h2>
        <p className="text-fg-muted text-lg mb-8 max-w-2xl mx-auto">
          A {entries.length}-node AI circuit that processes failures through
          specialized stages, each with its own purpose and expertise.
        </p>
        <div className="w-full" style={{ height: graphHeight }}>
          <ReactFlow
            nodes={flowNodes}
            edges={flowEdges}
            fitView
            panOnDrag={false}
            zoomOnScroll={false}
            zoomOnPinch={false}
            zoomOnDoubleClick={false}
            nodesDraggable={false}
            nodesConnectable={false}
            elementsSelectable={false}
            proOptions={{ hideAttribution: true }}
          >
            <Background gap={0} />
          </ReactFlow>
        </div>
        <div className="mt-6 grid grid-cols-2 md:grid-cols-4 gap-3 text-left">
          {entries.map(([name, desc]) => (
            <div key={name} className="rounded-lg border border-edge px-3 py-2 bg-raised">
              <div className="font-semibold text-fg capitalize text-sm">{name}</div>
              <div className="text-xs text-fg-muted mt-1">{desc}</div>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
