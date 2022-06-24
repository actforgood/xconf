// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xconf/LICENSE.

package xconf

import (
	"sync"
	"sync/atomic"
)

// MockConfig is a mock for xconf.Config contract, to be used in UT.
type MockConfig struct {
	cfg         *DefaultConfig
	getCallsCnt uint32
	getCallback func(key string, def ...interface{})
	configMap   map[string]interface{}
	mu          *sync.Mutex
}

// NewMockConfig instantiates new mocked Config with given key-values configuration.
// Make sure you pass an even number of elements and that the keys are strings.
// Usage example:
//		mock := xconf.NewMockConfig(
//			"foo", "bar",
//			"year", 2022,
//		)
//
func NewMockConfig(kv ...interface{}) *MockConfig {
	mock := &MockConfig{
		configMap: make(map[string]interface{}),
		mu:        new(sync.Mutex),
	}
	mock.SetKeyValues(kv...)

	return mock
}

// Get mock logic.
func (mock *MockConfig) Get(key string, def ...interface{}) interface{} {
	atomic.AddUint32(&mock.getCallsCnt, 1)
	if mock.getCallback != nil {
		mock.getCallback(key, def...)
	}

	return mock.cfg.Get(key, def...)
}

// SetKeyValues sets/resets given key-values.
// Make sure you pass an even number of elements and that the keys are strings.
func (mock *MockConfig) SetKeyValues(kv ...interface{}) {
	kvLen := len(kv)
	if len(kv)%2 == 1 {
		kvLen-- // skip last element
	}
	mock.mu.Lock()
	for i := 0; i < kvLen; i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			continue
		}
		value := kv[i+1]
		mock.configMap[key] = value
	}
	defCfg, _ := NewDefaultConfig(PlainLoader(mock.configMap))
	_ = defCfg.Close()
	mock.cfg = defCfg
	mock.mu.Unlock()
}

// SetGetCallback sets the given callback to be executed inside Get() method.
// You can inject yourself to make assertions upon passed parameter(s) this way.
// Usage example:
// 		mock.SetGetCallback(func(key string, def ...interface{}) {
// 			switch mock.GetCallsCount() {
// 			case 1:
//				if key != "expectedKeyAtCall1" {
//					t.Error("...")
//				}
//			case 2:
//				if key != "expectedKeyAtCall2" {
//					t.Error("...")
//				}
//			}
//		})
//
func (mock *MockConfig) SetGetCallback(callback func(key string, def ...interface{})) {
	mock.getCallback = callback
}

// GetCallsCount returns the no. of times Get() method was called.
func (mock *MockConfig) GetCallsCount() int {
	return int(atomic.LoadUint32(&mock.getCallsCnt))
}
