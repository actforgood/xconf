// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf

// DeepCopyConfigMap is a utility function to make a deep "copy"/clone of a config map.
func DeepCopyConfigMap(src map[string]any) map[string]any {
	// Note: Implementation is opinionated to basic types/types produced by current loaders/decoders.
	// In json you can have as value a nested structure which ends up being a map[string]any.
	// In yaml you can have as value a nested structure which ends up being either a map[string]any,
	// or map[any]any.
	// In json and yaml array-values end up being []any.
	// Otherwise (env/properties/ini) values resume to strings.
	// The PlainLoader is more flexible (nothing stops you from assigning to a key a pointer to a struct for example
	// - but it's your call if you do that).
	//
	// A general solution can be implemented with gob encoder/decoder, but the
	// results were not satisfying. For cached loaders, for example, in some cases,
	// benchmarks were actually worse than not having the cache in the first place because
	// of gob based deep copy strategy.
	dst := make(map[string]any, len(src))

	for key, value := range src {
		switch val := value.(type) {
		case []any:
			dst[key] = deepCopyInterfaceSlice(val)
		case []string:
			sliceCopy := make([]string, len(val))
			copy(sliceCopy, val)
			dst[key] = sliceCopy
		case []int:
			sliceCopy := make([]int, len(val))
			copy(sliceCopy, val)
			dst[key] = sliceCopy
		case map[string]any:
			dst[key] = DeepCopyConfigMap(val)
		case map[any]any:
			dst[key] = deepCopyInterfaceMap(val)
		default:
			dst[key] = value
		}
	}

	return dst
}

// deepCopyInterfaceMap makes a deep "copy" of a map[any]any.
// This kind of map is produced by YAML decoder.
func deepCopyInterfaceMap(src map[any]any) map[any]any {
	dst := make(map[any]any, len(src))

	for key, value := range src {
		switch val := value.(type) {
		case []any:
			dst[key] = deepCopyInterfaceSlice(val)
		case []string:
			sliceCopy := make([]string, len(val))
			copy(sliceCopy, val)
			dst[key] = sliceCopy
		case []int:
			sliceCopy := make([]int, len(val))
			copy(sliceCopy, val)
			dst[key] = sliceCopy
		case map[string]any:
			dst[key] = DeepCopyConfigMap(val)
		case map[any]any:
			dst[key] = deepCopyInterfaceMap(val)
		default:
			dst[key] = value
		}
	}

	return dst
}

// deepCopyInterfaceSlice makes a deep "copy" of a []any.
func deepCopyInterfaceSlice(src []any) []any {
	dst := make([]any, len(src))

	for key, value := range src {
		switch val := value.(type) {
		case []any:
			dst[key] = deepCopyInterfaceSlice(val)
		case []string:
			sliceCopy := make([]string, len(val))
			copy(sliceCopy, val)
			dst[key] = sliceCopy
		case []int:
			sliceCopy := make([]int, len(val))
			copy(sliceCopy, val)
			dst[key] = sliceCopy
		case map[string]any:
			dst[key] = DeepCopyConfigMap(val)
		case map[any]any:
			dst[key] = deepCopyInterfaceMap(val)
		default:
			dst[key] = value
		}
	}

	return dst
}
