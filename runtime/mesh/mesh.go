package mesh

import (
	"crypto/ed25519"
	"sync"
	"time"
)

// PeerInfo represents a known peer in the mesh.
type PeerInfo struct {
	DID          string           `json:"did"`
	Endpoint     string           `json:"endpoint"`
	PublicKey    ed25519.PublicKey `json:"-"`
	PublicKeyB64 string           `json:"public_key"`
	Skills       []string         `json:"skills"`
	Platforms    []string         `json:"platforms"`
	LastAnnounce time.Time        `json:"last_announce"`
	TTL          time.Duration    `json:"ttl"`
}

// Announcement is the signed payload a peer sends to register.
type Announcement struct {
	DID       string   `json:"did"`
	Endpoint  string   `json:"endpoint"`
	Skills    []string `json:"skills"`
	Platforms []string `json:"platforms"`
	Timestamp string   `json:"timestamp"`
	Signature string   `json:"signature"` // base64-encoded Ed25519 signature
	PublicKey string   `json:"public_key"` // base64-encoded public key
}

// DelegationRequest is a signed request to execute a skill on a remote peer.
type DelegationRequest struct {
	SkillHash    string `json:"skill_hash"`
	Input        string `json:"input"`
	CallbackURL  string `json:"callback_url,omitempty"`
	RequestorDID string `json:"requestor_did"`
	Timestamp    string `json:"timestamp"`
	Signature    string `json:"signature"`
}

// DelegationResponse is the result of a delegated skill execution.
type DelegationResponse struct {
	SkillHash   string `json:"skill_hash"`
	Output      string `json:"output"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
	ExecutorDID string `json:"executor_did"`
	Timestamp   string `json:"timestamp"`
	Signature   string `json:"signature"`
}

// MeshStatus reports the state of the mesh.
type MeshStatus struct {
	InstanceDID  string    `json:"instance_did"`
	PeerCount    int       `json:"peer_count"`
	LastAnnounce time.Time `json:"last_announce"`
	Healthy      bool      `json:"healthy"`
}

// Mesh manages the federated agent mesh.
type Mesh struct {
	identity *Identity
	mu       sync.RWMutex
	peers    map[string]*PeerInfo // keyed by DID
	ttl      time.Duration
}

// New creates a new Mesh with the given identity and peer TTL.
func New(identity *Identity, ttl time.Duration) *Mesh {
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	return &Mesh{
		identity: identity,
		peers:    make(map[string]*PeerInfo),
		ttl:      ttl,
	}
}

// Identity returns the mesh's identity.
func (m *Mesh) Identity() *Identity {
	return m.identity
}

// RegisterPeer adds or updates a peer from a verified announcement.
func (m *Mesh) RegisterPeer(peer *PeerInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	peer.LastAnnounce = time.Now().UTC()
	peer.TTL = m.ttl
	m.peers[peer.DID] = peer
}

// Peers returns all peers that have announced within the TTL window.
func (m *Mesh) Peers() []*PeerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	var active []*PeerInfo
	for _, p := range m.peers {
		if now.Sub(p.LastAnnounce) <= p.TTL {
			active = append(active, p)
		}
	}
	return active
}

// FindPeerForSkill finds a peer that advertises the given skill hash.
func (m *Mesh) FindPeerForSkill(skillHash string) *PeerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	for _, p := range m.peers {
		if now.Sub(p.LastAnnounce) > p.TTL {
			continue
		}
		for _, s := range p.Skills {
			if s == skillHash {
				return p
			}
		}
	}
	return nil
}

// Status returns the mesh status.
func (m *Mesh) Status() MeshStatus {
	peers := m.Peers()
	return MeshStatus{
		InstanceDID: m.identity.DID(),
		PeerCount:   len(peers),
		Healthy:     true,
	}
}
