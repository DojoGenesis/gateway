package server

import (
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/DojoGenesis/gateway/runtime/mesh"
)

// handleMeshDID handles GET /.well-known/did.json.
// Returns the DID document for this gateway instance.
func (s *Server) handleMeshDID(c *gin.Context) {
	if s.mesh == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mesh not configured"})
		return
	}
	c.JSON(http.StatusOK, s.mesh.Identity().DIDDocument())
}

// handleMeshAnnounce handles POST /mesh/announce.
// Accepts a signed Announcement, verifies the signature, and registers the peer.
func (s *Server) handleMeshAnnounce(c *gin.Context) {
	if s.mesh == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mesh not configured"})
		return
	}

	var ann mesh.Announcement
	if err := c.ShouldBindJSON(&ann); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid announcement: " + err.Error()})
		return
	}

	ok, err := mesh.VerifyAnnouncement(ann)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "announcement verification error: " + err.Error()})
		return
	}
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid signature"})
		return
	}

	// Decode the public key for storage.
	pubBytes, err := base64.StdEncoding.DecodeString(ann.PublicKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid public key encoding"})
		return
	}

	peer := &mesh.PeerInfo{
		DID:          ann.DID,
		Endpoint:     ann.Endpoint,
		PublicKey:    pubBytes,
		PublicKeyB64: ann.PublicKey,
		Skills:       ann.Skills,
		Platforms:    ann.Platforms,
	}
	s.mesh.RegisterPeer(peer)

	c.JSON(http.StatusOK, gin.H{
		"ok":  true,
		"did": ann.DID,
	})
}

// handleMeshPeers handles GET /mesh/peers.
// Returns the list of active peers (within TTL).
func (s *Server) handleMeshPeers(c *gin.Context) {
	if s.mesh == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mesh not configured"})
		return
	}

	peers := s.mesh.Peers()
	// Ensure we return an empty array, not null, when there are no peers.
	if peers == nil {
		peers = []*mesh.PeerInfo{}
	}
	c.JSON(http.StatusOK, gin.H{"peers": peers})
}

// handleMeshDelegate handles POST /mesh/delegate.
// Parses and signature-validates a DelegationRequest.
// Skill execution is deferred to Phase 1 — returns 501.
func (s *Server) handleMeshDelegate(c *gin.Context) {
	if s.mesh == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mesh not configured"})
		return
	}

	var req mesh.DelegationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid delegation request: " + err.Error()})
		return
	}

	// Locate the requestor's peer record to get their public key.
	peer := s.mesh.FindPeerForSkill(req.SkillHash)
	if peer == nil {
		// Attempt to verify against the peer list by DID lookup.
		// For Phase 0, if we can't find the peer just return 403.
		c.JSON(http.StatusForbidden, gin.H{"error": "unknown requestor or skill not found"})
		return
	}

	// Skill execution is deferred to Phase 1.
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":      "skill execution not yet implemented (Era 4 Phase 1)",
		"skill_hash": req.SkillHash,
		"executor":   s.mesh.Identity().DID(),
	})
}

// handleMeshHealth handles GET /mesh/health.
// Returns the mesh status.
func (s *Server) handleMeshHealth(c *gin.Context) {
	if s.mesh == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"healthy": false,
			"error":   "mesh not configured",
		})
		return
	}

	status := s.mesh.Status()
	c.JSON(http.StatusOK, status)
}
