package server

// POST /dispatch and POST /v1/dispatch (DGS-149).
//
// Wire contract (fixed by the Kata app → apps-edge-worker → gateway path):
//
//	body    {skill, input, mode?, session?:{app}, contract?}
//	200     {output, provenance:{skill, skill_version, contract?, model?, request_id, generated_at}, usage?, warnings?}
//	error   {"error": "<code>", "message": "..."}  — NOT the {error:{code,...}}
//	        shape errorResponse writes; the edge worker parses this flat one.
//
// Every 4xx/5xx except 503 carries X-No-Retry: true. subject_mismatch is 400,
// not 403: the Kata SDK treats any 401/403 from /dispatch as a stale token and
// force re-mints, which would push a user to re-auth over a malformed request.
//
// Structured ops call the provider layer directly — never prepareCompletion —
// so no base system prompt and no RAG context reach the model. The model sees
// exactly two messages: the op's system prompt and the input JSON.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/DojoGenesis/gateway/provider"
)

const maxDispatchBodyBytes = 1 << 20 // 1 MiB

type dispatchSession struct {
	App string `json:"app"`
}

type dispatchRequest struct {
	Skill    string           `json:"skill"`
	Input    json.RawMessage  `json:"input"`
	Mode     *string          `json:"mode,omitempty"`
	Session  *dispatchSession `json:"session,omitempty"`
	Contract *string          `json:"contract,omitempty"`
	// UserID is not part of the contract body, but if a caller sends one it
	// must agree with the token.
	UserID json.RawMessage `json:"user_id,omitempty"`
}

type dispatchProvenance struct {
	Skill        string `json:"skill"`
	SkillVersion string `json:"skill_version"`
	Contract     string `json:"contract,omitempty"`
	Model        string `json:"model,omitempty"`
	RequestID    string `json:"request_id"`
	GeneratedAt  string `json:"generated_at"`
}

type dispatchUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type dispatchResponse struct {
	Output     interface{}        `json:"output"`
	Provenance dispatchProvenance `json:"provenance"`
	Usage      *dispatchUsage     `json:"usage,omitempty"`
	Warnings   []string           `json:"warnings,omitempty"`
}

// dispatchError writes the flat contract error. Everything but 503 is marked
// non-retriable.
func dispatchError(c *gin.Context, status int, code, message string) {
	if status != http.StatusServiceUnavailable {
		c.Header("X-No-Retry", "true")
	}
	c.AbortWithStatusJSON(status, gin.H{"error": code, "message": message})
}

func dispatchRequestID(c *gin.Context) string {
	if v, ok := c.Get("request_id"); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return c.Writer.Header().Get("X-Request-ID")
}

// subjectMismatch reports whether a user_id value is present and disagrees
// with the authenticated subject. A present non-string user_id is a mismatch.
func subjectMismatch(raw interface{}, present bool, subject string) bool {
	if !present {
		return false
	}
	s, ok := raw.(string)
	return !ok || s != subject
}

func (s *Server) handleDispatch(c *gin.Context) {
	subject, ok := getUserIDFromContext(c)
	if !ok {
		// AuthMiddleware guarantees this; reaching here means a wiring bug.
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated", "message": "authentication required"})
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxDispatchBodyBytes))
	if err != nil {
		dispatchError(c, http.StatusBadRequest, "invalid_request", "request body unreadable or larger than 1 MiB")
		return
	}
	var req dispatchRequest
	if err := json.Unmarshal(body, &req); err != nil {
		dispatchError(c, http.StatusBadRequest, "invalid_request", "body must be a JSON object: "+err.Error())
		return
	}
	req.Skill = strings.TrimSpace(req.Skill)
	if req.Skill == "" {
		dispatchError(c, http.StatusBadRequest, "invalid_request", "skill is required")
		return
	}
	if len(bytes.TrimSpace(req.Input)) == 0 || string(bytes.TrimSpace(req.Input)) == "null" {
		dispatchError(c, http.StatusBadRequest, "invalid_request", "input is required")
		return
	}
	if req.Mode != nil && *req.Mode != "focused" && *req.Mode != "exploratory" {
		dispatchError(c, http.StatusBadRequest, "invalid_request", `mode must be "focused" or "exploratory"`)
		return
	}
	// Decoded with json.Number so the schema validator sees exact numbers.
	inputVal, err := jsonschema.UnmarshalJSON(bytes.NewReader(req.Input))
	if err != nil {
		dispatchError(c, http.StatusBadRequest, "invalid_request", "input is not valid JSON")
		return
	}
	inputObj, isObj := inputVal.(map[string]interface{})
	if !isObj {
		dispatchError(c, http.StatusBadRequest, "invalid_request", "input must be a JSON object")
		return
	}

	// Identity before anything else: a body that names another user is
	// refused regardless of which skill it targets.
	var bodyUID interface{}
	if len(req.UserID) > 0 {
		_ = json.Unmarshal(req.UserID, &bodyUID)
	}
	inUID, inPresent := inputObj["user_id"]
	if subjectMismatch(bodyUID, len(req.UserID) > 0, subject) || subjectMismatch(inUID, inPresent, subject) {
		dispatchError(c, http.StatusBadRequest, "subject_mismatch", "user_id does not match the authenticated subject")
		return
	}

	reg := s.dispatch
	if reg == nil {
		reg = &dispatchRegistry{}
	}

	if gop, ok := reg.goOps[req.Skill]; ok {
		if !gop.Implemented {
			dispatchError(c, http.StatusNotImplemented, "not_implemented", gop.Reason)
			return
		}
	}

	op, ok := reg.structured[req.Skill]
	if !ok {
		dispatchError(c, http.StatusNotFound, "unknown_skill", fmt.Sprintf("no dispatchable skill %q", req.Skill))
		return
	}
	if req.Contract == nil || strings.TrimSpace(*req.Contract) != op.Contract {
		got := "<missing>"
		if req.Contract != nil {
			got = *req.Contract
		}
		dispatchError(c, http.StatusUnprocessableEntity, "unsupported_contract",
			fmt.Sprintf("skill %q speaks contract %q; request named %s", op.ID, op.Contract, got))
		return
	}
	if err := op.inputSchema.Validate(inputVal); err != nil {
		dispatchError(c, http.StatusBadRequest, "invalid_input", "input does not match the skill's input schema: "+err.Error())
		return
	}

	s.runStructuredOp(c, reg, op, req.Input)
}

