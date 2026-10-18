package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// chainDomain prefixes every hashed entry so an audit hash can never be
// confused with a hash of the same bytes computed for another purpose.
const chainDomain = "dojo-audit-chain-v1\n"

// Recorder is the write path the running gateway uses. It wraps an AuditLog
// and links every entry it writes into a hash chain: entry N stores the hash
// of entry N-1 and a hash over its own fields plus that link. Editing,
// deleting or reordering any chained entry therefore breaks every hash after
// it, which Verify reports.
//
// A chain proves order and integrity of what is present. It cannot by itself
// prove that entries were not cut off the end; compare Head against a value
// recorded elsewhere for that.
//
// One Recorder should own an audit database. Two processes appending to the
// same log would each extend the chain from the head they last saw and fork it.
//
// A nil *Recorder is valid and records nothing, so call sites need no guard.
type Recorder struct {
	log            AuditLog
	recordToolArgs bool
	now            func() time.Time

	mu   sync.Mutex
	seq  int64
	head string
}

// RecorderOption configures a Recorder.
type RecorderOption func(*Recorder)

// WithToolArgs stores raw tool arguments in each tool-execution entry. Off by
// default: arguments can carry file contents, memory text or credentials, so
// by default only their SHA-256 and key names are kept.
func WithToolArgs(enabled bool) RecorderOption {
	return func(r *Recorder) { r.recordToolArgs = enabled }
}

// chainHeader is implemented by the built-in logs to find the chain head
// without reading the whole table.
type chainHeader interface {
	chainHead(ctx context.Context) (seq int64, hash string, err error)
}

// NewRecorder resumes the chain at the newest entry already in log.
func NewRecorder(ctx context.Context, log AuditLog, opts ...RecorderOption) (*Recorder, error) {
	if log == nil {
		return nil, fmt.Errorf("audit: recorder needs a log")
	}
	r := &Recorder{log: log, now: time.Now}
	for _, opt := range opts {
		opt(r)
	}

	var err error
	if h, ok := log.(chainHeader); ok {
		r.seq, r.head, err = h.chainHead(ctx)
	} else {
		r.seq, r.head, err = scanHead(ctx, log)
	}
	if err != nil {
		return nil, fmt.Errorf("audit: find chain head: %w", err)
	}
	return r, nil
}

func scanHead(ctx context.Context, log AuditLog) (int64, string, error) {
	entries, err := log.Query(ctx, AuditFilter{})
	if err != nil {
		return 0, "", err
	}
	var seq int64
	var hash string
	for _, e := range entries {
		if e.Seq > seq {
			seq, hash = e.Seq, e.EntryHash
		}
	}
	return seq, hash, nil
}

