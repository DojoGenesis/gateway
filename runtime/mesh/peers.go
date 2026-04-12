package mesh

// PeerStore defines the interface for persistent peer storage.
// Phase 0 uses in-memory only. Future phases may persist to CAS or D1.
type PeerStore interface {
	Save(peer *PeerInfo) error
	Load(did string) (*PeerInfo, error)
	All() ([]*PeerInfo, error)
	Remove(did string) error
}
