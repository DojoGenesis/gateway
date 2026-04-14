<script lang="ts">
	import { onMount, tick } from 'svelte';
	import { goto } from '$app/navigation';
	import { base } from '$app/paths';
	import {
		auth,
		messages,
		conversations,
		activeConversationId,
		currentConversationId,
		models,
		selectedModel,
		isStreaming,
		sidebarOpen,
		error,
		loadAuthFromStorage,
		clearAuth,
		setError,
		appendMessage,
		appendToLastAssistantMessage,
		clearMessages,
		setConversations,
		setModels
	} from '$lib/stores.svelte';
	import {
		streamChat,
		listModels,
		listConversations,
		createConversation,
		createMessage,
		listMessages
	} from '$lib/api';
	import { renderMarkdown } from '$lib/markdown';
	import type { ChatMessage } from '$lib/types';

	// ── DOM refs ──────────────────────────────────────────────────────────────
	let threadEl: HTMLElement | null = $state(null);
	let textareaEl: HTMLTextAreaElement | null = $state(null);
	let inputValue = $state('');

	// ── Init ──────────────────────────────────────────────────────────────────
	onMount(async () => {
		// Consume OAuth callback tokens from query params (GitHub OAuth flow).
		const params = new URLSearchParams(window.location.search);
		const oauthToken = params.get('access_token');
		if (oauthToken) {
			saveAuth(
				oauthToken,
				params.get('refresh_token') ?? '',
				params.get('user_id') ?? '',
				params.get('display_name') ?? ''
			);
			// Remove tokens from URL without triggering a navigation.
			history.replaceState({}, '', window.location.pathname);
		}

		loadAuthFromStorage();
		if (!auth.isAuthenticated) {
			goto(`${base}/login`);
			return;
		}
		await Promise.all([loadModels(), loadConversations()]);
	});

	async function loadModels() {
		try {
			const list = await listModels();
			setModels(list);
			if (list.length > 0 && !selectedModel.value) {
				selectedModel.value = list[0].id;
			}
		} catch {
			// Models load failure is non-fatal — user can still type a model name
		}
	}

	async function loadConversations() {
		try {
			const list = await listConversations();
			setConversations(list);
			// Restore most recent conversation on first load if none already active
			if (list.length > 0 && !currentConversationId.value) {
				await selectConversation(list[0].id, list[0].title);
			}
		} catch {
			// Not fatal — conversations sidebar just stays empty
		}
	}

	// ── Auto-resize textarea ──────────────────────────────────────────────────
	function handleTextareaInput(e: Event) {
		const el = e.target as HTMLTextAreaElement;
		inputValue = el.value;
		el.style.height = 'auto';
		el.style.height = Math.min(el.scrollHeight, 200) + 'px';
	}

	// ── Send on Enter (Shift+Enter = newline) ────────────────────────────────
	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			void sendMessage();
		}
	}

	// ── Scroll to bottom ──────────────────────────────────────────────────────
	async function scrollToBottom() {
		await tick();
		if (threadEl) {
			threadEl.scrollTop = threadEl.scrollHeight;
		}
	}

	// ── Send message ──────────────────────────────────────────────────────────
	async function sendMessage() {
		const text = inputValue.trim();
		if (!text || isStreaming.value) return;

		inputValue = '';
		if (textareaEl) {
			textareaEl.style.height = 'auto';
		}

		const model = selectedModel.value || 'claude-sonnet-4-20250514';

		// Ensure a conversation exists before sending
		if (!currentConversationId.value) {
			try {
				const title = text.slice(0, 50) || 'New conversation';
				const conv = await createConversation(title);
				currentConversationId.value = conv.id;
				activeConversationId.value = conv.id;
			} catch {
				// Persist failure is non-fatal — continue without a conversation ID
				setError('Could not create conversation — messages will not be saved');
			}
		}

		// Add user message to UI
		const userMsg: ChatMessage = { role: 'user', content: text };
		appendMessage(userMsg);
		await scrollToBottom();

		// Persist user message (non-blocking — don't await before showing UI)
		const convId = currentConversationId.value;
		if (convId) {
			createMessage(convId, 'user', text).catch(() => {
				// Non-fatal — continue even if persistence fails
			});
		}

		isStreaming.value = true;

		// Build context — include all current messages + new user message
		const context: ChatMessage[] = messages.map((m) => ({
			role: m.role,
			content: m.content
		}));

		let fullResponse = '';

		try {
			// Prime an empty assistant bubble
			appendMessage({ role: 'assistant', content: '' });
			await scrollToBottom();

			for await (const chunk of streamChat(context, model)) {
				appendToLastAssistantMessage(chunk);
				fullResponse += chunk;
				await scrollToBottom();
			}

			// Persist assistant message after stream completes
			if (convId && fullResponse) {
				createMessage(convId, 'assistant', fullResponse, model).catch(() => {
					// Non-fatal
				});
			}

			// Refresh sidebar conversation list
			loadConversations().catch(() => {});
		} catch (err) {
			const msg = err instanceof Error ? err.message : 'Stream error';
			setError(msg);
			// Remove the empty assistant bubble on error
			if (messages.length > 0 && messages[messages.length - 1].content === '') {
				messages.splice(messages.length - 1, 1);
			}
		} finally {
			isStreaming.value = false;
		}
	}

	// ── New conversation ──────────────────────────────────────────────────────
	function newConversation() {
		clearMessages();
		activeConversationId.value = '';
		currentConversationId.value = null;
		// Focus the input after clearing
		tick().then(() => {
			textareaEl?.focus();
		});
	}

	// ── Sign out ──────────────────────────────────────────────────────────────
	function signOut() {
		clearAuth();
		goto(`${base}/login`);
	}

	// ── Conversation select ───────────────────────────────────────────────────
	async function selectConversation(id: string, _title: string) {
		clearMessages();
		activeConversationId.value = id;
		currentConversationId.value = id;

		try {
			const msgs = await listMessages(id);
			// Map ConversationMessage → ChatMessage for the store
			for (const m of msgs) {
				appendMessage({ role: m.role, content: m.content });
			}
		} catch {
			// Non-fatal — conversation is selected but history couldn't load
			setError('Could not load conversation history');
		}

		await scrollToBottom();
	}

	// ── Format date for sidebar ───────────────────────────────────────────────
	function formatDate(iso: string): string {
		try {
			const d = new Date(iso);
			const now = new Date();
			const diff = now.getTime() - d.getTime();
			if (diff < 60_000) return 'just now';
			if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
			if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
			return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
		} catch {
			return iso;
		}
	}
