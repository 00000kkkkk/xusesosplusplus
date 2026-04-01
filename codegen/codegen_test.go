package codegen

import (
	"strings"
	"testing"

	"github.com/00000kkkkk/xusesosplusplus/lexer"
	"github.com/00000kkkkk/xusesosplusplus/parser"
)

func generate(t *testing.T, src string) string {
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
	gen := New()
	return gen.Generate(prog)
}

func assertContains(t *testing.T, code, substr string) {
	t.Helper()
	if !strings.Contains(code, substr) {
		t.Errorf("expected C code to contain %q, got:\n%s", substr, code)
	}
}

func TestHelloWorld(t *testing.T) {
	code := generate(t, `
		xuen main() {
			print("Hello, Xuesos++!")
		}
	`)
	assertContains(t, code, "#include <stdio.h>")
	assertContains(t, code, "int main(void)")
	assertContains(t, code, "Hello, Xuesos++!")
	assertContains(t, code, "return 0;")
}

func TestVariables(t *testing.T) {
	code := generate(t, `
		xuen main() {
			xuet x = 42
			xuiar y = 10
		}
	`)
	assertContains(t, code, "const int64_t x = 42LL")
	assertContains(t, code, "int64_t y = 10LL")
}

func TestFunction(t *testing.T) {
	code := generate(t, `
		xuen add(a int, b int) int {
			xueturn a + b
		}
	`)
	assertContains(t, code, "int64_t xpp_add(int64_t a, int64_t b)")
	assertContains(t, code, "return (a + b)")
}

func TestXuif(t *testing.T) {
	code := generate(t, `
		xuen main() {
			xuet x = 5
			xuif (x > 3) {
				print("yes")
			} xuelse {
				print("no")
			}
		}
	`)
	assertContains(t, code, "if ((x > 3LL))")
	assertContains(t, code, "} else {")
}

func TestXuiorRange(t *testing.T) {
	code := generate(t, `
		xuen main() {
			xuior (i xuin 0..10) {
				print(i)
			}
		}
	`)
	assertContains(t, code, "for (int64_t i = 0LL; i < 10LL; i++)")
}

func TestXuile(t *testing.T) {
	code := generate(t, `
		xuen main() {
			xuiar x = 0
			xuile (x < 10) {
				x = x + 1
			}
		}
	`)
	assertContains(t, code, "while ((x < 10LL))")
}

func TestStruct(t *testing.T) {
	code := generate(t, `
		xuiruct Point {
			x int
			y int
		}
		xuen main() {
			xuet p = Point { x = 10, y = 20 }
		}
	`)
	assertContains(t, code, "struct Point {")
	assertContains(t, code, "int64_t x;")
	assertContains(t, code, "int64_t y;")
	assertContains(t, code, "(struct Point){.x = 10LL, .y = 20LL}")
}

func TestBoolLiterals(t *testing.T) {
	code := generate(t, `
		xuen main() {
			xuet a = xuitru
			xuet b = xuinia
		}
	`)
	assertContains(t, code, "= true")
	assertContains(t, code, "= false")
}

func TestBreakContinue(t *testing.T) {
	code := generate(t, `
		xuen main() {
			xuior (i xuin 0..10) {
				xuif (i == 5) {
					xuieak
				}
				xuif (i == 3) {
					xuitinue
				}
			}
		}
	`)
	assertContains(t, code, "break;")
	assertContains(t, code, "continue;")
}

func TestReturnVoid(t *testing.T) {
	code := generate(t, `
		xuen doNothing() {
			xueturn
		}
	`)
	assertContains(t, code, "return;")
}

func TestStringLiteral(t *testing.T) {
	code := generate(t, `
		xuen main() {
			xuet name = "Xuesos"
		}
	`)
	assertContains(t, code, "XppString*")
	assertContains(t, code, `xpp_string_new("Xuesos")`)
}

func TestFibonacci(t *testing.T) {
	code := generate(t, `
		xuen fibonacci(n int) int {
			xuif (n <= 1) {
				xueturn n
			}
			xueturn fibonacci(n - 1) + fibonacci(n - 2)
		}
		xuen main() {
			xuior (i xuin 0..20) {
				print(fibonacci(i))
			}
		}
	`)
	assertContains(t, code, "int64_t xpp_fibonacci(int64_t n)")
	assertContains(t, code, "xpp_fibonacci((n - 1LL))")
	assertContains(t, code, "for (int64_t i = 0LL; i < 20LL; i++)")
}

func TestHeaders(t *testing.T) {
	code := generate(t, `xuen main() {}`)
	assertContains(t, code, "#include <stdio.h>")
	assertContains(t, code, "#include <stdlib.h>")
	assertContains(t, code, "#include <stdint.h>")
	assertContains(t, code, "#include <stdbool.h>")
}

