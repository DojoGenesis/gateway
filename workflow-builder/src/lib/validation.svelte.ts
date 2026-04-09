/**
 * Connection validation and cycle detection for the Workflow Builder.
 *
 * Per ADR-019: typed port compatibility via JSON Schema rules,
 * topological sort for cycle detection.
 *
 * Handle IDs encode: "{direction}:{portName}:{portType}"
 *   e.g. "output:result:string", "input:sources:string[]"
 */

import type { Node, Edge } from '@xyflow/svelte';
import type { SkillNodeData, PortDefinition } from '$lib/types.js';

// ---------------------------------------------------------------------------
// Handle ID encoding / decoding
// ---------------------------------------------------------------------------

export interface HandleInfo {
	direction: 'input' | 'output';
	portName: string;
	portType: string;
}

/**
 * Encode a handle ID from direction, port name, and type.
 */
export function encodeHandleId(direction: 'input' | 'output', port: PortDefinition): string {
	return `${direction}:${port.name}:${port.type}`;
}

/**
 * Decode a handle ID into its components.
 * Returns null if the ID does not follow the encoded format.
 */
export function decodeHandleId(handleId: string): HandleInfo | null {
	const parts = handleId.split(':');
	if (parts.length < 3) return null;
	const direction = parts[0] as 'input' | 'output';
	if (direction !== 'input' && direction !== 'output') return null;
	const portName = parts[1];
	// Type may contain colons (unlikely but safe)
	const portType = parts.slice(2).join(':');
	return { direction, portName, portType };
}

// ---------------------------------------------------------------------------
// Type compatibility (mirrors Go workflow.ValidateConnection)
// ---------------------------------------------------------------------------

/**
 * Check if an output type can connect to an input type.
 *
 * Type compatibility rules (from ADR-019):
 *   - Exact type match: always valid
 *   - "any" on either side: always valid
 *   - string -> string[]: valid (auto-wrap in array)
 *   - number -> string: valid (auto-coerce)
 *   - string[] -> string: invalid (lossy)
 *   - string -> number: invalid (may fail)
 */
export function areTypesCompatible(outputType: string, inputType: string): boolean {
	// Exact match
	if (outputType === inputType) return true;

	// "any" wildcard
	if (outputType === 'any' || inputType === 'any') return true;

	// string -> string[]: valid (auto-wrap)
	if (outputType === 'string' && inputType === 'string[]') return true;

	// number -> string: valid (auto-coerce)
	if (outputType === 'number' && inputType === 'string') return true;

	return false;
}

/**
 * Validate a proposed connection between two handles.
 * Returns null if valid, or an error message if incompatible.
 */
export function validateConnection(
	sourceHandleId: string | null | undefined,
	targetHandleId: string | null | undefined
): string | null {
	if (!sourceHandleId || !targetHandleId) return null; // no typed handles, allow

	const source = decodeHandleId(sourceHandleId);
	const target = decodeHandleId(targetHandleId);

	if (!source || !target) return null; // untyped handles, allow

	if (!areTypesCompatible(source.portType, target.portType)) {
		return `Type mismatch: ${source.portType} cannot connect to ${target.portType}`;
	}

	return null;
}

// ---------------------------------------------------------------------------
// Cycle detection via topological sort (Kahn's algorithm)
// ---------------------------------------------------------------------------

/**
 * Perform a topological sort on the graph. Returns the sorted node IDs
 * if the graph is a DAG, or null if a cycle exists.
 */
export function topologicalSort(nodes: Node[], edges: Edge[]): string[] | null {
	const nodeIds = new Set(nodes.map((n) => n.id));
	const inDegree = new Map<string, number>();
	const adjacency = new Map<string, string[]>();

	for (const id of nodeIds) {
		inDegree.set(id, 0);
		adjacency.set(id, []);
	}

	for (const edge of edges) {
		adjacency.get(edge.source)?.push(edge.target);
		inDegree.set(edge.target, (inDegree.get(edge.target) ?? 0) + 1);
	}

	// Seed queue with zero in-degree nodes
	const queue: string[] = [];
	for (const [id, deg] of inDegree) {
		if (deg === 0) queue.push(id);
	}

	const sorted: string[] = [];
	while (queue.length > 0) {
		const node = queue.shift()!;
		sorted.push(node);

		for (const neighbor of adjacency.get(node) ?? []) {
			const newDeg = (inDegree.get(neighbor) ?? 1) - 1;
			inDegree.set(neighbor, newDeg);
			if (newDeg === 0) queue.push(neighbor);
		}
	}

	// If not all nodes were visited, there's a cycle
	if (sorted.length !== nodeIds.size) return null;
	return sorted;
}

/**
 * Check if adding an edge from source to target would create a cycle.
 * Returns true if it would create a cycle (edge should be rejected).
 */
export function wouldCreateCycle(
	sourceId: string,
	targetId: string,
	nodes: Node[],
	existingEdges: Edge[]
): boolean {
	// Build adjacency from existing edges + proposed new edge
	const adj = new Map<string, string[]>();
	for (const node of nodes) {
		adj.set(node.id, []);
	}
	for (const edge of existingEdges) {
		adj.get(edge.source)?.push(edge.target);
	}
	adj.get(sourceId)?.push(targetId);

	// DFS from target: if we can reach source, adding source->target is a cycle
	const visited = new Set<string>();
	function dfs(node: string): boolean {
		if (node === sourceId) return true;
		if (visited.has(node)) return false;
		visited.add(node);
		for (const neighbor of adj.get(node) ?? []) {
			if (dfs(neighbor)) return true;
		}
		return false;
	}

	return dfs(targetId);
}

// ---------------------------------------------------------------------------
// Edge styling for validation state
// ---------------------------------------------------------------------------

export type EdgeValidation = 'valid' | 'invalid' | 'cycle';

/**
 * Get CSS class for an edge based on validation state.
 */
export function edgeValidationClass(state: EdgeValidation): string {
	switch (state) {
		case 'invalid':
			return 'edge-invalid';
		case 'cycle':
			return 'edge-cycle';
		default:
			return '';
	}
}
