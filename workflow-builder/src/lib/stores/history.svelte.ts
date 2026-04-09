/**
 * Snapshot undo/redo store for the Workflow Builder canvas.
 *
 * Per ADR-019 §7: max depth 50, snapshot-based (not op-based).
 * Keyboard: Cmd+Z / Cmd+Shift+Z (handled by the page component).
 *
 * Usage:
 *   import { history, pushSnapshot, undo, redo, canUndo, canRedo } from '$lib/stores/history.svelte.js';
 */

import type { Node, Edge } from '@xyflow/svelte';
import type { SkillNodeData } from '$lib/types.js';

const MAX_DEPTH = 50;

export interface Snapshot {
	nodes: Node<SkillNodeData>[];
	edges: Edge[];
}

export const history = $state({
	snapshots: [] as Snapshot[],
	index: -1,
});

/** Whether an undo operation is possible. */
export function canUndo(): boolean {
	return history.index > 0;
}

/** Whether a redo operation is possible. */
export function canRedo(): boolean {
	return history.index < history.snapshots.length - 1;
}

/**
 * Push a new snapshot, discarding any redo future.
 * Deep-clones nodes and edges to prevent mutation.
 */
export function pushSnapshot(nodes: Node<SkillNodeData>[], edges: Edge[]) {
	// Drop redo future
	const newSnapshots = history.snapshots.slice(0, history.index + 1);

	newSnapshots.push({
		nodes: structuredClone(nodes),
		edges: structuredClone(edges),
	});

	// Cap at MAX_DEPTH
	if (newSnapshots.length > MAX_DEPTH) {
		newSnapshots.shift();
	}

	history.snapshots = newSnapshots;
	history.index = history.snapshots.length - 1;
}

/**
 * Undo: step back one snapshot.
 * Returns the snapshot to apply, or null if at the beginning.
 */
export function undo(): Snapshot | null {
	if (!canUndo()) return null;
	history.index--;
	return structuredClone(history.snapshots[history.index]);
}

/**
 * Redo: step forward one snapshot.
 * Returns the snapshot to apply, or null if at the end.
 */
export function redo(): Snapshot | null {
	if (!canRedo()) return null;
	history.index++;
	return structuredClone(history.snapshots[history.index]);
}

/** Reset the history stack entirely. */
export function clearHistory() {
	history.snapshots = [];
	history.index = -1;
}