func TestStringVariable(t *testing.T) {
	code := generate(t, `xuen main() { xuet name = "world" }`)
	assertContains(t, code, "XppString*")
	assertContains(t, code, `xpp_string_new("world")`)
}

func TestPrintString(t *testing.T) {
	code := generate(t, `xuen main() { print("hello") }`)
	assertContains(t, code, `xpp_print_string(xpp_string_new("hello"))`)
}

func TestPrintStringVariable(t *testing.T) {
	code := generate(t, `
		xuen main() {
			xuet name = "Alice"
			print(name)
		}
	`)
	assertContains(t, code, `XppString* name`)
	assertContains(t, code, `xpp_print_string(name)`)
}

func TestPrintIntVariable(t *testing.T) {
	code := generate(t, `
		xuen main() {
			xuet x = 42
			print(x)
		}
	`)
	assertContains(t, code, `xpp_print_int(x)`)
}

func TestXuiatchStringArms(t *testing.T) {
	code := generate(t, `
		xuen main() {
			xuet x = "ok"
			xuiatch (x) {
				"ok" => print("good")
				_ => print("other")
			}
		}
	`)
	assertContains(t, code, `xpp_string_eq(x, xpp_string_new("ok"))`)
	assertContains(t, code, "} else {")
	assertContains(t, code, `xpp_print_string`)
}

func TestXuiatchIntArms(t *testing.T) {
	code := generate(t, `
		xuen main() {
			xuet x = 1
			xuiatch (x) {
				1 => print("one")
				2 => print("two")
				_ => print("other")
			}
		}
	`)
	assertContains(t, code, "if ((x) == (1LL))")
	assertContains(t, code, "} else if ((x) == (2LL))")
	assertContains(t, code, "} else {")
}

func TestTryCatch(t *testing.T) {
	code := generate(t, `
		xuen main() {
			xutry {
				xuet x = 1
			} xucatch (e) {
				print("error")
			}
		}
	`)
	assertContains(t, code, "_xpp_has_error = 0")
	assertContains(t, code, "if (_xpp_has_error)")
	assertContains(t, code, "const char* e = _xpp_error_msg")
}

func TestReturnZeroInsideMain(t *testing.T) {
	code := generate(t, `xuen main() { xuet x = 1 }`)
	// return 0 must come before the closing brace of main
	idx0 := strings.Index(code, "return 0;")
	idxClose := strings.LastIndex(code, "}")
	if idx0 < 0 || idxClose < 0 || idx0 >= idxClose {
		t.Errorf("return 0; should be inside main before closing }, got:\n%s", code)
	}
}

func TestStringComparisonHelper(t *testing.T) {
	code := generate(t, `xuen main() {}`)
	// The runtime provides xpp_string_eq for string comparison
	assertContains(t, code, "xpp_string_eq")
}

func TestMutableStringVariable(t *testing.T) {
	code := generate(t, `
		xuen main() {
			xuiar msg = "hi"
		}
	`)
	assertContains(t, code, `XppString* msg = xpp_string_new("hi")`)
}

func TestNullLiteral(t *testing.T) {
	code := generate(t, `
		xuen main() {
			xuet x = xuinull
		}
	`)
	assertContains(t, code, "NULL")
}

func TestPrintBool(t *testing.T) {
	code := generate(t, `
		xuen main() {
			print(xuitru)
		}
	`)
	assertContains(t, code, `xpp_print_bool(true)`)
}

func TestAddressOfAndDeref(t *testing.T) {
	code := generate(t, `
		xuen main() {
			xuiar x = 10
			xuet y = &x
		}
	`)
	assertContains(t, code, "&x")
}

func TestErrorHandlingGlobals(t *testing.T) {
	code := generate(t, `xuen main() {}`)
	// These globals are provided by the embedded runtime
	assertContains(t, code, "_xpp_has_error")
	assertContains(t, code, `_xpp_error_msg`)
}

// --- New tests for full runtime integration ---

func TestArrayLiteral(t *testing.T) {
	code := generate(t, `xuen main() { xuet arr = [1, 2, 3] }`)
	assertContains(t, code, "xpp_array_new")
	assertContains(t, code, "xpp_array_push")
	assertContains(t, code, "xpp_box_int")
}

func TestArrayLiteralEmpty(t *testing.T) {
	code := generate(t, `xuen main() { xuet arr = [] }`)
	assertContains(t, code, "xpp_array_new(0)")
}

func TestArrayInferType(t *testing.T) {
	code := generate(t, `xuen main() { xuet arr = [10, 20] }`)
	assertContains(t, code, "XppArray*")
}

func TestMapLiteralCodegen(t *testing.T) {
	code := generate(t, `xuen main() { xuet m = {"a": 1} }`)
	assertContains(t, code, "xpp_map_new")
	assertContains(t, code, "xpp_map_set")
	assertContains(t, code, "xpp_box_int")
}

