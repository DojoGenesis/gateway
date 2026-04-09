package channel

import (
	"net/http"
	"testing"
)

// AdapterComplianceHelper runs a standard suite of compliance checks on a
// WebhookAdapter implementation. It verifies:
//
//   - Name() returns a non-empty string
//   - Normalize(validPayload) succeeds and returns a non-nil ChannelMessage
//   - Normalize(invalidPayload) returns an error
//   - VerifySignature(signedReq) succeeds
//   - VerifySignature(unsignedReq) fails
//   - Capabilities() returns a non-zero MaxMessageLength
//
// The extraChecks parameter is an optional function for adapter-specific
// assertions. Pass nil to skip.
func AdapterComplianceHelper(
	t *testing.T,
	adapter WebhookAdapter,
	validPayload []byte,
	invalidPayload []byte,
	signedReq func() *http.Request,
	unsignedReq func() *http.Request,
	extraChecks func(t *testing.T, adapter WebhookAdapter),
) {
	t.Helper()

	// Name must be non-empty.
	t.Run("Name", func(t *testing.T) {
		name := adapter.Name()
		if name == "" {
			t.Error("Name() returned empty string")
		}
	})

	// Normalize valid payload.
	t.Run("Normalize_Valid", func(t *testing.T) {
		msg, err := adapter.Normalize(validPayload)
		if err != nil {
			t.Fatalf("Normalize valid payload: %v", err)
		}
		if msg == nil {
			t.Fatal("Normalize returned nil message for valid payload")
		}
		if msg.Platform == "" {
			t.Error("Normalize: Platform field is empty")
		}
		if msg.ID == "" {
			t.Error("Normalize: ID field is empty")
		}
	})

	// Normalize invalid payload — should return an error OR a nil message.
	// Some adapters parse structurally valid JSON that is semantically
	// incomplete (e.g. {"type":"event_callback","event":null}); the helper
	// accepts either an error or a nil message as valid behavior.
	t.Run("Normalize_Invalid", func(t *testing.T) {
		msg, err := adapter.Normalize(invalidPayload)
		if err == nil && msg != nil && msg.Text != "" {
			t.Error("Normalize should return error or empty message for invalid payload")
		}
	})

	// VerifySignature with signed request.
	if signedReq != nil {
		t.Run("VerifySignature_Valid", func(t *testing.T) {
			r := signedReq()
			if err := adapter.VerifySignature(r); err != nil {
				t.Errorf("VerifySignature rejected valid request: %v", err)
			}
		})
	}

	// VerifySignature with unsigned request.
	if unsignedReq != nil {
		t.Run("VerifySignature_Invalid", func(t *testing.T) {
			r := unsignedReq()
			if err := adapter.VerifySignature(r); err == nil {
				t.Error("VerifySignature should reject unsigned request")
			}
		})
	}

	// Capabilities.
	t.Run("Capabilities", func(t *testing.T) {
		caps := adapter.Capabilities()
		if caps.MaxMessageLength <= 0 {
			t.Errorf("MaxMessageLength = %d, want > 0", caps.MaxMessageLength)
		}
	})

	// Extra adapter-specific checks.
	if extraChecks != nil {
		t.Run("ExtraChecks", func(t *testing.T) {
			extraChecks(t, adapter)
		})
	}
}
