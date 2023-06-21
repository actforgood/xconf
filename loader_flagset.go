// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf

import (
	"flag"
	"sync/atomic"
)

// FlagSetLoader reduces flags to a configuration map.
// The first parameter is the [flag.FlagSet] holding flags.
// The second, optional, parameter indicates if all flags (even those not explicitly set)
// should be taken into consideration; by default, is true.
func FlagSetLoader(flgSet *flag.FlagSet, visitAll ...bool) Loader {
	all := true
	if len(visitAll) > 0 {
		all = visitAll[0]
	}
	configMap := make(map[string]any)
	storeFlagsIntoMap := func(f *flag.Flag) {
		configMap[f.Name] = f.Value.String()
	}
	var initialized int32

	return LoaderFunc(func() (map[string]any, error) {
		if flgSet.Parsed() && atomic.CompareAndSwapInt32(&initialized, 0, 1) {
			if all {
				flgSet.VisitAll(storeFlagsIntoMap)
			} else {
				flgSet.Visit(storeFlagsIntoMap)
			}
		}

		return DeepCopyConfigMap(configMap), nil // make a copy for an eventual (safe) later mutation.
	})
}
