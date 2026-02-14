package tools

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetStringParam(t *testing.T) {
	tests := []struct {
		name         string
		params       map[string]interface{}
		key          string
		defaultValue string
		expected     string
	}{
		{"existing value", map[string]interface{}{"key": "value"}, "key", "default", "value"},
		{"missing value", map[string]interface{}{}, "key", "default", "default"},
		{"wrong type", map[string]interface{}{"key": 123}, "key", "default", "default"},
		{"nil params", nil, "key", "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetStringParam(tt.params, tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetIntParam(t *testing.T) {
	tests := []struct {
		name         string
		params       map[string]interface{}
		key          string
		defaultValue int
		expected     int
	}{
		{"int value", map[string]interface{}{"key": 42}, "key", 0, 42},
		{"int64 value", map[string]interface{}{"key": int64(42)}, "key", 0, 42},
		{"float64 value", map[string]interface{}{"key": 42.0}, "key", 0, 42},
		{"missing value", map[string]interface{}{}, "key", 10, 10},
		{"wrong type", map[string]interface{}{"key": "string"}, "key", 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetIntParam(tt.params, tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetBoolParam(t *testing.T) {
	tests := []struct {
		name         string
		params       map[string]interface{}
		key          string
		defaultValue bool
		expected     bool
	}{
		{"true value", map[string]interface{}{"key": true}, "key", false, true},
		{"false value", map[string]interface{}{"key": false}, "key", true, false},
		{"missing value", map[string]interface{}{}, "key", true, true},
		{"wrong type", map[string]interface{}{"key": "true"}, "key", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBoolParam(tt.params, tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFloat64Param(t *testing.T) {
	tests := []struct {
		name         string
		params       map[string]interface{}
		key          string
		defaultValue float64
		expected     float64
	}{
		{"float64 value", map[string]interface{}{"key": 3.14}, "key", 0.0, 3.14},
		{"float32 value", map[string]interface{}{"key": float32(3.14)}, "key", 0.0, 3.14},
		{"int value", map[string]interface{}{"key": 42}, "key", 0.0, 42.0},
		{"int64 value", map[string]interface{}{"key": int64(42)}, "key", 0.0, 42.0},
		{"missing value", map[string]interface{}{}, "key", 1.5, 1.5},
		{"wrong type", map[string]interface{}{"key": "string"}, "key", 1.5, 1.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetFloat64Param(tt.params, tt.key, tt.defaultValue)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

func TestGetStringSliceParam(t *testing.T) {
	tests := []struct {
		name         string
		params       map[string]interface{}
		key          string
		defaultValue []string
		expected     []string
	}{
		{
			"interface slice",
			map[string]interface{}{"key": []interface{}{"a", "b", "c"}},
			"key",
			[]string{},
			[]string{"a", "b", "c"},
		},
		{
			"string slice",
			map[string]interface{}{"key": []string{"x", "y", "z"}},
			"key",
			[]string{},
			[]string{"x", "y", "z"},
		},
		{
			"missing value",
			map[string]interface{}{},
			"key",
			[]string{"default"},
			[]string{"default"},
		},
		{
			"wrong type",
			map[string]interface{}{"key": "string"},
			"key",
			[]string{"default"},
			[]string{"default"},
		},
		{
			"mixed types in slice",
			map[string]interface{}{"key": []interface{}{"a", 123, "b"}},
			"key",
			[]string{},
			[]string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetStringSliceParam(tt.params, tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetMapParam(t *testing.T) {
	tests := []struct {
		name         string
		params       map[string]interface{}
		key          string
		defaultValue map[string]interface{}
		expected     map[string]interface{}
	}{
		{
			"existing map",
			map[string]interface{}{"key": map[string]interface{}{"nested": "value"}},
			"key",
			map[string]interface{}{},
			map[string]interface{}{"nested": "value"},
		},
		{
			"missing value",
			map[string]interface{}{},
			"key",
			map[string]interface{}{"default": "value"},
			map[string]interface{}{"default": "value"},
		},
		{
			"wrong type",
			map[string]interface{}{"key": "string"},
			"key",
			map[string]interface{}{"default": "value"},
			map[string]interface{}{"default": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMapParam(tt.params, tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDurationParam(t *testing.T) {
	tests := []struct {
		name         string
		params       map[string]interface{}
		key          string
		defaultValue time.Duration
		expected     time.Duration
	}{
		{"int seconds", map[string]interface{}{"key": 30}, "key", 0, 30 * time.Second},
		{"int64 seconds", map[string]interface{}{"key": int64(60)}, "key", 0, 60 * time.Second},
		{"float64 seconds", map[string]interface{}{"key": 45.0}, "key", 0, 45 * time.Second},
		{"string duration", map[string]interface{}{"key": "2m"}, "key", 0, 2 * time.Minute},
		{"missing value", map[string]interface{}{}, "key", 10 * time.Second, 10 * time.Second},
		{"invalid string", map[string]interface{}{"key": "invalid"}, "key", 5 * time.Second, 5 * time.Second},
		{"wrong type", map[string]interface{}{"key": true}, "key", 5 * time.Second, 5 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDurationParam(tt.params, tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}
