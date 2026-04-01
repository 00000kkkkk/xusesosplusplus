package xuesos

import (
	"testing"

	"github.com/00000kkkkk/xusesosplusplus/interpreter"
	"github.com/00000kkkkk/xusesosplusplus/lexer"
	"github.com/00000kkkkk/xusesosplusplus/parser"
)

// TestReplExecBasic verifies that the REPL execution pipeline works for expressions.
func TestReplExecBasic(t *testing.T) {
	interp := interpreter.New()

	tests := []struct {
		input    string
		wantVal  string
		wantNull bool
	}{
		{"2 + 3", "5", false},
		{"10 * 4", "40", false},
		{`"hello"`, `"hello"`, false},
		{"xuitru", "xuitru", false},
		{"xuinia", "xuinia", false},
	}

	for _, tt := range tests {
		l := lexer.New("<test>", tt.input)
		tokens, lexErrs := l.ScanAll()
		if len(lexErrs) > 0 {
			t.Fatalf("lex errors for %q: %v", tt.input, lexErrs)
		}

		p := parser.New(tokens)
		program, parseErrs := p.Parse()
		if len(parseErrs) > 0 {
			t.Fatalf("parse errors for %q: %v", tt.input, parseErrs)
		}

		val, err := interp.RunLine(program)
		if err != nil {
			t.Fatalf("runtime error for %q: %v", tt.input, err)
		}

		if tt.wantNull {
			if val != nil && val.Type != interpreter.VAL_NULL {
				t.Errorf("expected null for %q, got %s", tt.input, val.Inspect())
			}
			continue
		}

		if val == nil {
			t.Fatalf("expected value for %q, got nil", tt.input)
		}
		if got := val.Inspect(); got != tt.wantVal {
			t.Errorf("for %q: got %s, want %s", tt.input, got, tt.wantVal)
		}
	}
}

// TestReplStatePersistence verifies that variables persist across REPL lines.
func TestReplStatePersistence(t *testing.T) {
	interp := interpreter.New()

	lines := []string{
		"xuet x = 42",
		"xuiar y = 10",
		"y = 20",
	}

	// Execute declarations
	for _, line := range lines {
		l := lexer.New("<test>", line)
		tokens, lexErrs := l.ScanAll()
		if len(lexErrs) > 0 {
			t.Fatalf("lex errors for %q: %v", line, lexErrs)
		}
		p := parser.New(tokens)
		program, parseErrs := p.Parse()
		if len(parseErrs) > 0 {
			t.Fatalf("parse errors for %q: %v", line, parseErrs)
		}
		_, err := interp.RunLine(program)
		if err != nil {
			t.Fatalf("runtime error for %q: %v", line, err)
		}
	}

	// Now read x back
	l := lexer.New("<test>", "x")
	tokens, _ := l.ScanAll()
	p := parser.New(tokens)
	program, _ := p.Parse()
	val, err := interp.RunLine(program)
	if err != nil {
		t.Fatalf("error reading x: %v", err)
	}
	if val == nil || val.Inspect() != "42" {
		t.Errorf("x: got %v, want 42", val)
	}

	// Read y back (should be 20 after reassignment)
	l = lexer.New("<test>", "y")
	tokens, _ = l.ScanAll()
	p = parser.New(tokens)
	program, _ = p.Parse()
	val, err = interp.RunLine(program)
	if err != nil {
		t.Fatalf("error reading y: %v", err)
	}
	if val == nil || val.Inspect() != "20" {
		t.Errorf("y: got %v, want 20", val)
	}
}

// TestReplFunctionPersistence verifies that functions defined in REPL persist.
func TestReplFunctionPersistence(t *testing.T) {
	interp := interpreter.New()

	// Define a function (multi-line combined as one input)
	funcDef := "xuen add(a int, b int) int {\n  xueturn a + b\n}"
	l := lexer.New("<test>", funcDef)
	tokens, lexErrs := l.ScanAll()
	if len(lexErrs) > 0 {
		t.Fatalf("lex errors: %v", lexErrs)
	}
	p := parser.New(tokens)
	program, parseErrs := p.Parse()
	if len(parseErrs) > 0 {
		t.Fatalf("parse errors: %v", parseErrs)
	}
	_, err := interp.RunLine(program)
	if err != nil {
		t.Fatalf("error defining function: %v", err)
	}

	// Call the function
	l = lexer.New("<test>", "add(3, 4)")
	tokens, _ = l.ScanAll()
	p = parser.New(tokens)
	program, _ = p.Parse()
	val, err := interp.RunLine(program)
	if err != nil {
		t.Fatalf("error calling add: %v", err)
	}
	if val == nil || val.Inspect() != "7" {
		t.Errorf("add(3,4): got %v, want 7", val)
	}
}

// TestCountBraces verifies brace counting for multi-line detection.
func TestCountBraces(t *testing.T) {
	tests := []struct {
		line string
		want int
	}{
		{"xuen foo() {", 1},
		{"  xueturn 42", 0},
		{"}", -1},
		{`xuet s = "hello { world }"`, 0}, // braces inside strings don't count
		{"// this is a { comment", 0},      // braces in comments don't count
		{"{ { }", 1},
		{"{}", 0},
	}

	for _, tt := range tests {
		got := countBraces(tt.line)
		if got != tt.want {
			t.Errorf("countBraces(%q) = %d, want %d", tt.line, got, tt.want)
		}
	}
}

// TestReplErrorRecovery verifies that errors don't crash the REPL pipeline.
func TestReplErrorRecovery(t *testing.T) {
	interp := interpreter.New()

	// Undefined variable should give error, not crash
	l := lexer.New("<test>", "undefined_var")
	tokens, _ := l.ScanAll()
	p := parser.New(tokens)
	program, _ := p.Parse()
	_, err := interp.RunLine(program)
	if err == nil {
		t.Error("expected error for undefined variable, got nil")
	}

	// After error, REPL should still work
	l = lexer.New("<test>", "42")
	tokens, _ = l.ScanAll()
	p = parser.New(tokens)
	program, _ = p.Parse()
	val, err := interp.RunLine(program)
	if err != nil {
		t.Fatalf("error after recovery: %v", err)
	}
	if val == nil || val.Inspect() != "42" {
		t.Errorf("after recovery: got %v, want 42", val)
	}
}
