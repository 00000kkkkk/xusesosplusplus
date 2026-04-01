package xuesos

import (
	"fmt"
	"os"
	"strings"

	"github.com/00000kkkkk/xusesosplusplus/interpreter"
	"github.com/00000kkkkk/xusesosplusplus/lexer"
	"github.com/00000kkkkk/xusesosplusplus/parser"
	"github.com/00000kkkkk/xusesosplusplus/typechecker"
)

func runRun(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("run requires a .xpp file argument\nUsage: xuesos run <file.xpp>")
	}

	filename := args[0]
	if !strings.HasSuffix(filename, ".xpp") {
		return fmt.Errorf("expected .xpp file, got %q", filename)
	}

	src, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", filename, err)
	}

	// Lex
	l := lexer.New(filename, string(src))
	tokens, lexErrs := l.ScanAll()
	if len(lexErrs) > 0 {
		for _, e := range lexErrs {
			fmt.Fprintf(os.Stderr, "error: %s\n", e)
		}
		return fmt.Errorf("lexing failed with %d error(s)", len(lexErrs))
	}

	// Parse
	p := parser.New(tokens)
	program, parseErrs := p.Parse()
	if len(parseErrs) > 0 {
		for _, e := range parseErrs {
			fmt.Fprintf(os.Stderr, "error: %s\n", e)
		}
		return fmt.Errorf("parsing failed with %d error(s)", len(parseErrs))
	}

	// Type check
	tc := typechecker.New()
	typeErrs := tc.Check(program)
	if len(typeErrs) > 0 {
		for _, e := range typeErrs {
			fmt.Fprintf(os.Stderr, "type error: %s\n", e)
		}
		return fmt.Errorf("type checking failed with %d error(s)", len(typeErrs))
	}

	// Interpret
	interp := interpreter.New()
	if err := interp.Run(program); err != nil {
		return fmt.Errorf("runtime error: %s", err)
	}

	return nil
}
