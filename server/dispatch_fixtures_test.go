package server

// Fixture eval for the kata-ai/v1 contract (DGS-149).
//
// For every contracts/kata-ai/v1/<op>/fixtures/*.json
// ({input, valid_output, invalid_outputs:[{why, output}]}):
//   - input validates against <op>/input.schema.json
//   - valid_output passes <op>/output.schema.json AND the contract lint
//   - each invalid whose why starts "schema:" fails the schema
//   - each invalid whose why starts "lint:" passes the schema but fails lint
//
// TestKataContractFixtures runs against the vendored contract
// (KATA_PLUGIN_DIR, else ../plugins/kata) and skips when it is absent.
// TestKataContractFixtures_Testdata runs the same walker against the mini
// plugin in testdata/ so the walker itself is always exercised.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"
)

type kataFixture struct {
	Input          json.RawMessage `json:"input"`
	ValidOutput    json.RawMessage `json:"valid_output"`
	InvalidOutputs []struct {
		Why    string          `json:"why"`
		Output json.RawMessage `json:"output"`
	} `json:"invalid_outputs"`
}

func runKataContractFixtures(t *testing.T, pluginDir string) int {
	t.Helper()
	contractDir := filepath.Join(pluginDir, "contracts", "kata-ai", "v1")
	rules, err := loadLintRules(pluginDir, "kata-ai/v1")
	require.NoError(t, err, "lint.json")

	files, err := filepath.Glob(filepath.Join(contractDir, "*", "fixtures", "*.json"))
	require.NoError(t, err)

	compiled := map[string][2]*jsonschema.Schema{}
	for _, f := range files {
		opDir := filepath.Dir(filepath.Dir(f))
		op := filepath.Base(opDir)
		pair, ok := compiled[opDir]
		if !ok {
			rel := func(name string) string {
				r, err := filepath.Rel(pluginDir, filepath.Join(opDir, name))
				require.NoError(t, err)
				return filepath.ToSlash(r)
			}
			in, _, err := compileSchemaFile(pluginDir, rel("input.schema.json"))
			require.NoError(t, err, "%s input schema", op)
			out, _, err := compileSchemaFile(pluginDir, rel("output.schema.json"))
			require.NoError(t, err, "%s output schema", op)
			pair = [2]*jsonschema.Schema{in, out}
			compiled[opDir] = pair
		}
		inSch, outSch := pair[0], pair[1]

		name := op + "/" + filepath.Base(f)
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(f)
			require.NoError(t, err)
			var fx kataFixture
			require.NoError(t, json.Unmarshal(raw, &fx))

			decode := func(b json.RawMessage) interface{} {
				v, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
				require.NoError(t, err)
				return v
			}
			require.NoError(t, inSch.Validate(decode(fx.Input)), "input must validate")

			valid := decode(fx.ValidOutput)
			require.NoError(t, outSch.Validate(valid), "valid_output must pass schema")
			require.Empty(t, lintViolations(rules, valid), "valid_output must pass lint")

			for i, inv := range fx.InvalidOutputs {
				v := decode(inv.Output)
				schemaErr := outSch.Validate(v)
				switch {
				case strings.HasPrefix(inv.Why, "schema:"):
					require.Error(t, schemaErr, "invalid_outputs[%d] (%s) must fail schema", i, inv.Why)
				case strings.HasPrefix(inv.Why, "lint:"):
					require.NoError(t, schemaErr, "invalid_outputs[%d] (%s) must pass schema", i, inv.Why)
					require.NotEmpty(t, lintViolations(rules, v), "invalid_outputs[%d] (%s) must fail lint", i, inv.Why)
				default:
					t.Fatalf("invalid_outputs[%d]: why %q must start with \"schema:\" or \"lint:\"", i, inv.Why)
				}
			}
		})
	}
	return len(files)
}

func TestKataContractFixtures(t *testing.T) {
	pluginDir := os.Getenv(envKataPluginDir)
	if pluginDir == "" {
		pluginDir = filepath.Join("..", "plugins", "kata")
	}
	if _, err := os.Stat(filepath.Join(pluginDir, "contracts", "kata-ai", "v1")); err != nil {
		t.Skipf("kata-ai/v1 contract not vendored at %s", pluginDir)
	}
	if n := runKataContractFixtures(t, pluginDir); n == 0 {
		t.Fatalf("contract present at %s but no fixtures found", pluginDir)
	}
}

func TestKataContractFixtures_Testdata(t *testing.T) {
	require.Equal(t, 1, runKataContractFixtures(t, testKataDir))
}
