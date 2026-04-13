// Svelte 5 reactive stores using $state runes.
// Must be a .svelte.ts file to use runes outside of .svelte components.

import type { ChatMessage, Conversation, Model } from './types';

// ── Auth ─────────────────────────────────────────────────────────────────────

export const auth = $state({
	token: '',
	refreshToken: '',
	userId: '',
	displayName: '',
	isAuthenticated: false
});

export function loadAuthFromStorage(): void {
	if (typeof localStorage === 'undefined') return;
	const token = localStorage.getItem('access_token') ?? '';
	const refreshTok = localStorage.getItem('refresh_token') ?? '';
	const userId = localStorage.getItem('user_id') ?? '';
	const displayName = localStorage.getItem('display_name') ?? '';
	auth.token = token;
	auth.refreshToken = refreshTok;
	auth.userId = userId;
	auth.displayName = displayName;
	auth.isAuthenticated = !!token;
}

export function saveAuth(
	token: string,
	refreshTok: string,
	userId: string,
	displayName: string
): void {
	auth.token = token;
	auth.refreshToken = refreshTok;
	auth.userId = userId;
	auth.displayName = displayName;
	auth.isAuthenticated = true;
	if (typeof localStorage !== 'undefined') {
		localStorage.setItem('access_token', token);
		localStorage.setItem('refresh_token', refreshTok);
		localStorage.setItem('user_id', userId);
		localStorage.setItem('display_name', displayName);
	}
}

export function clearAuth(): void {
	auth.token = '';
	auth.refreshToken = '';
	auth.userId = '';
	auth.displayName = '';
	auth.isAuthenticated = false;
	if (typeof localStorage !== 'undefined') {
		localStorage.removeItem('access_token');
		localStorage.removeItem('refresh_token');
		localStorage.removeItem('user_id');
		localStorage.removeItem('display_name');
	}
}

// ── Conversation / messages ───────────────────────────────────────────────────

export const messages = $state<ChatMessage[]>([]);

export function appendMessage(msg: ChatMessage): void {
	messages.push(msg);
}

export function appendToLastAssistantMessage(chunk: string): void {
	const last = messages[messages.length - 1];
	if (last && last.role === 'assistant') {
		last.content += chunk;
	} else {
		messages.push({ role: 'assistant', content: chunk });
	}
}

export function clearMessages(): void {
	messages.splice(0, messages.length);
}

// ── Conversation list ─────────────────────────────────────────────────────────

export const conversations = $state<Conversation[]>([]);

export function setConversations(list: Conversation[]): void {
	conversations.splice(0, conversations.length, ...list);
}

export const activeConversationId = $state({ value: '' });

// ── Models ───────────────────────────────────────────────────────────────────

export const models = $state<Model[]>([]);

export function setModels(list: Model[]): void {
	models.splice(0, models.length, ...list);
}

export const selectedModel = $state({ value: '' });

// ── Streaming ────────────────────────────────────────────────────────────────

export const isStreaming = $state({ value: false });

// ── UI ────────────────────────────────────────────────────────────────────────

export const sidebarOpen = $state({ value: true });

export const error = $state({ message: '' });

export function setError(msg: string): void {
	error.message = msg;
	if (msg) {
		setTimeout(() => {
			error.message = '';
		}, 5000);
	}
}