</script>

<svelte:head>
	<title>Dojo Chat</title>
</svelte:head>

<div class="app-shell">
	<!-- ── Sidebar ────────────────────────────────────────────────────────── -->
	<aside class="sidebar" class:collapsed={!sidebarOpen.value}>
		<div class="sidebar-header">
			<span class="sidebar-title">Conversations</span>
			<button
				class="btn-icon"
				title="New conversation"
				onclick={newConversation}
			>
				<svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.75">
					<line x1="8" y1="2" x2="8" y2="14" />
					<line x1="2" y1="8" x2="14" y2="8" />
				</svg>
			</button>
		</div>

		<div class="conversation-list">
			{#if conversations.length === 0}
				<p class="sidebar-empty">No conversations yet</p>
			{:else}
				{#each conversations as conv (conv.id)}
					<button
						class="conversation-item"
						class:active={activeConversationId.value === conv.id}
						onclick={() => void selectConversation(conv.id, conv.title)}
					>
						<span class="conv-title">{conv.title}</span>
						<span class="conv-meta">{formatDate(conv.updated_at)} · {conv.message_count} msg</span>
					</button>
				{/each}
			{/if}
		</div>
	</aside>

	<!-- ── Main panel ─────────────────────────────────────────────────────── -->
	<main class="main-panel">
		<!-- Header -->
		<header class="chat-header">
			<div class="header-left">
				<button
					class="btn-icon"
					title="Toggle sidebar"
					onclick={() => (sidebarOpen.value = !sidebarOpen.value)}
				>
					<svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.75">
						<line x1="2" y1="4" x2="14" y2="4" />
						<line x1="2" y1="8" x2="14" y2="8" />
						<line x1="2" y1="12" x2="14" y2="12" />
					</svg>
				</button>
				<span class="app-logo">Dojo <span>Chat</span></span>
			</div>

			<div class="header-right">
				{#if auth.displayName}
					<span class="user-badge">{auth.displayName}</span>
				{/if}
				<button class="btn-text" onclick={signOut}>Sign out</button>
			</div>
		</header>

		<!-- Message thread -->
		<div class="message-thread" bind:this={threadEl}>
			<div class="thread-inner">
				{#if messages.length === 0}
					<div class="empty-state">
						<p class="empty-state-title">What would you like to explore?</p>
						<p class="empty-state-sub">Type a message below to start a conversation.</p>
					</div>
				{:else}
					{#each messages as msg, i (i)}
						<div class="message-row {msg.role}">
							<div class="message-bubble">
								{#if msg.role === 'assistant'}
									<div
										class="markdown-content"
										role="article"
										aria-label="Assistant message"
									>{@html renderMarkdown(msg.content)}</div>
								{:else}
									{msg.content}
								{/if}
							</div>
						</div>
					{/each}
				{/if}

				{#if isStreaming.value && messages.length > 0 && messages[messages.length - 1].content === ''}
					<div class="typing-indicator">
						<div class="typing-dots">
							<span></span>
							<span></span>
							<span></span>
						</div>
					</div>
				{/if}
			</div>
		</div>

		<!-- Input bar -->
		<div class="input-bar">
			<div class="input-bar-inner">
				<div class="input-controls">
					<div class="input-wrapper">
						<textarea
							bind:this={textareaEl}
							class="chat-textarea"
							placeholder="Message Dojo Chat…"
							rows="1"
							value={inputValue}
							oninput={handleTextareaInput}
							onkeydown={handleKeydown}
							disabled={isStreaming.value}
						></textarea>
					</div>

					<button
						class="btn-send"
						title="Send message"
						disabled={!inputValue.trim() || isStreaming.value}
						onclick={() => void sendMessage()}
					>
						<svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
							<path d="M2 14L14 8 2 2v4.5l8 1.5-8 1.5z" />
						</svg>
					</button>
				</div>

				<div class="input-meta">
					{#if models.length > 0}
						<select
							class="model-select"
							bind:value={selectedModel.value}
							disabled={isStreaming.value}
						>
							{#each models as m (m.id)}
								<option value={m.id}>{m.id}</option>
							{/each}
						</select>
					{:else}
						<span class="input-hint">No models loaded</span>
					{/if}

					<span class="input-hint">Enter to send · Shift+Enter for newline</span>
				</div>
			</div>
		</div>
	</main>
</div>

{#if error.message}
	<div class="error-toast" role="alert">{error.message}</div>
{/if}
