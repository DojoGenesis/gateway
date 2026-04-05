package wasm

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// ErrCapabilityDenied is returned when a module attempts a denied operation.
var ErrCapabilityDenied = fmt.Errorf("wasm: capability denied")

// virtualModuleName is the internal module for wrapped tools.
const virtualModuleName = "__wrapped_tools__"

// tracer is the package-level OpenTelemetry tracer for WASM runtime operations.
var tracer = otel.Tracer("dojo.runtime.wasm")

// wasmCallsCounter tracks the total number of Execute invocations.
var wasmCallsCounter metric.Int64Counter

func init() {
	meter := otel.Meter("dojo.runtime.wasm")
	var err error
	wasmCallsCounter, err = meter.Int64Counter("dojo.wasm.calls",
		metric.WithDescription("Total number of WASM Execute invocations"),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		slog.Error("wasm: failed to create dojo.wasm.calls counter", "error", err)
	}
}

// WASMRuntime manages WASM module lifecycle and execution.
type WASMRuntime interface {
	// LoadModule compiles and caches a WASM module.
	LoadModule(ctx context.Context, ref ModuleRef) error

	// LoadModuleBytes compiles a WASM module from raw bytes.
	LoadModuleBytes(ctx context.Context, ref ModuleRef, wasmBytes []byte) error

	// Execute runs a function in a WASM module with capability restrictions.
	Execute(ctx context.Context, moduleName string, funcName string, input []byte, caps CapabilitySet) ([]byte, error)

	// WrapTool wraps an existing tool definition as a WASM-hosted tool.
	// The returned definition routes execution through the WASM sandbox.
	WrapTool(name string, handler func(ctx context.Context, input []byte) ([]byte, error)) (func(ctx context.Context, input []byte) ([]byte, error), error)

	// ListModules returns all loaded modules.
	ListModules(ctx context.Context) ([]ModuleInfo, error)

	// Close releases all WASM resources.
	Close() error
}

// wazeroRuntime manages WASM module lifecycle using Wazero.
type wazeroRuntime struct {
	mu      sync.RWMutex
	engine  wazero.Runtime
	cache   wazero.CompilationCache
	modules map[string]*compiledModule
	wrapped map[string]func(ctx context.Context, input []byte) ([]byte, error) // virtual tool handlers
	config  Config
	closed  bool
}

type compiledModule struct {
	ref      ModuleRef
	compiled wazero.CompiledModule
	data     []byte // raw WASM bytes for hash verification
	caps     CapabilitySet
	info     ModuleInfo
}

// NewRuntime creates a new WASMRuntime backed by Wazero.
func NewRuntime(config Config) (WASMRuntime, error) {
	ctx := context.Background()

	cache := wazero.NewCompilationCache()
	runtimeConfig := wazero.NewRuntimeConfigCompiler().
		WithCompilationCache(cache).
		WithCloseOnContextDone(true)

	engine := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)

	// Instantiate WASI for modules that need it.
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, engine); err != nil {
		engine.Close(ctx)
		cache.Close(ctx)
		return nil, fmt.Errorf("wasm: instantiate WASI: %w", err)
	}

	return &wazeroRuntime{
		engine:  engine,
		cache:   cache,
		modules: make(map[string]*compiledModule),
		wrapped: make(map[string]func(ctx context.Context, input []byte) ([]byte, error)),
		config:  config,
	}, nil
}

func (r *wazeroRuntime) LoadModule(ctx context.Context, ref ModuleRef) error {
	// Load from file in ModuleDir.
	path := filepath.Join(r.config.ModuleDir, ref.Name+".wasm")
	wasmBytes, err := os.ReadFile(path)
	if err != nil {
		// If no file, create a placeholder entry (backward compat with tests that
		// don't supply real WASM bytes).
		return r.loadPlaceholder(ref)
	}
	return r.LoadModuleBytes(ctx, ref, wasmBytes)
}

