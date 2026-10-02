package server

// Dispatch registry for POST /dispatch (DGS-149).
//
// A structured op is a plugins/kata/skills/<id>/SKILL.md whose frontmatter has
// `kind: structured`. It names an input schema and an output schema (paths
// relative to the plugin root) and its markdown body is the system prompt.
// The registry is built once at startup. A missing plugin root is not an error
// (zero structured ops, the gateway still boots); a malformed op is logged and
// skipped, never fatal.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

const (
	envKataPluginDir       = "KATA_PLUGIN_DIR"
	envKataDispatchTimeout = "KATA_DISPATCH_TIMEOUT"
	envKataDefaultModel    = "KATA_DEFAULT_MODEL"

	defaultKataPluginDir       = "plugins/kata"
	defaultKataDispatchTimeout = 20 * time.Second

	// emitResultTool is the single tool every structured op offers the model.
	emitResultTool = "emit_result"
)

// structuredOp is one loaded, validated `kind: structured` skill.
type structuredOp struct {
	ID          string
	Version     string
	Contract    string
	Model       string
	Temperature *float64
	MaxTokens   *int
	System      string

	inputSchema  *jsonschema.Schema
	outputSchema *jsonschema.Schema
	// toolParams is the output schema as sent to the provider as the
	// emit_result tool's parameters ($schema / $id stripped).
	toolParams map[string]interface{}
	lint       []lintRule
}

type lintRule struct {
	ID string
	re *regexp.Regexp
}

// goHandlerOp names an op implemented in Go rather than by a model.
type goHandlerOp struct {
	ID string
	// Implemented is false while the op is registered but blocked; the
	// handler answers 501 instead of 404 so the caller can tell "not built
	// yet" from "no such skill".
	Implemented bool
	Reason      string
}

// dispatchRegistry maps skill id → handler.
type dispatchRegistry struct {
	pluginDir    string
	defaultModel string
	timeout      time.Duration
	structured   map[string]*structuredOp
	goOps        map[string]*goHandlerOp
}

// kataGoOps are the Go-handler ops (DGS-150). They are registered but NOT
// implemented: the memory store has no per-user scoping (the memories table
// has no owner column and /v1/memory list/get/delete are global), so a
// subject-scoped write or "delete all for this user" cannot be built safely
// without an operator ruling. See the DGS-150 report.
var kataGoOps = []goHandlerOp{
	{ID: "kata-memory-write", Reason: "memory storage is not scoped per user; blocked pending operator ruling (DGS-150)"},
	{ID: "kata-account-delete", Reason: "memory storage is not scoped per user; blocked pending operator ruling (DGS-150)"},
}

// kataPluginDirFromEnv resolves the kata plugin root: KATA_PLUGIN_DIR, else
// plugins/kata relative to the working directory — the same convention the
// provider plugin dir uses (config default "plugins", relative).
func kataPluginDirFromEnv() string {
	if d := strings.TrimSpace(os.Getenv(envKataPluginDir)); d != "" {
		return d
	}
	return defaultKataPluginDir
}

func kataDispatchTimeoutFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv(envKataDispatchTimeout))
	if raw == "" {
		return defaultKataDispatchTimeout
	}
	if d, err := time.ParseDuration(raw); err == nil && d > 0 {
		return d
	}
	if secs, err := time.ParseDuration(raw + "s"); err == nil && secs > 0 {
		return secs
	}
	slog.Warn("dispatch: invalid KATA_DISPATCH_TIMEOUT, using default", "value", raw, "default", defaultKataDispatchTimeout)
	return defaultKataDispatchTimeout
}

// loadDispatchRegistryFromEnv builds the registry from the environment. It
// never fails: every problem is logged and degrades to fewer ops.
func loadDispatchRegistryFromEnv() *dispatchRegistry {
	return loadDispatchRegistry(kataPluginDirFromEnv(), strings.TrimSpace(os.Getenv(envKataDefaultModel)), kataDispatchTimeoutFromEnv())
}

