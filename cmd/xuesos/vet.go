package xuesos

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/00000kkkkk/xusesosplusplus/lexer"
	"github.com/00000kkkkk/xusesosplusplus/parser"
	"github.com/00000kkkkk/xusesosplusplus/typechecker"
)

func runVet(args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && strings.HasSuffix(path, ".xpp") && !strings.HasSuffix(path, "_test.xpp") {
			files = append(files, path)
		}
		return nil
	})

	if len(files) == 0 {
		fmt.Println("No .xpp files found")
		return nil
	}

	totalIssues := 0
	for _, file := range files {
		issues := vetFile(file)
		totalIssues += len(issues)
		for _, issue := range issues {
			fmt.Println(issue)
		}
	}

	if totalIssues == 0 {
		fmt.Printf("vet: %d files checked, no issues found\n", len(files))
	} else {
		fmt.Printf("\nvet: %d issue(s) found in %d file(s)\n", totalIssues, len(files))
	}

	if totalIssues > 0 {
		return fmt.Errorf("%d issue(s) found", totalIssues)
	}
	return nil
}

func vetFile(filename string) []string {
	var issues []string

	src, err := os.ReadFile(filename)
	if err != nil {
		return []string{fmt.Sprintf("%s: cannot read file: %s", filename, err)}
	}

	// Lex check
	l := lexer.New(filename, string(src))
	tokens, lexErrs := l.ScanAll()
	for _, e := range lexErrs {
		issues = append(issues, fmt.Sprintf("  %s: lex: %s", filename, e.Message))
	}

	// Parse check (recover from panics in parser)
	var program *parser.Program
	var parseErrs []parser.ParseError
	func() {
		defer func() {
			if r := recover(); r != nil {
				issues = append(issues, fmt.Sprintf("  %s: parse: internal error: %v", filename, r))
			}
		}()
		p := parser.New(tokens)
		program, parseErrs = p.Parse()
	}()
	for _, e := range parseErrs {
		issues = append(issues, fmt.Sprintf("  %s: parse: %s", filename, e.Message))
	}

	if program == nil || len(lexErrs) > 0 || len(parseErrs) > 0 {
		return issues
	}

	// Type check
	tc := typechecker.New()
	typeErrs := tc.Check(program)
	for _, e := range typeErrs {
		issues = append(issues, fmt.Sprintf("  %s: type: %s", filename, e.Message))
	}

	// Custom vet checks
	for _, stmt := range program.Statements {
		issues = append(issues, vetStatement(filename, stmt)...)
	}

	return issues
}

func vetStatement(file string, stmt parser.Statement) []string {
	var issues []string

	switch s := stmt.(type) {
	case *parser.XuenStatement:
		// Check for empty functions
		if len(s.Body.Statements) == 0 && s.Name != "main" {
			issues = append(issues, fmt.Sprintf("  %s:%d: empty function %q", file, s.Pos.Line, s.Name))
		}
		// Check function body
		for _, bodyStmt := range s.Body.Statements {
			issues = append(issues, vetStatement(file, bodyStmt)...)
		}

	case *parser.XuetStatement:
		// Check for empty variable name
		if s.Name == "" {
			issues = append(issues, fmt.Sprintf("  %s:%d: empty variable name", file, s.Pos.Line))
		}

	case *parser.XuifStatement:
		// Check for empty if body
		if len(s.Consequence.Statements) == 0 {
			issues = append(issues, fmt.Sprintf("  %s:%d: empty xuif body", file, s.Pos.Line))
		}
		for _, bodyStmt := range s.Consequence.Statements {
			issues = append(issues, vetStatement(file, bodyStmt)...)
		}
		if s.Alternative != nil {
			issues = append(issues, vetStatement(file, s.Alternative)...)
		}

	case *parser.XuiorStatement:
		if len(s.Body.Statements) == 0 {
			issues = append(issues, fmt.Sprintf("  %s:%d: empty xuior body", file, s.Pos.Line))
		}
		for _, bodyStmt := range s.Body.Statements {
			issues = append(issues, vetStatement(file, bodyStmt)...)
		}

	case *parser.XuileStatement:
		if len(s.Body.Statements) == 0 {
			issues = append(issues, fmt.Sprintf("  %s:%d: empty xuile body", file, s.Pos.Line))
		}
		for _, bodyStmt := range s.Body.Statements {
			issues = append(issues, vetStatement(file, bodyStmt)...)
		}

	case *parser.BlockStatement:
		for _, bodyStmt := range s.Statements {
			issues = append(issues, vetStatement(file, bodyStmt)...)
		}

	case *parser.XuimplStatement:
		for _, method := range s.Methods {
			issues = append(issues, vetStatement(file, method)...)
		}
	}

	return issues
}
