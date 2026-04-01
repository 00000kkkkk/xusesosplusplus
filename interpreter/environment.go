package interpreter

import "fmt"

// Environment represents a variable scope.
type Environment struct {
	store  map[string]*Value
	mutable map[string]bool // tracks which variables are mutable (xuiar)
	outer  *Environment
}

// NewEnvironment creates a new top-level environment.
func NewEnvironment() *Environment {
	return &Environment{
		store:   make(map[string]*Value),
		mutable: make(map[string]bool),
	}
}

// NewEnclosedEnvironment creates a child scope.
func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

// Get retrieves a variable's value, searching up the scope chain.
func (e *Environment) Get(name string) (*Value, bool) {
	val, ok := e.store[name]
	if !ok && e.outer != nil {
		return e.outer.Get(name)
	}
	return val, ok
}

// Define creates a new variable in the current scope.
func (e *Environment) Define(name string, val *Value, isMutable bool) {
	e.store[name] = val
	e.mutable[name] = isMutable
}

// Set updates an existing variable's value (searching up the scope chain).
// Returns error if variable is immutable or doesn't exist.
func (e *Environment) Set(name string, val *Value) error {
	if _, ok := e.store[name]; ok {
		if !e.mutable[name] {
			return fmt.Errorf("cannot assign to immutable variable %q (declared with xuet)", name)
		}
		e.store[name] = val
		return nil
	}
	if e.outer != nil {
		return e.outer.Set(name, val)
	}
	return fmt.Errorf("undefined variable %q", name)
}

// AllVars returns all variables in the current scope (not parent scopes).
func (e *Environment) AllVars() map[string]*Value {
	result := make(map[string]*Value)
	for name, val := range e.store {
		result[name] = val
	}
	return result
}
