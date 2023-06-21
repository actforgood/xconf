// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf_test

import (
	"testing"

	"github.com/actforgood/xconf"
)

func TestDeepCopyConfigMap(t *testing.T) {
	t.Parallel()

	// arrange
	var (
		input = map[string]any{
			"string":          "a string",
			"int":             1234,
			"float":           1234.56,
			"slice_interface": []any{"a", "b", "c"},
			"slice_interface_deep": []any{
				[]any{"x", "y", "z"},
				[]string{"x", "y", "z"},
				[]int{1, 2, 3},
				map[string]any{"foo": "bar"},
				map[any]any{"foo": "bar"},
			},
			"slice_int":    []int{1, 2, 3},
			"slice_string": []string{"a", "b", "c"},
			"map_string": map[string]any{
				"x": "X",
				"y": "Y",
				"z": "Z",
			},
			"map_string_deep": map[string]any{
				"slice": []any{"foo", "bar", "baz"},
				"map": map[string]any{
					"en": "Hello",
					"es": "Ola",
				},
			},
			"map_interface": map[any]any{
				"x": "X",
				"y": "Y",
				"z": "Z",
			},
			"map_interface_deep": map[any]any{
				"slice_interface": []any{"foo", "bar", "baz"},
				"slice_string":    []string{"foo", "bar", "baz"},
				"slice_int":       []int{1, 2, 3},
				"map": map[any]any{
					"en": "Hello",
					"es": "Ola",
				},
			},
		}
		subject = xconf.DeepCopyConfigMap
	)

	// act
	result := subject(input)

	// assert
	if assertEqual(t, input, result) { // apply some changes and see original config map does not get modified
		result["string"] = "a string modified"
		assertEqual(t, "a string", input["string"])

		result["int"] = 9876
		assertEqual(t, 1234, input["int"])

		result["float"] = 9876.54
		assertEqual(t, 1234.56, input["float"])

		result["slice_interface"].([]any)[0] = "aaa"
		assertEqual(t, "a", input["slice_interface"].([]any)[0])

		result["slice_interface_deep"].([]any)[0].([]any)[2] = "zzz"
		assertEqual(t, "z", input["slice_interface_deep"].([]any)[0].([]any)[2])
		result["slice_interface_deep"].([]any)[1].([]string)[2] = "zzz"
		assertEqual(t, "z", input["slice_interface_deep"].([]any)[1].([]string)[2])
		result["slice_interface_deep"].([]any)[2].([]int)[2] = 333
		assertEqual(t, 3, input["slice_interface_deep"].([]any)[2].([]int)[2])
		result["slice_interface_deep"].([]any)[3].(map[string]any)["foo"] = "B_A_R"
		assertEqual(t, "bar", input["slice_interface_deep"].([]any)[3].(map[string]any)["foo"])
		result["slice_interface_deep"].([]any)[4].(map[any]any)["foo"] = "B_A_R"
		assertEqual(t, "bar", input["slice_interface_deep"].([]any)[4].(map[any]any)["foo"])

		result["slice_int"].([]int)[0] = 111
		assertEqual(t, 1, input["slice_int"].([]int)[0])

		result["slice_string"].([]string)[0] = "aaa"
		assertEqual(t, "a", input["slice_string"].([]string)[0])

		result["map_string"].(map[string]any)["z"] = "ZZZ"
		assertEqual(t, "Z", input["map_string"].(map[string]any)["z"])

		result["map_string_deep"].(map[string]any)["slice"].([]any)[0] = "F_O_O"
		assertEqual(t, "foo", input["map_string_deep"].(map[string]any)["slice"].([]any)[0])
		result["map_string_deep"].(map[string]any)["map"].(map[string]any)["en"] = "Hi"
		assertEqual(t, "Hello", input["map_string_deep"].(map[string]any)["map"].(map[string]any)["en"])

		result["map_interface"].(map[any]any)["z"] = "ZZZ"
		assertEqual(t, "Z", input["map_interface"].(map[any]any)["z"])

		result["map_interface_deep"].(map[any]any)["slice_interface"].([]any)[0] = "FoO"
		assertEqual(t, "foo", input["map_interface_deep"].(map[any]any)["slice_interface"].([]any)[0])
		result["map_interface_deep"].(map[any]any)["slice_string"].([]string)[0] = "fOO"
		assertEqual(t, "foo", input["map_interface_deep"].(map[any]any)["slice_string"].([]string)[0])
		result["map_interface_deep"].(map[any]any)["slice_int"].([]int)[0] = 111
		assertEqual(t, 1, input["map_interface_deep"].(map[any]any)["slice_int"].([]int)[0])
		result["map_interface_deep"].(map[any]any)["map"].(map[any]any)["en"] = "Hi"
		assertEqual(t,
			"Hello",
			input["map_interface_deep"].(map[any]any)["map"].(map[any]any)["en"],
		)
	}
}

func BenchmarkDeepCopyConfigMap(b *testing.B) {
	input := map[string]any{
		"foo":           "bar",
		"year":          2022,
		"temperature":   37.5,
		"shopping_list": []any{"bread", "milk", "eggs"},
		"timeouts":      []any{10, 15, 20},
	}
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_ = xconf.DeepCopyConfigMap(input)
	}
}
