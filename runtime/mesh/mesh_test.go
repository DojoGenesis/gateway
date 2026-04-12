package mesh

import (
	"strings"
	"testing"
	"time"
)

// TestGenerateIdentity verifies that a new identity is created with a valid
// Ed25519 keypair and a correctly-formatted DID.
func TestGenerateIdentity(t *testing.T) {
	id, err := GenerateIdentity("example.com")
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	if id.DID() != "did:web:example.com" {
		t.Errorf("DID = %q, want %q", id.DID(), "did:web:example.com")
	}

	if id.PublicKeyBase64() == "" {
		t.Error("PublicKeyBase64 is empty")
	}

	if id.PrivateKeyBase64() == "" {
		t.Error("PrivateKeyBase64 is empty")
	}
}

// TestLoadIdentity verifies round-trip serialisation of the private key.
func TestLoadIdentity(t *testing.T) {
	original, err := GenerateIdentity("example.com")
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	loaded, err := LoadIdentity(original.DID(), original.PrivateKeyBase64())
	if err != nil {
		t.Fatalf("LoadIdentity: %v", err)
	}

	if loaded.DID() != original.DID() {
		t.Errorf("DID mismatch: got %q, want %q", loaded.DID(), original.DID())
	}

	if loaded.PublicKeyBase64() != original.PublicKeyBase64() {
		t.Error("public key mismatch after load")
	}
}

// TestSignVerify verifies that Sign + Verify form a correct round-trip.
func TestSignVerify(t *testing.T) {
	id, err := GenerateIdentity("example.com")
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	message := []byte("hello mesh")
	sig := id.Sign(message)

	ok, err := Verify(id.PublicKeyBase64(), message, sig)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !ok {
		t.Error("signature verification failed")
	}

	// Tampered message must not verify.
	ok, err = Verify(id.PublicKeyBase64(), []byte("tampered"), sig)
	if err != nil {
		t.Fatalf("Verify (tampered): %v", err)
	}
	if ok {
		t.Error("tampered message should not verify")
	}
}

// TestSignAnnouncement verifies that a signed announcement round-trips correctly.
func TestSignAnnouncement(t *testing.T) {
	id, err := GenerateIdentity("gateway.example.com")
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	ann := id.SignAnnouncement(
		"https://gateway.example.com/mesh",
		[]string{"sha256:abc123", "sha256:def456"},
		[]string{"claude", "openai"},
	)

	if ann.DID != id.DID() {
		t.Errorf("announcement DID = %q, want %q", ann.DID, id.DID())
	}
	if ann.Signature == "" {
		t.Error("announcement signature is empty")
	}

	ok, err := VerifyAnnouncement(ann)
	if err != nil {
		t.Fatalf("VerifyAnnouncement: %v", err)
	}
	if !ok {
		t.Error("announcement verification failed")
	}

	// Tamper with endpoint and ensure verification fails.
	ann.Endpoint = "https://evil.example.com/mesh"
	ok, err = VerifyAnnouncement(ann)
	if err != nil {
		t.Fatalf("VerifyAnnouncement (tampered): %v", err)
	}
	if ok {
		t.Error("tampered announcement should not verify")
	}
}

// TestMeshRegisterPeerAndPeers tests peer registration and active peer listing.
func TestMeshRegisterPeerAndPeers(t *testing.T) {
	id, err := GenerateIdentity("host.example.com")
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	m := New(id, 5*time.Minute)

	// No peers initially.
	if got := m.Peers(); len(got) != 0 {
		t.Errorf("expected 0 peers, got %d", len(got))
	}

	// Register a peer.
	m.RegisterPeer(&PeerInfo{
		DID:      "did:web:peer.example.com",
		Endpoint: "https://peer.example.com/mesh",
		Skills:   []string{"sha256:skill1"},
	})

	peers := m.Peers()
	if len(peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(peers))
	}
	if peers[0].DID != "did:web:peer.example.com" {
		t.Errorf("peer DID = %q", peers[0].DID)
	}
}

