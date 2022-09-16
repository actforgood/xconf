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
		input = map[string]interface{}{
			"string":          "a string",
			"int":             1234,
			"float":           1234.56,
			"slice_interface": []interface{}{"a", "b", "c"},
			"slice_interface_deep": []interface{}{
				[]interface{}{"x", "y", "z"},
				[]string{"x", "y", "z"},
				[]int{1, 2, 3},
				map[string]interface{}{"foo": "bar"},
				map[interface{}]interface{}{"foo": "bar"},
			},
			"slice_int":    []int{1, 2, 3},
			"slice_string": []string{"a", "b", "c"},
			"map_string": map[string]interface{}{
				"x": "X",
				"y": "Y",
				"z": "Z",
			},
			"map_string_deep": map[string]interface{}{
				"slice": []interface{}{"foo", "bar", "baz"},
				"map": map[string]interface{}{
					"en": "Hello",
					"es": "Ola",
				},
			},
			"map_interface": map[interface{}]interface{}{
				"x": "X",
				"y": "Y",
				"z": "Z",
			},
			"map_interface_deep": map[interface{}]interface{}{
				"slice_interface": []interface{}{"foo", "bar", "baz"},
				"slice_string":    []string{"foo", "bar", "baz"},
				"slice_int":       []int{1, 2, 3},
				"map": map[interface{}]interface{}{
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

		result["slice_interface"].([]interface{})[0] = "aaa"
		assertEqual(t, "a", input["slice_interface"].([]interface{})[0])

		result["slice_interface_deep"].([]interface{})[0].([]interface{})[2] = "zzz"
		assertEqual(t, "z", input["slice_interface_deep"].([]interface{})[0].([]interface{})[2])
		result["slice_interface_deep"].([]interface{})[1].([]string)[2] = "zzz"
		assertEqual(t, "z", input["slice_interface_deep"].([]interface{})[1].([]string)[2])
		result["slice_interface_deep"].([]interface{})[2].([]int)[2] = 333
		assertEqual(t, 3, input["slice_interface_deep"].([]interface{})[2].([]int)[2])
		result["slice_interface_deep"].([]interface{})[3].(map[string]interface{})["foo"] = "B_A_R"
		assertEqual(t, "bar", input["slice_interface_deep"].([]interface{})[3].(map[string]interface{})["foo"])
		result["slice_interface_deep"].([]interface{})[4].(map[interface{}]interface{})["foo"] = "B_A_R"
		assertEqual(t, "bar", input["slice_interface_deep"].([]interface{})[4].(map[interface{}]interface{})["foo"])

		result["slice_int"].([]int)[0] = 111
		assertEqual(t, 1, input["slice_int"].([]int)[0])

		result["slice_string"].([]string)[0] = "aaa"
		assertEqual(t, "a", input["slice_string"].([]string)[0])

		result["map_string"].(map[string]interface{})["z"] = "ZZZ"
		assertEqual(t, "Z", input["map_string"].(map[string]interface{})["z"])

		result["map_string_deep"].(map[string]interface{})["slice"].([]interface{})[0] = "F_O_O"
		assertEqual(t, "foo", input["map_string_deep"].(map[string]interface{})["slice"].([]interface{})[0])
		result["map_string_deep"].(map[string]interface{})["map"].(map[string]interface{})["en"] = "Hi"
		assertEqual(t, "Hello", input["map_string_deep"].(map[string]interface{})["map"].(map[string]interface{})["en"])

		result["map_interface"].(map[interface{}]interface{})["z"] = "ZZZ"
		assertEqual(t, "Z", input["map_interface"].(map[interface{}]interface{})["z"])

		result["map_interface_deep"].(map[interface{}]interface{})["slice_interface"].([]interface{})[0] = "FoO"
		assertEqual(t, "foo", input["map_interface_deep"].(map[interface{}]interface{})["slice_interface"].([]interface{})[0])
		result["map_interface_deep"].(map[interface{}]interface{})["slice_string"].([]string)[0] = "fOO"
		assertEqual(t, "foo", input["map_interface_deep"].(map[interface{}]interface{})["slice_string"].([]string)[0])
		result["map_interface_deep"].(map[interface{}]interface{})["slice_int"].([]int)[0] = 111
		assertEqual(t, 1, input["map_interface_deep"].(map[interface{}]interface{})["slice_int"].([]int)[0])
		result["map_interface_deep"].(map[interface{}]interface{})["map"].(map[interface{}]interface{})["en"] = "Hi"
		assertEqual(t, "Hello", input["map_interface_deep"].(map[interface{}]interface{})["map"].(map[interface{}]interface{})["en"])
	}
}

func BenchmarkDeepCopyConfigMap(b *testing.B) {
	input := map[string]interface{}{
		"foo":           "bar",
		"year":          2022,
		"temperature":   37.5,
		"shopping_list": []interface{}{"bread", "milk", "eggs"},
		"timeouts":      []interface{}{10, 15, 20},
	}
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_ = xconf.DeepCopyConfigMap(input)
	}
}
