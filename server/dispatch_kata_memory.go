package server

// Go-handler dispatch ops for Kata (DGS-150): kata-memory-write and
// kata-account-delete. They are writes, not model calls, and both act only on
// the authenticated subject's memories (DGS-163 owner scoping). handleDispatch
// has already refused a body or input user_id that disagrees with the token.

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/DojoGenesis/gateway/memory"
)

// kataRollMemoryType is the memories.type of a Roll written by Kata.
const kataRollMemoryType = "kata.roll"

// maxIdempotencyKeyLen bounds the header value; the Kata SDK sends a UUIDv7.
const maxIdempotencyKeyLen = 200

// kataMemoryIDNamespace seeds kataMemoryID. Fixed forever: changing it would
// make a retried write after a deploy land as a second memory.
var kataMemoryIDNamespace = uuid.MustParse("6f1e2d1c-3b8a-5c47-9d0e-4a1b2c3d4e5f")

// kataMemoryID derives the memory ID from (subject, Idempotency-Key). A replay
// of the same key by the same user therefore names the same row, and two users
// sending the same key never collide.
func kataMemoryID(subject, idempotencyKey string) string {
	return uuid.NewSHA1(kataMemoryIDNamespace, []byte(subject+"\x00"+idempotencyKey)).String()
}

func goOpResponse(c *gin.Context, op *goHandlerOp, output interface{}) {
	c.JSON(http.StatusOK, dispatchResponse{
		Output: output,
		Provenance: dispatchProvenance{
			Skill:        op.ID,
			SkillVersion: op.Version,
			RequestID:    dispatchRequestID(c),
			GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// runKataMemoryWrite stores input.roll as one memory owned by subject and
// returns {memory_id}.
//
// Idempotent on the Idempotency-Key header: the Kata SDK keeps one key per
// queued write across retries, so a replay finds the row the first attempt
// wrote and returns its memory_id without writing again. Without the header
// every call is a new write.
func (s *Server) runKataMemoryWrite(c *gin.Context, op *goHandlerOp, subject string, rawInput json.RawMessage) {
	var in struct {
		Roll json.RawMessage `json:"roll"`
	}
	if err := json.Unmarshal(rawInput, &in); err != nil {
		dispatchError(c, http.StatusBadRequest, "invalid_input", "input is not valid JSON")
		return
	}
	var roll map[string]interface{}
	if err := json.Unmarshal(in.Roll, &roll); err != nil || roll == nil {
		dispatchError(c, http.StatusBadRequest, "invalid_input", "input.roll must be a JSON object")
		return
	}
	// A Roll carries its own user_id; it must be the caller too.
	if uid, present := roll["user_id"]; subjectMismatch(uid, present, subject) {
		dispatchError(c, http.StatusBadRequest, "subject_mismatch", "roll.user_id does not match the authenticated subject")
		return
	}

	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if len(key) > maxIdempotencyKeyLen {
		dispatchError(c, http.StatusBadRequest, "invalid_request", "Idempotency-Key is too long")
		return
	}
	memoryID := uuid.New().String()
	if key != "" {
		memoryID = kataMemoryID(subject, key)
	}

	ctx := c.Request.Context()
	scope := memory.ForOwner(subject)
	if key != "" {
		if _, err := s.memoryManager.GetMemoryScoped(ctx, scope, memoryID); err == nil {
			goOpResponse(c, op, gin.H{"memory_id": memoryID})
			return
		} else if !errors.Is(err, memory.ErrMemoryNotFound) {
			slog.Error("dispatch: kata-memory-write lookup failed", "request_id", dispatchRequestID(c), "error", err)
			dispatchError(c, http.StatusServiceUnavailable, "memory_unavailable", "memory store failed")
			return
		}
	}

	var content bytes.Buffer
	if err := json.Compact(&content, in.Roll); err != nil {
		dispatchError(c, http.StatusBadRequest, "invalid_input", "input.roll is not valid JSON")
		return
	}
	meta := map[string]interface{}{"source": "kata", "skill": op.ID}
	if id, ok := roll["id"].(string); ok && id != "" {
		meta["roll_id"] = id
	}
	now := time.Now()
	err := s.memoryManager.Store(ctx, memory.Memory{
		ID:          memoryID,
		Type:        kataRollMemoryType,
		Content:     content.String(),
		Metadata:    meta,
		ContextType: "private",
		OwnerID:     subject,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		// ErrMemoryOwnerConflict cannot happen for a derived ID unless SHA-1
		// collides; treat it like any other store failure.
		slog.Error("dispatch: kata-memory-write store failed", "request_id", dispatchRequestID(c), "error", err)
		dispatchError(c, http.StatusServiceUnavailable, "memory_unavailable", "memory store failed")
		return
	}
	goOpResponse(c, op, gin.H{"memory_id": memoryID})
}

// runKataAccountDelete erases every memory owned by subject and returns how
// many went. Repeating it is harmless: the second call deletes zero.
//
// It does not revoke the subject's outstanding JWTs: the gateway has no
// session epoch for user tokens yet (see DGS-163).
func (s *Server) runKataAccountDelete(c *gin.Context, op *goHandlerOp, subject string) {
	n, err := s.memoryManager.DeleteAllForOwner(c.Request.Context(), subject)
	if err != nil {
		slog.Error("dispatch: kata-account-delete failed", "request_id", dispatchRequestID(c), "error", err)
		dispatchError(c, http.StatusServiceUnavailable, "memory_unavailable", "memory store failed")
		return
	}
	goOpResponse(c, op, gin.H{"deleted_memories": n})
}
