package typechecker

import (
	"strings"
	"testing"

	"github.com/00000kkkkk/xusesosplusplus/lexer"
	"github.com/00000kkkkk/xusesosplusplus/parser"
)

func check(t *testing.T, src string) []TypeError {
	t.Helper()
	l := lexer.New("test.xpp", src)
	tokens, lexErrs := l.ScanAll()
	if len(lexErrs) > 0 {
		t.Fatalf("lexer errors: %v", lexErrs)
	}
	p := parser.New(tokens)
	prog, parseErrs := p.Parse()
	if len(parseErrs) > 0 {
		t.Fatalf("parser errors: %v", parseErrs)
	}
	c := New()
	return c.Check(prog)
}

func expectNoErrors(t *testing.T, src string) {
	t.Helper()
	errs := check(t, src)
	if len(errs) > 0 {
		t.Fatalf("expected no errors, got %d: %v", len(errs), errs)
	}
}

func expectError(t *testing.T, src string, substr string) {
	t.Helper()
	errs := check(t, src)
	if len(errs) == 0 {
		t.Fatal("expected type error, got none")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, substr) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error containing %q, got: %v", substr, errs)
	}
}

// --- Valid programs ---

func TestValidXuet(t *testing.T) {
	expectNoErrors(t, `xuet x = 42`)
}

func TestValidXuetWithType(t *testing.T) {
	expectNoErrors(t, `xuet x int = 42`)
}

func TestValidXuiar(t *testing.T) {
	expectNoErrors(t, `
		xuiar x = 10
		x = 20
	`)
}

func TestValidFunction(t *testing.T) {
	expectNoErrors(t, `
		xuen add(a int, b int) int {
			xueturn a + b
		}
	`)
}

func TestValidFunctionNoReturn(t *testing.T) {
	expectNoErrors(t, `
		xuen greet(name str) {
			print(name)
		}
	`)
}

func TestValidXuif(t *testing.T) {
	expectNoErrors(t, `
		xuet x = 5
		xuif (x > 3) {
			print("yes")
		}
	`)
}

func TestValidXuior(t *testing.T) {
	expectNoErrors(t, `
		xuior (i xuin 0..10) {
			print(i)
		}
	`)
}

func TestValidXuile(t *testing.T) {
	expectNoErrors(t, `
		xuiar x = 0
		xuile (x < 10) {
			x = x + 1
		}
	`)
}

func TestValidStruct(t *testing.T) {
	expectNoErrors(t, `
		xuiruct Point {
			x int
			y int
		}
		xuet p = Point { x = 10, y = 20 }
		print(p.x)
	`)
}

func TestValidStructMethod(t *testing.T) {
	expectNoErrors(t, `
		xuiruct Counter {
			value int
		}
		xuimpl Counter {
			xuen get(self) int {
				xueturn self.value
			}
		}
	`)
}

func TestValidEnum(t *testing.T) {
	expectNoErrors(t, `
		xuenum Color {
			Red
			Green
			Blue
		}
		xuet c = Red
	`)
}

func TestValidXuiatch(t *testing.T) {
	expectNoErrors(t, `
		xuet x = "ok"
		xuiatch (x) {
			"ok" => print("good")
			_ => print("other")
		}
	`)
}

func TestValidArray(t *testing.T) {
	expectNoErrors(t, `
		xuet nums = [1, 2, 3]
		print(nums[0])
	`)
}

func TestValidStringConcat(t *testing.T) {
	expectNoErrors(t, `
		xuet result = "hello" + " " + "world"
	`)
}

func TestValidArithmetic(t *testing.T) {
	expectNoErrors(t, `
		xuet a = 5 + 3
		xuet b = 10 - 2
		xuet c = 3 * 4
		xuet d = 15 / 3
		xuet e = 17 % 5
	`)
}

func TestValidComparison(t *testing.T) {
	expectNoErrors(t, `
		xuet a = 5 > 3
		xuet b = 5 == 5
		xuet c = 5 != 3
	`)
}

func TestValidNullable(t *testing.T) {
	expectNoErrors(t, `xuet x ?int = xuinull`)
}

func TestValidIntToFloat(t *testing.T) {
	expectNoErrors(t, `xuet x float = 42`)
}

func TestValidFullProgram(t *testing.T) {
	expectNoErrors(t, `
		xuen fibonacci(n int) int {
			xuif (n <= 1) {
				xueturn n
			}
			xueturn fibonacci(n - 1) + fibonacci(n - 2)
		}
		xuen main() {
			xuior (i xuin 0..10) {
				print(fibonacci(i))
			}
		}
	`)
}

// --- Type errors ---

func TestImmutableAssign(t *testing.T) {
	expectError(t, `
		xuet x = 10
		x = 20
	`, "immutable")
}

func TestUndefinedVariable(t *testing.T) {
	expectError(t, `print(xyz)`, "undefined variable")
}

func TestTypeMismatchAssign(t *testing.T) {
	expectError(t, `xuet x int = "hello"`, "cannot assign str to int")
}

func TestCannotNegateString(t *testing.T) {
	expectError(t, `xuet x = -"hello"`, "cannot negate str")
}

func TestCannotAddStringAndBool(t *testing.T) {
	// String concat is allowed with anything via .String(), so this should check non-string non-numeric
	expectError(t, `xuet x = xuitru + xuinia`, "cannot apply")
}

func TestUndefinedStruct(t *testing.T) {
	expectError(t, `xuet p = Unknown { x = 1 }`, "undefined struct")
}

func TestStructNoField(t *testing.T) {
	expectError(t, `
		xuiruct Point {
			x int
			y int
		}
		xuet p = Point { x = 1, y = 2 }
		print(p.z)
	`, "no field")
}

func TestWrongReturnType(t *testing.T) {
	expectError(t, `
		xuen foo() int {
			xueturn "hello"
		}
	`, "cannot return str")
}

func TestUndefinedStructInImpl(t *testing.T) {
	expectError(t, `
		xuimpl Unknown {
			xuen foo(self) {}
		}
	`, "undefined struct")
}

func TestRangeWithFloat(t *testing.T) {
	expectError(t, `xuet r = 1.5..10`, "range start must be int")
}

func TestIndexWithFloat(t *testing.T) {
	expectError(t, `
		xuet arr = [1, 2, 3]
		print(arr[1.5])
	`, "index must be int")
}