func loadDispatchRegistry(pluginDir, defaultModel string, timeout time.Duration) *dispatchRegistry {
	r := &dispatchRegistry{
		pluginDir:    pluginDir,
		defaultModel: defaultModel,
		timeout:      timeout,
		structured:   map[string]*structuredOp{},
		goOps:        map[string]*goHandlerOp{},
	}
	for i := range kataGoOps {
		op := kataGoOps[i]
		r.goOps[op.ID] = &op
	}

	skillsDir := filepath.Join(pluginDir, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		slog.Info("dispatch: kata plugin skills dir not found; zero structured ops", "dir", skillsDir, "error", err)
		return r
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	lintCache := map[string][]lintRule{}
	for _, name := range names {
		path := filepath.Join(skillsDir, name, "SKILL.md")
		raw, err := os.ReadFile(path)
		if err != nil {
			continue // a directory without SKILL.md is not an op
		}
		op, isStructured, err := parseStructuredOp(pluginDir, path, raw, lintCache)
		if err != nil {
			slog.Error("dispatch: skipping malformed structured op", "path", path, "error", err)
			continue
		}
		if !isStructured {
			continue
		}
		if _, clash := r.goOps[op.ID]; clash {
			slog.Error("dispatch: structured op name collides with a Go-handler op; skipped", "id", op.ID)
			continue
		}
		if _, dup := r.structured[op.ID]; dup {
			slog.Error("dispatch: duplicate structured op name; skipped", "id", op.ID, "path", path)
			continue
		}
		r.structured[op.ID] = op
	}
	slog.Info("dispatch: registry loaded", "plugin_dir", pluginDir, "structured_ops", len(r.structured), "go_ops", len(r.goOps))
	return r
}

type skillFrontmatter struct {
	Name         string   `yaml:"name"`
	Version      string   `yaml:"version"`
	Contract     string   `yaml:"contract"`
	Kind         string   `yaml:"kind"`
	Model        string   `yaml:"model"`
	Temperature  *float64 `yaml:"temperature"`
	MaxTokens    *int     `yaml:"max_tokens"`
	InputSchema  string   `yaml:"input_schema"`
	OutputSchema string   `yaml:"output_schema"`
}

var semverRe = regexp.MustCompile(`^v?\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

// splitFrontmatter separates a `---`-delimited YAML header from the body.
func splitFrontmatter(raw []byte) ([]byte, string, error) {
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return nil, "", fmt.Errorf("no frontmatter")
	}
	rest := text[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, "", fmt.Errorf("unterminated frontmatter")
	}
	header := rest[:end]
	body := rest[end+len("\n---"):]
	// Drop the remainder of the closing delimiter line.
	if nl := strings.IndexByte(body, '\n'); nl >= 0 {
		body = body[nl+1:]
	} else {
		body = ""
	}
	return []byte(header), strings.TrimSpace(body), nil
}

// parseStructuredOp returns isStructured=false (and no error) for any skill
// whose kind is not "structured" — those are not dispatchable.
func parseStructuredOp(pluginDir, path string, raw []byte, lintCache map[string][]lintRule) (*structuredOp, bool, error) {
	header, body, err := splitFrontmatter(raw)
	if err != nil {
		// Not every skill has frontmatter; only a structured op must.
		return nil, false, nil
	}
	var fm skillFrontmatter
	if err := yaml.Unmarshal(header, &fm); err != nil {
		// Can't tell whether it was meant to be structured; if the header
		// mentions it, say so loudly.
		if bytes.Contains(header, []byte("structured")) {
			return nil, false, fmt.Errorf("frontmatter: %w", err)
		}
		return nil, false, nil
	}
	if strings.TrimSpace(fm.Kind) != "structured" {
		return nil, false, nil
	}

	var problems []string
	if strings.TrimSpace(fm.Name) == "" {
		problems = append(problems, "name is required")
	} else if want := filepath.Base(filepath.Dir(path)); fm.Name != want {
		problems = append(problems, fmt.Sprintf("name %q does not match directory %q", fm.Name, want))
	}
	if !semverRe.MatchString(strings.TrimSpace(fm.Version)) {
		problems = append(problems, fmt.Sprintf("version %q is not semver", fm.Version))
	}
	if strings.TrimSpace(fm.Contract) == "" {
		problems = append(problems, "contract is required")
	}
	if fm.InputSchema == "" || fm.OutputSchema == "" {
		problems = append(problems, "input_schema and output_schema are required")
	}
	if body == "" {
		problems = append(problems, "system prompt (markdown body) is empty")
	}
	if fm.Temperature != nil && (*fm.Temperature < 0 || *fm.Temperature > 2) {
		problems = append(problems, "temperature must be within [0, 2]")
	}
	if fm.MaxTokens != nil && *fm.MaxTokens <= 0 {
		problems = append(problems, "max_tokens must be positive")
	}
	if len(problems) > 0 {
		return nil, true, fmt.Errorf("%s", strings.Join(problems, "; "))
	}

	inSch, _, err := compileSchemaFile(pluginDir, fm.InputSchema)
	if err != nil {
		return nil, true, fmt.Errorf("input_schema: %w", err)
	}
	outSch, outDoc, err := compileSchemaFile(pluginDir, fm.OutputSchema)
	if err != nil {
		return nil, true, fmt.Errorf("output_schema: %w", err)
	}
	params, err := toolParamsFromSchema(outDoc)
	if err != nil {
		return nil, true, fmt.Errorf("output_schema: %w", err)
	}

	contract := strings.TrimSpace(fm.Contract)
	rules, ok := lintCache[contract]
	if !ok {
		rules, err = loadLintRules(pluginDir, contract)
		if err != nil {
			// Fail closed: an op whose shame-lint can't be loaded is not served.
			return nil, true, fmt.Errorf("lint rules for contract %q: %w", contract, err)
		}
		lintCache[contract] = rules
	}

	return &structuredOp{
		ID:           fm.Name,
		Version:      strings.TrimSpace(fm.Version),
		Contract:     contract,
		Model:        strings.TrimSpace(fm.Model),
		Temperature:  fm.Temperature,
		MaxTokens:    fm.MaxTokens,
		System:       body,
		inputSchema:  inSch,
		outputSchema: outSch,
		toolParams:   params,
		lint:         rules,
	}, true, nil
}

// resolveUnderRoot joins rel onto root and refuses anything that escapes it.
func resolveUnderRoot(root, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("path %q must be relative to the plugin root", rel)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	full := filepath.Join(absRoot, filepath.FromSlash(rel))
	if full != absRoot && !strings.HasPrefix(full, absRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes the plugin root", rel)
	}
	return full, nil
}

// compileSchemaFile compiles a JSON Schema file under the plugin root and
// returns it plus its raw decoded document.
func compileSchemaFile(pluginDir, rel string) (*jsonschema.Schema, map[string]interface{}, error) {
	full, err := resolveUnderRoot(pluginDir, rel)
	if err != nil {
		return nil, nil, err
	}
	raw, err := os.ReadFile(full)
	if err != nil {
		return nil, nil, err
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, nil, fmt.Errorf("%s: not a JSON object: %w", rel, err)
	}
	siblings, err := contractSiblingSchemas(pluginDir, full)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", rel, err)
	}
	sch, err := compileSchemaDoc(full, raw, siblings)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", rel, err)
	}
	return sch, doc, nil
}

// contractSiblingSchemas collects the shared schemas a contract's files may
// $ref by absolute $id (e.g. kata-ai/v1/_defs.schema.json, which input
// schemas reference as https://contracts.dojogenesis.com/kata-ai/v1/_defs.schema.json).
// It walks from the schema's directory up to (not above) the plugin root and
// returns every *.schema.json found directly in those directories, keyed by
// its $id. Nothing is fetched: an $id that is not on disk under the plugin
// root still fails to resolve.
func contractSiblingSchemas(pluginDir, schemaPath string) (map[string]any, error) {
	absRoot, err := filepath.Abs(pluginDir)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for dir := filepath.Dir(schemaPath); ; dir = filepath.Dir(dir) {
		if dir != absRoot && !strings.HasPrefix(dir, absRoot+string(filepath.Separator)) {
			break
		}
		matches, err := filepath.Glob(filepath.Join(dir, "*.schema.json"))
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			if m == schemaPath {
				continue
			}
			b, err := os.ReadFile(m)
			if err != nil {
				return nil, err
			}
			d, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
			if err != nil {
				return nil, fmt.Errorf("%s: %w", m, err)
			}
			obj, ok := d.(map[string]any)
			if !ok {
				continue
			}
			id, _ := obj["$id"].(string)
			if id == "" {
				continue
			}
			if _, dup := out[id]; !dup {
				out[id] = d
			}
		}
		if dir == absRoot {
			break
		}
	}
	return out, nil
}

// compileSchemaDoc compiles raw schema bytes registered at the file's path,
// defaulting to draft 2020-12, with resources (shared schemas keyed by $id)
// pre-registered so local $refs resolve. Only the file loader is reachable
// (the library default) — no network fetches.
func compileSchemaDoc(location string, raw []byte, resources map[string]any) (*jsonschema.Schema, error) {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	for id, r := range resources {
		if err := c.AddResource(id, r); err != nil {
			return nil, fmt.Errorf("registering %s: %w", id, err)
		}
	}
	if err := c.AddResource(location, doc); err != nil {
		return nil, err
	}
	return c.Compile(location)
}

// toolParamsFromSchema prepares an output schema for use as a tool's
// parameters. Top-level $schema and $id are dropped: some provider
// function-declaration validators (Gemini) reject unknown keywords. A $ref to
// another document is refused at load time — the provider would receive an
// unresolvable reference.
func toolParamsFromSchema(doc map[string]interface{}) (map[string]interface{}, error) {
	if t, _ := doc["type"].(string); t != "object" {
		return nil, fmt.Errorf("tool parameters must be a schema with type \"object\"")
	}
	if ref, ok := findExternalRef(doc); ok {
		return nil, fmt.Errorf("external $ref %q is not supported in an output schema", ref)
	}
	out := make(map[string]interface{}, len(doc))
	for k, v := range doc {
		if k == "$schema" || k == "$id" {
			continue
		}
		out[k] = v
	}
	return out, nil
}

func findExternalRef(v interface{}) (string, bool) {
	switch t := v.(type) {
	case map[string]interface{}:
		for k, child := range t {
			if k == "$ref" {
				if s, ok := child.(string); ok && !strings.HasPrefix(s, "#") {
					return s, true
				}
			}
			if ref, ok := findExternalRef(child); ok {
				return ref, true
			}
		}
	case []interface{}:
		for _, child := range t {
			if ref, ok := findExternalRef(child); ok {
				return ref, true
			}
		}
	}
	return "", false
}

type lintFile struct {
	Rules []struct {
		ID      string `json:"id"`
		Pattern string `json:"pattern"`
		Flags   string `json:"flags"`
	} `json:"rules"`
}

// loadLintRules reads contracts/<contract>/lint.json under the plugin root.
func loadLintRules(pluginDir, contract string) ([]lintRule, error) {
	full, err := resolveUnderRoot(pluginDir, filepath.ToSlash(filepath.Join("contracts", contract, "lint.json")))
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(full)
	if err != nil {
		return nil, err
	}
	return parseLintRules(raw)
}

func parseLintRules(raw []byte) ([]lintRule, error) {
	var lf lintFile
	if err := json.Unmarshal(raw, &lf); err != nil {
		return nil, err
	}
	rules := make([]lintRule, 0, len(lf.Rules))
	for i, r := range lf.Rules {
		if r.ID == "" || r.Pattern == "" {
			return nil, fmt.Errorf("rules[%d]: id and pattern are required", i)
		}
		pattern := r.Pattern
		if strings.Contains(r.Flags, "i") {
			pattern = "(?i)" + pattern
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("rules[%d] (%s): %w", i, r.ID, err)
		}
		rules = append(rules, lintRule{ID: r.ID, re: re})
	}
	return rules, nil
}

// lintViolations applies every rule to every string value in v, recursively.
// It returns one entry per (rule, path) hit.
func lintViolations(rules []lintRule, v interface{}) []string {
	var hits []string
	var walk func(path string, v interface{})
	walk = func(path string, v interface{}) {
		switch t := v.(type) {
		case string:
			for _, r := range rules {
				if r.re.MatchString(t) {
					hits = append(hits, r.ID+" at "+path)
				}
			}
		case map[string]interface{}:
			keys := make([]string, 0, len(t))
			for k := range t {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				walk(path+"/"+k, t[k])
			}
		case []interface{}:
			for i, child := range t {
				walk(fmt.Sprintf("%s/%d", path, i), child)
			}
		}
	}
	walk("", v)
	return hits
}
