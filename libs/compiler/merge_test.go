package compiler

import (
	"reflect"
	"testing"
)

// TestDeepMerge_ScalarOverrides tests that scalar values follow last-wins semantics.
func TestDeepMerge_ScalarOverrides(t *testing.T) {
	tests := []struct {
		name     string
		dst      map[string]any
		src      map[string]any
		expected map[string]any
	}{
		{
			name:     "override string",
			dst:      map[string]any{"key": "original"},
			src:      map[string]any{"key": "updated"},
			expected: map[string]any{"key": "updated"},
		},
		{
			name:     "override int",
			dst:      map[string]any{"count": 1},
			src:      map[string]any{"count": 2},
			expected: map[string]any{"count": 2},
		},
		{
			name:     "override bool",
			dst:      map[string]any{"enabled": false},
			src:      map[string]any{"enabled": true},
			expected: map[string]any{"enabled": true},
		},
		{
			name:     "nil overwrites value",
			dst:      map[string]any{"key": "value"},
			src:      map[string]any{"key": nil},
			expected: map[string]any{"key": nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeepMerge(tt.dst, tt.src)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("DeepMerge() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestDeepMerge_ArrayReplacement tests that arrays are replaced, not merged.
func TestDeepMerge_ArrayReplacement(t *testing.T) {
	tests := []struct {
		name     string
		dst      map[string]any
		src      map[string]any
		expected map[string]any
	}{
		{
			name:     "replace array completely",
			dst:      map[string]any{"items": []any{1, 2, 3}},
			src:      map[string]any{"items": []any{4, 5}},
			expected: map[string]any{"items": []any{4, 5}},
		},
		{
			name:     "replace empty array",
			dst:      map[string]any{"items": []any{}},
			src:      map[string]any{"items": []any{1, 2}},
			expected: map[string]any{"items": []any{1, 2}},
		},
		{
			name:     "replace array with empty array",
			dst:      map[string]any{"items": []any{1, 2}},
			src:      map[string]any{"items": []any{}},
			expected: map[string]any{"items": []any{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeepMerge(tt.dst, tt.src)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("DeepMerge() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestDeepMerge_MapDeepMerge tests that maps are deep-merged.
func TestDeepMerge_MapDeepMerge(t *testing.T) {
	tests := []struct {
		name     string
		dst      map[string]any
		src      map[string]any
		expected map[string]any
	}{
		{
			name: "merge two simple maps",
			dst:  map[string]any{"a": 1},
			src:  map[string]any{"b": 2},
			expected: map[string]any{
				"a": 1,
				"b": 2,
			},
		},
		{
			name: "merge nested maps",
			dst: map[string]any{
				"config": map[string]any{
					"port": 8080,
					"host": "localhost",
				},
			},
			src: map[string]any{
				"config": map[string]any{
					"port": 9000,
					"tls":  true,
				},
			},
			expected: map[string]any{
				"config": map[string]any{
					"port": 9000,
					"host": "localhost",
					"tls":  true,
				},
			},
		},
		{
			name: "deep nested maps",
			dst: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"level3": map[string]any{
							"value": "original",
						},
					},
				},
			},
			src: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"level3": map[string]any{
							"value":   "updated",
							"newprop": "added",
						},
					},
				},
			},
			expected: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"level3": map[string]any{
							"value":   "updated",
							"newprop": "added",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeepMerge(tt.dst, tt.src)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("DeepMerge() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestDeepMerge_MixedTypes tests behavior when types conflict.
func TestDeepMerge_MixedTypes(t *testing.T) {
	tests := []struct {
		name     string
		dst      map[string]any
		src      map[string]any
		expected map[string]any
	}{
		{
			name:     "scalar replaces map",
			dst:      map[string]any{"key": map[string]any{"nested": "value"}},
			src:      map[string]any{"key": "scalar"},
			expected: map[string]any{"key": "scalar"},
		},
		{
			name:     "map replaces scalar",
			dst:      map[string]any{"key": "scalar"},
			src:      map[string]any{"key": map[string]any{"nested": "value"}},
			expected: map[string]any{"key": map[string]any{"nested": "value"}},
		},
		{
			name:     "array replaces map",
			dst:      map[string]any{"key": map[string]any{"nested": "value"}},
			src:      map[string]any{"key": []any{1, 2}},
			expected: map[string]any{"key": []any{1, 2}},
		},
		{
			name:     "map replaces array",
			dst:      map[string]any{"key": []any{1, 2}},
			src:      map[string]any{"key": map[string]any{"nested": "value"}},
			expected: map[string]any{"key": map[string]any{"nested": "value"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeepMerge(tt.dst, tt.src)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("DeepMerge() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestDeepMerge_EdgeCases tests edge cases like empty maps and nil values.
func TestDeepMerge_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		dst      map[string]any
		src      map[string]any
		expected map[string]any
	}{
		{
			name:     "empty dst",
			dst:      map[string]any{},
			src:      map[string]any{"key": "value"},
			expected: map[string]any{"key": "value"},
		},
		{
			name:     "empty src",
			dst:      map[string]any{"key": "value"},
			src:      map[string]any{},
			expected: map[string]any{"key": "value"},
		},
		{
			name:     "both empty",
			dst:      map[string]any{},
			src:      map[string]any{},
			expected: map[string]any{},
		},
		{
			name: "multiple keys",
			dst: map[string]any{
				"a": 1,
				"b": 2,
				"c": 3,
			},
			src: map[string]any{
				"b": 20,
				"d": 4,
			},
			expected: map[string]any{
				"a": 1,
				"b": 20,
				"c": 3,
				"d": 4,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeepMerge(tt.dst, tt.src)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("DeepMerge() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestDeepMerge_NonMutating tests that DeepMerge does not mutate input maps.
func TestDeepMerge_NonMutating(t *testing.T) {
	dst := map[string]any{"key": "original"}
	src := map[string]any{"key": "updated"}

	// Deep copy for comparison
	dstCopy := map[string]any{"key": "original"}
	srcCopy := map[string]any{"key": "updated"}

	result := DeepMerge(dst, src)

	// Verify inputs weren't mutated
	if !reflect.DeepEqual(dst, dstCopy) {
		t.Errorf("DeepMerge mutated dst: got %v, want %v", dst, dstCopy)
	}
	if !reflect.DeepEqual(src, srcCopy) {
		t.Errorf("DeepMerge mutated src: got %v, want %v", src, srcCopy)
	}

	// Verify result is correct
	expected := map[string]any{"key": "updated"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("DeepMerge() = %v, want %v", result, expected)
	}
}

// TestDeepMergeWithProvenance_BasicTracking tests provenance recording for overwritten keys.
func TestDeepMergeWithProvenance_BasicTracking(t *testing.T) {
	tests := []struct {
		name               string
		dst                map[string]any
		dstSource          string
		src                map[string]any
		srcSource          string
		expectedData       map[string]any
		expectedProvenance map[string]Provenance
	}{
		{
			name:      "track scalar override",
			dst:       map[string]any{"key": "original"},
			dstSource: "file1.csl",
			src:       map[string]any{"key": "updated"},
			srcSource: "file2.csl",
			expectedData: map[string]any{
				"key": "updated",
			},
			expectedProvenance: map[string]Provenance{
				"key": {Source: "file2.csl"},
			},
		},
		{
			name:      "track multiple keys",
			dst:       map[string]any{"a": 1, "b": 2},
			dstSource: "base.csl",
			src:       map[string]any{"b": 20, "c": 3},
			srcSource: "override.csl",
			expectedData: map[string]any{
				"a": 1,
				"b": 20,
				"c": 3,
			},
			expectedProvenance: map[string]Provenance{
				"a": {Source: "base.csl"},
				"b": {Source: "override.csl"},
				"c": {Source: "override.csl"},
			},
		},
		{
			name:      "track deep merge preserves dst provenance for non-overwritten keys",
			dst:       map[string]any{"config": map[string]any{"port": 8080, "host": "localhost"}},
			dstSource: "config.csl",
			src:       map[string]any{"config": map[string]any{"port": 9000}},
			srcSource: "override.csl",
			expectedData: map[string]any{
				"config": map[string]any{
					"port": 9000,
					"host": "localhost",
				},
			},
			expectedProvenance: map[string]Provenance{
				"config": {Source: "override.csl"}, // Last file to touch this top-level key
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provenance := make(map[string]Provenance)
			result := DeepMergeWithProvenance(tt.dst, tt.dstSource, tt.src, tt.srcSource, provenance)

			if !reflect.DeepEqual(result, tt.expectedData) {
				t.Errorf("DeepMergeWithProvenance() data = %v, want %v", result, tt.expectedData)
			}

			if !reflect.DeepEqual(provenance, tt.expectedProvenance) {
				t.Errorf("DeepMergeWithProvenance() provenance = %v, want %v", provenance, tt.expectedProvenance)
			}
		})
	}
}

// TestDeepMergeWithProvenance_ArrayReplacement tests that array replacement updates provenance.
func TestDeepMergeWithProvenance_ArrayReplacement(t *testing.T) {
	dst := map[string]any{"items": []any{1, 2, 3}}
	src := map[string]any{"items": []any{4, 5}}

	provenance := make(map[string]Provenance)
	result := DeepMergeWithProvenance(dst, "base.csl", src, "override.csl", provenance)

	expectedData := map[string]any{"items": []any{4, 5}}
	expectedProvenance := map[string]Provenance{
		"items": {Source: "override.csl"},
	}

	if !reflect.DeepEqual(result, expectedData) {
		t.Errorf("data = %v, want %v", result, expectedData)
	}

	if !reflect.DeepEqual(provenance, expectedProvenance) {
		t.Errorf("provenance = %v, want %v", provenance, expectedProvenance)
	}
}
