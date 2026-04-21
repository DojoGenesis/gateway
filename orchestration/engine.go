package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/DojoGenesis/gateway/disposition"
)

// ErrorType classifies errors for retry/replan/abort decisions.
type ErrorType string

const (
	ErrorTypeTransient  ErrorType = "transient"
	ErrorTypePersistent ErrorType = "persistent"
	ErrorTypeFatal      ErrorType = "fatal"
)

// EngineConfig controls orchestration behavior.
type EngineConfig struct {
	MaxRetries              int
	RetryBackoff            time.Duration
	MaxBackoff              time.Duration
	MaxParallelNodes        int
	EnableAutoReplanning    bool
	MaxReplanningAttempts   int
	EnableJitter            bool
	EnableCircuitBreaker    bool
	CircuitBreakerThreshold int
	CircuitBreakerTimeout   time.Duration
}

// DefaultEngineConfig returns sensible default configuration.
func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		MaxRetries:              3,
		RetryBackoff:            1 * time.Second,
		MaxBackoff:              30 * time.Second,
		MaxParallelNodes:        5,
		EnableAutoReplanning:    true,
		MaxReplanningAttempts:   2,
		EnableJitter:            true,
		EnableCircuitBreaker:    true,
		CircuitBreakerThreshold: 5,
		CircuitBreakerTimeout:   60 * time.Second,
	}
}

type circuitBreakerState struct {
	consecutiveFailures int
	lastFailureTime     time.Time
	isOpen              bool
}

// ToolHealthMetrics tracks per-tool execution statistics.
type ToolHealthMetrics struct {
	TotalAttempts   int
	SuccessfulCalls int
	FailedCalls     int
	LastErrorType   ErrorType
}

// Engine is the DAG-based orchestration engine that executes plans
// with auto-replanning, circuit breakers, and cost estimation.
//
// Engine is safe for concurrent Execute calls. The event emitter is supplied
// per-Execute invocation (see Execute) rather than stored on the struct, so
// parallel orchestrations never clobber each other's event stream.
type Engine struct {
	config          *EngineConfig
	planner         PlannerInterface
	toolInvoker     ToolInvokerInterface
	traceLogger     TraceLoggerInterface
	budgetTracker   BudgetTrackerInterface
	disposition     *disposition.DispositionConfig
	mu              sync.RWMutex
	circuitBreakers map[string]*circuitBreakerState
	toolHealth      map[string]*ToolHealthMetrics
	healthMu        sync.RWMutex
}

// EngineOption is a functional option for configuring the Engine.
type EngineOption func(*Engine)

// WithDisposition sets the disposition configuration for the engine.
// This controls pacing behavior per the Gateway-ADA Contract.
func WithDisposition(disp *disposition.DispositionConfig) EngineOption {
	return func(e *Engine) {
		e.disposition = disp
	}
}

