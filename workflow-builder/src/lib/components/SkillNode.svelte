<script lang="ts">
	import { Handle, Position } from '@xyflow/svelte';
	import type { NodeProps } from '@xyflow/svelte';
	import type { SkillNodeData } from '$lib/types';

	// Svelte Flow passes node props via the NodeProps interface.
	let { data } = $props<NodeProps<SkillNodeData>>();

	let expanded = $state(false);

	// Map execution status to CSS modifier class.
	const statusClass = $derived(
		data.status === 'running'
			? 'skill-node--running'
			: data.status === 'completed'
				? 'skill-node--completed'
				: data.status === 'failed'
					? 'skill-node--failed'
					: data.status === 'skipped'
						? 'skill-node--skipped'
						: ''
	);
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="skill-node {statusClass}"
	onclick={() => (expanded = !expanded)}
>
	<div class="skill-node__header">
		<span class="skill-node__name">{data.label}</span>
		<span class="skill-node__toggle">{expanded ? '▲' : '▼'}</span>
	</div>

	{#if expanded && data.outputs && data.outputs.length > 0}
		<div class="skill-node__description">
			<span class="skill-node__skill-label">Skill: {data.skill}</span>
		</div>
	{/if}

	<div class="skill-node__ports">
		<!-- Input handles (left side) -->
		{#each data.inputs ?? [] as input, i}
			<Handle
				type="target"
				position={Position.Left}
				id={input.name}
				style="top: {(i + 1) * 24 + 36}px"
			/>
			<div class="skill-node__port-label skill-node__port-label--left">
				<span class="port-name">{input.name}</span>
				{#if input.required}
					<span class="port-required" title="Required">*</span>
				{/if}
				<span class="port-type">{input.type}</span>
			</div>
		{/each}

		<!-- Output handles (right side) -->
		{#each data.outputs ?? [] as output, i}
			<Handle
				type="source"
				position={Position.Right}
				id={output.name}
				style="top: {(i + 1) * 24 + 36}px"
			/>
			<div class="skill-node__port-label skill-node__port-label--right">
				<span class="port-type">{output.type}</span>
				<span class="port-name">{output.name}</span>
			</div>
		{/each}
	</div>
</div>

<style>
	.skill-node {
		background: #ffffff;
		border: 2px solid #6366f1;
		border-radius: 8px;
		min-width: 180px;
		max-width: 240px;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.12);
		cursor: pointer;
		font-family: 'Inter', 'Segoe UI', sans-serif;
		font-size: 12px;
		transition: box-shadow 0.15s ease;
	}

	.skill-node:hover {
		box-shadow: 0 4px 16px rgba(99, 102, 241, 0.25);
	}

	/* Status variants */
	.skill-node--running {
		border-color: #22c55e;
		box-shadow: 0 0 0 3px rgba(34, 197, 94, 0.2);
	}

	.skill-node--completed {
		border-color: #10b981;
	}

	.skill-node--failed {
		border-color: #ef4444;
		box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.2);
	}

	.skill-node--skipped {
		border-color: #94a3b8;
		opacity: 0.6;
	}

	.skill-node__header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 8px 12px;
		background: #eef2ff;
		border-radius: 6px 6px 0 0;
		border-bottom: 1px solid #e0e7ff;
		gap: 8px;
	}

	.skill-node__name {
		font-weight: 600;
		color: #3730a3;
		font-size: 12px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.skill-node__toggle {
		color: #6366f1;
		font-size: 10px;
		flex-shrink: 0;
	}

	.skill-node__description {
		padding: 6px 12px;
		background: #f8fafc;
		border-bottom: 1px solid #e2e8f0;
	}

	.skill-node__skill-label {
		color: #64748b;
		font-size: 11px;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
	}

	.skill-node__ports {
		padding: 8px 0;
		min-height: 20px;
		position: relative;
	}

	.skill-node__port-label {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 2px 12px;
		height: 24px;
		font-size: 11px;
		color: #475569;
	}

	.skill-node__port-label--left {
		justify-content: flex-start;
		padding-left: 20px;
	}

	.skill-node__port-label--right {
		justify-content: flex-end;
		padding-right: 20px;
	}

	.port-name {
		font-weight: 500;
		color: #334155;
	}

	.port-type {
		color: #94a3b8;
		font-size: 10px;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
	}

	.port-required {
		color: #ef4444;
		font-weight: 700;
		font-size: 13px;
		line-height: 1;
	}
</style>
