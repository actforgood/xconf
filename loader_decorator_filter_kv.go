// Copyright The ActForGood Authors.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/blob/main/LICENSE.

package xconf

import "strings"

// FilterType is just an alias for byte.
type FilterType byte

const (
	// FilterTypeWhitelist represents a whitelist filter.
	FilterTypeWhitelist FilterType = 1
	// FilterTypeBlacklist represents a blacklist filter.
	FilterTypeBlacklist FilterType = 2
)

// FilterKV is the contract for a key-value filter.
type FilterKV interface {
	// IsAllowed returns true if a key-value is eligible to be returned
	// in the configuration map.
	IsAllowed(key string, value interface{}) bool

	// Type returns filter's type (FilterTypeWhitelist / FilterTypeBlacklist).
	Type() FilterType
}

// The FilterKVWhitelistFunc type is an adapter to allow the use of
// ordinary functions as FilterKV of "whitelist" type. If fn is a function
// with the appropriate signature, FilterKVWhitelistFunc(fn) is a
// FilterKV that calls fn of type FilterTypeWhitelist.
// fn should return true if the KV is whitelisted.
//
// Example:
//
//	xconf.FilterKVWhitelistFunc(func(key string, _ interface{}) bool {
//		return key == "KEEP_ME_1" || key == "KEEP_ME_2"
//	})
type FilterKVWhitelistFunc func(key string, value interface{}) bool

// IsAllowed returns true if a key-value is whitelisted.
func (filter FilterKVWhitelistFunc) IsAllowed(key string, value interface{}) bool {
	return filter(key, value)
}

// Type returns filter's type (FilterTypeWhitelist).
func (filter FilterKVWhitelistFunc) Type() FilterType {
	return FilterTypeWhitelist
}

// The FilterKVBlacklistFunc type is an adapter to allow the use of
// ordinary functions as FilterKV of "blacklist" type. If fn is a function
// with the appropriate signature, FilterKVBlacklistFunc(fn) is a
// FilterKV that calls fn and has type FilterTypeBlacklist.
// fn should return true if the KV is blacklisted.
//
// Example:
//
//	xconf.FilterKVBlacklistFunc(func(key string, _ interface{}) bool {
//		return key == "DENY_ME_1" || key == "DENY_ME_2"
//	})
type FilterKVBlacklistFunc func(key string, value interface{}) bool

// IsAllowed returns false if a key-value is blacklisted.
func (filter FilterKVBlacklistFunc) IsAllowed(key string, value interface{}) bool {
	return !filter(key, value)
}

// Type returns filter's type (FilterTypeBlacklist).
func (filter FilterKVBlacklistFunc) Type() FilterType {
	return FilterTypeBlacklist
}

// FilterKVLoader decorates another loader to whitelist/blacklist key-values.
//
// A blacklist filter has more weight than a whitelist filter, as if a blacklist denies a KV
// and a whitelist allows it, that KV will not be returned in the configuration map.
//
// If there are only whitelist filters, a KV will be returned into the configuration map
// if at least one filter allows it.
//
// If there are only blacklist filters, a KV will be returned into the configuration map
// if no filter denies it.
func FilterKVLoader(loader Loader, filters ...FilterKV) Loader {
	// make 2 buckets of filters.
	var (
		blacklistFilters = make([]FilterKV, 0, len(filters))
		whitelistFilters = make([]FilterKV, 0, len(filters))
	)
	for _, filter := range filters {
		switch filter.Type() {
		case FilterTypeWhitelist:
			whitelistFilters = append(whitelistFilters, filter)
		case FilterTypeBlacklist:
			blacklistFilters = append(blacklistFilters, filter)
		}
	}

	return LoaderFunc(func() (map[string]interface{}, error) {
		configMap, err := loader.Load()
		if err != nil {
			return configMap, err
		}

	KvLoop:
		for key, value := range configMap {
			// check if KV is blacklisted
			for _, blFilter := range blacklistFilters {
				if !blFilter.IsAllowed(key, value) {
					delete(configMap, key)

					continue KvLoop
				}
			}

			// check if it is whitelisted
			if len(whitelistFilters) > 0 {
				isAllowed := false
				for _, wlFilter := range whitelistFilters {
					if wlFilter.IsAllowed(key, value) {
						isAllowed = true

						break
					}
				}

				if !isAllowed {
					delete(configMap, key)
				}
			}
		}

		return configMap, nil
	})
}

// FilterKeyWithPrefix returns true if a key has given prefix.
// It can be used as a FilterKV like:
//
//	xconf.FilterKVWhitelistFunc(xconf.FilterKeyWithPrefix(prefix))
//	xconf.FilterKVBlacklistFunc(xconf.FilterKeyWithPrefix(prefix))
func FilterKeyWithPrefix(prefix string) func(key string, _ interface{}) bool {
	return func(key string, _ interface{}) bool {
		return strings.HasPrefix(key, prefix)
	}
}

// FilterKeyWithSuffix returns true if a key has given suffix.
// It can be used as a FilterKV like:
//
//	xconf.FilterKVWhitelistFunc(xconf.FilterKeyWithSuffix(suffix))
//	xconf.FilterKVBlacklistFunc(xconf.FilterKeyWithSuffix(suffix))
func FilterKeyWithSuffix(suffix string) func(key string, _ interface{}) bool {
	return func(key string, _ interface{}) bool {
		return strings.HasSuffix(key, suffix)
	}
}

// FilterExactKeys returns true if a key is present in the provided list.
// It can be used as a FilterKV like:
//
//	xconf.FilterKVWhitelistFunc(xconf.FilterExactKeys(key1, key2))
//	xconf.FilterKVBlacklistFunc(xconf.FilterExactKeys(key1, key2))
func FilterExactKeys(keys ...string) func(key string, _ interface{}) bool {
	return func(key string, _ interface{}) bool {
		for _, k := range keys {
			if key == k {
				return true
			}
		}

		return false
	}
}

// FilterEmptyValue returns true if a value is nil or "".
// It can be used as a FilterKV like:
//
//	xconf.FilterKVBlacklistFunc(xconf.FilterEmptyValue)
func FilterEmptyValue(_ string, value interface{}) bool {
	if value == nil {
		return true
	}

	if valueStr, ok := value.(string); ok && valueStr == "" {
		return true
	}

	return false
}
