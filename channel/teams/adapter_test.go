package teams

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/channel"
)

// ---------------------------------------------------------------------------
// Test fixtures — generated once for the whole package test run.
// ---------------------------------------------------------------------------

var (
	testPrivKey *rsa.PrivateKey
	testKID     = "test-key-001"
	testAppID   = "test-app-id-12345"
)

func TestMain(m *testing.M) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("teams test: generate RSA key: " + err.Error())
	}
	testPrivKey = key
	os.Exit(m.Run())
}

// makeJWT creates a real RS256-signed JWT with the given claims.
// exp and nbf are Unix timestamps; pass nbf=0 to omit the claim.
func makeJWT(t *testing.T, key *rsa.PrivateKey, kid, iss, aud string, exp, nbf int64) string {
	t.Helper()

	headerJSON, _ := json.Marshal(map[string]string{
		"alg": "RS256",
		"typ": "JWT",
		"kid": kid,
	})
	claims := map[string]interface{}{
		"iss": iss,
		"aud": aud,
		"exp": exp,
	}
	if nbf != 0 {
		claims["nbf"] = nbf
	}
	payloadJSON, _ := json.Marshal(claims)

	hdr := base64.RawURLEncoding.EncodeToString(headerJSON)
	pay := base64.RawURLEncoding.EncodeToString(payloadJSON)
	sigInput := hdr + "." + pay

	digest := sha256.Sum256([]byte(sigInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("makeJWT: sign: %v", err)
	}
	return sigInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// jwksServerFor starts a mock JWKS HTTP server that serves the given public key.
func jwksServerFor(t *testing.T, kid string, pub *rsa.PublicKey) *httptest.Server {
	t.Helper()

	nBytes := pub.N.Bytes()

	e := pub.E
	eBytes := make([]byte, 4)
	eBytes[0] = byte(e >> 24) //nolint:gosec // G115 -- bounded test input, RSA public exponent fits int
	eBytes[1] = byte(e >> 16) //nolint:gosec // G115 -- bounded test input, RSA public exponent fits int
	eBytes[2] = byte(e >> 8)  //nolint:gosec // G115 -- bounded test input, RSA public exponent fits int
	eBytes[3] = byte(e) //nolint:gosec // G115 -- bounded test input, RSA public exponent fits int
	// Trim leading zero bytes.
	i := 0
	for i < len(eBytes)-1 && eBytes[i] == 0 {
		i++
	}
	eBytes = eBytes[i:]

	body, _ := json.Marshal(map[string]interface{}{
		"keys": []map[string]string{
			{
				"kty": "RSA",
				"use": "sig",
				"kid": kid,
				"n":   base64.RawURLEncoding.EncodeToString(nBytes),
				"e":   base64.RawURLEncoding.EncodeToString(eBytes),
			},
		},
	})

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
}

// newTestAdapter creates a TeamsAdapter pointing at mockJWKSSrv for key
// fetches, without touching the real Microsoft endpoint.
func newTestAdapter(t *testing.T, jwksSrv *httptest.Server) *TeamsAdapter {
	t.Helper()
	return newTeamsAdapterInternal("test-bot-token", testAppID, jwksSrv.URL, http.DefaultClient)
}

// ---------------------------------------------------------------------------
// 1. TestTeamsAdapter_Name
// ---------------------------------------------------------------------------

func TestTeamsAdapter_Name(t *testing.T) {
	a := NewTeamsAdapter("test-bot-token", testAppID)
	if got := a.Name(); got != "teams" {
		t.Errorf("Name() = %q, want %q", got, "teams")
	}
}

// ---------------------------------------------------------------------------
// 2. TestTeamsAdapter_Capabilities
// ---------------------------------------------------------------------------

func TestTeamsAdapter_Capabilities(t *testing.T) {
	a := NewTeamsAdapter("test-bot-token", testAppID)
	caps := a.Capabilities()

	if !caps.SupportsThreads {
		t.Error("SupportsThreads should be true for Teams")
	}
	if !caps.SupportsReactions {
		t.Error("SupportsReactions should be true for Teams")
	}
	if !caps.SupportsAttachments {
		t.Error("SupportsAttachments should be true for Teams")
	}
	if caps.SupportsEdits {
		t.Error("SupportsEdits should be false for Teams")
	}
	if caps.MaxMessageLength != 28000 {
		t.Errorf("MaxMessageLength = %d, want 28000", caps.MaxMessageLength)
	}
}

// ---------------------------------------------------------------------------
// 3. TestTeamsAdapter_Normalize_Message
// ---------------------------------------------------------------------------

func TestTeamsAdapter_Normalize_Message(t *testing.T) {
	a := NewTeamsAdapter("test-bot-token", testAppID)

	act := Activity{
		Type: "message",
		ID:   "msg-001",
		From: ChannelAccount{ID: "user-abc", Name: "Alice"},
		Conversation: ConversationAccount{ID: "conv-xyz", Name: "General"},
		Text:       "hello teams",
		ServiceURL: "https://smba.trafficmanager.net/teams/",
		Timestamp:  "2024-04-05T12:00:00Z",
	}

	raw, _ := json.Marshal(act)
	msg, err := a.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	assertField(t, "Platform", msg.Platform, "teams")
	assertField(t, "ID", msg.ID, "msg-001")
	assertField(t, "ChannelID", msg.ChannelID, "conv-xyz")
	assertField(t, "UserID", msg.UserID, "user-abc")
	assertField(t, "UserName", msg.UserName, "Alice")
	assertField(t, "Text", msg.Text, "hello teams")
	assertField(t, "ThreadID", msg.ThreadID, "")

	if msg.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
	if len(msg.Attachments) != 0 {
		t.Errorf("expected 0 attachments, got %d", len(msg.Attachments))
	}
}

// ---------------------------------------------------------------------------
// 4. TestTeamsAdapter_Normalize_Reply
// ---------------------------------------------------------------------------

func TestTeamsAdapter_Normalize_Reply(t *testing.T) {
	a := NewTeamsAdapter("test-bot-token", testAppID)

	act := Activity{
		Type:      "message",
		ID:        "msg-002",
		From:      ChannelAccount{ID: "user-bob", Name: "Bob"},
		Conversation: ConversationAccount{ID: "conv-xyz"},
		Text:      "replying now",
		ReplyToID: "msg-001",
		ServiceURL: "https://smba.trafficmanager.net/teams/",
	}

	raw, _ := json.Marshal(act)
	msg, err := a.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	assertField(t, "ThreadID", msg.ThreadID, "msg-001")
	assertField(t, "Text", msg.Text, "replying now")
}

// ---------------------------------------------------------------------------
// 5. TestTeamsAdapter_VerifySignature
// ---------------------------------------------------------------------------

func TestTeamsAdapter_VerifySignature(t *testing.T) {
	jwksSrv := jwksServerFor(t, testKID, &testPrivKey.PublicKey)
	defer jwksSrv.Close()
	a := newTestAdapter(t, jwksSrv)

	futureExp := time.Now().Add(time.Hour).Unix()
	pastExp := time.Now().Add(-2 * time.Hour).Unix()

	t.Run("valid_token", func(t *testing.T) {
		token := makeJWT(t, testPrivKey, testKID, botFrameworkIssuer, testAppID, futureExp, 0)
		req := httptest.NewRequest(http.MethodPost, "/webhook/teams", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		if err := a.VerifySignature(req); err != nil {
			t.Errorf("expected valid token to pass, got: %v", err)
		}
	})

	t.Run("missing_header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook/teams", nil)
		if err := a.VerifySignature(req); err == nil {
			t.Error("expected error for missing Authorization header, got nil")
		}
	})

	t.Run("non_bearer_scheme", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook/teams", nil)
		req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
		if err := a.VerifySignature(req); err == nil {
			t.Error("expected error for non-Bearer scheme, got nil")
		}
	})

	t.Run("malformed_jwt", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook/teams", nil)
		req.Header.Set("Authorization", "Bearer notajwt")
		if err := a.VerifySignature(req); err == nil {
			t.Error("expected error for malformed JWT, got nil")
		}
	})

	t.Run("expired_token", func(t *testing.T) {
		token := makeJWT(t, testPrivKey, testKID, botFrameworkIssuer, testAppID, pastExp, 0)
		req := httptest.NewRequest(http.MethodPost, "/webhook/teams", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		err := a.VerifySignature(req)
		if err == nil {
			t.Error("expected error for expired token, got nil")
		}
		if !strings.Contains(err.Error(), "expired") {
			t.Errorf("expected 'expired' in error, got: %v", err)
		}
	})

	t.Run("wrong_issuer", func(t *testing.T) {
		token := makeJWT(t, testPrivKey, testKID, "https://evil.example.com", testAppID, futureExp, 0)
		req := httptest.NewRequest(http.MethodPost, "/webhook/teams", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		err := a.VerifySignature(req)
		if err == nil {
			t.Error("expected error for wrong issuer, got nil")
		}
		if !strings.Contains(err.Error(), "issuer") {
			t.Errorf("expected 'issuer' in error, got: %v", err)
		}
	})

	t.Run("wrong_audience", func(t *testing.T) {
		token := makeJWT(t, testPrivKey, testKID, botFrameworkIssuer, "wrong-app-id", futureExp, 0)
		req := httptest.NewRequest(http.MethodPost, "/webhook/teams", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		err := a.VerifySignature(req)
		if err == nil {
			t.Error("expected error for wrong audience, got nil")
		}
		if !strings.Contains(err.Error(), "audience") {
			t.Errorf("expected 'audience' in error, got: %v", err)
		}
	})

	t.Run("unknown_kid", func(t *testing.T) {
		token := makeJWT(t, testPrivKey, "unknown-kid", botFrameworkIssuer, testAppID, futureExp, 0)
		req := httptest.NewRequest(http.MethodPost, "/webhook/teams", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		err := a.VerifySignature(req)
		if err == nil {
			t.Error("expected error for unknown kid, got nil")
		}
	})

	t.Run("tampered_payload", func(t *testing.T) {
		// Build a valid token then replace the payload with different claims.
		token := makeJWT(t, testPrivKey, testKID, botFrameworkIssuer, testAppID, futureExp, 0)
		parts := strings.Split(token, ".")

		// Swap in a payload with a different audience — signature won't match.
		fakeClaims, _ := json.Marshal(map[string]interface{}{
			"iss": botFrameworkIssuer,
			"aud": "attacker-app-id",
			"exp": futureExp,
		})
		parts[1] = base64.RawURLEncoding.EncodeToString(fakeClaims)
		tampered := strings.Join(parts, ".")

		req := httptest.NewRequest(http.MethodPost, "/webhook/teams", nil)
		req.Header.Set("Authorization", "Bearer "+tampered)
		if err := a.VerifySignature(req); err == nil {
			t.Error("expected error for tampered payload, got nil")
		}
	})

	t.Run("nbf_in_future", func(t *testing.T) {
		// nbf well beyond clock-skew tolerance
		futurNbf := time.Now().Add(30 * time.Minute).Unix()
		token := makeJWT(t, testPrivKey, testKID, botFrameworkIssuer, testAppID, futureExp, futurNbf)
		req := httptest.NewRequest(http.MethodPost, "/webhook/teams", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		err := a.VerifySignature(req)
		if err == nil {
			t.Error("expected error for future nbf, got nil")
		}
		if !strings.Contains(err.Error(), "not yet valid") {
			t.Errorf("expected 'not yet valid' in error, got: %v", err)
		}
	})

	t.Run("signed_by_different_key", func(t *testing.T) {
		// Generate a different key and sign with it — JWKS has the original key.
		otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("generate key: %v", err)
		}
		token := makeJWT(t, otherKey, testKID, botFrameworkIssuer, testAppID, futureExp, 0)
		req := httptest.NewRequest(http.MethodPost, "/webhook/teams", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		if err := a.VerifySignature(req); err == nil {
			t.Error("expected signature error, got nil")
		}
	})

	t.Run("jwks_cache_hit", func(t *testing.T) {
		// Wrap the shared JWKS server with a counting proxy to confirm that
		// 3 verification calls only trigger 1 JWKS fetch (cache warms on first).
		callCount := 0
		proxySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			// Delegate to a fresh jwksServerFor handler to avoid duplicating
			// the public-key encoding logic.
			inner := jwksServerFor(t, testKID, &testPrivKey.PublicKey)
			defer inner.Close()
			resp, err := http.Get(inner.URL)
			if err != nil {
				http.Error(w, "proxy fetch failed", http.StatusInternalServerError)
				return
			}
			defer func() { _ = resp.Body.Close() }()
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.Copy(w, resp.Body)
		}))
		defer proxySrv.Close()

		ca := newTeamsAdapterInternal("tok", testAppID, proxySrv.URL, http.DefaultClient)
		token := makeJWT(t, testPrivKey, testKID, botFrameworkIssuer, testAppID, futureExp, 0)

		for i := 0; i < 3; i++ {
			req := httptest.NewRequest(http.MethodPost, "/webhook/teams", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			if err := ca.VerifySignature(req); err != nil {
				t.Fatalf("request %d: %v", i, err)
			}
		}
		if callCount != 1 {
			t.Errorf("JWKS endpoint hit %d times, want 1 (cache miss only on first)", callCount)
		}
	})

	t.Run("oversized_token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook/teams", nil)
		req.Header.Set("Authorization", "Bearer "+strings.Repeat("a", maxTokenBytes+1))
		if err := a.VerifySignature(req); err == nil {
			t.Error("expected error for oversized token, got nil")
		}
	})
}