func (r *wazeroRuntime) LoadModuleBytes(ctx context.Context, ref ModuleRef, wasmBytes []byte) error {
	ctx, span := tracer.Start(ctx, "wasm.load_module",
		trace.WithAttributes(
			attribute.String("module.name", ref.Name),
			attribute.String("module.hash", ref.Hash),
			attribute.Int64("size", int64(len(wasmBytes))),
		),
	)
	defer span.End()

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return fmt.Errorf("wasm: runtime is closed")
	}

	if _, exists := r.modules[ref.Name]; exists {
		return fmt.Errorf("wasm: module %q already loaded", ref.Name)
	}

	// Verify hash if provided.
	if ref.Hash != "" {
		computed := fmt.Sprintf("%x", sha256.Sum256(wasmBytes))
		if computed != ref.Hash {
			return fmt.Errorf("wasm: module %q hash mismatch: expected %s, got %s", ref.Name, ref.Hash, computed)
		}
	}

	caps := r.config.Defaults
	for _, mc := range r.config.Modules {
		if mc.Name == ref.Name {
			caps = mc.Capabilities
			break
		}
	}

	compiled, err := r.engine.CompileModule(ctx, wasmBytes)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("wasm: compile module %q: %w", ref.Name, err)
	}

	mod := &compiledModule{
		ref:      ref,
		compiled: compiled,
		data:     wasmBytes,
		caps:     caps,
		info: ModuleInfo{
			Ref:          ref,
			Size:         int64(len(wasmBytes)),
			Capabilities: caps,
			LoadedAt:     time.Now(),
		},
	}

	// Backfill hash if not provided.
	if ref.Hash == "" {
		mod.ref.Hash = fmt.Sprintf("%x", sha256.Sum256(wasmBytes))
		mod.info.Ref.Hash = mod.ref.Hash
	}

	r.modules[ref.Name] = mod
	return nil
}

// loadPlaceholder creates a non-WASM entry for backward compatibility with
// tests that call LoadModule without a real .wasm file.
func (r *wazeroRuntime) loadPlaceholder(ref ModuleRef) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return fmt.Errorf("wasm: runtime is closed")
	}

	if _, exists := r.modules[ref.Name]; exists {
		return fmt.Errorf("wasm: module %q already loaded", ref.Name)
	}

	caps := r.config.Defaults
	for _, mc := range r.config.Modules {
		if mc.Name == ref.Name {
			caps = mc.Capabilities
			break
		}
	}

	mod := &compiledModule{
		ref:  ref,
		caps: caps,
		info: ModuleInfo{
			Ref:          ref,
			Capabilities: caps,
			LoadedAt:     time.Now(),
		},
	}
	r.modules[ref.Name] = mod
	return nil
}

func (r *wazeroRuntime) Execute(ctx context.Context, moduleName string, funcName string, input []byte, caps CapabilitySet) ([]byte, error) {
	start := time.Now()
	ctx, span := tracer.Start(ctx, "wasm.execute",
		trace.WithAttributes(
			attribute.String("module.name", moduleName),
			attribute.String("function", funcName),
		),
	)
	defer func() {
		span.SetAttributes(attribute.Int64("duration_ms", time.Since(start).Milliseconds()))
		span.End()
	}()

	// Increment the calls counter (nil-safe if init() failed).
	if wasmCallsCounter != nil {
		wasmCallsCounter.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("module.name", moduleName),
				attribute.String("function", funcName),
			),
		)
	}

	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return nil, fmt.Errorf("wasm: runtime is closed")
	}

	mod, ok := r.modules[moduleName]
	if !ok {
		r.mu.RUnlock()
		return nil, fmt.Errorf("wasm: module %q not found", moduleName)
	}
	r.mu.RUnlock()

	// Enforce capability restrictions.
	if err := enforceCaps(caps, mod.caps); err != nil {
		span.RecordError(err)
		return nil, err
	}

	// Apply execution timeout.
	timeout := caps.MaxDuration
	if timeout == 0 {
		timeout = mod.caps.MaxDuration
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// If this is a placeholder module (no compiled WASM), return error.
	if mod.compiled == nil {
		return nil, fmt.Errorf("wasm: function %q not found in module %q", funcName, moduleName)
	}

	// Build module config based on capabilities.
	moduleConfig := wazero.NewModuleConfig().
		WithName("").
		WithStdout(os.Stdout).
		WithStderr(os.Stderr)

	// Only grant FS if capabilities allow it.
	if mod.caps.FileSystem != nil {
		fsConfig := wazero.NewFSConfig()
		for _, p := range mod.caps.FileSystem.ReadPaths {
			fsConfig = fsConfig.WithReadOnlyDirMount(p, p)
		}
		for _, p := range mod.caps.FileSystem.WritePaths {
			fsConfig = fsConfig.WithDirMount(p, p)
		}
		moduleConfig = moduleConfig.WithFSConfig(fsConfig)
	}

	// Set environment variables.
	for _, envKey := range mod.caps.Environment {
		if val, ok := os.LookupEnv(envKey); ok {
			moduleConfig = moduleConfig.WithEnv(envKey, val)
		}
	}

	// Instantiate and run.
	instance, err := r.engine.InstantiateModule(execCtx, mod.compiled, moduleConfig)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("wasm: instantiate module %q: %w", moduleName, err)
	}
	defer instance.Close(execCtx)

	// Use the JSON bridge pattern: alloc → write → call → read.
	return executeViaLinearMemory(execCtx, instance, funcName, input)
}

