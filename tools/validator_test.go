package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateParameters(t *testing.T) {
	t.Run("nil schema", func(t *testing.T) {
		err := ValidateParameters(map[string]interface{}{"key": "value"}, nil)
		assert.NoError(t, err)
	})

	t.Run("empty schema", func(t *testing.T) {
		err := ValidateParameters(map[string]interface{}{"key": "value"}, map[string]interface{}{})
		assert.NoError(t, err)
	})

	t.Run("valid string parameter", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type": "string",
				},
			},
		}

		params := map[string]interface{}{
			"name": "test",
		}

		err := ValidateParameters(params, schema)
		assert.NoError(t, err)
	})

	t.Run("valid number parameter", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"age": map[string]interface{}{
					"type": "number",
				},
			},
		}

		params := map[string]interface{}{
			"age": 25.5,
		}

		err := ValidateParameters(params, schema)
		assert.NoError(t, err)
	})

	t.Run("valid integer parameter", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"count": map[string]interface{}{
					"type": "integer",
				},
			},
		}

		params := map[string]interface{}{
			"count": 42,
		}

		err := ValidateParameters(params, schema)
		assert.NoError(t, err)
	})

	t.Run("valid boolean parameter", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"enabled": map[string]interface{}{
					"type": "boolean",
				},
			},
		}

		params := map[string]interface{}{
			"enabled": true,
		}

		err := ValidateParameters(params, schema)
		assert.NoError(t, err)
	})

	t.Run("valid array parameter", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"items": map[string]interface{}{
					"type": "array",
				},
			},
		}

		params := map[string]interface{}{
			"items": []string{"a", "b", "c"},
		}

		err := ValidateParameters(params, schema)
		assert.NoError(t, err)
	})

	t.Run("valid object parameter", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"metadata": map[string]interface{}{
					"type": "object",
				},
			},
		}

		params := map[string]interface{}{
			"metadata": map[string]interface{}{
				"key": "value",
			},
		}

		err := ValidateParameters(params, schema)
		assert.NoError(t, err)
	})

	t.Run("required parameter missing", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type": "string",
				},
			},
			"required": []interface{}{"name"},
		}

		params := map[string]interface{}{}

		err := ValidateParameters(params, schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required parameter missing")
	})

	t.Run("type mismatch - string expected number", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"age": map[string]interface{}{
					"type": "number",
				},
			},
		}

		params := map[string]interface{}{
			"age": "twenty",
		}

		err := ValidateParameters(params, schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected type number")
	})

	t.Run("type mismatch - number expected string", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type": "string",
				},
			},
		}

		params := map[string]interface{}{
			"name": 123,
		}

		err := ValidateParameters(params, schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected type string")
	})

	t.Run("nil value is allowed", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"optional": map[string]interface{}{
					"type": "string",
				},
			},
		}

		params := map[string]interface{}{
			"optional": nil,
		}

		err := ValidateParameters(params, schema)
		assert.NoError(t, err)
	})

	t.Run("extra parameters allowed", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type": "string",
				},
			},
		}

		params := map[string]interface{}{
			"name":  "test",
			"extra": "ignored",
		}

		err := ValidateParameters(params, schema)
		assert.NoError(t, err)
	})

	t.Run("multiple required parameters", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type": "string",
				},
				"age": map[string]interface{}{
					"type": "number",
				},
			},
			"required": []interface{}{"name", "age"},
		}

		params := map[string]interface{}{
			"name": "test",
		}

		err := ValidateParameters(params, schema)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required parameter missing: age")
	})

	t.Run("float as integer accepted", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"count": map[string]interface{}{
					"type": "integer",
				},
			},
		}

		params := map[string]interface{}{
			"count": 42.0,
		}

		err := ValidateParameters(params, schema)
		assert.NoError(t, err)
	})

	t.Run("float with decimal as integer rejected", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"count": map[string]interface{}{
					"type": "integer",
				},
			},
		}

		params := map[string]interface{}{
			"count": 42.5,
		}

		err := ValidateParameters(params, schema)
		assert.Error(t, err)
	})
}

func TestGetJSONType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"nil", nil, "null"},
		{"bool", true, "boolean"},
		{"int", 42, "integer"},
		{"int8", int8(42), "integer"},
		{"int16", int16(42), "integer"},
		{"int32", int32(42), "integer"},
		{"int64", int64(42), "integer"},
		{"uint", uint(42), "integer"},
		{"uint8", uint8(42), "integer"},
		{"uint16", uint16(42), "integer"},
		{"uint32", uint32(42), "integer"},
		{"uint64", uint64(42), "integer"},
		{"float32", float32(3.14), "number"},
		{"float64", float64(3.14), "number"},
		{"string", "hello", "string"},
		{"slice", []int{1, 2, 3}, "array"},
		{"array", [3]int{1, 2, 3}, "array"},
		{"map", map[string]string{"key": "value"}, "object"},
		{"struct", struct{ Name string }{Name: "test"}, "object"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getJSONType(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}
