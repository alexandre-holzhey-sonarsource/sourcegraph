// Code generated by go-mockgen 1.3.7; DO NOT EDIT.
//
// This file was generated by running `sg generate` (or `go-mockgen`) at the root of
// this repository. To add additional mocks to this or another package, add a new entry
// to the mockgen.yaml file in the root of this repository.

package background

import "sync"

// MockRoutine is a mock implementation of the Routine interface (from the
// package github.com/sourcegraph/sourcegraph/lib/background) used for unit
// testing.
type MockRoutine struct {
	// StartFunc is an instance of a mock function object controlling the
	// behavior of the method Start.
	StartFunc *RoutineStartFunc
	// StopFunc is an instance of a mock function object controlling the
	// behavior of the method Stop.
	StopFunc *RoutineStopFunc
}

// NewMockRoutine creates a new mock of the Routine interface. All methods
// return zero values for all results, unless overwritten.
func NewMockRoutine() *MockRoutine {
	return &MockRoutine{
		StartFunc: &RoutineStartFunc{
			defaultHook: func() {
				return
			},
		},
		StopFunc: &RoutineStopFunc{
			defaultHook: func() {
				return
			},
		},
	}
}

// NewStrictMockRoutine creates a new mock of the Routine interface. All
// methods panic on invocation, unless overwritten.
func NewStrictMockRoutine() *MockRoutine {
	return &MockRoutine{
		StartFunc: &RoutineStartFunc{
			defaultHook: func() {
				panic("unexpected invocation of MockRoutine.Start")
			},
		},
		StopFunc: &RoutineStopFunc{
			defaultHook: func() {
				panic("unexpected invocation of MockRoutine.Stop")
			},
		},
	}
}

// NewMockRoutineFrom creates a new mock of the MockRoutine interface. All
// methods delegate to the given implementation, unless overwritten.
func NewMockRoutineFrom(i Routine) *MockRoutine {
	return &MockRoutine{
		StartFunc: &RoutineStartFunc{
			defaultHook: i.Start,
		},
		StopFunc: &RoutineStopFunc{
			defaultHook: i.Stop,
		},
	}
}

// RoutineStartFunc describes the behavior when the Start method of the
// parent MockRoutine instance is invoked.
type RoutineStartFunc struct {
	defaultHook func()
	hooks       []func()
	history     []RoutineStartFuncCall
	mutex       sync.Mutex
}

// Start delegates to the next hook function in the queue and stores the
// parameter and result values of this invocation.
func (m *MockRoutine) Start() {
	m.StartFunc.nextHook()()
	m.StartFunc.appendCall(RoutineStartFuncCall{})
	return
}

// SetDefaultHook sets function that is called when the Start method of the
// parent MockRoutine instance is invoked and the hook queue is empty.
func (f *RoutineStartFunc) SetDefaultHook(hook func()) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// Start method of the parent MockRoutine instance invokes the hook at the
// front of the queue and discards it. After the queue is empty, the default
// hook function is invoked for any future action.
func (f *RoutineStartFunc) PushHook(hook func()) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultHook with a function that returns the
// given values.
func (f *RoutineStartFunc) SetDefaultReturn() {
	f.SetDefaultHook(func() {
		return
	})
}

// PushReturn calls PushHook with a function that returns the given values.
func (f *RoutineStartFunc) PushReturn() {
	f.PushHook(func() {
		return
	})
}

func (f *RoutineStartFunc) nextHook() func() {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *RoutineStartFunc) appendCall(r0 RoutineStartFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of RoutineStartFuncCall objects describing the
// invocations of this function.
func (f *RoutineStartFunc) History() []RoutineStartFuncCall {
	f.mutex.Lock()
	history := make([]RoutineStartFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// RoutineStartFuncCall is an object that describes an invocation of method
// Start on an instance of MockRoutine.
type RoutineStartFuncCall struct{}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c RoutineStartFuncCall) Args() []interface{} {
	return []interface{}{}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c RoutineStartFuncCall) Results() []interface{} {
	return []interface{}{}
}

// RoutineStopFunc describes the behavior when the Stop method of the parent
// MockRoutine instance is invoked.
type RoutineStopFunc struct {
	defaultHook func()
	hooks       []func()
	history     []RoutineStopFuncCall
	mutex       sync.Mutex
}

// Stop delegates to the next hook function in the queue and stores the
// parameter and result values of this invocation.
func (m *MockRoutine) Stop() {
	m.StopFunc.nextHook()()
	m.StopFunc.appendCall(RoutineStopFuncCall{})
	return
}

// SetDefaultHook sets function that is called when the Stop method of the
// parent MockRoutine instance is invoked and the hook queue is empty.
func (f *RoutineStopFunc) SetDefaultHook(hook func()) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// Stop method of the parent MockRoutine instance invokes the hook at the
// front of the queue and discards it. After the queue is empty, the default
// hook function is invoked for any future action.
func (f *RoutineStopFunc) PushHook(hook func()) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultHook with a function that returns the
// given values.
func (f *RoutineStopFunc) SetDefaultReturn() {
	f.SetDefaultHook(func() {
		return
	})
}

// PushReturn calls PushHook with a function that returns the given values.
func (f *RoutineStopFunc) PushReturn() {
	f.PushHook(func() {
		return
	})
}

func (f *RoutineStopFunc) nextHook() func() {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *RoutineStopFunc) appendCall(r0 RoutineStopFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of RoutineStopFuncCall objects describing the
// invocations of this function.
func (f *RoutineStopFunc) History() []RoutineStopFuncCall {
	f.mutex.Lock()
	history := make([]RoutineStopFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// RoutineStopFuncCall is an object that describes an invocation of method
// Stop on an instance of MockRoutine.
type RoutineStopFuncCall struct{}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c RoutineStopFuncCall) Args() []interface{} {
	return []interface{}{}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c RoutineStopFuncCall) Results() []interface{} {
	return []interface{}{}
}