func TestMapLiteralEmpty(t *testing.T) {
	code := generate(t, `xuen main() { xuet m = map_new() }`)
	assertContains(t, code, "xpp_map_new()")
}

func TestMapInferType(t *testing.T) {
	code := generate(t, `xuen main() { xuet m = {"x": 42} }`)
	assertContains(t, code, "XppMap*")
}

func TestForEachArray(t *testing.T) {
	code := generate(t, `xuen main() {
		xuet arr = [1, 2, 3]
		xuior (x xuin arr) { print(x) }
	}`)
	assertContains(t, code, "xpp_array_len")
	assertContains(t, code, "xpp_array_get")
	assertContains(t, code, "xpp_unbox_int")
}

func TestMethodCallCodegen(t *testing.T) {
	code := generate(t, `
		xuiruct Dog { name str }
		xuimpl Dog { xuen bark(self) { print("woof") } }
		xuen main() {
			xuet d = Dog { name = "Rex" }
			d.bark()
		}
	`)
	assertContains(t, code, "xpp_Dog_bark")
}

func TestChannelCodegen(t *testing.T) {
	code := generate(t, `xuen main() { xuet ch = channel() }`)
	assertContains(t, code, "xpp_channel_new")
}

func TestChannelInferType(t *testing.T) {
	code := generate(t, `xuen main() { xuet ch = channel() }`)
	assertContains(t, code, "XppChannel*")
}

func TestBuiltinLen(t *testing.T) {
	code := generate(t, `xuen main() {
		xuet arr = [1, 2]
		xuet n = len(arr)
	}`)
	assertContains(t, code, "xpp_array_len")
}

func TestBuiltinPush(t *testing.T) {
	code := generate(t, `xuen main() {
		xuet arr = [1, 2]
		push(arr, 3)
	}`)
	assertContains(t, code, "xpp_array_push")
	assertContains(t, code, "xpp_box_int")
}

func TestPrintMultipleArgs(t *testing.T) {
	code := generate(t, `xuen main() { print("x", "y") }`)
	// Multiple args should use non-newline prints with space separation
	assertContains(t, code, "printf(\" \")")
	assertContains(t, code, "printf(\"\\n\")")
}

func TestDeferStack(t *testing.T) {
	code := generate(t, `xuen main() {}`)
	// Main should initialize the defer stack
	assertContains(t, code, "XppDeferStack")
	assertContains(t, code, "xpp_defer_init")
	assertContains(t, code, "xpp_defer_run_all")
}

func TestThrowExpression(t *testing.T) {
	code := generate(t, `
		xuen main() {
			xutry {
				xuthrow "boom"
			} xucatch (e) {
				print(e)
			}
		}
	`)
	assertContains(t, code, "xpp_throw")
	assertContains(t, code, `"boom"`)
}

func TestMathBuiltins(t *testing.T) {
	code := generate(t, `xuen main() {
		xuet a = sqrt(4.0)
		xuet b = abs(5)
	}`)
	assertContains(t, code, "xpp_math_sqrt")
	assertContains(t, code, "xpp_math_abs")
}

func TestEnumCodegen(t *testing.T) {
	code := generate(t, `
		xuenum Color { Red Green Blue }
		xuen main() {}
	`)
	assertContains(t, code, "enum Color")
	assertContains(t, code, "Color_Red")
	assertContains(t, code, "Color_Green")
	assertContains(t, code, "Color_Blue")
}

func TestMethodForwardDecl(t *testing.T) {
	code := generate(t, `
		xuiruct Cat { name str }
		xuimpl Cat { xuen meow(self) { print("meow") } }
		xuen main() {}
	`)
	// Forward declaration should appear before implementation
	assertContains(t, code, "xpp_Cat_meow(struct Cat *self);")
	assertContains(t, code, "xpp_Cat_meow(struct Cat *self) {")
}

func TestLambdaPlaceholder(t *testing.T) {
	code := generate(t, `xuen main() { xuet f = (x) => x }`)
	// Lambda should be documented as a placeholder
	assertContains(t, code, "lambda")
	assertContains(t, code, "NULL")
}

func TestSleepBuiltin(t *testing.T) {
	code := generate(t, `xuen main() { sleep(100) }`)
	assertContains(t, code, "usleep")
}

func TestIndexOnArray(t *testing.T) {
	code := generate(t, `xuen main() {
		xuet arr = [10, 20, 30]
		xuet x = arr[0]
	}`)
	assertContains(t, code, "xpp_array_get")
	assertContains(t, code, "xpp_unbox_int")
}

func TestPrintFloat(t *testing.T) {
	code := generate(t, `xuen main() { print(3.14) }`)
	assertContains(t, code, "xpp_print_float(3.14)")
}
