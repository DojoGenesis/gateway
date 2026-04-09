<script lang="ts">
	import { createWorkflow, validateWorkflow, saveCanvas as apiSaveCanvas } from '$lib/api';
	import type { WorkflowDefinition, CanvasState } from '$lib/types';
	import { canUndo, canRedo } from '$lib/stores/history.svelte.js';

	// Reactive derivation of undo/redo availability
	const undoAvailable = $derived(canUndo());
	const redoAvailable = $derived(canRedo());

	interface Props {
		workflowName: string;
		onWorkflowNameChange?: (name: string) => void;
		getDefinition?: () => WorkflowDefinition;
		getCanvasState?: () => CanvasState;
		onAutoLayout?: () => void;
		onLoadWorkflow?: () => void;
		onRunWorkflow?: () => void;
		onUndo?: () => void;
		onRedo?: () => void;
		isRunning?: boolean;
	}

	let {
		workflowName,
		onWorkflowNameChange,
		getDefinition,
		getCanvasState,
		onAutoLayout,
		onLoadWorkflow,
		onRunWorkflow,
		onUndo,
		onRedo,
		isRunning = false
	}: Props = $props();

	type ValidationState = 'idle' | 'validating' | 'valid' | 'invalid';
	type SaveState = 'idle' | 'saving' | 'saved' | 'error';

	let validationState = $state<ValidationState>('idle');
	let validationError = $state<string | null>(null);
	let saveState = $state<SaveState>('idle');
	let saveError = $state<string | null>(null);

	async function handleSave() {
		if (!getDefinition) return;
		const def = getDefinition();
		if (!def.name) {
			saveError = 'Workflow name is required.';
			saveState = 'error';
			return;
		}
		saveState = 'saving';
		saveError = null;
		try {
			// 1. Save workflow.json to CAS
			const result = await createWorkflow(def);

			// 2. Save workflow.canvas.json to CAS (if canvas state is available)
			if (getCanvasState) {
				const canvas = getCanvasState();
				canvas.workflow_ref = result.ref;
				await apiSaveCanvas(def.name, canvas);
			}

			saveState = 'saved';
			setTimeout(() => {
				saveState = 'idle';
			}, 2500);
		} catch (e) {
			saveError = e instanceof Error ? e.message : 'Save failed';
			saveState = 'error';
		}
	}

	async function handleValidate() {
		if (!getDefinition) return;
		const def = getDefinition();
		if (!def.name) {
			validationError = 'Set a workflow name before validating.';
			validationState = 'invalid';
			return;
		}
		validationState = 'validating';
		validationError = null;
		try {
			const result = await validateWorkflow(def.name, def);
			if (result.valid) {
				validationState = 'valid';
				setTimeout(() => {
					validationState = 'idle';
				}, 3000);
			} else {
				validationState = 'invalid';
				validationError = result.error ?? 'Validation failed';
			}
		} catch (e) {
			validationState = 'invalid';
			validationError = e instanceof Error ? e.message : 'Validation error';
		}
	}

	function handleRun() {
		onRunWorkflow?.();
	}
</script>

