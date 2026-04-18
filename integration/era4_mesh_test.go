package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/runtime/mesh"
)

// ---------------------------------------------------------------------------
// meshTestNode — wraps a mesh.Mesh + httptest.Server for integration tests.
// Mirrors the routes from server/handle_mesh.go + server/router.go
// without requiring the full Server struct.
// ---------------------------------------------------------------------------

type meshTestNode struct {
	mesh   *mesh.Mesh
	server *httptest.Server
}

func newMeshTestNode(t *testing.T, hostname string, ttl time.Duration) *meshTestNode {
	t.Helper()

	identity, err := mesh.GenerateIdentity(hostname)
	if err != nil {
		t.Fatalf("GenerateIdentity(%s): %v", hostname, err)
	}

	m := mesh.New(identity, ttl)
	mux := http.NewServeMux()

	// GET /.well-known/did.json
	mux.HandleFunc("GET /.well-known/did.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(m.Identity().DIDDocument())
	})

	// POST /mesh/announce
	mux.HandleFunc("POST /mesh/announce", func(w http.ResponseWriter, r *http.Request) {
		var ann mesh.Announcement
		if err := json.NewDecoder(r.Body).Decode(&ann); err != nil {
			http.Error(w, `{"error":"invalid announcement"}`, http.StatusBadRequest)
			return
		}
		ok, err := mesh.VerifyAnnouncement(ann)
		if err != nil {
			http.Error(w, `{"error":"verification error"}`, http.StatusBadRequest)
			return
		}
		if !ok {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid signature"})
			return
		}
		peer := &mesh.PeerInfo{
			DID:          ann.DID,
			Endpoint:     ann.Endpoint,
			PublicKeyB64: ann.PublicKey,
			Skills:       ann.Skills,
			Platforms:    ann.Platforms,
		}
		m.RegisterPeer(peer)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "did": ann.DID})
	})

	// GET /mesh/peers
	mux.HandleFunc("GET /mesh/peers", func(w http.ResponseWriter, r *http.Request) {
		peers := m.Peers()
		if peers == nil {
			peers = []*mesh.PeerInfo{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"peers": peers})
	})

	// POST /mesh/delegate
	mux.HandleFunc("POST /mesh/delegate", func(w http.ResponseWriter, r *http.Request) {
		var req mesh.DelegationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
			return
		}
		peer := m.FindPeerForSkill(req.SkillHash)
		if peer == nil {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "unknown requestor or skill not found"})
			return
		}
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":      "skill execution not yet implemented (Era 4 Phase 1)",
			"skill_hash": req.SkillHash,
			"executor":   m.Identity().DID(),
		})
	})

	// GET /mesh/health
	mux.HandleFunc("GET /mesh/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(m.Status())
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return &meshTestNode{mesh: m, server: srv}
}

// ---------------------------------------------------------------------------
// Track D Integration Tests — Era 4
// ---------------------------------------------------------------------------

