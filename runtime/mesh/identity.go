package mesh

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// Identity holds the instance's Ed25519 keypair and DID.
type Identity struct {
	did        string
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
}

// GenerateIdentity creates a new Ed25519 identity with the given DID.
// The DID should be in the format "did:web:hostname".
func GenerateIdentity(hostname string) (*Identity, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("mesh: generate key: %w", err)
	}
	return &Identity{
		did:        fmt.Sprintf("did:web:%s", hostname),
		publicKey:  pub,
		privateKey: priv,
	}, nil
}

// LoadIdentity restores an identity from a stored private key and DID.
func LoadIdentity(did string, privateKeyB64 string) (*Identity, error) {
	privBytes, err := base64.StdEncoding.DecodeString(privateKeyB64)
	if err != nil {
		return nil, fmt.Errorf("mesh: decode private key: %w", err)
	}
	priv := ed25519.PrivateKey(privBytes)
	pub := priv.Public().(ed25519.PublicKey)
	return &Identity{
		did:        did,
		publicKey:  pub,
		privateKey: priv,
	}, nil
}

// DID returns the instance's DID string.
func (id *Identity) DID() string { return id.did }

// PublicKeyBase64 returns the base64-encoded public key.
func (id *Identity) PublicKeyBase64() string {
	return base64.StdEncoding.EncodeToString(id.publicKey)
}

// PrivateKeyBase64 returns the base64-encoded private key (for storage).
func (id *Identity) PrivateKeyBase64() string {
	return base64.StdEncoding.EncodeToString(id.privateKey)
}

// Sign signs a message with the instance's private key.
func (id *Identity) Sign(message []byte) string {
	sig := ed25519.Sign(id.privateKey, message)
	return base64.StdEncoding.EncodeToString(sig)
}

// Verify checks a signature against a public key.
func Verify(publicKeyB64 string, message []byte, signatureB64 string) (bool, error) {
	pubBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return false, fmt.Errorf("mesh: decode public key: %w", err)
	}
	sigBytes, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return false, fmt.Errorf("mesh: decode signature: %w", err)
	}
	return ed25519.Verify(ed25519.PublicKey(pubBytes), message, sigBytes), nil
}

// DIDDocument returns the DID document as a map.
func (id *Identity) DIDDocument() map[string]interface{} {
	return map[string]interface{}{
		"@context": []string{"https://www.w3.org/ns/did/v1"},
		"id":       id.did,
		"verificationMethod": []map[string]interface{}{
			{
				"id":                 id.did + "#key-1",
				"type":               "Ed25519VerificationKey2020",
				"controller":         id.did,
				"publicKeyMultibase": "z" + base64.RawStdEncoding.EncodeToString(id.publicKey),
			},
		},
		"authentication":  []string{id.did + "#key-1"},
		"assertionMethod": []string{id.did + "#key-1"},
		"service": []map[string]interface{}{
			{
				"id":              id.did + "#dojo-mesh",
				"type":            "DojoMeshEndpoint",
				"serviceEndpoint": fmt.Sprintf("https://%s/mesh", id.did[8:]),
			},
		},
	}
}

// VerifyAnnouncement validates a signed announcement from a peer.
func VerifyAnnouncement(ann Announcement) (bool, error) {
	// Reconstruct the signed payload (everything except signature)
	payload := map[string]interface{}{
		"did":       ann.DID,
		"endpoint":  ann.Endpoint,
		"skills":    ann.Skills,
		"platforms": ann.Platforms,
		"timestamp": ann.Timestamp,
	}
	msg, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}
	return Verify(ann.PublicKey, msg, ann.Signature)
}

// SignAnnouncement creates a signed announcement for this identity.
func (id *Identity) SignAnnouncement(endpoint string, skills, platforms []string) Announcement {
	ann := Announcement{
		DID:       id.did,
		Endpoint:  endpoint,
		Skills:    skills,
		Platforms: platforms,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		PublicKey: id.PublicKeyBase64(),
	}
	payload := map[string]interface{}{
		"did":       ann.DID,
		"endpoint":  ann.Endpoint,
		"skills":    ann.Skills,
		"platforms": ann.Platforms,
		"timestamp": ann.Timestamp,
	}
	msg, _ := json.Marshal(payload)
	ann.Signature = id.Sign(msg)
	return ann
}
