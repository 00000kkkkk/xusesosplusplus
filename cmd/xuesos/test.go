package xuesos

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/00000kkkkk/xusesosplusplus/interpreter"
	"github.com/00000kkkkk/xusesosplusplus/lexer"
	"github.com/00000kkkkk/xusesosplusplus/parser"
)

func runTest(args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	// Find all _test.xpp files
	var testFiles []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && strings.HasSuffix(path, "_test.xpp") {
			testFiles = append(testFiles, path)
		}
		return nil
	})

	if len(testFiles) == 0 {
		fmt.Println("No test files found (*_test.xpp)")
		return nil
	}

	totalTests := 0
	passed := 0
	failed := 0
	start := time.Now()

	for _, file := range testFiles {
		src, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL: cannot read %s: %s\n", file, err)
			failed++
			continue
		}

		l := lexer.New(file, string(src))
		tokens, lexErrs := l.ScanAll()
		if len(lexErrs) > 0 {
			fmt.Fprintf(os.Stderr, "FAIL: %s: lex error: %s\n", file, lexErrs[0])
			failed++
			continue
		}

		p := parser.New(tokens)
		program, parseErrs := p.Parse()
		if len(parseErrs) > 0 {
			fmt.Fprintf(os.Stderr, "FAIL: %s: parse error: %s\n", file, parseErrs[0])
			failed++
			continue
		}

		interp := interpreter.New()
		// Add assert built-in for tests
		interp.AddTestBuiltins()

		err = interp.Run(program)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL: %s: %s\n", file, err)
			failed++
		} else {
			fmt.Printf("PASS: %s\n", file)
			passed++
		}
		totalTests++
	}

	elapsed := time.Since(start)
	fmt.Printf("\n%d/%d tests passed in %s\n", passed, totalTests, elapsed.Round(time.Millisecond))

	if failed > 0 {
		return fmt.Errorf("%d test(s) failed", failed)
	}
	return nil
}
