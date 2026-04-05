<script lang="ts">
	import { listSkills } from '$lib/api';
	import type { SkillInfo } from '$lib/types';

	interface Props {
		ondragskill?: (skill: SkillInfo) => void;
	}

	let { ondragskill }: Props = $props();

	let query = $state('');
	let skills = $state<SkillInfo[]>([]);
	let loading = $state(false);
	let error = $state<string | null>(null);

	// Fetch skills whenever query changes (debounced).
	let debounceTimer: ReturnType<typeof setTimeout>;
	$effect(() => {
		const q = query; // capture for closure
		clearTimeout(debounceTimer);
		debounceTimer = setTimeout(() => {
			fetchSkills(q);
		}, 250);
		return () => clearTimeout(debounceTimer);
	});

	async function fetchSkills(q: string) {
		loading = true;
		error = null;
		try {
			skills = await listSkills(q.trim() || undefined);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load skills';
			skills = [];
		} finally {
			loading = false;
		}
	}

	function handleDragStart(event: DragEvent, skill: SkillInfo) {
		if (!event.dataTransfer) return;
		event.dataTransfer.setData('application/dojo-skill', JSON.stringify(skill));
		event.dataTransfer.effectAllowed = 'copy';
		ondragskill?.(skill);
	}

	function inputCount(skill: SkillInfo): number {
		return skill.inputs?.length ?? 0;
	}

	function outputCount(skill: SkillInfo): number {
		return skill.outputs?.length ?? 0;
	}
</script>

<aside class="skill-palette">
	<div class="palette-header">
		<h2 class="palette-title">Skills</h2>
		<input
			class="palette-search"
			type="search"
			placeholder="Search skills…"
			bind:value={query}
			aria-label="Search skills"
		/>
	</div>

	<div class="palette-body">
		{#if loading}
			<div class="palette-state">Loading…</div>
		{:else if error}
			<div class="palette-state palette-state--error">
				<span>{error}</span>
				<button class="retry-btn" onclick={() => fetchSkills(query)}>Retry</button>
			</div>
		{:else if skills.length === 0}
			<div class="palette-state">
				{query ? 'No skills match your search.' : 'No skills available. Start the Gateway and ensure skills are registered in CAS.'}
			</div>
		{:else}
			<ul class="skill-list">
				{#each skills as skill (skill.name + '@' + skill.version)}
					<li
						class="skill-card"
						draggable="true"
						ondragstart={(e) => handleDragStart(e, skill)}
						title={skill.description}
						role="listitem"
					>
						<div class="skill-card__header">
							<span class="skill-card__name">{skill.name}</span>
							<span class="skill-card__version">v{skill.version}</span>
						</div>
						{#if skill.description}
							<p class="skill-card__description">{skill.description}</p>
						{/if}
						<div class="skill-card__ports">
							<span class="port-badge port-badge--in" title="Inputs">
								↓ {inputCount(skill)} in
							</span>
							<span class="port-badge port-badge--out" title="Outputs">
								↑ {outputCount(skill)} out
							</span>
						</div>
					</li>
				{/each}
			</ul>
		{/if}
	</div>
</aside>

<style>
	.skill-palette {
		width: 240px;
		min-width: 240px;
		background: #1e293b;
		color: #e2e8f0;
		display: flex;
		flex-direction: column;
		height: 100%;
		overflow: hidden;
		border-right: 1px solid #334155;
	}

	.palette-header {
		padding: 12px;
		border-bottom: 1px solid #334155;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.palette-title {
		font-size: 13px;
		font-weight: 600;
		color: #94a3b8;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		margin: 0;
	}

	.palette-search {
		width: 100%;
		background: #0f172a;
		border: 1px solid #334155;
		border-radius: 6px;
		padding: 6px 10px;
		color: #e2e8f0;
		font-size: 13px;
		outline: none;
		box-sizing: border-box;
		transition: border-color 0.15s;
	}

	.palette-search:focus {
		border-color: #6366f1;
	}

	.palette-search::placeholder {
		color: #475569;
	}

	.palette-body {
		flex: 1;
		overflow-y: auto;
		padding: 8px;
	}

	.palette-state {
		padding: 16px 8px;
		font-size: 12px;
		color: #64748b;
		text-align: center;
		display: flex;
		flex-direction: column;
		gap: 8px;
		align-items: center;
	}

	.palette-state--error {
		color: #f87171;
	}

	.retry-btn {
		background: #1e293b;
		border: 1px solid #ef4444;
		color: #f87171;
		border-radius: 4px;
		padding: 4px 10px;
		font-size: 12px;
		cursor: pointer;
	}

	.retry-btn:hover {
		background: #ef4444;
		color: white;
	}

	.skill-list {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.skill-card {
		background: #0f172a;
		border: 1px solid #334155;
		border-radius: 8px;
		padding: 10px;
		cursor: grab;
		transition: border-color 0.15s, background 0.15s;
		user-select: none;
	}

	.skill-card:hover {
		border-color: #6366f1;
		background: #1e293b;
	}

	.skill-card:active {
		cursor: grabbing;
	}

	.skill-card__header {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 4px;
		margin-bottom: 4px;
	}

	.skill-card__name {
		font-weight: 600;
		font-size: 13px;
		color: #e2e8f0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.skill-card__version {
		font-size: 10px;
		color: #475569;
		flex-shrink: 0;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
	}

	.skill-card__description {
		font-size: 11px;
		color: #94a3b8;
		margin: 0 0 6px;
		overflow: hidden;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		line-height: 1.4;
	}

	.skill-card__ports {
		display: flex;
		gap: 6px;
	}

	.port-badge {
		font-size: 10px;
		padding: 2px 6px;
		border-radius: 4px;
		font-family: 'JetBrains Mono', 'Fira Code', monospace;
	}

	.port-badge--in {
		background: #1e3a5f;
		color: #60a5fa;
	}

	.port-badge--out {
		background: #1a3a2e;
		color: #34d399;
	}
</style>
