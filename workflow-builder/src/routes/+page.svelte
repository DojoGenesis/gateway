<script lang="ts">
	import {
		SvelteFlow,
		Controls,
		Background,
		MiniMap,
		addEdge,
		type Node,
		type Edge,
		type Connection
	} from '@xyflow/svelte';
	import '@xyflow/svelte/dist/style.css';

	import SkillPalette from '$lib/components/SkillPalette.svelte';
	import WorkflowToolbar from '$lib/components/WorkflowToolbar.svelte';
	import SkillNode from '$lib/components/SkillNode.svelte';

	import { applyDagreLayout } from '$lib/layout';
	import {
		validateConnection,
		wouldCreateCycle,
		encodeHandleId
	} from '$lib/validation.svelte.js';
	import {
		pushSnapshot,
		undo as historyUndo,
		redo as historyRedo,
		clearHistory
	} from '$lib/stores/history.svelte.js';
	import {
		listWorkflows,
		getWorkflow,
		saveCanvas,
		getCanvas,
		executeWorkflow,
		subscribeExecution
	} from '$lib/api';
	import type {
		WorkflowDefinition,
		SkillInfo,
		SkillNodeData,
		CanvasState,
		Step,
		StepStatus
	} from '$lib/types';
	import {
		execution,
		startExecution,
		updateStep,
		finishExecution
	} from '$lib/stores/execution.svelte.js';

	// -------------------------------------------------------------------------
	// Node types registry for Svelte Flow
	// -------------------------------------------------------------------------
	const nodeTypes = { skill: SkillNode } as Record<string, typeof SkillNode>;

	// -------------------------------------------------------------------------
	// Reactive state -- Svelte 5 runes
	// -------------------------------------------------------------------------
	let nodes = $state<Node<SkillNodeData>[]>([]);
	let edges = $state<Edge[]>([]);
	let workflowName = $state('');
	let nodeIdCounter = $state(0);

	// Connection validation feedback
	let connectionError = $state<string | null>(null);
	let connectionErrorTimeout: ReturnType<typeof setTimeout> | undefined;
	let flashNodeId = $state<string | null>(null);

	// Load workflow modal state
	let showLoadModal = $state(false);
	let availableWorkflows = $state<Array<{ name: string; version: string; ref: string }>>([]);
	let loadingWorkflows = $state(false);

	// -------------------------------------------------------------------------
	// History helpers (delegate to store)
	// -------------------------------------------------------------------------
	function recordSnapshot() {
		pushSnapshot(nodes, edges);
	}

	function handleUndo() {
		const snap = historyUndo();
		if (snap) {
			nodes = snap.nodes;
			edges = snap.edges;
		}
	}

	function handleRedo() {
		const snap = historyRedo();
		if (snap) {
			nodes = snap.nodes;
			edges = snap.edges;
		}
	}

	// -------------------------------------------------------------------------
	// Keyboard shortcuts
	// -------------------------------------------------------------------------
	function handleKeydown(e: KeyboardEvent) {
		const mod = e.metaKey || e.ctrlKey;
		if (mod && e.key === 'z' && !e.shiftKey) {
			e.preventDefault();
			handleUndo();
		}
		if ((mod && e.shiftKey && e.key === 'z') || (mod && e.key === 'y')) {
			e.preventDefault();
			handleRedo();
		}
	}

	// -------------------------------------------------------------------------
	// Connection error flash
	// -------------------------------------------------------------------------
	function showConnectionError(msg: string, targetNodeId?: string) {
		connectionError = msg;
		if (targetNodeId) flashNodeId = targetNodeId;

		if (connectionErrorTimeout) clearTimeout(connectionErrorTimeout);
		connectionErrorTimeout = setTimeout(() => {
			connectionError = null;
			flashNodeId = null;
		}, 3000);
	}

	// -------------------------------------------------------------------------
	// Drag-and-drop from skill palette onto canvas
	// -------------------------------------------------------------------------
	let canvasEl = $state<HTMLDivElement | null>(null);

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy';
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		const raw = e.dataTransfer?.getData('application/dojo-skill');
		if (!raw) return;

		let skill: SkillInfo;
		try {
			skill = JSON.parse(raw) as SkillInfo;
		} catch {
			return;
		}

		// Compute drop position relative to the canvas element.
		const rect = canvasEl?.getBoundingClientRect();
		const x = rect ? e.clientX - rect.left - 100 : e.clientX - 100;
		const y = rect ? e.clientY - rect.top - 50 : e.clientY - 50;

		const id = `step-${++nodeIdCounter}`;
		const newNode: Node<SkillNodeData> = {
			id,
			type: 'skill',
			position: { x, y },
			data: {
				label: skill.name,
				skill: skill.name,
				inputs: skill.inputs ?? [],
				outputs: skill.outputs ?? [],
				status: 'pending'
			}
		};

		nodes = [...nodes, newNode];
		recordSnapshot();
	}

	// -------------------------------------------------------------------------
	// Edge connection handler with type validation + cycle detection
	// -------------------------------------------------------------------------
	function handleConnect(connection: Connection) {
		const sourceId = connection.source;
		const targetId = connection.target;

		if (!sourceId || !targetId) return;

		// 1. Cycle detection
		if (wouldCreateCycle(sourceId, targetId, nodes, edges)) {
			showConnectionError('Cycle detected -- connection rejected', targetId);
			return;
		}

		// 2. Type compatibility check
		const typeError = validateConnection(
			connection.sourceHandle,
			connection.targetHandle
		);
		if (typeError) {
			showConnectionError(typeError, targetId);
			return;
		}

		// Valid connection -- add the edge
		edges = addEdge(connection, edges);
		recordSnapshot();
	}

	// -------------------------------------------------------------------------
	// Toolbar: build WorkflowDefinition from canvas state
	// -------------------------------------------------------------------------
	function buildDefinition(): WorkflowDefinition {
		const steps: Step[] = nodes.map((node) => {
			// Derive depends_on from edges targeting this node.
			const dependsOn = edges
				.filter((e) => e.target === node.id)
				.map((e) => e.source);

			// Build inputs map from edge connections (port-level bindings)
			const inputBindings: Record<string, string> = {};
			for (const edge of edges) {
				if (edge.target !== node.id) continue;
				if (edge.sourceHandle && edge.targetHandle) {
					// Extract port names from encoded handles
					const sourceParts = edge.sourceHandle.split(':');
					const targetParts = edge.targetHandle.split(':');
					if (sourceParts.length >= 2 && targetParts.length >= 2) {
						const sourceStepId = edge.source;
						const sourcePortName = sourceParts[1];
						const targetPortName = targetParts[1];
						inputBindings[targetPortName] = `{{ steps.${sourceStepId}.outputs.${sourcePortName} }}`;
					}
				}
			}

			return {
				id: node.id,
				skill: node.data.skill,
				inputs: inputBindings,
				depends_on: [...new Set(dependsOn)]
			};
		});

		return {
			version: '1.0.0',
			name: workflowName || 'untitled',
			artifact_type: 'application/vnd.dojo.workflow.v1',
			steps
		};
	}

	// -------------------------------------------------------------------------
	// Build CanvasState for persistence
	// -------------------------------------------------------------------------
	function buildCanvasState(): CanvasState {
		const positions: Record<string, { x: number; y: number }> = {};
		for (const n of nodes) {
			positions[n.id] = { x: n.position.x, y: n.position.y };
		}
		return {
			workflow_ref: '',
			viewport: { x: 0, y: 0, zoom: 1 },
			node_positions: positions
		};
	}

	// -------------------------------------------------------------------------
	// Auto-layout via dagre
	// -------------------------------------------------------------------------
	function handleAutoLayout() {
		if (nodes.length === 0) return;
		nodes = applyDagreLayout(nodes, edges, 'LR') as Node<SkillNodeData>[];
		recordSnapshot();
	}

	// -------------------------------------------------------------------------
	// Load existing workflow
	// -------------------------------------------------------------------------
	async function openLoadModal() {
		showLoadModal = true;
		loadingWorkflows = true;
		try {
			const res = await listWorkflows();
			availableWorkflows = res.workflows;
		} catch {
			availableWorkflows = [];
		} finally {
			loadingWorkflows = false;
		}
	}

	async function loadWorkflow(name: string) {
		showLoadModal = false;
		try {
			const def = await getWorkflow(name);
			workflowName = def.name;

			// Rebuild nodes from steps.
			let x = 80;
			const newNodes: Node<SkillNodeData>[] = def.steps.map((step, i) => ({
				id: step.id,
				type: 'skill' as const,
				position: { x: x + i * 220, y: 200 },
				data: {
					label: step.skill,
					skill: step.skill,
					inputs: [],
					outputs: [],
					status: 'pending' as const
				}
			}));

			// Rebuild edges from depends_on.
			const newEdges: Edge[] = [];
			for (const step of def.steps) {
				for (const dep of step.depends_on ?? []) {
					newEdges.push({
						id: `${dep}->${step.id}`,
						source: dep,
						target: step.id
					});
				}
			}

			nodes = newNodes;
			edges = newEdges;
			nodeIdCounter = newNodes.length;

			// Try to load saved canvas positions.
			try {
				const canvas = await getCanvas(name);
				nodes = nodes.map((n) => {
					const pos = canvas.node_positions[n.id];
					return pos ? { ...n, position: { x: pos.x, y: pos.y } } : n;
				});
			} catch {
				// Canvas not found -- use auto-layout.
				nodes = applyDagreLayout(nodes, edges, 'LR') as Node<SkillNodeData>[];
			}

			clearHistory();
			recordSnapshot();
		} catch (e) {
			console.error('Failed to load workflow:', e);
		}
	}

	// -------------------------------------------------------------------------
	// Save canvas state after drag/layout operations
	// -------------------------------------------------------------------------
	async function persistCanvas() {
		if (!workflowName) return;
		try {
			await saveCanvas(workflowName, buildCanvasState());
		} catch {
			// Non-critical -- canvas save failure doesn't block workflow save.
		}
	}

	// -------------------------------------------------------------------------
	// Execution: POST -> run_id -> SSE -> node status updates
	// -------------------------------------------------------------------------
	let cleanupExecution = $state<(() => void) | null>(null);

	async function handleRunWorkflow() {
		if (!workflowName || execution.running) return;

		// Save the workflow first so the executor finds the latest definition.
		try {
			const { createWorkflow } = await import('$lib/api');
			await createWorkflow(buildDefinition());
		} catch {
			// Non-blocking -- proceed even if save fails (workflow may already exist).
		}

		let result: { run_id: string; workflow: string };
		try {
			result = await executeWorkflow(workflowName);
		} catch (e) {
			console.error('Failed to start execution:', e);
			return;
		}

		// Reset all node statuses to pending.
		nodes = nodes.map((n) => ({
			...n,
			data: { ...n.data, status: 'pending' as StepStatus }
		}));
		startExecution(result.run_id);

		// Subscribe to the SSE stream.
		if (cleanupExecution) cleanupExecution();
		cleanupExecution = subscribeExecution(
			result.run_id,
			(stepId, status) => {
				updateStep(stepId, status as StepStatus);
				// Mirror status into the Svelte Flow nodes array so SkillNode re-renders.
				nodes = nodes.map((n) =>
					n.id === stepId
						? { ...n, data: { ...n.data, status: status as StepStatus } }
						: n
				);
			},
			() => {
				finishExecution();
			}
		);
	}

	// -------------------------------------------------------------------------
	// Initialize history on first load
	// -------------------------------------------------------------------------
	$effect(() => {
		if (nodes !== undefined && edges !== undefined) {
			// Only push initial snapshot once
			recordSnapshot();
		}
	});
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="workflow-builder">
	<WorkflowToolbar
		{workflowName}
		onWorkflowNameChange={(name) => (workflowName = name)}
		getDefinition={buildDefinition}
		getCanvasState={buildCanvasState}
		onAutoLayout={handleAutoLayout}
		onLoadWorkflow={openLoadModal}
		onRunWorkflow={handleRunWorkflow}
		onUndo={handleUndo}
		onRedo={handleRedo}
		isRunning={execution.running}
	/>

	<div class="builder-body">
		<SkillPalette />

		<!-- Canvas drop zone -->
		<div
			class="canvas-area"
			bind:this={canvasEl}
			ondragover={handleDragOver}
			ondrop={handleDrop}
			role="region"
			aria-label="Workflow canvas"
		>
			<SvelteFlow
				bind:nodes
				bind:edges
				{nodeTypes}
				onconnect={handleConnect}
				onnodedragstop={persistCanvas}
				fitView
			>
				<Controls />
				<Background />
				<MiniMap />
			</SvelteFlow>

			{#if nodes.length === 0}
				<div class="canvas-empty">
					<p class="canvas-empty__title">
						Drag skills from the palette to build your workflow
					</p>
					<p class="canvas-empty__hint">
						Connect nodes by dragging from output (right) handles to input (left) handles
					</p>
				</div>
			{/if}

			<!-- Connection validation error toast -->
			{#if connectionError}
				<div class="connection-error-toast" role="alert">
					<span class="connection-error-toast__icon">✗</span>
					<span>{connectionError}</span>
				</div>
			{/if}
		</div>
	</div>
</div>

<!-- Load workflow modal -->
{#if showLoadModal}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="modal-overlay" onclick={() => (showLoadModal = false)}>
		<div
			class="modal"
			onclick={(e) => e.stopPropagation()}
			role="dialog"
			aria-modal="true"
			aria-label="Load workflow"
			tabindex="-1"
		>
			<div class="modal__header">
				<h2 class="modal__title">Load Workflow</h2>
				<button
					class="modal__close"
					onclick={() => (showLoadModal = false)}
					aria-label="Close"
				>
					✕
				</button>
			</div>
			<div class="modal__body">
				{#if loadingWorkflows}
					<p class="modal__loading">Loading...</p>
				{:else if availableWorkflows.length === 0}
					<p class="modal__empty">No workflows found. Save a workflow first.</p>
				{:else}
					<ul class="modal__list">
						{#each availableWorkflows as wf (wf.name + '@' + wf.version)}
							<li>
								<button
									class="modal__workflow-btn"
									onclick={() => loadWorkflow(wf.name)}
								>
									<span class="modal__wf-name">{wf.name}</span>
									<span class="modal__wf-version">v{wf.version}</span>
								</button>
							</li>
						{/each}
					</ul>
				{/if}
			</div>
		</div>
	</div>
{/if}

<style>
	:global(html, body) {
		margin: 0;
		padding: 0;
		height: 100%;
		overflow: hidden;
	}

	.workflow-builder {
		display: flex;
		flex-direction: column;
		height: 100vh;
		background: #0f172a;
		font-family: 'Inter', 'Segoe UI', system-ui, sans-serif;
	}

	.builder-body {
		display: flex;
		flex: 1;
		overflow: hidden;
	}

	.canvas-area {
		flex: 1;
		position: relative;
		background: #f8fafc;
		overflow: hidden;
	}

	/* Canvas empty state overlay */
	.canvas-empty {
		position: absolute;
		inset: 0;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		pointer-events: none;
		z-index: 4;
	}

	.canvas-empty__title {
		font-size: 16px;
		font-weight: 500;
		color: #94a3b8;
		margin: 0 0 8px;
	}

	.canvas-empty__hint {
		font-size: 13px;
		color: #cbd5e1;
		margin: 0;
	}

	/* Connection validation error toast */
	.connection-error-toast {
		position: absolute;
		bottom: 24px;
		left: 50%;
		transform: translateX(-50%);
		background: #7f1d1d;
		border: 1px solid #ef4444;
		color: #fca5a5;
		padding: 8px 16px;
		border-radius: 8px;
		font-size: 13px;
		display: flex;
		align-items: center;
		gap: 8px;
		z-index: 100;
		box-shadow: 0 4px 20px rgba(239, 68, 68, 0.3);
		animation: toast-in 0.2s ease-out;
	}

	.connection-error-toast__icon {
		font-weight: 700;
		font-size: 14px;
		color: #ef4444;
	}

	@keyframes toast-in {
		from {
			opacity: 0;
			transform: translateX(-50%) translateY(10px);
		}
		to {
			opacity: 1;
			transform: translateX(-50%) translateY(0);
		}
	}

	/* Modal overlay */
	.modal-overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.6);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
	}

	.modal {
		background: #1e293b;
		border: 1px solid #334155;
		border-radius: 10px;
		width: 400px;
		max-height: 500px;
		display: flex;
		flex-direction: column;
		overflow: hidden;
		box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
	}

	.modal__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 20px;
		border-bottom: 1px solid #334155;
	}

	.modal__title {
		margin: 0;
		font-size: 16px;
		font-weight: 600;
		color: #e2e8f0;
	}

	.modal__close {
		background: none;
		border: none;
		color: #64748b;
		font-size: 16px;
		cursor: pointer;
		padding: 4px;
		line-height: 1;
	}

	.modal__close:hover {
		color: #e2e8f0;
	}

	.modal__body {
		padding: 12px;
		overflow-y: auto;
		flex: 1;
	}

	.modal__loading,
	.modal__empty {
		text-align: center;
		color: #64748b;
		font-size: 13px;
		padding: 20px;
	}

	.modal__list {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.modal__workflow-btn {
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: space-between;
		background: #0f172a;
		border: 1px solid #334155;
		border-radius: 6px;
		padding: 10px 14px;
		cursor: pointer;
		transition: border-color 0.15s, background 0.15s;
	}

	.modal__workflow-btn:hover {
		border-color: #6366f1;
		background: #1e293b;
	}

	.modal__wf-name {
		color: #e2e8f0;
		font-size: 13px;
		font-weight: 500;
	}

	.modal__wf-version {
		color: #475569;
		font-size: 11px;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
	}

	/* Red flash on invalid edge target nodes */
	:global(.svelte-flow__node.flash-invalid) {
		animation: flash-red 0.4s ease-out;
	}

	@keyframes flash-red {
		0% {
			box-shadow: 0 0 0 4px rgba(239, 68, 68, 0.6);
		}
		100% {
			box-shadow: none;
		}
	}

	/* Invalid edge styling (applied via class) */
	:global(.svelte-flow__edge.edge-invalid .svelte-flow__edge-path) {
		stroke: #ef4444 !important;
		stroke-dasharray: 5 3;
	}

	:global(.svelte-flow__edge.edge-cycle .svelte-flow__edge-path) {
		stroke: #f59e0b !important;
		stroke-dasharray: 8 4;
	}
</style>