// TestMeshPeerTTLExpiry verifies that expired peers are not returned.
func TestMeshPeerTTLExpiry(t *testing.T) {
	id, err := GenerateIdentity("host.example.com")
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	// Use a very short TTL so we can test expiry.
	m := New(id, 10*time.Millisecond)

	m.RegisterPeer(&PeerInfo{
		DID:      "did:web:peer.example.com",
		Endpoint: "https://peer.example.com/mesh",
	})

	// Peer should be visible immediately.
	if got := m.Peers(); len(got) != 1 {
		t.Errorf("expected 1 peer before TTL, got %d", len(got))
	}

	// Wait for TTL to expire.
	time.Sleep(20 * time.Millisecond)

	if got := m.Peers(); len(got) != 0 {
		t.Errorf("expected 0 peers after TTL expiry, got %d", len(got))
	}
}

// TestMeshFindPeerForSkill verifies skill-based peer lookup.
func TestMeshFindPeerForSkill(t *testing.T) {
	id, err := GenerateIdentity("host.example.com")
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	m := New(id, 5*time.Minute)

	m.RegisterPeer(&PeerInfo{
		DID:      "did:web:peer-a.example.com",
		Endpoint: "https://peer-a.example.com/mesh",
		Skills:   []string{"sha256:skill-alpha", "sha256:skill-beta"},
	})
	m.RegisterPeer(&PeerInfo{
		DID:      "did:web:peer-b.example.com",
		Endpoint: "https://peer-b.example.com/mesh",
		Skills:   []string{"sha256:skill-gamma"},
	})

	// Find by exact hash.
	peer := m.FindPeerForSkill("sha256:skill-gamma")
	if peer == nil {
		t.Fatal("expected to find peer for skill-gamma, got nil")
	}
	if peer.DID != "did:web:peer-b.example.com" {
		t.Errorf("wrong peer: %q", peer.DID)
	}

	// Unknown skill returns nil.
	if p := m.FindPeerForSkill("sha256:unknown"); p != nil {
		t.Errorf("expected nil for unknown skill, got %q", p.DID)
	}
}

// TestDIDDocument verifies the DID document structure.
func TestDIDDocument(t *testing.T) {
	id, err := GenerateIdentity("gateway.example.com")
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	doc := id.DIDDocument()

	// Check required top-level fields.
	if _, ok := doc["@context"]; !ok {
		t.Error("DIDDocument missing @context")
	}
	if docID, ok := doc["id"].(string); !ok || docID != id.DID() {
		t.Errorf("DIDDocument id = %v, want %q", doc["id"], id.DID())
	}

	// Verify verificationMethod contains the key.
	vms, ok := doc["verificationMethod"].([]map[string]interface{})
	if !ok || len(vms) == 0 {
		t.Fatal("DIDDocument verificationMethod missing or empty")
	}
	vm := vms[0]
	if !strings.HasPrefix(vm["id"].(string), id.DID()) {
		t.Errorf("verificationMethod id should start with DID")
	}
	if vm["type"] != "Ed25519VerificationKey2020" {
		t.Errorf("verificationMethod type = %q", vm["type"])
	}

	// Verify service endpoint.
	svcs, ok := doc["service"].([]map[string]interface{})
	if !ok || len(svcs) == 0 {
		t.Fatal("DIDDocument service missing or empty")
	}
	if svcs[0]["type"] != "DojoMeshEndpoint" {
		t.Errorf("service type = %q", svcs[0]["type"])
	}
}

// TestMeshStatus verifies the Status report.
func TestMeshStatus(t *testing.T) {
	id, err := GenerateIdentity("host.example.com")
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	m := New(id, 5*time.Minute)

	status := m.Status()
	if status.InstanceDID != id.DID() {
		t.Errorf("status DID = %q, want %q", status.InstanceDID, id.DID())
	}
	if !status.Healthy {
		t.Error("expected status.Healthy = true")
	}
	if status.PeerCount != 0 {
		t.Errorf("expected PeerCount = 0, got %d", status.PeerCount)
	}

	m.RegisterPeer(&PeerInfo{DID: "did:web:peer.example.com"})
	status = m.Status()
	if status.PeerCount != 1 {
		t.Errorf("expected PeerCount = 1 after registration, got %d", status.PeerCount)
	}
}
