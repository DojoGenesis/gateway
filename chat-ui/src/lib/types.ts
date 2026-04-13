// TypeScript types for the Dojo Gateway chat API.

export interface ChatMessage {
	role: 'user' | 'assistant' | 'system';
	content: string;
}

export interface AuthResponse {
	user_id: string;
	display_name: string;
	access_token: string;
	refresh_token: string;
}

export interface Model {
	id: string;
	object: string;
	owned_by: string;
}

export interface ModelsResponse {
	object: string;
	data: Model[];
}

export interface Conversation {
	id: string;
	title: string;
	updated_at: string;
	message_count: number;
}

export interface ConversationMessage {
	id: string;
	conversation_id: string;
	role: 'user' | 'assistant' | 'system';
	content: string;
	created_at: string;
	model?: string;
}

export interface ConversationDetail {
	id: string;
	title: string;
	updated_at: string;
	messages: ConversationMessage[];
}

// OpenAI-compatible SSE chunk shape
export interface ChatCompletionChunk {
	id: string;
	object: string;
	created: number;
	model: string;
	choices: Array<{
		index: number;
		delta: {
			role?: string;
			content?: string;
		};
		finish_reason: string | null;
	}>;
}
