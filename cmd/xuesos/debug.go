package xuesos

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/00000kkkkk/xusesosplusplus/interpreter"
	"github.com/00000kkkkk/xusesosplusplus/lexer"
	"github.com/00000kkkkk/xusesosplusplus/parser"
)

func runDebug(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("debug requires a .xpp file\nUsage: xuesos debug <file.xpp>")
	}

	filename := args[0]
	src, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", filename, err)
	}

	// Lex and parse
	l := lexer.New(filename, string(src))
	tokens, lexErrs := l.ScanAll()
	if len(lexErrs) > 0 {
		for _, e := range lexErrs {
			fmt.Fprintf(os.Stderr, "error: %s\n", e)
		}
		return fmt.Errorf("lexing failed")
	}

	p := parser.New(tokens)
	program, parseErrs := p.Parse()
	if len(parseErrs) > 0 {
		for _, e := range parseErrs {
			fmt.Fprintf(os.Stderr, "error: %s\n", e)
		}
		return fmt.Errorf("parsing failed")
	}

	lines := strings.Split(string(src), "\n")

	fmt.Printf("Xuesos++ Debugger — %s (%d lines)\n", filename, len(lines))
	fmt.Println("Commands: (s)tep, (c)ontinue, (p) var, (l)ocals, (b) line, (q)uit, (h)elp")
	fmt.Println()

	dbg := &debugger{
		lines:       lines,
		breakpoints: make(map[int]bool),
		stepping:    true,
	}

	// Create interpreter with debug hook
	interp := interpreter.New()
	interp.SetDebugHook(func(pos lexer.Position, env *interpreter.Environment) bool {
		return dbg.onStatement(pos, env)
	})

	err = interp.Run(program)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nRuntime error: %s\n", err)
	}

	fmt.Println("\nProgram finished.")
	return nil
}

type debugger struct {
	lines       []string
	breakpoints map[int]bool
	stepping    bool
	continuing  bool
}

func (d *debugger) onStatement(pos lexer.Position, env *interpreter.Environment) bool {
	line := pos.Line

	// If continuing and no breakpoint, skip
	if d.continuing && !d.breakpoints[line] {
		return true
	}
	d.continuing = false

	// Show current position
	d.showContext(line)

	// Debug REPL
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("debug:%d> ", line)
		if !scanner.Scan() {
			return false
		}

		cmd := strings.TrimSpace(scanner.Text())
		if cmd == "" {
			cmd = "s" // default: step
		}

		parts := strings.SplitN(cmd, " ", 2)
		switch parts[0] {
		case "s", "step":
			return true
		case "c", "continue":
			d.continuing = true
			return true
		case "q", "quit":
			fmt.Println("Debugger: quit")
			os.Exit(0)
		case "p", "print":
			if len(parts) < 2 {
				fmt.Println("Usage: p <variable>")
				continue
			}
			d.printVar(parts[1], env)
		case "l", "locals":
			d.printLocals(env)
		case "b", "break":
			if len(parts) < 2 {
				// Show breakpoints
				if len(d.breakpoints) == 0 {
					fmt.Println("No breakpoints set")
				} else {
					for bp := range d.breakpoints {
						fmt.Printf("  breakpoint at line %d\n", bp)
					}
				}
				continue
			}
			lineNum, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				fmt.Println("Invalid line number")
				continue
			}
			d.breakpoints[lineNum] = true
			fmt.Printf("Breakpoint set at line %d\n", lineNum)
		case "h", "help":
			fmt.Println("Debugger commands:")
			fmt.Println("  s, step       Step to next statement")
			fmt.Println("  c, continue   Continue to next breakpoint")
			fmt.Println("  p <var>       Print variable value")
			fmt.Println("  l, locals     Show all local variables")
			fmt.Println("  b <line>      Set breakpoint at line")
			fmt.Println("  b             Show all breakpoints")
			fmt.Println("  q, quit       Quit debugger")
			fmt.Println("  h, help       Show this help")
		default:
			fmt.Printf("Unknown command: %s (type 'h' for help)\n", cmd)
		}
	}
}

func (d *debugger) showContext(line int) {
	fmt.Println()
	start := line - 2
	if start < 1 {
		start = 1
	}
	end := line + 2
	if end > len(d.lines) {
		end = len(d.lines)
	}

	for i := start; i <= end; i++ {
		marker := "  "
		if i == line {
			marker = "-> "
		}
		bp := " "
		if d.breakpoints[i] {
			bp = "*"
		}
		lineContent := ""
		if i-1 < len(d.lines) {
			lineContent = d.lines[i-1]
		}
		fmt.Printf(" %s%s%4d | %s\n", bp, marker, i, lineContent)
	}
	fmt.Println()
}

func (d *debugger) printVar(name string, env *interpreter.Environment) {
	val, ok := env.Get(name)
	if !ok {
		fmt.Printf("  %s: <undefined>\n", name)
		return
	}
	fmt.Printf("  %s = %s (type: %s)\n", name, val.Inspect(), val.Type)
}

func (d *debugger) printLocals(env *interpreter.Environment) {
	vars := env.AllVars()
	if len(vars) == 0 {
		fmt.Println("  (no local variables)")
		return
	}
	for name, val := range vars {
		fmt.Printf("  %s = %s (type: %s)\n", name, val.Inspect(), val.Type)
	}
}
