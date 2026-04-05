// API client for the Workflow Gateway endpoints.
// Base URL is configurable via VITE_GATEWAY_URL env var (defaults to localhost:8080).

import type { CanvasState, SkillInfo, WorkflowDefinition } from './types';

const BASE = (import.meta.env.VITE_GATEWAY_URL as string | undefined) ?? 'http://localhost:8080';

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(`${BASE}${path}`, {
		headers: { 'Content-Type': 'application/json', ...init?.headers },
		...init
	});
	if (!res.ok) {
		let msg = `HTTP ${res.status}`;
		try {
			const body = (await res.json()) as { error?: string };
			if (body.error) msg = body.error;
		} catch {
			// ignore parse errors
		}
		throw new Error(msg);
	}
	return res.json() as Promise<T>;
}

// GET /api/workflows
export async function listWorkflows(): Promise<{ workflows: Array<{ name: string; version: string; ref: string }> }> {
	return apiFetch('/api/workflows');
}

// POST /api/workflows
export async function createWorkflow(
	def: WorkflowDefinition
): Promise<{ ref: string; name: string; version: string }> {
	return apiFetch('/api/workflows', {
		method: 'POST',
		body: JSON.stringify(def)
	});
}

// GET /api/workflows/{name}[?version=X]
export async function getWorkflow(name: string, version?: string): Promise<WorkflowDefinition> {
	const qs = version ? `?version=${encodeURIComponent(version)}` : '';
	return apiFetch(`/api/workflows/${encodeURIComponent(name)}${qs}`);
}

// PUT /api/workflows/{name}/canvas
export async function saveCanvas(name: string, state: CanvasState): Promise<{ ref: string }> {
	return apiFetch(`/api/workflows/${encodeURIComponent(name)}/canvas`, {
		method: 'PUT',
		body: JSON.stringify(state)
	});
}

// GET /api/workflows/{name}/canvas
export async function getCanvas(name: string): Promise<CanvasState> {
	return apiFetch(`/api/workflows/${encodeURIComponent(name)}/canvas`);
}

// POST /api/workflows/{name}/validate
export async function validateWorkflow(
	name: string,
	def: WorkflowDefinition
): Promise<{ valid: boolean; error?: string }> {
	return apiFetch(`/api/workflows/${encodeURIComponent(name)}/validate`, {
		method: 'POST',
		body: JSON.stringify(def)
	});
}

// GET /api/skills[?q=search]
export async function listSkills(query?: string): Promise<SkillInfo[]> {
	const qs = query ? `?q=${encodeURIComponent(query)}` : '';
	const data = await apiFetch<{ skills: SkillInfo[] }>(`/api/skills${qs}`);
	return data.skills ?? [];
}

// POST /api/workflows/{name}/execute
// Returns a run_id used to subscribe to the SSE execution stream.
export async function executeWorkflow(name: string): Promise<{ run_id: string; workflow: string }> {
	return apiFetch(`/api/workflows/${encodeURIComponent(name)}/execute`, { method: 'POST' });
}

// Connect to the SSE execution stream for a given run_id.
// Calls onStep for each step event, onDone when the run completes.
// Returns a cleanup function that closes the EventSource.
export function subscribeExecution(
	runId: string,
	onStep: (stepId: string, status: string) => void,
	onDone: () => void
): () => void {
	const es = new EventSource(`${BASE}/api/workflows/${encodeURIComponent(runId)}/execution`);

	es.onmessage = (e: MessageEvent) => {
		let payload: { stepId?: string; status?: string; type?: string };
		try {
			payload = JSON.parse(e.data as string) as typeof payload;
		} catch {
			return;
		}
		if (payload.type === 'done') {
			onDone();
			es.close();
		} else if (payload.stepId && payload.status) {
			onStep(payload.stepId, payload.status);
		}
	};

	es.onerror = () => {
		onDone();
		es.close();
	};

	return () => es.close();
}