// NewEngine creates a new orchestration engine.
// traceLogger and budgetTracker are optional (can be nil).
//
// The eventEmitter parameter is retained for call-site backward compatibility
// but is intentionally ignored: emitters must be supplied per-Execute to avoid
// the cross-orchestration race surfaced by the previous SetEventEmitter API
// (see ADR-022 P0). Callers should pass nil for this parameter.
func NewEngine(
	config *EngineConfig,
	planner PlannerInterface,
	toolInvoker ToolInvokerInterface,
	traceLogger TraceLoggerInterface,
	eventEmitter EventEmitterInterface,
	budgetTracker BudgetTrackerInterface,
	opts ...EngineOption,
) *Engine {
	_ = eventEmitter // intentionally unused — pass emitter per-Execute instead

	if config == nil {
		config = DefaultEngineConfig()
	}

	e := &Engine{
		config:          config,
		planner:         planner,
		toolInvoker:     toolInvoker,
		traceLogger:     traceLogger,
		budgetTracker:   budgetTracker,
		disposition:     disposition.DefaultDisposition(),
		circuitBreakers: make(map[string]*circuitBreakerState),
		toolHealth:      make(map[string]*ToolHealthMetrics),
	}

	// Apply options
	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Execute runs a plan to completion with auto-replanning on failure.
// emitter receives node start/end and replanning events during execution.
// Pass nil to disable event emission. Safe for concurrent calls on the same engine.
func (e *Engine) Execute(ctx context.Context, plan *Plan, task *Task, userID string, emitter EventEmitterInterface) error {
	estimatedCost, err := e.EstimatePlanCost(plan)
	if err != nil {
		return fmt.Errorf("failed to estimate plan cost: %w", err)
	}

	if err := e.checkBudget(userID, estimatedCost); err != nil {
		return fmt.Errorf("budget check failed: %w", err)
	}

	var orchestrationSpan SpanHandle
	if e.traceLogger != nil {
		orchestrationSpan, err = e.traceLogger.StartSpan(ctx, task.ID, "orchestration.execute", map[string]interface{}{
			"task_id":        task.ID,
			"plan_id":        plan.ID,
			"node_count":     len(plan.Nodes),
			"user_id":        userID,
			"plan_version":   plan.Version,
			"estimated_cost": estimatedCost,
		})
		if err == nil && orchestrationSpan != nil {
			defer func() {
				if plan.HasFailedNodes() {
					_ = e.traceLogger.FailSpan(ctx, orchestrationSpan, "orchestration failed with errors")
				} else {
					_ = e.traceLogger.EndSpan(ctx, orchestrationSpan, map[string]interface{}{
						"completed_nodes": len(plan.Nodes),
						"status":          "success",
					})
				}
			}()
		}
	}

	if orchestrationSpan != nil {
		planJSON, _ := json.Marshal(plan)
		orchestrationSpan.AddMetadata("plan", string(planJSON))
	}

	replanningAttempts := 0
	currentPlan := plan

	for {
		execErr := e.executePlan(ctx, currentPlan, task, emitter)
		if execErr == nil {
			return nil
		}

		errorType := e.ClassifyError(execErr)
		if errorType == ErrorTypeFatal {
			return execErr
		}

		if !e.config.EnableAutoReplanning {
			return execErr
		}

		if replanningAttempts >= e.config.MaxReplanningAttempts {
			return fmt.Errorf("replanning attempts exhausted (%d): %w", e.config.MaxReplanningAttempts, execErr)
		}

		failedNode := e.findFirstFailedNode(currentPlan)
		if failedNode == nil {
			return execErr
		}

		nodeError := fmt.Errorf("node error: %s", failedNode.Error)
		if e.ClassifyError(nodeError) == ErrorTypeFatal {
			return fmt.Errorf("fatal error in node %s: %s", failedNode.ID, failedNode.Error)
		}

		newPlan, replanErr := e.handlePersistentFailure(ctx, currentPlan, task, failedNode, emitter)
		if replanErr != nil {
			return fmt.Errorf("replanning failed: %w", replanErr)
		}

		replanningAttempts++
		currentPlan = newPlan

		if orchestrationSpan != nil {
			orchestrationSpan.AddMetadata(fmt.Sprintf("replanning_attempt_%d", replanningAttempts), map[string]interface{}{
				"new_plan_id": newPlan.ID,
				"failed_node": failedNode.ID,
			})
		}
	}
}

func (e *Engine) executePlan(ctx context.Context, plan *Plan, task *Task, emitter EventEmitterInterface) error {
	maxIterations := 1000
	iteration := 0

	for !plan.AllNodesCompleted() {
		iteration++
		if iteration > maxIterations {
			return fmt.Errorf("execution exceeded maximum iterations: possible infinite loop")
		}

		executableNodes := plan.GetExecutableNodes()
		if len(executableNodes) == 0 {
			if !plan.AllNodesCompleted() {
				return fmt.Errorf("no executable nodes available but plan is not complete")
			}
			break
		}

		if err := e.executeNodesInParallel(ctx, executableNodes, plan, task.ID, emitter); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	if plan.HasFailedNodes() {
		failedNode := e.findFirstFailedNode(plan)
		if failedNode != nil {
			return fmt.Errorf("plan execution failed at node %s: %s", failedNode.ID, failedNode.Error)
		}
		return fmt.Errorf("plan execution failed")
	}

	return nil
}

func (e *Engine) executeNodesInParallel(ctx context.Context, nodes []*PlanNode, plan *Plan, traceID string, emitter EventEmitterInterface) error {
	parallelBatchSize := e.config.MaxParallelNodes

	for i := 0; i < len(nodes); i += parallelBatchSize {
		end := i + parallelBatchSize
		if end > len(nodes) {
			end = len(nodes)
		}
		batch := nodes[i:end]

		var wg sync.WaitGroup
		errChan := make(chan error, len(batch))

		for _, node := range batch {
			wg.Add(1)
			go func(n *PlanNode) {
				defer wg.Done()
				if err := e.executeNode(ctx, n, plan, traceID, emitter); err != nil {
					errChan <- err
				}
			}(node)
		}

		wg.Wait()
		close(errChan)

		for err := range errChan {
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (e *Engine) executeNode(ctx context.Context, node *PlanNode, plan *Plan, traceID string, emitter EventEmitterInterface) error {
	// Apply pacing delay before executing node (per Gateway-ADA Contract §3.1)
	if err := e.applyPacingDelay(ctx); err != nil {
		return err
	}

	e.mu.Lock()
	node.State = NodeStateRunning
	now := time.Now()
	node.StartTime = &now
	e.mu.Unlock()

	if emitter != nil {
		e.sendNodeStartEvent(node, plan.ID, emitter)
	}

	var nodeSpan SpanHandle
	if e.traceLogger != nil {
		var err error
		nodeSpan, err = e.traceLogger.StartSpan(ctx, traceID, fmt.Sprintf("node.%s", node.ToolName), map[string]interface{}{
			"node_id":      node.ID,
			"tool_name":    node.ToolName,
			"parameters":   node.Parameters,
			"dependencies": node.Dependencies,
		})
		if err != nil {
			nodeSpan = nil
		}
	}

	result, execErr := e.invokeToolWithRetry(ctx, node, plan, traceID)

	e.mu.Lock()
	endTime := time.Now()
	node.EndTime = &endTime

	if execErr != nil {
		node.State = NodeStateFailed
		node.Error = execErr.Error()
		if nodeSpan != nil && e.traceLogger != nil {
			_ = e.traceLogger.FailSpan(ctx, nodeSpan, execErr.Error())
		}
	} else {
		node.State = NodeStateSuccess
		node.Result = result
		if nodeSpan != nil && e.traceLogger != nil {
			_ = e.traceLogger.EndSpan(ctx, nodeSpan, map[string]interface{}{
				"result": result,
			})
		}
	}
	e.mu.Unlock()

	if emitter != nil {
		e.sendNodeEndEvent(node, plan.ID, emitter)
	}

	return execErr
}

func (e *Engine) invokeToolWithRetry(ctx context.Context, node *PlanNode, plan *Plan, traceID string) (map[string]interface{}, error) {
	if e.config.EnableCircuitBreaker {
		if isOpen, err := e.checkCircuitBreaker(node.ToolName); isOpen {
			return nil, err
		}
	}

	var lastErr error
	errorType := ErrorTypeTransient

	for attempt := 0; attempt <= e.config.MaxRetries; attempt++ {
		if attempt > 0 {
			if !e.shouldRetry(node, lastErr) {
				break
			}

			backoffDuration := e.calculateAdaptiveBackoff(attempt, errorType, node.ToolName)
			if retryErr := e.retryWithBackoff(ctx, backoffDuration); retryErr != nil {
				return nil, retryErr
			}
		}

		e.recordToolAttempt(node.ToolName)

		result, err := e.toolInvoker.InvokeTool(ctx, node.ToolName, node.Parameters)
		if err == nil {
			e.recordToolSuccess(node.ToolName)
			if e.config.EnableCircuitBreaker {
				e.resetCircuitBreaker(node.ToolName)
			}
			return result, nil
		}

		lastErr = err
		errorType = e.ClassifyError(err)
		node.RetryCount = attempt

		e.recordToolFailure(node.ToolName, errorType)

		if e.config.EnableCircuitBreaker {
			e.updateCircuitBreaker(node.ToolName, errorType)
		}
	}

	return nil, fmt.Errorf("tool execution failed after %d retries: %w", e.config.MaxRetries, lastErr)
}

func (e *Engine) shouldRetry(node *PlanNode, err error) bool {
	if node.RetryCount >= e.config.MaxRetries {
		return false
	}

	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	transientErrors := []string{
		"timeout",
		"temporary",
		"rate limit",
		"too many requests",
		"service unavailable",
		"connection",
		"network",
		"deadline",
	}

	for _, transient := range transientErrors {
		if strings.Contains(errMsg, transient) {
			return true
		}
	}

	return false
}

func (e *Engine) calculateAdaptiveBackoff(attempt int, errorType ErrorType, toolName string) time.Duration {
	baseBackoff := time.Duration(math.Pow(2, float64(attempt))) * e.config.RetryBackoff

	switch errorType {
	case ErrorTypeTransient:
		e.healthMu.RLock()
		if health, exists := e.toolHealth[toolName]; exists && health.LastErrorType == ErrorTypeTransient {
			baseBackoff = baseBackoff * 2
		}
		e.healthMu.RUnlock()
	case ErrorTypePersistent:
		baseBackoff = e.config.RetryBackoff
	case ErrorTypeFatal:
		return 0
	}

	if baseBackoff > e.config.MaxBackoff {
		baseBackoff = e.config.MaxBackoff
	}

	if e.config.EnableJitter {
		source := rand.NewSource(time.Now().UnixNano())
		rng := rand.New(source)
		jitter := time.Duration(rng.Float64() * float64(baseBackoff) * 0.3)
		baseBackoff = baseBackoff + jitter
	}

	return baseBackoff
}

func (e *Engine) retryWithBackoff(ctx context.Context, backoffDuration time.Duration) error {
	timer := time.NewTimer(backoffDuration)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *Engine) sendNodeStartEvent(node *PlanNode, planID string, emitter EventEmitterInterface) {
	if emitter == nil {
		return
	}

	emitter.Emit(StreamEvent{
		Type: "orchestration.node.start",
		Data: map[string]interface{}{
			"node_id":    node.ID,
			"plan_id":    planID,
			"tool_name":  node.ToolName,
			"parameters": node.Parameters,
		},
		Timestamp: time.Now(),
	})
}

func (e *Engine) sendNodeEndEvent(node *PlanNode, planID string, emitter EventEmitterInterface) {
	if emitter == nil {
		return
	}

	var durationMs int64
	if node.StartTime != nil && node.EndTime != nil {
		durationMs = node.EndTime.Sub(*node.StartTime).Milliseconds()
	}

	emitter.Emit(StreamEvent{
		Type: "orchestration.node.end",
		Data: map[string]interface{}{
			"node_id":     node.ID,
			"plan_id":     planID,
			"tool_name":   node.ToolName,
			"state":       string(node.State),
			"result":      node.Result,
			"error":       node.Error,
			"duration_ms": durationMs,
			"retry_count": node.RetryCount,
		},
		Timestamp: time.Now(),
	})
}

// ClassifyError categorizes an error for retry/replan/abort decisions.
func (e *Engine) ClassifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypeTransient
	}

	errMsg := strings.ToLower(err.Error())

	fatalErrors := []string{
		"budget exceeded",
		"insufficient budget",
		"invalid plan structure",
		"cyclic dependencies",
		"unauthorized",
		"forbidden",
		"authentication failed",
		"permission denied",
		"replanning attempts exhausted",
		"circuit_breaker_open",
	}

	for _, fatal := range fatalErrors {
		if strings.Contains(errMsg, fatal) {
			return ErrorTypeFatal
		}
	}

	transientErrors := []string{
		"timeout",
		"temporary",
		"rate limit",
		"too many requests",
		"service unavailable",
		"connection",
		"network",
		"deadline",
		"unavailable",
		"try again",
	}

	for _, transient := range transientErrors {
		if strings.Contains(errMsg, transient) {
			return ErrorTypeTransient
		}
	}

	persistentErrors := []string{
		"invalid parameter",
		"missing parameter",
		"not found",
		"does not exist",
		"invalid input",
		"validation failed",
		"schema mismatch",
		"parse error",
		"invalid format",
	}

	for _, persistent := range persistentErrors {
		if strings.Contains(errMsg, persistent) {
			return ErrorTypePersistent
		}
	}

	return ErrorTypePersistent
}

func (e *Engine) findFirstFailedNode(plan *Plan) *PlanNode {
	for _, node := range plan.Nodes {
		if node.State == NodeStateFailed {
			return node
		}
	}
	return nil
}

func (e *Engine) handlePersistentFailure(ctx context.Context, plan *Plan, task *Task, failedNode *PlanNode, emitter EventEmitterInterface) (*Plan, error) {
	if !e.config.EnableAutoReplanning {
		return nil, fmt.Errorf("auto-replanning is disabled")
	}

	completedNodes := make([]*PlanNode, 0)
	failedNodes := make([]string, 0)

	for _, node := range plan.Nodes {
		if node.State == NodeStateSuccess {
			completedNodes = append(completedNodes, node)
		} else if node.State == NodeStateFailed {
			failedNodes = append(failedNodes, node.ID)
		}
	}

	errorContext := fmt.Sprintf("Node %s (%s) failed with error: %s\nRetries attempted: %d\nCompleted nodes: %d/%d",
		failedNode.ID,
		failedNode.ToolName,
		failedNode.Error,
		failedNode.RetryCount,
		len(completedNodes),
		len(plan.Nodes),
	)

	if emitter != nil {
		emitter.Emit(StreamEvent{
			Type: "orchestration.replanning",
			Data: map[string]interface{}{
				"plan_id":      plan.ID,
				"task_id":      task.ID,
				"reason":       errorContext,
				"failed_nodes": failedNodes,
			},
			Timestamp: time.Now(),
		})
	}

	if e.planner == nil {
		return nil, fmt.Errorf("planner is not available for replanning")
	}

	newPlan, err := e.planner.RegeneratePlan(ctx, task, plan, errorContext)
	if err != nil {
		return nil, fmt.Errorf("failed to regenerate plan: %w", err)
	}

	return newPlan, nil
}

// EstimatePlanCost estimates the total token cost of executing a plan.
func (e *Engine) EstimatePlanCost(plan *Plan) (int, error) {
	if plan == nil || len(plan.Nodes) == 0 {
		return 0, nil
	}

	totalEstimatedTokens := 0
	dependencyDepths := e.calculateDependencyDepths(plan)
	maxDepth := e.getMaxDepth(dependencyDepths)

	for _, node := range plan.Nodes {
		nodeDepth := dependencyDepths[node.ID]
		nodeTokens := e.estimateNodeTokens(node, nodeDepth, maxDepth)
		totalEstimatedTokens += nodeTokens
	}

	planningOverhead := e.calculatePlanningOverhead(plan, totalEstimatedTokens)
	totalEstimatedTokens += planningOverhead

	return totalEstimatedTokens, nil
}

func (e *Engine) calculateDependencyDepths(plan *Plan) map[string]int {
	depths := make(map[string]int)

	for _, node := range plan.Nodes {
		depths[node.ID] = e.calculateNodeDepth(node, plan, depths, make(map[string]bool))
	}

	return depths
}

func (e *Engine) calculateNodeDepth(node *PlanNode, plan *Plan, depths map[string]int, visited map[string]bool) int {
	if depth, exists := depths[node.ID]; exists {
		return depth
	}

	if visited[node.ID] {
		return 0
	}
	visited[node.ID] = true

	if len(node.Dependencies) == 0 {
		return 0
	}

	maxDepth := 0
	for _, depID := range node.Dependencies {
		depNode := plan.GetNodeByID(depID)
		if depNode != nil {
			depDepth := e.calculateNodeDepth(depNode, plan, depths, visited)
			if depDepth > maxDepth {
				maxDepth = depDepth
			}
		}
	}

	return maxDepth + 1
}

func (e *Engine) getMaxDepth(depths map[string]int) int {
	maxDepth := 0
	for _, depth := range depths {
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	return maxDepth
}

func (e *Engine) calculatePlanningOverhead(plan *Plan, baseTokens int) int {
	nodeCount := len(plan.Nodes)

	baseOverhead := int(float64(baseTokens) * 0.15)

	if nodeCount > 10 {
		baseOverhead = int(float64(baseTokens) * 0.25)
	} else if nodeCount > 5 {
		baseOverhead = int(float64(baseTokens) * 0.20)
	}

	complexityFactor := 1.0
	parallelGroups := e.countParallelGroups(plan)
	if parallelGroups > 1 {
		complexityFactor += float64(parallelGroups) * 0.05
	}

	return int(float64(baseOverhead) * complexityFactor)
}

func (e *Engine) countParallelGroups(plan *Plan) int {
	groups := 0
	for _, node := range plan.Nodes {
		if len(node.Dependencies) == 0 {
			groups++
		}
	}
	return groups
}

func (e *Engine) estimateNodeTokens(node *PlanNode, depth int, maxDepth int) int {
	toolCategory := e.categorizeToolByName(node.ToolName)

	inputTokens := e.estimateInputTokens(node, toolCategory)
	outputTokens := e.estimateOutputTokens(toolCategory)
	contextTokens := e.estimateContextTokens(depth, maxDepth)

	totalTokens := inputTokens + outputTokens + contextTokens

	safetyMultiplier := e.getSafetyMultiplier(toolCategory)
	return int(float64(totalTokens) * safetyMultiplier)
}

func (e *Engine) categorizeToolByName(toolName string) string {
	webTools := map[string]bool{
		"web_search": true, "fetch_url": true, "api_call": true,
		"web_navigate": true, "web_scrape_structured": true,
		"advanced_web_search": true, "multi_source_search": true,
	}

	fileTools := map[string]bool{
		"read_file": true, "write_file": true, "list_directory": true,
		"search_files": true, "create_directory": true, "delete_file": true,
	}

	memoryTools := map[string]bool{
		"memory_store": true, "memory_search": true, "memory_list": true,
		"memory_delete": true, "memory_stats": true,
	}

	computeTools := map[string]bool{
		"calculate": true, "analyze_data": true, "process_json": true,
		"transform_data": true,
	}

	if webTools[toolName] {
		return "web"
	} else if fileTools[toolName] {
		return "file"
	} else if memoryTools[toolName] {
		return "memory"
	} else if computeTools[toolName] {
		return "compute"
	}

	return "generic"
}

func (e *Engine) estimateInputTokens(node *PlanNode, category string) int {
	baseInputTokens := 80

	toolNameTokens := len(node.ToolName) / 4

	paramComplexity := e.analyzeParameterComplexity(node.Parameters)

	categoryMultiplier := 1.0
	switch category {
	case "web":
		categoryMultiplier = 1.2
	case "file":
		categoryMultiplier = 0.9
	case "memory":
		categoryMultiplier = 1.1
	case "compute":
		categoryMultiplier = 1.3
	}

	return int(float64(baseInputTokens+toolNameTokens+paramComplexity) * categoryMultiplier)
}

func (e *Engine) analyzeParameterComplexity(params map[string]interface{}) int {
	if len(params) == 0 {
		return 20
	}

	totalComplexity := 0

	for key, value := range params {
		keyTokens := len(key) / 4
		totalComplexity += keyTokens

		valueTokens := e.estimateValueTokens(value, 0)
		totalComplexity += valueTokens
	}

	return totalComplexity
}

func (e *Engine) estimateValueTokens(value interface{}, depth int) int {
	if depth > 5 {
		return 10
	}

	switch v := value.(type) {
	case string:
		return len(v) / 4
	case int, int32, int64, float32, float64, bool:
		return 5
	case []interface{}:
		tokens := 10
		for _, item := range v {
			tokens += e.estimateValueTokens(item, depth+1)
		}
		return tokens
	case map[string]interface{}:
		tokens := 15
		for key, val := range v {
			tokens += len(key)/4 + e.estimateValueTokens(val, depth+1)
		}
		return tokens
	default:
		jsonBytes, _ := json.Marshal(value)
		return len(string(jsonBytes)) / 4
	}
}

func (e *Engine) estimateOutputTokens(category string) int {
	switch category {
	case "web":
		return 800
	case "file":
		return 400
	case "memory":
		return 300
	case "compute":
		return 500
	default:
		return 350
	}
}

func (e *Engine) estimateContextTokens(depth int, maxDepth int) int {
	if maxDepth == 0 {
		return 50
	}

	contextPerDepth := 100

	return int(float64(depth*contextPerDepth) * 0.8)
}

func (e *Engine) getSafetyMultiplier(category string) float64 {
	switch category {
	case "web":
		return 1.4
	case "file":
		return 1.3
	case "memory":
		return 1.2
	case "compute":
		return 1.5
	default:
		return 1.35
	}
}

func (e *Engine) checkBudget(userID string, estimatedCost int) error {
	if e.budgetTracker == nil {
		return nil
	}

	if userID == "" {
		return nil
	}

	remaining, err := e.budgetTracker.GetRemaining(userID)
	if err != nil {
		return fmt.Errorf("failed to get remaining budget: %w", err)
	}

	if estimatedCost > remaining {
		return fmt.Errorf("insufficient budget: estimated cost %d tokens exceeds remaining budget %d tokens (%.2f%% over budget)",
			estimatedCost, remaining, float64(estimatedCost-remaining)/float64(remaining)*100)
	}

	return nil
}

// Circuit breaker methods

func (e *Engine) checkCircuitBreaker(toolName string) (bool, error) {
	e.healthMu.RLock()
	defer e.healthMu.RUnlock()

	state, exists := e.circuitBreakers[toolName]
	if !exists {
		return false, nil
	}

	if !state.isOpen {
		return false, nil
	}

	if time.Since(state.lastFailureTime) > e.config.CircuitBreakerTimeout {
		return false, nil
	}

	return true, fmt.Errorf("circuit_breaker_open for tool %s: %d consecutive failures, retry after %v",
		toolName, state.consecutiveFailures, e.config.CircuitBreakerTimeout-time.Since(state.lastFailureTime))
}

func (e *Engine) updateCircuitBreaker(toolName string, errorType ErrorType) {
	e.healthMu.Lock()
	defer e.healthMu.Unlock()

	state, exists := e.circuitBreakers[toolName]
	if !exists {
		state = &circuitBreakerState{}
		e.circuitBreakers[toolName] = state
	}

	if errorType == ErrorTypeFatal || errorType == ErrorTypePersistent {
		state.consecutiveFailures++
		state.lastFailureTime = time.Now()

		if state.consecutiveFailures >= e.config.CircuitBreakerThreshold {
			state.isOpen = true
		}
	}
}

func (e *Engine) resetCircuitBreaker(toolName string) {
	e.healthMu.Lock()
	defer e.healthMu.Unlock()

	state, exists := e.circuitBreakers[toolName]
	if !exists {
		return
	}

	state.consecutiveFailures = 0
	state.isOpen = false
}

// Health metrics methods

func (e *Engine) recordToolAttempt(toolName string) {
	e.healthMu.Lock()
	defer e.healthMu.Unlock()

	health, exists := e.toolHealth[toolName]
	if !exists {
		health = &ToolHealthMetrics{}
		e.toolHealth[toolName] = health
	}

	health.TotalAttempts++
}

func (e *Engine) recordToolSuccess(toolName string) {
	e.healthMu.Lock()
	defer e.healthMu.Unlock()

	health, exists := e.toolHealth[toolName]
	if !exists {
		health = &ToolHealthMetrics{}
		e.toolHealth[toolName] = health
	}

	health.SuccessfulCalls++
}

func (e *Engine) recordToolFailure(toolName string, errorType ErrorType) {
	e.healthMu.Lock()
	defer e.healthMu.Unlock()

	health, exists := e.toolHealth[toolName]
	if !exists {
		health = &ToolHealthMetrics{}
		e.toolHealth[toolName] = health
	}

	health.FailedCalls++
	health.LastErrorType = errorType
}

// GetToolHealthMetrics returns a copy of the health metrics for a tool.
func (e *Engine) GetToolHealthMetrics(toolName string) *ToolHealthMetrics {
	e.healthMu.RLock()
	defer e.healthMu.RUnlock()

	health, exists := e.toolHealth[toolName]
	if !exists {
		return nil
	}

	copy := *health
	return &copy
}

// applyPacingDelay applies a delay before tool execution based on disposition.Pacing.
// Per Gateway-ADA Contract §3.1:
//   - deliberate: 2-5s delay (using 3s as midpoint)
//   - measured: 1-2s delay (using 1.5s as midpoint)
//   - responsive: 0.5-1s delay (using 0.75s as midpoint)
//   - rapid: no delay, parallel execution enabled elsewhere
func (e *Engine) applyPacingDelay(ctx context.Context) error {
	delay := e.pacingDelay()
	if delay == 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// pacingDelay returns the delay duration based on disposition.Pacing.
func (e *Engine) pacingDelay() time.Duration {
	if e.disposition == nil {
		return 1500 * time.Millisecond // Default: measured pacing
	}

	switch e.disposition.Pacing {
	case "deliberate":
		return 3 * time.Second
	case "measured":
		return 1500 * time.Millisecond
	case "responsive":
		return 750 * time.Millisecond
	case "rapid":
		return 0 // No delay
	default:
		return 1500 * time.Millisecond // Default: measured
	}
}