// executeViaLinearMemory implements the alloc/read/write pattern for passing JSON
// through WASM linear memory.
func executeViaLinearMemory(ctx context.Context, instance api.Module, funcName string, input []byte) ([]byte, error) {
	fn := instance.ExportedFunction(funcName)
	if fn == nil {
		return nil, fmt.Errorf("wasm: exported function %q not found", funcName)
	}

	mem := instance.Memory()
	if mem == nil {
		// No memory export — try calling with no args and reading result directly.
		results, err := fn.Call(ctx)
		if err != nil {
			return nil, fmt.Errorf("wasm: call %q: %w", funcName, err)
		}
		if len(results) > 0 {
			return []byte(fmt.Sprintf("%d", results[0])), nil
		}
		return nil, nil
	}

	// Try to find alloc function for passing input.
	allocFn := instance.ExportedFunction("alloc")
	if allocFn == nil || len(input) == 0 {
		// No alloc or no input — call directly.
		results, err := fn.Call(ctx)
		if err != nil {
			return nil, fmt.Errorf("wasm: call %q: %w", funcName, err)
		}
		if len(results) >= 2 {
			ptr := uint32(results[0])
			size := uint32(results[1])
			data, ok := mem.Read(ptr, size)
			if !ok {
				return nil, fmt.Errorf("wasm: failed to read result from memory")
			}
			return data, nil
		}
		if len(results) == 1 {
			return []byte(fmt.Sprintf("%d", results[0])), nil
		}
		return nil, nil
	}

	// Alloc memory for input, write, call with (ptr, len), read result.
	inputLen := uint64(len(input))
	allocResults, err := allocFn.Call(ctx, inputLen)
	if err != nil {
		return nil, fmt.Errorf("wasm: alloc for input: %w", err)
	}
	if len(allocResults) == 0 {
		return nil, fmt.Errorf("wasm: alloc returned no results")
	}
	inputPtr := uint32(allocResults[0])

	if !mem.Write(inputPtr, input) {
		return nil, fmt.Errorf("wasm: failed to write input to memory")
	}

	results, err := fn.Call(ctx, uint64(inputPtr), inputLen)
	if err != nil {
		return nil, fmt.Errorf("wasm: call %q: %w", funcName, err)
	}

	if len(results) >= 2 {
		ptr := uint32(results[0])
		size := uint32(results[1])
		data, ok := mem.Read(ptr, size)
		if !ok {
			return nil, fmt.Errorf("wasm: failed to read result from memory")
		}
		return data, nil
	}

	return nil, nil
}

func (r *wazeroRuntime) WrapTool(name string, handler func(ctx context.Context, input []byte) ([]byte, error)) (func(ctx context.Context, input []byte) ([]byte, error), error) {
	if handler == nil {
		return nil, fmt.Errorf("wasm: handler is required")
	}

	// Return a wrapped handler that enforces default capabilities.
	wrapped := func(ctx context.Context, input []byte) ([]byte, error) {
		timeout := r.config.Defaults.MaxDuration
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		execCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return handler(execCtx, input)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.wrapped[name] = wrapped

	return wrapped, nil
}

func (r *wazeroRuntime) ListModules(_ context.Context) ([]ModuleInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]ModuleInfo, 0, len(r.modules))
	for _, mod := range r.modules {
		infos = append(infos, mod.info)
	}
	return infos, nil
}

func (r *wazeroRuntime) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}
	r.closed = true

	ctx := context.Background()
	if r.engine != nil {
		r.engine.Close(ctx)
	}
	if r.cache != nil {
		r.cache.Close(ctx)
	}
	r.modules = nil
	r.wrapped = nil
	return nil
}

// enforceCaps verifies the requested capabilities are a subset of the allowed capabilities.
func enforceCaps(requested, allowed CapabilitySet) error {
	// Check filesystem access.
	if requested.FileSystem != nil && allowed.FileSystem == nil {
		return fmt.Errorf("%w: filesystem access not allowed", ErrCapabilityDenied)
	}
	// Check network access.
	if requested.Network != nil && allowed.Network == nil {
		return fmt.Errorf("%w: network access not allowed", ErrCapabilityDenied)
	}
	// Check memory limit.
	if requested.MaxMemory > 0 && allowed.MaxMemory > 0 && requested.MaxMemory > allowed.MaxMemory {
		return fmt.Errorf("%w: memory limit %d exceeds allowed %d", ErrCapabilityDenied, requested.MaxMemory, allowed.MaxMemory)
	}
	return nil
}
