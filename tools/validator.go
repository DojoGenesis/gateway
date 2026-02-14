package tools

import (
	"fmt"
	"reflect"
)

func ValidateParameters(params map[string]interface{}, schema map[string]interface{}) error {
	if schema == nil {
		return nil
	}

	schemaType, ok := schema["type"].(string)
	if !ok || schemaType != "object" {
		return nil
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return nil
	}

	required, _ := schema["required"].([]interface{})
	requiredFields := make(map[string]bool)
	for _, field := range required {
		if fieldName, ok := field.(string); ok {
			requiredFields[fieldName] = true
		}
	}

	for fieldName := range requiredFields {
		if _, exists := params[fieldName]; !exists {
			return fmt.Errorf("required parameter missing: %s", fieldName)
		}
	}

	for paramName, paramValue := range params {
		propSchema, exists := properties[paramName]
		if !exists {
			continue
		}

		propMap, ok := propSchema.(map[string]interface{})
		if !ok {
			continue
		}

		expectedType, ok := propMap["type"].(string)
		if !ok {
			continue
		}

		if err := validateType(paramName, paramValue, expectedType); err != nil {
			return err
		}
	}

	return nil
}

func validateType(paramName string, value interface{}, expectedType string) error {
	if value == nil {
		return nil
	}

	actualType := getJSONType(value)

	if expectedType == "integer" && actualType == "number" {
		if _, ok := value.(float64); ok {
			if float64(int(value.(float64))) == value.(float64) {
				return nil
			}
		}
	}

	if actualType != expectedType {
		return fmt.Errorf("parameter %s: expected type %s, got %s", paramName, expectedType, actualType)
	}

	return nil
}

func getJSONType(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch reflect.TypeOf(value).Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.String:
		return "string"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	default:
		return "unknown"
	}
}
