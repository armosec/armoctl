package patcher

import (
	"encoding/json"
	"reflect"
	"unicode"

	awsecs "github.com/aws/aws-sdk-go-v2/service/ecs"
)

// TaskDefinition is a type alias for the AWS SDK RegisterTaskDefinitionInput.
// Using the SDK type directly ensures all ECS fields (privileged, tags,
// placement constraints, etc.) are preserved during round-trip serialization
// without maintaining a parallel struct.
type TaskDefinition = awsecs.RegisterTaskDefinitionInput

// PatchOptions holds targeting configuration for patching an ECS task definition.
type PatchOptions struct {
	// Containers is an optional list of container names to patch.
	// If empty, all containers are patched.
	Containers []string

	// VolumeFixer includes a volume-fixer init container that runs
	// chmod/chown on the shared volume. Disabled by default.
	VolumeFixer bool
}

// SidecarConfig holds configuration for the injected sidecar container.
type SidecarConfig struct {
	// Image is the ptrace-agent container image (e.g. quay.io/armosec/ptrace-agent:latest).
	Image string

	// CustomerGUID is the ARMO customer GUID.
	CustomerGUID string

	// AccessKey is the ARMO API access key.
	AccessKey string
}

// UnmarshalTaskDef reads camelCase ECS task definition JSON into the SDK type.
// This works because encoding/json performs case-insensitive struct field matching.
func UnmarshalTaskDef(data []byte) (*TaskDefinition, error) {
	var td TaskDefinition
	if err := json.Unmarshal(data, &td); err != nil {
		return nil, err
	}
	return &td, nil
}

// MarshalTaskDef marshals an SDK task definition type to camelCase JSON
// matching the standard ECS task definition file format.
//
// It uses reflection to walk the typed struct so that struct field names are
// converted from PascalCase to camelCase, while user-defined map keys
// (e.g. DockerLabels, LogConfiguration.Options) are preserved as-is.
func MarshalTaskDef(td *TaskDefinition) ([]byte, error) {
	result := marshalStruct(reflect.ValueOf(td).Elem())
	return json.MarshalIndent(result, "", "  ")
}

// marshalStruct converts a struct value to a map with camelCase keys.
// Only exported, non-zero fields are included.
func marshalStruct(v reflect.Value) map[string]any {
	t := v.Type()
	result := make(map[string]any)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		fv := v.Field(i)
		if shouldOmit(fv) {
			continue
		}

		result[pascalToCamel(field.Name)] = marshalValue(fv)
	}

	return result
}

// shouldOmit returns true if a struct field should be omitted from the output.
// - nil pointers, slices, maps → omitted
// - empty non-pointer strings (enum zero values like PidMode "") → omitted
func shouldOmit(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.IsNil()
	case reflect.String:
		// Non-pointer string fields in the SDK are enum types (PidMode, NetworkMode, etc.).
		// Skip their zero value (empty string). Actual string data uses *string pointers.
		return v.String() == ""
	}
	return false
}

// marshalValue converts a reflect.Value to a JSON-compatible Go value.
// Struct fields get camelCase keys; map keys are preserved as-is.
func marshalValue(v reflect.Value) any {
	// Dereference pointers.
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		return marshalStruct(v)
	case reflect.Slice:
		result := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			result[i] = marshalValue(v.Index(i))
		}
		return result
	case reflect.Map:
		// Map keys are user data (DockerLabels, Options) — preserve as-is.
		result := make(map[string]any, v.Len())
		for _, key := range v.MapKeys() {
			result[key.String()] = marshalValue(v.MapIndex(key))
		}
		return result
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint()
	case reflect.Float32, reflect.Float64:
		return v.Float()
	default:
		return nil
	}
}

// pascalToCamel converts a PascalCase string to camelCase by lowercasing the first rune.
func pascalToCamel(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}
