package xuesos

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/00000kkkkk/xusesosplusplus/interpreter"
	"github.com/00000kkkkk/xusesosplusplus/lexer"
	"github.com/00000kkkkk/xusesosplusplus/parser"
)

const (
	replPrompt     = "xuesos>> "
	replContPrompt = "...      "
)

func runRepl() error {
	fmt.Printf("Xuesos++ REPL v%s\n", Version)
	fmt.Println("Type :help for help, :quit to exit.")
	fmt.Println()

	interp := interpreter.New()
	scanner := bufio.NewScanner(os.Stdin)

	var accumulator strings.Builder
	braceDepth := 0
	continuing := false

	for {
		// Print prompt
		if continuing {
			fmt.Print(replContPrompt)
		} else {
			fmt.Print(replPrompt)
		}

		if !scanner.Scan() {
			// EOF (e.g. Ctrl+D)
			fmt.Println()
			break
		}

		line := scanner.Text()

		// Handle REPL commands only on fresh input (not continuation)
		if !continuing {
			trimmed := strings.TrimSpace(line)
			if trimmed == ":quit" || trimmed == ":exit" {
				fmt.Println("Bye!")
				return nil
			}
			if trimmed == ":help" {
				printReplHelp()
				continue
			}
			if trimmed == "" {
				continue
			}
		}

		// Accumulate lines
		if continuing {
			accumulator.WriteString("\n")
		}
		accumulator.WriteString(line)

		// Count braces to detect multi-line blocks
		braceDepth += countBraces(line)

		if braceDepth > 0 {
			continuing = true
			continue
		}

		// We have a complete input -- process it
		src := accumulator.String()
		accumulator.Reset()
		braceDepth = 0
		continuing = false

		replExec(interp, src)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read error: %w", err)
	}
	return nil
}

// replExec lexes, parses, and interprets a single REPL input.
func replExec(interp *interpreter.Interpreter, src string) {
	// Lex
	l := lexer.New("<repl>", src)
	tokens, lexErrs := l.ScanAll()
	if len(lexErrs) > 0 {
		for _, e := range lexErrs {
			fmt.Fprintf(os.Stderr, "  lex error: %s\n", e)
		}
		return
	}

	// Parse
	p := parser.New(tokens)
	program, parseErrs := p.Parse()
	if len(parseErrs) > 0 {
		for _, e := range parseErrs {
			fmt.Fprintf(os.Stderr, "  parse error: %s\n", e)
		}
		return
	}

	// Interpret (using RunLine so we get the expression result back)
	val, err := interp.RunLine(program)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  runtime error: %s\n", err)
		return
	}

	// Print result of expression (skip null to avoid noise from print() calls etc.)
	if val != nil && val.Type != interpreter.VAL_NULL {
		fmt.Println(val.Inspect())
	}
}

// countBraces returns the net brace depth change in a line.
// It respects string literals and comments to avoid false counts.
func countBraces(line string) int {
	depth := 0
	inString := false
	escaped := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		// Skip line comments
		if ch == '/' && i+1 < len(line) && line[i+1] == '/' {
			break
		}
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
		}
	}
	return depth
}

func printReplHelp() {
	fmt.Println(`Xuesos++ REPL Commands:
  :help          Show this help message
  :quit, :exit   Exit the REPL

Language quick reference:
  xuet x = 42              Immutable variable
  xuiar y = 10             Mutable variable
  y = 20                   Reassign mutable variable
  xuen add(a int, b int) int {   Function definition
    xueturn a + b
  }
  print("hello")           Print to stdout
  xuif x > 0 { ... }      Conditional
  xuior i xuin 0..10 { }  For loop
  xuile cond { }           While loop`)
}