<header class="toolbar">
	<div class="toolbar__brand">
		<span class="toolbar__logo">◆</span>
		<span class="toolbar__label">Workflow Builder</span>
	</div>

	<div class="toolbar__name-field">
		<input
			class="toolbar__name-input"
			type="text"
			placeholder="workflow-name"
			value={workflowName}
			oninput={(e) => onWorkflowNameChange?.((e.target as HTMLInputElement).value)}
			aria-label="Workflow name"
			spellcheck={false}
		/>
	</div>

	<div class="toolbar__actions">
		<!-- Undo -->
		<button
			class="toolbar__btn"
			onclick={() => onUndo?.()}
			disabled={!undoAvailable}
			title="Undo (Cmd+Z)"
		>
			↩ Undo
		</button>

		<!-- Redo -->
		<button
			class="toolbar__btn"
			onclick={() => onRedo?.()}
			disabled={!redoAvailable}
			title="Redo (Cmd+Shift+Z)"
		>
			↪ Redo
		</button>

		<!-- Validate -->
		<button
			class="toolbar__btn toolbar__btn--validate"
			class:toolbar__btn--valid={validationState === 'valid'}
			class:toolbar__btn--invalid={validationState === 'invalid'}
			onclick={handleValidate}
			disabled={validationState === 'validating'}
			title={validationError ?? 'Validate workflow DAG'}
		>
			{#if validationState === 'validating'}
				<span class="spinner"></span> Validating…
			{:else if validationState === 'valid'}
				✓ Valid
			{:else if validationState === 'invalid'}
				✗ Invalid
			{:else}
				Validate
			{/if}
		</button>

		<!-- Auto-layout -->
		<button
			class="toolbar__btn"
			onclick={onAutoLayout}
			title="Auto-arrange nodes using dagre layout"
		>
			Layout
		</button>

		<!-- Load -->
		<button
			class="toolbar__btn"
			onclick={onLoadWorkflow}
			title="Load existing workflow from Gateway"
		>
			Load
		</button>

		<!-- Save -->
		<button
			class="toolbar__btn toolbar__btn--primary"
			class:toolbar__btn--saved={saveState === 'saved'}
			class:toolbar__btn--error={saveState === 'error'}
			onclick={handleSave}
			disabled={saveState === 'saving'}
			title={saveError ?? 'Save workflow to Gateway CAS'}
		>
			{#if saveState === 'saving'}
				<span class="spinner"></span> Saving…
			{:else if saveState === 'saved'}
				✓ Saved
			{:else if saveState === 'error'}
				✗ Error
			{:else}
				Save
			{/if}
		</button>

		<!-- Run -->
		<button
			class="toolbar__btn toolbar__btn--run"
			class:toolbar__btn--running={isRunning}
			onclick={handleRun}
			disabled={isRunning || !workflowName}
			title={isRunning ? 'Execution in progress…' : 'Execute workflow (saves first)'}
		>
			{#if isRunning}
				<span class="spinner"></span> Running…
			{:else}
				▶ Run
			{/if}
		</button>
	</div>

	{#if validationError && validationState === 'invalid'}
		<div class="toolbar__error-banner">
			{validationError}
		</div>
	{/if}
	{#if saveError && saveState === 'error'}
		<div class="toolbar__error-banner">
			{saveError}
		</div>
	{/if}
</header>

<style>
	.toolbar {
		display: flex;
		align-items: center;
		height: 48px;
		background: #0f172a;
		border-bottom: 1px solid #1e293b;
		padding: 0 16px;
		gap: 12px;
		flex-shrink: 0;
		position: relative;
	}

	.toolbar__brand {
		display: flex;
		align-items: center;
		gap: 8px;
		color: #6366f1;
		font-weight: 700;
		font-size: 14px;
		white-space: nowrap;
	}

	.toolbar__logo {
		font-size: 16px;
	}

	.toolbar__label {
		color: #e2e8f0;
		font-size: 14px;
	}

	.toolbar__name-field {
		flex: 1;
		max-width: 280px;
	}

	.toolbar__name-input {
		width: 100%;
		background: #1e293b;
		border: 1px solid #334155;
		border-radius: 6px;
		padding: 6px 10px;
		color: #e2e8f0;
		font-size: 13px;
		outline: none;
		box-sizing: border-box;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
		transition: border-color 0.15s;
	}

	.toolbar__name-input:focus {
		border-color: #6366f1;
	}

	.toolbar__name-input::placeholder {
		color: #475569;
	}

	.toolbar__actions {
		display: flex;
		gap: 8px;
		align-items: center;
		margin-left: auto;
	}

	.toolbar__btn {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		padding: 6px 14px;
		background: #1e293b;
		border: 1px solid #334155;
		border-radius: 6px;
		color: #cbd5e1;
		font-size: 13px;
		cursor: pointer;
		white-space: nowrap;
		transition: background 0.15s, border-color 0.15s, color 0.15s;
	}

	.toolbar__btn:hover:not(:disabled) {
		background: #334155;
		color: #e2e8f0;
	}

	.toolbar__btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.toolbar__btn--primary {
		background: #4f46e5;
		border-color: #4f46e5;
		color: #fff;
		font-weight: 600;
	}

	.toolbar__btn--primary:hover:not(:disabled) {
		background: #4338ca;
		border-color: #4338ca;
	}

	.toolbar__btn--validate.toolbar__btn--valid {
		border-color: #22c55e;
		color: #22c55e;
	}

	.toolbar__btn--validate.toolbar__btn--invalid {
		border-color: #ef4444;
		color: #ef4444;
	}

	.toolbar__btn--saved {
		background: #166534 !important;
		border-color: #22c55e !important;
		color: #22c55e !important;
	}

	.toolbar__btn--error {
		background: #7f1d1d !important;
		border-color: #ef4444 !important;
		color: #ef4444 !important;
	}

	.toolbar__btn--run {
		border-color: #f59e0b;
		color: #f59e0b;
	}

	.toolbar__btn--run:hover:not(:disabled) {
		background: #78350f;
		border-color: #f59e0b;
	}

	.toolbar__btn--running {
		background: #78350f !important;
		border-color: #f59e0b !important;
		color: #f59e0b !important;
	}

	.spinner {
		display: inline-block;
		width: 10px;
		height: 10px;
		border: 2px solid currentColor;
		border-top-color: transparent;
		border-radius: 50%;
		animation: spin 0.6s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	.toolbar__error-banner {
		position: absolute;
		bottom: -32px;
		left: 50%;
		transform: translateX(-50%);
		background: #7f1d1d;
		border: 1px solid #ef4444;
		color: #fca5a5;
		font-size: 12px;
		padding: 5px 14px;
		border-radius: 0 0 6px 6px;
		white-space: nowrap;
		z-index: 100;
	}
</style>