// Head returns the sequence number and hash of the newest chained entry.
func (r *Recorder) Head() (int64, string) {
	if r == nil {
		return 0, ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.seq, r.head
}

// Log returns the underlying log, for queries and export.
func (r *Recorder) Log() AuditLog {
	if r == nil {
		return nil
	}
	return r.log
}

// Close closes the underlying log.
func (r *Recorder) Close() error {
	if r == nil {
		return nil
	}
	return r.log.Close()
}

// Record fills in ID and Timestamp when empty, links the entry to the chain
// and writes it. The chain only advances when the write succeeds. It returns
// the entry as stored.
func (r *Recorder) Record(ctx context.Context, entry AuditEntry) (AuditEntry, error) {
	if r == nil {
		return entry, nil
	}
	if entry.ID == "" {
		entry.ID = uuid.NewString()
	}
	if entry.AgentID == "" {
		entry.AgentID = AgentIDFrom(ctx)
	}
	// The stored maps come back from JSON, so hash the JSON form too; otherwise
	// an int argument read back as float64 could change the hash.
	var err error
	if entry.ToolArgs, err = normalizeMap(entry.ToolArgs); err != nil {
		return entry, fmt.Errorf("audit: tool args: %w", err)
	}
	if entry.Metadata, err = normalizeMap(entry.Metadata); err != nil {
		return entry, fmt.Errorf("audit: metadata: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if entry.Timestamp.IsZero() {
		entry.Timestamp = r.now()
	}
	entry.Timestamp = entry.Timestamp.UTC().Round(0)
	entry.Seq = r.seq + 1
	entry.PrevHash = r.head
	if entry.EntryHash, err = HashEntry(entry); err != nil {
		return entry, err
	}

	if err := r.log.Record(ctx, entry); err != nil {
		return entry, err
	}
	r.seq, r.head = entry.Seq, entry.EntryHash
	return entry, nil
}

// record is Record for instrumentation call sites: an audit failure is logged
// and does not fail the action being audited. The write ignores cancellation
// of ctx, so a caller that hangs up mid-call is still recorded.
func (r *Recorder) record(ctx context.Context, entry AuditEntry) {
	if _, err := r.Record(context.WithoutCancel(ctx), entry); err != nil {
		slog.Warn("audit: record failed", "action", entry.Action, "tool", entry.Tool, "error", err)
	}
}

// ToolExecution records one tool call. err is the call's outcome; a refused
// call is recorded too, with its error.
func (r *Recorder) ToolExecution(ctx context.Context, tool string, args, result map[string]interface{}, callErr error, d time.Duration) {
	if r == nil {
		return
	}
	meta := map[string]interface{}{"outcome": outcome(callErr)}
	if callErr != nil {
		meta["error"] = callErr.Error()
	}
	entry := AuditEntry{
		Action:   ActionToolExecution,
		Tool:     tool,
		Duration: d,
		Metadata: meta,
	}
	if r.recordToolArgs {
		entry.ToolArgs = args
	} else if len(args) > 0 {
		keys := make([]string, 0, len(args))
		for k := range args {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		meta["arg_keys"] = keys
		meta["args_sha256"] = HashValue(args)
	}
	if callErr == nil {
		entry.ResultHash = HashValue(result)
	}
	r.record(ctx, entry)
}

// AgentSpawn records an agent being created.
func (r *Recorder) AgentSpawn(ctx context.Context, agentID string, meta map[string]interface{}) {
	if r == nil {
		return
	}
	r.record(ctx, AuditEntry{AgentID: agentID, Action: ActionAgentSpawn, Metadata: meta})
}

// MemoryAccess records one memory operation (store, get, update, delete,
// search, list). It never records memory content: meta should carry IDs,
// counts and hashes only.
func (r *Recorder) MemoryAccess(ctx context.Context, op string, meta map[string]interface{}, callErr error, d time.Duration) {
	if r == nil {
		return
	}
	m := map[string]interface{}{"op": op, "outcome": outcome(callErr)}
	for k, v := range meta {
		m[k] = v
	}
	if callErr != nil {
		m["error"] = callErr.Error()
	}
	r.record(ctx, AuditEntry{Action: ActionMemoryAccess, Duration: d, Metadata: m})
}

func outcome(err error) string {
	if err != nil {
		return "error"
	}
	return "ok"
}

// ---------------------------------------------------------------------------
// Hashing
// ---------------------------------------------------------------------------

// chainFields is the exact, ordered content an entry hash covers.
type chainFields struct {
	Seq                 int64                  `json:"seq"`
	ID                  string                 `json:"id"`
	Timestamp           string                 `json:"timestamp"`
	AgentID             string                 `json:"agent_id"`
	Action              string                 `json:"action"`
	Tool                string                 `json:"tool"`
	ToolArgs            map[string]interface{} `json:"tool_args"`
	CapabilitiesGranted []string               `json:"capabilities_granted"`
	ResultHash          string                 `json:"result_hash"`
	DurationNs          int64                  `json:"duration_ns"`
	Metadata            map[string]interface{} `json:"metadata"`
	PrevHash            string                 `json:"prev_hash"`
}

// HashEntry computes the chain hash of e from its fields and PrevHash. The
// EntryHash field itself is not an input.
func HashEntry(e AuditEntry) (string, error) {
	b, err := json.Marshal(chainFields{
		Seq:                 e.Seq,
		ID:                  e.ID,
		Timestamp:           e.Timestamp.UTC().Format(time.RFC3339Nano),
		AgentID:             e.AgentID,
		Action:              string(e.Action),
		Tool:                e.Tool,
		ToolArgs:            e.ToolArgs,
		CapabilitiesGranted: e.CapabilitiesGranted,
		ResultHash:          e.ResultHash,
		DurationNs:          e.Duration.Nanoseconds(),
		Metadata:            e.Metadata,
		PrevHash:            e.PrevHash,
	})
	if err != nil {
		return "", fmt.Errorf("audit: hash entry: %w", err)
	}
	sum := sha256.Sum256(append([]byte(chainDomain), b...))
	return hex.EncodeToString(sum[:]), nil
}

// HashValue returns the hex SHA-256 of v's JSON encoding, or "" if v cannot be
// encoded. Map keys are sorted by encoding/json, so equal maps hash equally.
func HashValue(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func normalizeMap(m map[string]interface{}) (map[string]interface{}, error) {
	if m == nil {
		return nil, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Verification
// ---------------------------------------------------------------------------

// VerifyReport is the result of checking a log's hash chain.
type VerifyReport struct {
	// Chained is the number of entries that carry a sequence number.
	Chained int `json:"chained"`
	// Unchained is the number of entries written without a Recorder.
	Unchained int `json:"unchained"`
	// HeadSeq and HeadHash identify the newest chained entry.
	HeadSeq  int64  `json:"head_seq"`
	HeadHash string `json:"head_hash"`
	// OK is true when every chained entry checks out.
	OK bool `json:"ok"`
	// Problems lists each break found, oldest first.
	Problems []string `json:"problems,omitempty"`
}

// Verify reads every entry in log and checks the chain: sequence numbers run
// 1..N without gaps or repeats, each PrevHash equals the previous EntryHash,
// and each EntryHash matches the entry's recomputed hash.
func Verify(ctx context.Context, log AuditLog) (VerifyReport, error) {
	entries, err := log.Query(ctx, AuditFilter{})
	if err != nil {
		return VerifyReport{}, err
	}

	chain := make([]AuditEntry, 0, len(entries))
	var rep VerifyReport
	for _, e := range entries {
		if e.Seq == 0 {
			rep.Unchained++
			continue
		}
		chain = append(chain, e)
	}
	sort.SliceStable(chain, func(i, j int) bool { return chain[i].Seq < chain[j].Seq })
	rep.Chained = len(chain)

	var prevSeq int64
	var prevHash string
	for _, e := range chain {
		switch {
		case e.Seq == prevSeq:
			rep.Problems = append(rep.Problems, fmt.Sprintf("seq %d: repeated (entry %s)", e.Seq, e.ID))
		case e.Seq > prevSeq+1:
			rep.Problems = append(rep.Problems, fmt.Sprintf("seq %d: entries %d..%d missing", e.Seq, prevSeq+1, e.Seq-1))
		}
		if e.PrevHash != prevHash {
			rep.Problems = append(rep.Problems, fmt.Sprintf("seq %d: prev_hash does not match the entry before it", e.Seq))
		}
		got, err := HashEntry(e)
		if err != nil {
			return rep, err
		}
		if got != e.EntryHash {
			rep.Problems = append(rep.Problems, fmt.Sprintf("seq %d: entry_hash does not match its contents (entry %s)", e.Seq, e.ID))
		}
		prevSeq, prevHash = e.Seq, e.EntryHash
	}
	if n := len(chain); n > 0 {
		rep.HeadSeq, rep.HeadHash = chain[n-1].Seq, chain[n-1].EntryHash
	}
	rep.OK = len(rep.Problems) == 0
	return rep, nil
}

// ---------------------------------------------------------------------------
// Context
// ---------------------------------------------------------------------------

type agentIDKey struct{}

// DefaultAgentID is recorded when no agent is bound to the context.
const DefaultAgentID = "gateway"

// WithAgentID binds the acting agent's ID to ctx so entries recorded beneath
// it name that agent.
func WithAgentID(ctx context.Context, agentID string) context.Context {
	if agentID == "" {
		return ctx
	}
	return context.WithValue(ctx, agentIDKey{}, agentID)
}

// InheritAgentID copies the agent bound to src (if any) onto dst, for code that
// detaches from a request context.
func InheritAgentID(dst, src context.Context) context.Context {
	if src != nil {
		if id, ok := src.Value(agentIDKey{}).(string); ok {
			return WithAgentID(dst, id)
		}
	}
	return dst
}

// AgentIDFrom returns the agent bound by WithAgentID, or DefaultAgentID.
func AgentIDFrom(ctx context.Context) string {
	if ctx != nil {
		if id, ok := ctx.Value(agentIDKey{}).(string); ok && id != "" {
			return id
		}
	}
	return DefaultAgentID
}
