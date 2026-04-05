// TypeScript types mirroring the Go structs in AgenticGatewayByDojoGenesis/workflow/
// Follows ADR-019 two-layer schema split.

export interface WorkflowDefinition {
	version: string;
	name: string;
	description?: string;
	artifact_type: string;
	steps: Step[];
	trigger?: Trigger;
	metadata?: Record<string, string>;
}

export interface Step {
	id: string;
	skill: string;
	inputs: Record<string, string>;
	depends_on: string[];
}

export interface Trigger {
	type: 'channel_message' | 'schedule' | 'manual' | 'webhook';
	platform?: string;
	pattern?: string;
	cron?: string;
}

export interface CanvasState {
	workflow_ref: string;
	viewport: { x: number; y: number; zoom: number };
	node_positions: Record<string, { x: number; y: number }>;
	edge_styles?: Record<string, { color?: string; animated?: boolean }>;
	selection?: string[];
}

export interface PortDefinition {
	name: string;
	type: string;
	description?: string;
	required?: boolean;
	default?: unknown;
	enum?: string[];
}

export interface SkillInfo {
	name: string;
	version: string;
	description: string;
	inputs?: PortDefinition[];
	outputs?: PortDefinition[];
}

// Status for execution state (WebSocket bridge, Phase 2+)
export type StepStatus = 'pending' | 'running' | 'completed' | 'failed' | 'skipped';

// Node data shape used by SkillNode
export interface SkillNodeData extends Record<string, unknown> {
	label: string;
	skill: string;
	inputs: PortDefinition[];
	outputs: PortDefinition[];
	status?: StepStatus;
}
