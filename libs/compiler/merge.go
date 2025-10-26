package compiler

// DeepMerge performs a deep merge of two maps following Nomos composition semantics:
// - Maps are deep-merged recursively
// - Arrays are replaced (no deep-array merge)
// - Scalars follow last-wins policy
// - The function does not mutate input maps; it returns a new merged map
func DeepMerge(dst, src map[string]any) map[string]any {
	// Create a new result map to avoid mutating inputs
	result := make(map[string]any, len(dst)+len(src))

	// Copy all entries from dst
	for k, v := range dst {
		result[k] = deepCopyValue(v)
	}

	// Merge entries from src
	for k, srcVal := range src {
		dstVal, existsInDst := result[k]

		// If key doesn't exist in dst, just copy from src
		if !existsInDst {
			result[k] = deepCopyValue(srcVal)
			continue
		}

		// Both dst and src have this key - apply merge rules
		result[k] = mergeValues(dstVal, srcVal)
	}

	return result
}

// DeepMergeWithProvenance performs a deep merge with provenance tracking.
// It records the source file for each top-level key in the provenance map.
// The provenance map is updated in-place to record origins.
func DeepMergeWithProvenance(dst map[string]any, dstSource string, src map[string]any, srcSource string, provenance map[string]Provenance) map[string]any {
	// Create a new result map to avoid mutating inputs
	result := make(map[string]any, len(dst)+len(src))

	// Copy all entries from dst and record provenance
	for k, v := range dst {
		result[k] = deepCopyValue(v)
		// Record dst source for this key
		provenance[k] = Provenance{Source: dstSource}
	}

	// Merge entries from src and update provenance
	for k, srcVal := range src {
		dstVal, existsInDst := result[k]

		// Record src as the source for this key (overwrites dst provenance if key existed)
		provenance[k] = Provenance{Source: srcSource}

		// If key doesn't exist in dst, just copy from src
		if !existsInDst {
			result[k] = deepCopyValue(srcVal)
			continue
		}

		// Both dst and src have this key - apply merge rules
		result[k] = mergeValues(dstVal, srcVal)
	}

	return result
}

// mergeValues merges two values according to composition semantics.
func mergeValues(dst, src any) any {
	// If src is nil, it overwrites dst (last-wins)
	if src == nil {
		return nil
	}

	// If dst is nil, src wins
	if dst == nil {
		return deepCopyValue(src)
	}

	// Check if both are maps - if so, deep merge
	dstMap, dstIsMap := dst.(map[string]any)
	srcMap, srcIsMap := src.(map[string]any)

	if dstIsMap && srcIsMap {
		// Both are maps - recursively deep merge
		return DeepMerge(dstMap, srcMap)
	}

	// For all other cases (scalars, arrays, type mismatches):
	// src wins (last-wins / replacement)
	return deepCopyValue(src)
}

// deepCopyValue creates a deep copy of a value to prevent mutation.
func deepCopyValue(val any) any {
	if val == nil {
		return nil
	}

	switch v := val.(type) {
	case map[string]any:
		// Deep copy map
		copied := make(map[string]any, len(v))
		for k, mapVal := range v {
			copied[k] = deepCopyValue(mapVal)
		}
		return copied

	case []any:
		// Deep copy array
		copied := make([]any, len(v))
		for i, arrayVal := range v {
			copied[i] = deepCopyValue(arrayVal)
		}
		return copied

	default:
		// Primitives (string, int, bool, etc.) are immutable in Go
		// so no deep copy needed
		return v
	}
}