func (s *Server) runStructuredOp(c *gin.Context, reg *dispatchRegistry, op *structuredOp, rawInput json.RawMessage) {
	requestID := dispatchRequestID(c)

	var compact bytes.Buffer
	if err := json.Compact(&compact, rawInput); err != nil {
		dispatchError(c, http.StatusBadRequest, "invalid_request", "input is not valid JSON")
		return
	}

	model := op.Model
	if model == "" {
		model = reg.defaultModel
	}
	provName, prov, err := s.resolveProvider(model)
	if err != nil {
		slog.Error("dispatch: no provider", "skill", op.ID, "model", model, "request_id", requestID, "error", err)
		dispatchError(c, http.StatusServiceUnavailable, "provider_unavailable", "no model provider is available")
		return
	}

	temp := 0.0
	if op.Temperature != nil {
		temp = *op.Temperature
	}
	maxTokens := 4096
	if op.MaxTokens != nil {
		maxTokens = *op.MaxTokens
	}
	creq := &provider.CompletionRequest{
		Model: model,
		Messages: []provider.Message{
			{Role: "system", Content: op.System},
			{Role: "user", Content: compact.String()},
		},
		Temperature: temp,
		MaxTokens:   maxTokens,
		Tools: []provider.Tool{{
			Name:        emitResultTool,
			Description: "Return the result for this request.",
			Parameters:  op.toolParams,
		}},
		ToolChoice: "required",
	}
	// Nothing was prepended: say so, matching /v1/chat/completions' header.
	c.Header("X-Dojo-Injected", "none")

	timeout := reg.timeout
	if timeout <= 0 {
		timeout = defaultKataDispatchTimeout
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	start := time.Now()
	resp, err := prov.GenerateCompletion(ctx, creq)
	if s.latencyTracker != nil {
		s.latencyTracker.Record(provName, time.Since(start).Milliseconds(), err != nil)
	}
	if err != nil || resp == nil {
		if err == nil {
			err = errors.New("nil response")
		}
		slog.Error("dispatch: provider call failed", "skill", op.ID, "provider", provName, "request_id", requestID, "error", err)
		dispatchError(c, http.StatusServiceUnavailable, "provider_unavailable", "model provider failed or timed out")
		return
	}

	output, reason := extractStructuredOutput(op, resp)
	if reason != "" {
		slog.Warn("dispatch: invalid model output", "skill", op.ID, "request_id", requestID, "reason", reason)
		dispatchError(c, http.StatusBadGateway, "invalid_model_output", reason)
		return
	}

	usedModel := resp.Model
	if usedModel == "" {
		usedModel = model
	}
	out := dispatchResponse{
		Output: output,
		Provenance: dispatchProvenance{
			Skill:        op.ID,
			SkillVersion: op.Version,
			Contract:     op.Contract,
			Model:        usedModel,
			RequestID:    requestID,
			GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		},
	}
	if resp.Usage.TotalTokens > 0 || resp.Usage.InputTokens > 0 || resp.Usage.OutputTokens > 0 {
		out.Usage = &dispatchUsage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}
	c.JSON(http.StatusOK, out)
}

// extractStructuredOutput takes the first emit_result tool call, validates it
// against the output schema, and runs the shame-lint. A non-empty reason means
// the output must not be served.
func extractStructuredOutput(op *structuredOp, resp *provider.CompletionResponse) (interface{}, string) {
	if len(resp.ToolCalls) == 0 {
		return nil, "model returned no tool call"
	}
	call := resp.ToolCalls[0]
	if call.Name != emitResultTool {
		return nil, fmt.Sprintf("model called %q, not %q", call.Name, emitResultTool)
	}
	// Providers decode arguments themselves and some swallow the parse error,
	// leaving a nil map (openai_compat.go) — that is the unparsable case.
	if call.Arguments == nil {
		return nil, "tool call arguments were missing or not valid JSON"
	}
	raw, err := json.Marshal(call.Arguments)
	if err != nil {
		return nil, "tool call arguments could not be encoded"
	}
	val, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return nil, "tool call arguments were not valid JSON"
	}
	if err := op.outputSchema.Validate(val); err != nil {
		return nil, "output does not match the skill's output schema: " + err.Error()
	}
	if hits := lintViolations(op.lint, val); len(hits) > 0 {
		return nil, "output failed lint: " + strings.Join(hits, ", ")
	}
	return call.Arguments, ""
}
