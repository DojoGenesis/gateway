package actor

import "context"

// AgentRef is a handle to a running agent actor.
type AgentRef struct {
	// ID is the unique agent identifier.
	ID string

	// IdentityName is the name of the ADA disposition identity loaded.
	IdentityName string
}

// AgentHandler processes messages for an agent.
type AgentHandler interface {
	// HandleMessage processes a single message.
	HandleMessage(ctx context.Context, self *AgentRef, msg Message) error

	// OnStart is called when the agent starts.
	OnStart(ctx context.Context, self *AgentRef) error

	// OnStop is called when the agent stops.
	OnStop(ctx context.Context, self *AgentRef) error
}

// SpawnOptions configures a new agent actor.
type SpawnOptions struct {
	// ID is the unique agent identifier.
	ID string

	// IdentityName is the ADA disposition identity to load.
	IdentityName string

	// Strategy is the supervision strategy for this agent.
	Strategy SupervisionStrategy

	// Handler implements the agent's message processing logic.
	Handler AgentHandler

	// MailboxSize is the channel buffer size (0 = use default).
	MailboxSize int
}

// SupervisorStats provides metrics about the supervisor.
type SupervisorStats struct {
	ActiveAgents    int
	TotalSpawned    int
	TotalRestarted  int
	TotalFailed     int
	MessagesHandled int64
}
