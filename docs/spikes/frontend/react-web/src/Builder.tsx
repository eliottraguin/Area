import { useCallback, useState } from 'react';
import { ReactFlow, Background, Controls, addEdge, useNodesState, useEdgesState, type Connection, type Node } from '@xyflow/react';
import '@xyflow/react/dist/style.css';

// Representative workflow builder: trigger -> condition -> two reactions (MVP shape from #3).
const initialNodes: Node[] = [
  { id: 'trigger', type: 'input', position: { x: 0, y: 80 }, data: { label: 'Trigger: GitHub issue opened' } },
  { id: 'cond', position: { x: 260, y: 80 }, data: { label: 'Condition: label = bug' } },
  { id: 'slack', type: 'output', position: { x: 520, y: 0 }, data: { label: 'Reaction: Slack message' } },
  { id: 'mail', type: 'output', position: { x: 520, y: 160 }, data: { label: 'Reaction: Send email' } },
];
const initialEdges = [
  { id: 'e1', source: 'trigger', target: 'cond' },
  { id: 'e2', source: 'cond', target: 'slack' },
];

export default function Builder() {
  const [nodes, , onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);
  const [count, setCount] = useState(initialEdges.length);
  const onConnect = useCallback((c: Connection) => { setEdges((es) => addEdge(c, es)); setCount((n) => n + 1); }, [setEdges]);
  return (
    <section aria-label="Workflow builder">
      <h2>Workflow builder</h2>
      <p data-testid="edge-count">Connections: {count}</p>
      <div style={{ height: 320 }}>
        <ReactFlow nodes={nodes} edges={edges} onNodesChange={onNodesChange} onEdgesChange={onEdgesChange} onConnect={onConnect} fitView>
          <Background />
          <Controls />
        </ReactFlow>
      </div>
    </section>
  );
}
