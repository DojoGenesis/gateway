/**
 * Execution store — tracks real-time step status for a running workflow.
 *
 * Usage in +page.svelte:
 *   import { execution, startExecution, updateStep, finishExecution } from './stores/execution.svelte.js';
 *
 *   // Reactive read:
 *   const status = execution.statuses.get(stepId); // StepStatus | undefined
 */

import type { StepStatus } from '$lib/types.js';

// execution is the single shared reactive state object.
// Svelte 5 runes transform $state() in .svelte.ts files.
export const execution = $state({
	/** Maps step ID → current execution status. */
	statuses: new Map<string, StepStatus>(),

	/** UUID of the current run, or null when idle. */
	runId: null as string | null,

	/** True while an execution is in progress. */
	running: false,
});

/** Call when a new execution starts. Resets statuses and records the run ID. */
export function startExecution(runId: string) {
	execution.statuses = new Map();
	execution.runId = runId;
	execution.running = true;
}

/** Update a single step's status (called on each SSE event). */
export function updateStep(stepId: string, status: StepStatus) {
	// Svelte tracks Map mutations via a new Map assignment.
	execution.statuses = new Map(execution.statuses).set(stepId, status);
}

/** Call when the SSE stream sends the terminal {"type":"done"} event. */
export function finishExecution() {
	execution.running = false;
}
