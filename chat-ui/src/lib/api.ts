// Gateway API client for the chat UI.
// VITE_API_BASE defaults to empty string so fetch calls go to the same origin.
// In dev, set VITE_API_BASE=http://localhost:7340 in .env.local.

import type {
	AuthResponse,
	ChatMessage,
	ChatCompletionChunk,
	Conversation,
	ConversationDetail,
	Model,
	ModelsResponse
} from './types';

const API_BASE = (import.meta.env.VITE_API_BASE as string | undefined) ?? '';

// ── Auth helpers ─────────────────────────────────────────────────────────────

function getToken(): string {
	return localStorage.getItem('access_token') ?? '';
}

function authHeaders(): Record<string, string> {
	const token = getToken();
	return token ? { Authorization: `Bearer ${token}` } : {};
}

// Generic JSON fetch with auth
async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
	const res = await fetch(`${API_BASE}${path}`, {
		...init,
		headers: {
			'Content-Type': 'application/json',
			...authHeaders(),
			...(init.headers as Record<string, string> | undefined)
		}
	});
	if (!res.ok) {
		let msg = `HTTP ${res.status}`;
		try {
			const body = (await res.json()) as { error?: string; message?: string };
			msg = body.error ?? body.message ?? msg;
		} catch {
			// ignore parse errors
		}
		throw new Error(msg);
	}
	return res.json() as Promise<T>;
}

// ── Auth ─────────────────────────────────────────────────────────────────────

export async function login(email: string, password: string): Promise<AuthResponse> {
	return apiFetch<AuthResponse>('/auth/login', {
		method: 'POST',
		body: JSON.stringify({ email, password }),
		headers: {} // no auth header needed for login
	});
}

export async function refreshToken(token: string): Promise<AuthResponse> {
	return apiFetch<AuthResponse>('/auth/refresh', {
		method: 'POST',
		body: JSON.stringify({ refresh_token: token }),
		headers: {} // no auth header needed for refresh
	});
}

// ── Models ───────────────────────────────────────────────────────────────────

export async function listModels(): Promise<Model[]> {
	const data = await apiFetch<ModelsResponse>('/v1/models');
	return data.data ?? [];
}

// ── Chat (SSE streaming) ─────────────────────────────────────────────────────

/**
 * Send messages and stream back assistant response chunks.
 * Yields content string fragments as they arrive.
 * Handles OpenAI SSE format: `data: {...}\n\ndata: [DONE]\n\n`
 */
export async function* streamChat(
	messages: ChatMessage[],
	model: string
): AsyncGenerator<string> {
	const res = await fetch(`${API_BASE}/v1/chat/completions`, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
			...authHeaders()
		},
		body: JSON.stringify({
			model,
			messages,
			stream: true
		})
	});

	if (!res.ok) {
		let msg = `HTTP ${res.status}`;
		try {
			const body = (await res.json()) as { error?: string };
			msg = body.error ?? msg;
		} catch {
			// ignore
		}
		throw new Error(msg);
	}

	if (!res.body) {
		throw new Error('Response body is null');
	}

	const reader = res.body.getReader();
	const decoder = new TextDecoder();
	let buffer = '';

	try {
		while (true) {
			const { done, value } = await reader.read();
			if (done) break;

			buffer += decoder.decode(value, { stream: true });

			// Split on double newline — SSE event boundaries
			const events = buffer.split('\n\n');
			// Keep the last (potentially incomplete) chunk in the buffer
			buffer = events.pop() ?? '';

			for (const event of events) {
				const lines = event.split('\n');
				for (const line of lines) {
					if (!line.startsWith('data: ')) continue;
					const raw = line.slice(6).trim();
					if (raw === '[DONE]') return;

					let chunk: ChatCompletionChunk;
					try {
						chunk = JSON.parse(raw) as ChatCompletionChunk;
					} catch {
						continue;
					}

					const content = chunk.choices?.[0]?.delta?.content;
					if (content) {
						yield content;
					}
				}
			}
		}
	} finally {
		reader.releaseLock();
	}
}

// ── Conversations ─────────────────────────────────────────────────────────────

export async function listConversations(): Promise<Conversation[]> {
	try {
		const data = await apiFetch<{ conversations: Conversation[] }>('/v1/conversations');
		return data.conversations ?? [];
	} catch {
		// Conversations endpoint may not exist yet — return empty list
		return [];
	}
}

export async function getConversation(id: string): Promise<ConversationDetail> {
	return apiFetch<ConversationDetail>(`/v1/conversations/${encodeURIComponent(id)}`);
}

export async function deleteConversation(id: string): Promise<void> {
	await apiFetch<unknown>(`/v1/conversations/${encodeURIComponent(id)}`, {
		method: 'DELETE'
	});
}

// ── Health ────────────────────────────────────────────────────────────────────

export async function healthCheck(): Promise<boolean> {
	try {
		await fetch(`${API_BASE}/health`);
		return true;
	} catch {
		return false;
	}
}