// TestEra4_DIDDocument verifies that GET /.well-known/did.json returns a
// well-formed DID Document with required fields.
func TestEra4_DIDDocument(t *testing.T) {
	t.Parallel()

	node := newMeshTestNode(t, "gateway.example.com", 5*time.Minute)

	resp, err := http.Get(node.server.URL + "/.well-known/did.json")
	if err != nil {
		t.Fatalf("GET did.json: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var doc map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Required DID fields
	if _, ok := doc["@context"]; !ok {
		t.Error("missing @context")
	}
	id, ok := doc["id"].(string)
	if !ok || id != "did:web:gateway.example.com" {
		t.Errorf("id = %v, want did:web:gateway.example.com", doc["id"])
	}
	if _, ok := doc["verificationMethod"]; !ok {
		t.Error("missing verificationMethod")
	}
	if _, ok := doc["authentication"]; !ok {
		t.Error("missing authentication")
	}
	if _, ok := doc["service"]; !ok {
		t.Error("missing service")
	}
}

// TestEra4_AnnounceRoundTrip verifies that a signed announcement is accepted,
// and the peer appears in GET /mesh/peers.
func TestEra4_AnnounceRoundTrip(t *testing.T) {
	t.Parallel()

	nodeA := newMeshTestNode(t, "peer-a.example.com", 5*time.Minute)
	nodeB := newMeshTestNode(t, "peer-b.example.com", 5*time.Minute)

	// A signs an announcement and sends it to B
	ann := nodeA.mesh.Identity().SignAnnouncement(
		nodeA.server.URL+"/mesh",
		[]string{"sha256:skill-alpha", "sha256:skill-beta"},
		[]string{"claude", "openai"},
	)

	body, _ := json.Marshal(ann)
	resp, err := http.Post(nodeB.server.URL+"/mesh/announce", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST announce: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("announce status = %d, body = %s", resp.StatusCode, respBody)
	}

	var announceResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&announceResp)
	if announceResp["ok"] != true {
		t.Errorf("announce response ok = %v", announceResp["ok"])
	}

	// B's peer list should now contain A
	resp2, err := http.Get(nodeB.server.URL + "/mesh/peers")
	if err != nil {
		t.Fatalf("GET peers: %v", err)
	}
	defer resp2.Body.Close()

	var peersResp struct {
		Peers []struct {
			DID      string   `json:"did"`
			Skills   []string `json:"skills"`
			Endpoint string   `json:"endpoint"`
		} `json:"peers"`
	}
	json.NewDecoder(resp2.Body).Decode(&peersResp)

	if len(peersResp.Peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(peersResp.Peers))
	}
	if peersResp.Peers[0].DID != "did:web:peer-a.example.com" {
		t.Errorf("peer DID = %q", peersResp.Peers[0].DID)
	}
	if len(peersResp.Peers[0].Skills) != 2 {
		t.Errorf("peer skills = %v, want 2 entries", peersResp.Peers[0].Skills)
	}
}

// TestEra4_AnnounceTamperedSignature verifies that a tampered announcement
// is rejected with 403.
func TestEra4_AnnounceTamperedSignature(t *testing.T) {
	t.Parallel()

	node := newMeshTestNode(t, "gateway.example.com", 5*time.Minute)

	identity, err := mesh.GenerateIdentity("evil.example.com")
	if err != nil {
		t.Fatal(err)
	}

	ann := identity.SignAnnouncement(
		"https://evil.example.com/mesh",
		[]string{"sha256:skill-x"},
		[]string{"claude"},
	)

	// Tamper with the endpoint after signing
	ann.Endpoint = "https://hijacked.example.com/mesh"

	body, _ := json.Marshal(ann)
	resp, err := http.Post(node.server.URL+"/mesh/announce", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("tampered announce status = %d, want 403", resp.StatusCode)
	}
}