// ---------------------------------------------------------------------------
// 6. TestTeamsAdapter_HandleWebhook
// ---------------------------------------------------------------------------

func TestTeamsAdapter_HandleWebhook(t *testing.T) {
	jwksSrv := jwksServerFor(t, testKID, &testPrivKey.PublicKey)
	defer jwksSrv.Close()
	a := newTestAdapter(t, jwksSrv)

	validAct := Activity{
		Type: "message",
		ID:   "msg-003",
		From: ChannelAccount{ID: "user-charlie", Name: "Charlie"},
		Conversation: ConversationAccount{ID: "conv-abc"},
		Text:       "webhook round trip",
		ServiceURL: "https://smba.trafficmanager.net/teams/",
	}

	t.Run("valid_request_returns_200", func(t *testing.T) {
		raw, _ := json.Marshal(validAct)
		token := makeJWT(t, testPrivKey, testKID, botFrameworkIssuer, testAppID, time.Now().Add(time.Hour).Unix(), 0)

		req := httptest.NewRequest(http.MethodPost, "/webhook/teams", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		a.HandleWebhook(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("HandleWebhook status = %d, want 200", rec.Code)
		}
	})

	t.Run("invalid_token_returns_401", func(t *testing.T) {
		raw, _ := json.Marshal(validAct)

		req := httptest.NewRequest(http.MethodPost, "/webhook/teams", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer notavalidtoken.atall.nope")
		rec := httptest.NewRecorder()

		a.HandleWebhook(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("HandleWebhook status = %d, want 401", rec.Code)
		}
	})

	t.Run("missing_auth_returns_401", func(t *testing.T) {
		raw, _ := json.Marshal(validAct)

		req := httptest.NewRequest(http.MethodPost, "/webhook/teams", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		a.HandleWebhook(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("HandleWebhook status = %d, want 401", rec.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// 7. TestTeamsAdapter_Send
// ---------------------------------------------------------------------------

func TestTeamsAdapter_Send(t *testing.T) {
	var capturedBody []byte
	var capturedAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"reply-001"}`))
	}))
	defer srv.Close()

	a := NewTeamsAdapterWithClient("my-bot-token", testAppID, &http.Client{
		Transport: &urlRewriteTransport{
			base:       http.DefaultTransport,
			serverAddr: srv.URL,
		},
	})

	msg := &channel.ChannelMessage{
		Platform:  "teams",
		ChannelID: "conv-xyz",
		Text:      "hello from send test",
		Metadata: map[string]interface{}{
			"service_url": srv.URL,
		},
	}

	if err := a.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send: %v", err)
	}

	// Verify Bearer auth was set.
	if !strings.HasPrefix(capturedAuth, "Bearer ") {
		t.Errorf("Authorization header = %q, expected Bearer token", capturedAuth)
	}

	// Verify the activity body.
	var act Activity
	if err := json.Unmarshal(capturedBody, &act); err != nil {
		t.Fatalf("unmarshal captured body: %v", err)
	}
	if act.Text != "hello from send test" {
		t.Errorf("activity.text = %q, want %q", act.Text, "hello from send test")
	}
	if act.Type != "message" {
		t.Errorf("activity.type = %q, want %q", act.Type, "message")
	}
}

// ---------------------------------------------------------------------------
// 8. TestTeamsAdapter_Send_MissingServiceURL
// ---------------------------------------------------------------------------

func TestTeamsAdapter_Send_MissingServiceURL(t *testing.T) {
	a := NewTeamsAdapter("test-bot-token", testAppID)

	msg := &channel.ChannelMessage{
		Platform:  "teams",
		ChannelID: "conv-xyz",
		Text:      "no service url",
	}

	if err := a.Send(context.Background(), msg); err == nil {
		t.Error("expected error when service_url missing from metadata, got nil")
	}
}

// ---------------------------------------------------------------------------
// 9. TestRSAPublicKeyFromJWK
// ---------------------------------------------------------------------------

func TestRSAPublicKeyFromJWK(t *testing.T) {
	pub := &testPrivKey.PublicKey

	nBytes := pub.N.Bytes()
	eVal := pub.E
	eBytes := big.NewInt(int64(eVal)).Bytes()

	entry := jwkEntry{
		Kid: "k1",
		Kty: "RSA",
		Use: "sig",
		N:   base64.RawURLEncoding.EncodeToString(nBytes),
		E:   base64.RawURLEncoding.EncodeToString(eBytes),
	}

	got, err := rsaPublicKeyFromJWK(entry)
	if err != nil {
		t.Fatalf("rsaPublicKeyFromJWK: %v", err)
	}
	if got.N.Cmp(pub.N) != 0 {
		t.Error("modulus mismatch")
	}
	if got.E != pub.E {
		t.Errorf("exponent = %d, want %d", got.E, pub.E)
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func assertField(t *testing.T, name, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", name, got, want)
	}
}

// urlRewriteTransport redirects all requests to the given mock server.
type urlRewriteTransport struct {
	base       http.RoundTripper
	serverAddr string
}

func (t *urlRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = "http"
	cloned.URL.Host = strings.TrimPrefix(t.serverAddr, "http://")
	return t.base.RoundTrip(cloned)
}
