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
