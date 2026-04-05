package workflow

// CanvasState stores the visual layout of a workflow in the builder.
// It is stored as workflow.canvas.json in CAS, separate from the execution definition.
// Moving nodes without changing logic only creates a new canvas hash.
type CanvasState struct {
	// WorkflowRef is the CAS hash of the corresponding workflow.json.
	WorkflowRef string `json:"workflow_ref"`

	// Viewport stores the current pan and zoom of the canvas.
	Viewport Viewport `json:"viewport"`

	// NodePositions maps step IDs to their (x, y) positions on the canvas.
	NodePositions map[string]Position `json:"node_positions"`

	// EdgeStyles maps edge keys to visual styling overrides.
	EdgeStyles map[string]EdgeStyle `json:"edge_styles,omitempty"`

	// Selection holds the IDs of currently selected nodes.
	Selection []string `json:"selection,omitempty"`
}

// Viewport stores the pan and zoom state of the canvas.
type Viewport struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Zoom float64 `json:"zoom"`
}

// Position represents a 2D coordinate on the canvas.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// EdgeStyle holds visual styling overrides for a workflow edge.
type EdgeStyle struct {
	Color    string `json:"color,omitempty"`
	Animated bool   `json:"animated,omitempty"`
}