// TestEra4_DelegateReturns501 verifies that POST /mesh/delegate returns 501
// (skill execution is deferred to Phase 1).
func TestEra4_DelegateReturns501(t *testing.T) {
	t.Parallel()

	nodeA := newMeshTestNode(t, "peer-a.example.com", 5*time.Minute)
	nodeB := newMeshTestNode(t, "peer-b.example.com", 5*time.Minute)

	// A announces to B with a specific skill
	ann := nodeA.mesh.Identity().SignAnnouncement(
		nodeA.server.URL+"/mesh",
		[]string{"sha256:delegated-skill"},
		[]string{"claude"},
	)
	body, _ := json.Marshal(ann)
	resp, err := http.Post(nodeB.server.URL+"/mesh/announce", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Now try to delegate that skill to B
	delegateReq := mesh.DelegationRequest{
		SkillHash:    "sha256:delegated-skill",
		Input:        "test input",
		RequestorDID: nodeA.mesh.Identity().DID(),
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}
	delegateBody, _ := json.Marshal(delegateReq)
	resp2, err := http.Post(nodeB.server.URL+"/mesh/delegate", "application/json", bytes.NewReader(delegateBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusNotImplemented {
		t.Fatalf("delegate status = %d, want 501", resp2.StatusCode)
	}

	var delegateResp map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&delegateResp)

	errMsg, _ := delegateResp["error"].(string)
	if errMsg == "" {
		t.Error("expected error message in 501 response")
	}

	executor, _ := delegateResp["executor"].(string)
	if executor != nodeB.mesh.Identity().DID() {
		t.Errorf("executor = %q, want %q", executor, nodeB.mesh.Identity().DID())
	}
}

// TestEra4_DelegateUnknownSkill verifies that delegation for an unknown
// skill returns 403.
func TestEra4_DelegateUnknownSkill(t *testing.T) {
	t.Parallel()

	node := newMeshTestNode(t, "gateway.example.com", 5*time.Minute)

	req := mesh.DelegationRequest{
		SkillHash:    "sha256:nonexistent",
		Input:        "test",
		RequestorDID: "did:web:unknown.example.com",
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}
	body, _ := json.Marshal(req)
	resp, err := http.Post(node.server.URL+"/mesh/delegate", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("unknown skill delegate status = %d, want 403", resp.StatusCode)
	}
}

// TestEra4_MeshHealth verifies the health endpoint returns correct status.
func TestEra4_MeshHealth(t *testing.T) {
	t.Parallel()

	node := newMeshTestNode(t, "gateway.example.com", 5*time.Minute)

	resp, err := http.Get(node.server.URL + "/mesh/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var status mesh.MeshStatus
	json.NewDecoder(resp.Body).Decode(&status)

	if status.InstanceDID != "did:web:gateway.example.com" {
		t.Errorf("instance_did = %q", status.InstanceDID)
	}
	if !status.Healthy {
		t.Error("expected healthy = true")
	}
	if status.PeerCount != 0 {
		t.Errorf("peer_count = %d, want 0", status.PeerCount)
	}
}

// TestEra4_MeshHealthAfterAnnounce verifies peer count updates after an announcement.
func TestEra4_MeshHealthAfterAnnounce(t *testing.T) {
	t.Parallel()

	nodeA := newMeshTestNode(t, "peer-a.example.com", 5*time.Minute)
	nodeB := newMeshTestNode(t, "peer-b.example.com", 5*time.Minute)

	// Announce A to B
	ann := nodeA.mesh.Identity().SignAnnouncement(
		nodeA.server.URL+"/mesh",
		[]string{"sha256:s1"},
		[]string{"claude"},
	)
	body, _ := json.Marshal(ann)
	resp, _ := http.Post(nodeB.server.URL+"/mesh/announce", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// Check B's health
	resp2, err := http.Get(nodeB.server.URL + "/mesh/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()

	var status mesh.MeshStatus
	json.NewDecoder(resp2.Body).Decode(&status)

	if status.PeerCount != 1 {
		t.Errorf("peer_count = %d after announce, want 1", status.PeerCount)
	}
}

// TestEra4_PeerTTLExpiry verifies that peers with expired TTL are pruned.
func TestEra4_PeerTTLExpiry(t *testing.T) {
	t.Parallel()

	// Use a very short TTL
	node := newMeshTestNode(t, "gateway.example.com", 500*time.Millisecond)

	identity, err := mesh.GenerateIdentity("ephemeral.example.com")
	if err != nil {
		t.Fatal(err)
	}

	ann := identity.SignAnnouncement(
		"https://ephemeral.example.com/mesh",
		[]string{"sha256:tmp"},
		[]string{"claude"},
	)

	body, _ := json.Marshal(ann)
	resp, err := http.Post(node.server.URL+"/mesh/announce", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Peer should be visible immediately
	resp2, _ := http.Get(node.server.URL + "/mesh/peers")
	var peersResp struct {
		Peers []struct{ DID string } `json:"peers"`
	}
	json.NewDecoder(resp2.Body).Decode(&peersResp)
	resp2.Body.Close()

	if len(peersResp.Peers) != 1 {
		t.Fatalf("expected 1 peer before TTL, got %d", len(peersResp.Peers))
	}

	// Wait for TTL to expire
	time.Sleep(700 * time.Millisecond)

	resp3, _ := http.Get(node.server.URL + "/mesh/peers")
	var peersResp2 struct {
		Peers []struct{ DID string } `json:"peers"`
	}
	json.NewDecoder(resp3.Body).Decode(&peersResp2)
	resp3.Body.Close()

	if len(peersResp2.Peers) != 0 {
		t.Errorf("expected 0 peers after TTL expiry, got %d", len(peersResp2.Peers))
	}
}

// TestEra4_TwoInstanceFederation is the end-to-end federation test:
// Instance A and Instance B each have their own identity.
// A announces to B. B announces to A.
// Both see each other as peers. Delegation flows correctly.
func TestEra4_TwoInstanceFederation(t *testing.T) {
	t.Parallel()

	nodeA := newMeshTestNode(t, "instance-a.dojo.dev", 5*time.Minute)
	nodeB := newMeshTestNode(t, "instance-b.dojo.dev", 5*time.Minute)

	// A announces to B
	annA := nodeA.mesh.Identity().SignAnnouncement(
		nodeA.server.URL+"/mesh",
		[]string{"sha256:skill-research", "sha256:skill-code-gen"},
		[]string{"claude", "openai"},
	)
	bodyA, _ := json.Marshal(annA)
	r1, _ := http.Post(nodeB.server.URL+"/mesh/announce", "application/json", bytes.NewReader(bodyA))
	r1.Body.Close()

	// B announces to A
	annB := nodeB.mesh.Identity().SignAnnouncement(
		nodeB.server.URL+"/mesh",
		[]string{"sha256:skill-scout", "sha256:skill-forge"},
		[]string{"claude"},
	)
	bodyB, _ := json.Marshal(annB)
	r2, _ := http.Post(nodeA.server.URL+"/mesh/announce", "application/json", bytes.NewReader(bodyB))
	r2.Body.Close()

	// A sees B as a peer
	r3, _ := http.Get(nodeA.server.URL + "/mesh/peers")
	var peersA struct {
		Peers []struct {
			DID    string   `json:"did"`
			Skills []string `json:"skills"`
		} `json:"peers"`
	}
	json.NewDecoder(r3.Body).Decode(&peersA)
	r3.Body.Close()

	if len(peersA.Peers) != 1 {
		t.Fatalf("A sees %d peers, want 1", len(peersA.Peers))
	}
	if peersA.Peers[0].DID != "did:web:instance-b.dojo.dev" {
		t.Errorf("A's peer DID = %q", peersA.Peers[0].DID)
	}

	// B sees A as a peer
	r4, _ := http.Get(nodeB.server.URL + "/mesh/peers")
	var peersB struct {
		Peers []struct {
			DID    string   `json:"did"`
			Skills []string `json:"skills"`
		} `json:"peers"`
	}
	json.NewDecoder(r4.Body).Decode(&peersB)
	r4.Body.Close()

	if len(peersB.Peers) != 1 {
		t.Fatalf("B sees %d peers, want 1", len(peersB.Peers))
	}
	if peersB.Peers[0].DID != "did:web:instance-a.dojo.dev" {
		t.Errorf("B's peer DID = %q", peersB.Peers[0].DID)
	}

	// A delegates a skill to B — B should return 501 (Phase 0 gate)
	// First, B needs to have A's skill registered so FindPeerForSkill works.
	// B has A registered with skill-research. Try delegating skill-research to B.
	delegateReq := mesh.DelegationRequest{
		SkillHash:    "sha256:skill-research",
		Input:        `{"topic": "federated AI agents"}`,
		RequestorDID: nodeA.mesh.Identity().DID(),
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}
	delegateBody, _ := json.Marshal(delegateReq)
	r5, err := http.Post(nodeB.server.URL+"/mesh/delegate", "application/json", bytes.NewReader(delegateBody))
	if err != nil {
		t.Fatalf("delegate: %v", err)
	}
	defer r5.Body.Close()

	if r5.StatusCode != http.StatusNotImplemented {
		body, _ := io.ReadAll(r5.Body)
		t.Fatalf("delegate status = %d, want 501, body = %s", r5.StatusCode, body)
	}

	var delegateResp map[string]interface{}
	json.NewDecoder(r5.Body).Decode(&delegateResp)

	executor, _ := delegateResp["executor"].(string)
	if executor != nodeB.mesh.Identity().DID() {
		t.Errorf("executor DID = %q, want %q", executor, nodeB.mesh.Identity().DID())
	}
}

// TestEra4_EmptyPeersReturnsArray verifies the peers endpoint returns an
// empty JSON array (not null) when no peers exist.
func TestEra4_EmptyPeersReturnsArray(t *testing.T) {
	t.Parallel()

	node := newMeshTestNode(t, "gateway.example.com", 5*time.Minute)

	resp, err := http.Get(node.server.URL + "/mesh/peers")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var raw map[string]json.RawMessage
	json.Unmarshal(body, &raw)

	// Must be [] not null
	if string(raw["peers"]) == "null" {
		t.Error("peers should be [] not null when empty")
	}
}
