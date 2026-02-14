package artifacts

import "time"

type ArtifactType string

type Artifact struct {
	ID            string                 `json:"id"`
	ProjectID     string                 `json:"project_id"`
	SessionID     string                 `json:"session_id,omitempty"`
	Type          ArtifactType           `json:"type"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description,omitempty"`
	LatestVersion int                    `json:"latest_version"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	Metadata      map[string]interface{} `json:"metadata"`
}

type ArtifactVersion struct {
	ID            string                 `json:"id"`
	ArtifactID    string                 `json:"artifact_id"`
	Version       int                    `json:"version"`
	Content       string                 `json:"content"`
	Diff          string                 `json:"diff,omitempty"`
	CommitMessage string                 `json:"commit_message,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	CreatedBy     string                 `json:"created_by"`
	Metadata      map[string]interface{} `json:"metadata"`
}

const (
	TypeDocument    ArtifactType = "document"
	TypeDiagram     ArtifactType = "diagram"
	TypeCodeProject ArtifactType = "code_project"
	TypeDataViz     ArtifactType = "data_viz"
	TypeImage       ArtifactType = "image"
)

const (
	CreatedByAgent = "agent"
	CreatedByUser  = "user"
)
