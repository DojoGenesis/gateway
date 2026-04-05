// Auto-layout utility using dagre for hierarchical DAG arrangement.
// Called by the "Layout" toolbar button.

import dagre from 'dagre';
import type { Node, Edge } from '@xyflow/svelte';

const NODE_WIDTH = 200;
const NODE_HEIGHT = 100; // estimated height; dagre uses this for spacing

export function applyDagreLayout(
	nodes: Node[],
	edges: Edge[],
	direction: 'TB' | 'LR' = 'LR'
): Node[] {
	const g = new dagre.graphlib.Graph();
	g.setDefaultEdgeLabel(() => ({}));
	g.setGraph({ rankdir: direction, ranksep: 80, nodesep: 40 });

	for (const node of nodes) {
		g.setNode(node.id, { width: NODE_WIDTH, height: NODE_HEIGHT });
	}

	for (const edge of edges) {
		g.setEdge(edge.source, edge.target);
	}

	dagre.layout(g);

	return nodes.map((node) => {
		const pos = g.node(node.id);
		return {
			...node,
			position: {
				x: pos.x - NODE_WIDTH / 2,
				y: pos.y - NODE_HEIGHT / 2
			}
		};
	});
}
