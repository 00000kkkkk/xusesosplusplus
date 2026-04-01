package codegen

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/00000kkkkk/xusesosplusplus/parser"
)

//go:embed runtime/runtime.c
var runtimeC string

// CCodegen generates C code from the AST.
type CCodegen struct {
	output      strings.Builder
	indent      int
	structDefs  map[string]*parser.XuiructStatement
	implDefs    map[string]*parser.XuimplStatement
	varTypes    map[string]string // variable name -> C type
	tempCounter int
}

// New creates a new C code generator.
func New() *CCodegen {
	return &CCodegen{
		structDefs: make(map[string]*parser.XuiructStatement),
		implDefs:   make(map[string]*parser.XuimplStatement),
		varTypes:   make(map[string]string),
	}
}

// Generate produces C code from a parsed program.
func (g *CCodegen) Generate(program *parser.Program) string {
	// First pass: collect struct and impl definitions
	for _, stmt := range program.Statements {
		switch s := stmt.(type) {
		case *parser.XuiructStatement:
			g.structDefs[s.Name] = s
		case *parser.XuimplStatement:
			g.implDefs[s.Name] = s
		}
	}

	// Emit runtime library (includes all headers, types, and helpers)
	// The runtime provides: XppString*, XppArray*, XppMap*, xpp_print_*,
	// xpp_string_new/concat/eq, error handling globals, etc.
	g.writeln(runtimeC)
	g.writeln("")

	// Forward declare all functions
	for _, stmt := range program.Statements {
		if fn, ok := stmt.(*parser.XuenStatement); ok {
			g.emitFuncForwardDecl(fn)
		}
	}
	// Forward declare methods
	for name, impl := range g.implDefs {
		for _, method := range impl.Methods {
			g.emitMethodForwardDecl(name, method)
		}
	}
	g.writeln("")

	// Emit enums
	for _, stmt := range program.Statements {
		if s, ok := stmt.(*parser.XuenumStatement); ok {
			g.emitEnum(s)
		}
	}

	// Emit structs
	for _, stmt := range program.Statements {
		if s, ok := stmt.(*parser.XuiructStatement); ok {
			g.emitStruct(s)
		}
	}

	// Emit all functions and impl methods
	for _, stmt := range program.Statements {
		switch s := stmt.(type) {
		case *parser.XuenStatement:
			g.emitFunction(s)
		case *parser.XuimplStatement:
			g.emitImpl(s)
		}
	}

	// Emit main wrapper
	g.emitMainWrapper(program)

	return g.output.String()
}

func (g *CCodegen) write(s string) {
	g.output.WriteString(s)
}

func (g *CCodegen) writeln(s string) {
	g.output.WriteString(s)
	g.output.WriteString("\n")
}

func (g *CCodegen) writeIndent() {
	for i := 0; i < g.indent; i++ {
		g.write("    ")
	}
}

func (g *CCodegen) emitLine(format string, args ...interface{}) {
	g.writeIndent()
	g.writeln(fmt.Sprintf(format, args...))
}

func (g *CCodegen) newTemp() string {
	g.tempCounter++
	return fmt.Sprintf("_tmp%d", g.tempCounter)
}

// --- Type mapping ---

func (g *CCodegen) mapType(typeName string) string {
	switch typeName {
	case "int", "int64":
		return "int64_t"
	case "int8":
		return "int8_t"
	case "int16":
		return "int16_t"
	case "int32":
		return "int32_t"
	case "uint":
		return "uint64_t"
	case "float", "float32":
		return "double"
	case "bool":
		return "bool"
	case "str":
		return "XppString*"
	case "[]str":
		return "XppArray*"
	case "[]int":
		return "XppArray*"
	case "char":
		return "char"
	case "byte":
		return "uint8_t"
	case "void", "":
		return "void"
	default:
		// Check for array type prefix
		if strings.HasPrefix(typeName, "[]") {
			return "XppArray*"
		}
		// Check for channel type
		if typeName == "channel" || typeName == "chan" {
			return "XppChannel*"
		}
		// Check for map type
		if strings.HasPrefix(typeName, "map[") {
			return "XppMap*"
		}
		// Struct type
		return "struct " + typeName
	}
}

// --- Forward declarations ---

func (g *CCodegen) emitFuncForwardDecl(fn *parser.XuenStatement) {
	if fn.Name == "main" {
		return // main is handled separately
	}
	retType := g.mapType(fn.ReturnType)
	params := g.buildParamList(fn.Params)
	g.writeln(fmt.Sprintf("%s xpp_%s(%s);", retType, fn.Name, params))
}

func (g *CCodegen) emitMethodForwardDecl(structName string, method *parser.XuenStatement) {
	retType := g.mapType(method.ReturnType)
	params := g.buildMethodParamList(structName, method.Params)
	g.writeln(fmt.Sprintf("%s xpp_%s_%s(%s);", retType, structName, method.Name, params))
}

func (g *CCodegen) buildParamList(params []parser.Parameter) string {
	if len(params) == 0 {
		return "void"
	}
	var parts []string
	for _, p := range params {
		parts = append(parts, fmt.Sprintf("%s %s", g.mapType(p.TypeName), p.Name))
	}
	return strings.Join(parts, ", ")
}

func (g *CCodegen) buildMethodParamList(structName string, params []parser.Parameter) string {
	var parts []string
	for _, p := range params {
		if p.Name == "self" {
			parts = append(parts, fmt.Sprintf("struct %s *self", structName))
		} else {
			parts = append(parts, fmt.Sprintf("%s %s", g.mapType(p.TypeName), p.Name))
		}
	}
	if len(parts) == 0 {
		return "void"
	}
	return strings.Join(parts, ", ")
}

// --- Enum emission ---

func (g *CCodegen) emitEnum(s *parser.XuenumStatement) {
	g.writeln(fmt.Sprintf("enum %s {", s.Name))
	g.indent++
	for i, v := range s.Variants {
		comma := ","
		if i == len(s.Variants)-1 {
			comma = ""
		}
		g.emitLine("%s_%s = %d%s", s.Name, v, i, comma)
	}
	g.indent--
	g.writeln("};")
	g.writeln("")
}

// --- Struct emission ---

func (g *CCodegen) emitStruct(s *parser.XuiructStatement) {
	g.writeln(fmt.Sprintf("struct %s {", s.Name))
	g.indent++
	for _, f := range s.Fields {
		g.emitLine("%s %s;", g.mapType(f.TypeName), f.Name)
	}
	g.indent--
	g.writeln("};")
	g.writeln("")
}

// --- Function emission ---

func (g *CCodegen) emitFunction(fn *parser.XuenStatement) {
	if fn.Name == "main" {
		return // handled by emitMainWrapper
	}
	retType := g.mapType(fn.ReturnType)
	params := g.buildParamList(fn.Params)
	g.writeln(fmt.Sprintf("%s xpp_%s(%s) {", retType, fn.Name, params))
	g.indent++
	// Register parameter types
	for _, p := range fn.Params {
		g.varTypes[p.Name] = g.mapType(p.TypeName)
	}
	g.emitBlock(fn.Body)
	g.indent--
	g.writeln("}")
	g.writeln("")
}

func (g *CCodegen) emitImpl(s *parser.XuimplStatement) {
	for _, method := range s.Methods {
		retType := g.mapType(method.ReturnType)
		params := g.buildMethodParamList(s.Name, method.Params)
		g.writeln(fmt.Sprintf("%s xpp_%s_%s(%s) {", retType, s.Name, method.Name, params))
		g.indent++
		// Register parameter types
		for _, p := range method.Params {
			if p.Name == "self" {
				g.varTypes["self"] = "struct " + s.Name + " *"
			} else {
				g.varTypes[p.Name] = g.mapType(p.TypeName)
			}
		}
		g.emitBlock(method.Body)
		g.indent--
		g.writeln("}")
		g.writeln("")
	}
}

func (g *CCodegen) emitMainWrapper(program *parser.Program) {
	// Find the user's main function
	var mainFn *parser.XuenStatement
	var topLevel []parser.Statement

	for _, stmt := range program.Statements {
		if fn, ok := stmt.(*parser.XuenStatement); ok && fn.Name == "main" {
			mainFn = fn
		} else {
			switch stmt.(type) {
			case *parser.XuiructStatement, *parser.XuimplStatement, *parser.XuenumStatement,
				*parser.XuenStatement, *parser.XuimportStatement, *parser.XuinterfaceStatement:
				// Skip declarations — already emitted
			default:
				topLevel = append(topLevel, stmt)
			}
		}
	}

	g.writeln("int main(void) {")
	g.indent++

	// Initialize defer stack for main scope
	g.emitLine("XppDeferStack _defer_stack;")
	g.emitLine("xpp_defer_init(&_defer_stack);")

	// Emit top-level statements
	for _, stmt := range topLevel {
		g.emitStatement(stmt)
	}

	// Call user's main if it exists
	if mainFn != nil {
		g.emitBlock(mainFn.Body)
	}

	// Run deferred calls before exiting
	g.emitLine("xpp_defer_run_all(&_defer_stack);")
	g.emitLine("return 0;")
	g.indent--
	g.writeln("}")
}

// --- Statement emission ---

func (g *CCodegen) emitBlock(block *parser.BlockStatement) {
	for _, stmt := range block.Statements {
		g.emitStatement(stmt)
	}
}

func (g *CCodegen) emitStatement(stmt parser.Statement) {
	switch s := stmt.(type) {
	case *parser.XuetStatement:
		g.emitXuet(s)
	case *parser.XuiarStatement:
		g.emitXuiar(s)
	case *parser.AssignStatement:
		g.emitAssign(s)
	case *parser.XueturnStatement:
		g.emitXueturn(s)
	case *parser.XueakStatement:
		g.emitLine("break;")
	case *parser.XuitinueStatement:
		g.emitLine("continue;")
	case *parser.XuifStatement:
		g.emitXuif(s)
	case *parser.XuiorStatement:
		g.emitXuior(s)
	case *parser.XuiorClassicStatement:
		g.emitXuiorClassic(s)
	case *parser.XuileStatement:
		g.emitXuile(s)
	case *parser.XuiatchStatement:
		g.emitXuiatch(s)
	case *parser.TryStatement:
		g.emitTry(s)
	case *parser.XudeferStatement:
		g.emitDefer(s)
	case *parser.XuselectStatement:
		g.emitSelect(s)
	case *parser.ExpressionStatement:
		g.writeIndent()
		g.emitExpression(s.Expr)
		g.writeln(";")
	case *parser.BlockStatement:
		g.emitLine("{")
		g.indent++
		g.emitBlock(s)
		g.indent--
		g.emitLine("}")
	}
}

func (g *CCodegen) emitXuet(s *parser.XuetStatement) {
	cType := g.inferCType(s.Value, s.TypeName)
	g.varTypes[s.Name] = cType
	g.writeIndent()
	g.write(fmt.Sprintf("const %s %s = ", cType, s.Name))
	g.emitExpression(s.Value)
	g.writeln(";")
}

func (g *CCodegen) emitXuiar(s *parser.XuiarStatement) {
	cType := g.inferCType(s.Value, s.TypeName)
	g.varTypes[s.Name] = cType
	g.writeIndent()
	g.write(fmt.Sprintf("%s %s = ", cType, s.Name))
	g.emitExpression(s.Value)
	g.writeln(";")
}

func (g *CCodegen) emitAssign(s *parser.AssignStatement) {
	g.writeIndent()
	g.emitExpression(s.Target)
	g.write(" = ")
	g.emitExpression(s.Value)
	g.writeln(";")
}

func (g *CCodegen) emitXueturn(s *parser.XueturnStatement) {
	if s.Value == nil {
		g.emitLine("return;")
		return
	}
	g.writeIndent()
	g.write("return ")
	g.emitExpression(s.Value)
	g.writeln(";")
}

func (g *CCodegen) emitXuif(s *parser.XuifStatement) {
	g.writeIndent()
	g.write("if (")
	g.emitExpression(s.Condition)
	g.writeln(") {")
	g.indent++
	g.emitBlock(s.Consequence)
	g.indent--

	if s.Alternative != nil {
		switch alt := s.Alternative.(type) {
		case *parser.BlockStatement:
			g.emitLine("} else {")
			g.indent++
			g.emitBlock(alt)
			g.indent--
			g.emitLine("}")
		case *parser.XuifStatement:
			g.writeIndent()
			g.write("} else ")
			g.emitXuif(alt)
			return
		}
	} else {
		g.emitLine("}")
	}
}

func (g *CCodegen) emitXuior(s *parser.XuiorStatement) {
	// Check if iterable is a range expression
	if rng, ok := s.Iterable.(*parser.RangeExpression); ok {
		g.writeIndent()
		g.write(fmt.Sprintf("for (int64_t %s = ", s.Variable))
		g.emitExpression(rng.Start)
		g.write(fmt.Sprintf("; %s < ", s.Variable))
		g.emitExpression(rng.End)
		g.write(fmt.Sprintf("; %s++", s.Variable))
		g.writeln(") {")
		g.indent++
		g.emitBlock(s.Body)
		g.indent--
		g.emitLine("}")
	} else {
		// Array iteration using XppArray runtime
		tmpArr := g.newTemp()
		tmpIdx := g.newTemp()
		g.writeIndent()
		g.writeln("{")
		g.indent++
		g.writeIndent()
		g.write(fmt.Sprintf("XppArray* %s = ", tmpArr))
		g.emitExpression(s.Iterable)
		g.writeln(";")
		g.writeIndent()
		g.write(fmt.Sprintf("for (int64_t %s = 0; %s < xpp_array_len(%s); %s++)", tmpIdx, tmpIdx, tmpArr, tmpIdx))
		g.writeln(" {")
		g.indent++
		g.emitLine("int64_t %s = xpp_unbox_int(xpp_array_get(%s, %s));", s.Variable, tmpArr, tmpIdx)
		g.emitBlock(s.Body)
		g.indent--
		g.emitLine("}")
		g.indent--
		g.emitLine("}")
	}
}

func (g *CCodegen) emitXuiorClassic(s *parser.XuiorClassicStatement) {
	g.writeIndent()
	g.writeln("{")
	g.indent++
	// Emit init as statement
	if s.Init != nil {
		g.emitStatement(s.Init)
	}
	g.writeIndent()
	g.write("while (")
	g.emitExpression(s.Condition)
	g.writeln(") {")
	g.indent++
	g.emitBlock(s.Body)
	if s.Post != nil {
		g.emitStatement(s.Post)
	}
	g.indent--
	g.emitLine("}")
	g.indent--
	g.emitLine("}")
}

func (g *CCodegen) emitXuile(s *parser.XuileStatement) {
	g.writeIndent()
	g.write("while (")
	g.emitExpression(s.Condition)
	g.writeln(") {")
	g.indent++
	g.emitBlock(s.Body)
	g.indent--
	g.emitLine("}")
}

func (g *CCodegen) emitXuiatch(s *parser.XuiatchStatement) {
	for i, arm := range s.Arms {
		isDefault := false
		if ident, ok := arm.Pattern.(*parser.Identifier); ok && ident.Value == "_" {
			isDefault = true
		}

		if isDefault {
			if i > 0 {
				g.writeIndent()
				g.writeln("} else {")
			} else {
				g.emitLine("{")
			}
		} else {
			g.writeIndent()
			if i > 0 {
				g.write("} else if (")
			} else {
				g.write("if (")
			}
			// Use xpp_string_eq for string patterns, == for integers
			if sl, ok := arm.Pattern.(*parser.StringLiteral); ok {
				g.write("xpp_string_eq(")
				g.emitExpression(s.Value)
				g.write(fmt.Sprintf(", xpp_string_new(%q))", sl.Value))
			} else {
				g.write("(")
				g.emitExpression(s.Value)
				g.write(") == (")
				g.emitExpression(arm.Pattern)
				g.write(")")
			}
			g.writeln(") {")
		}

		g.indent++
		switch body := arm.Body.(type) {
		case *parser.BlockStatement:
			g.emitBlock(body)
		case *parser.ExpressionStatement:
			g.writeIndent()
			g.emitExpression(body.Expr)
			g.writeln(";")
		default:
			g.emitStatement(arm.Body)
		}
		g.indent--
	}
	g.emitLine("}")
}

func (g *CCodegen) emitTry(s *parser.TryStatement) {
	// Emit try body directly, with error checking via the global flag
	g.emitLine("_xpp_has_error = 0;")
	g.emitLine("{")
	g.indent++
	g.emitBlock(s.Body)
	g.indent--
	g.emitLine("}")
	g.writeIndent()
	g.writeln("if (_xpp_has_error) {")
	g.indent++
	g.emitLine("const char* %s = _xpp_error_msg;", s.CatchVar)
	g.emitBlock(s.CatchBody)
	g.indent--
	g.emitLine("}")
}

func (g *CCodegen) emitDefer(s *parser.XudeferStatement) {
	// Defer requires closures for full support. Emit as a comment
	// documenting the deferred call, since C does not support arbitrary
	// closures. The defer stack is initialized but only usable with
	// function-pointer-based deferred actions.
	g.writeIndent()
	g.write("/* xudefer: ")
	g.emitExpression(s.Call)
	g.writeln(" */")
}

func (g *CCodegen) emitSelect(s *parser.XuselectStatement) {
	// xuselect with channel cases. Generate as if/else chain checking
	// which channel is ready (simplified: first non-default case wins).
	g.emitLine("/* xuselect */")
	g.emitLine("{")
	g.indent++
	for i, c := range s.Cases {
		if c.IsDefault {
			if i > 0 {
				g.emitLine("} else {")
			} else {
				g.emitLine("{")
			}
		} else {
			g.writeIndent()
			if i > 0 {
				g.write("} else if (")
			} else {
				g.write("if (")
			}
			g.emitExpression(c.Channel)
			g.writeln(") {")
		}
		g.indent++
		g.emitBlock(c.Body)
		g.indent--
	}
	g.emitLine("}")
	g.indent--
	g.emitLine("}")
}

// --- Expression emission ---

func (g *CCodegen) emitExpression(expr parser.Expression) {
	switch e := expr.(type) {
	case *parser.IntegerLiteral:
		g.write(fmt.Sprintf("%dLL", e.Value))
	case *parser.FloatLiteral:
		g.write(e.Raw)
	case *parser.StringLiteral:
		g.write(fmt.Sprintf("xpp_string_new(%q)", e.Value))
	case *parser.CharLiteral:
		g.write(fmt.Sprintf("'%c'", e.Value))
	case *parser.BoolLiteral:
		if e.Value {
			g.write("true")
		} else {
			g.write("false")
		}
	case *parser.NullLiteral:
		g.write("NULL")
	case *parser.Identifier:
		g.write(e.Value)
	case *parser.PrefixExpression:
		g.write("(")
		g.write(e.Operator)
		g.emitExpression(e.Right)
		g.write(")")
	case *parser.InfixExpression:
		g.emitInfix(e)
	case *parser.CallExpression:
		g.emitCall(e)
	case *parser.MemberExpression:
		g.emitMember(e)
	case *parser.IndexExpression:
		g.emitIndex(e)
	case *parser.ArrayLiteral:
		g.emitArrayLiteral(e)
	case *parser.RangeExpression:
		// Range as expression — just emit start (used in for loops directly)
		g.emitExpression(e.Start)
	case *parser.StructLiteral:
		g.write(fmt.Sprintf("(struct %s){", e.Name))
		for i, f := range e.Fields {
			if i > 0 {
				g.write(", ")
			}
			g.write(fmt.Sprintf(".%s = ", f.Name))
			g.emitExpression(f.Value)
		}
		g.write("}")
	case *parser.MapLiteral:
		g.emitMapLiteral(e)
	case *parser.InterpolatedString:
		g.emitInterpolatedString(e)
	case *parser.LambdaExpression:
		g.emitLambda(e)
	case *parser.ThrowExpression:
		g.emitThrow(e)
	case *parser.AddressOfExpression:
		g.write("&")
		g.emitExpression(e.Value)
	case *parser.DerefExpression:
		g.write("(*")
		g.emitExpression(e.Value)
		g.write(")")
	}
}

func (g *CCodegen) emitInfix(e *parser.InfixExpression) {
	leftIsStr := g.isStringTypedExpr(e.Left)
	rightIsStr := g.isStringTypedExpr(e.Right)

	// String comparison
	if e.Operator == "==" || e.Operator == "!=" {
		if leftIsStr || rightIsStr {
			if e.Operator == "!=" {
				g.write("!")
			}
			g.write("xpp_string_eq(")
			g.emitExpression(e.Left)
			g.write(", ")
			g.emitExpression(e.Right)
			g.write(")")
			return
		}
	}

	// String concatenation using runtime library
	if e.Operator == "+" && (leftIsStr || rightIsStr) {
		g.write("xpp_string_concat(")
		g.emitExpression(e.Left)
		g.write(", ")
		g.emitExpression(e.Right)
		g.write(")")
		return
	}

	g.write("(")
	g.emitExpression(e.Left)
	g.write(" ")

	// Map operators
	switch e.Operator {
	case "&&":
		g.write("&&")
	case "||":
		g.write("||")
	default:
		g.write(e.Operator)
	}

	g.write(" ")
	g.emitExpression(e.Right)
	g.write(")")
}

func (g *CCodegen) emitCall(e *parser.CallExpression) {
	// Method call: obj.method(args)
	if member, ok := e.Function.(*parser.MemberExpression); ok {
		g.emitMethodCall(member, e.Arguments)
		return
	}

	// Regular function call
	if ident, ok := e.Function.(*parser.Identifier); ok {
		// Built-in function dispatch
		switch ident.Value {
		case "print", "println":
			g.emitPrintCall(e.Arguments)
			return
		case "channel":
			g.write("xpp_channel_new()")
			return
		case "send":
			if len(e.Arguments) >= 2 {
				g.write("xpp_channel_send(")
				g.emitExpression(e.Arguments[0])
				g.write(", xpp_box_int(")
				g.emitExpression(e.Arguments[1])
				g.write("))")
			}
			return
		case "recv":
			if len(e.Arguments) >= 1 {
				g.write("xpp_unbox_int(xpp_channel_recv(")
				g.emitExpression(e.Arguments[0])
				g.write("))")
			}
			return
		case "spawn":
			g.write("/* spawn: requires closure support */ (void)0")
			return
		case "sleep":
			if len(e.Arguments) >= 1 {
				g.write("usleep((unsigned int)(")
				g.emitExpression(e.Arguments[0])
				g.write(") * 1000)")
			}
			return
		case "len":
			if len(e.Arguments) >= 1 {
				arg := e.Arguments[0]
				if ident2, ok := arg.(*parser.Identifier); ok {
					if t, exists := g.varTypes[ident2.Value]; exists {
						if t == "XppString*" {
							g.write("xpp_string_len(")
							g.emitExpression(arg)
							g.write(")")
							return
						}
						if t == "XppMap*" {
							g.write("xpp_map_len(")
							g.emitExpression(arg)
							g.write(")")
							return
						}
					}
				}
				g.write("xpp_array_len(")
				g.emitExpression(arg)
				g.write(")")
			}
			return
		case "push", "append":
			if len(e.Arguments) >= 2 {
				g.write("xpp_array_push(")
				g.emitExpression(e.Arguments[0])
				g.write(", xpp_box_int(")
				g.emitExpression(e.Arguments[1])
				g.write("))")
			}
			return
		case "pop":
			if len(e.Arguments) >= 1 {
				g.write("xpp_unbox_int(xpp_array_pop(")
				g.emitExpression(e.Arguments[0])
				g.write("))")
			}
			return
		case "read_line":
			g.write("xpp_read_line()")
			return
		case "parse_int":
			if len(e.Arguments) >= 1 {
				g.write("xpp_parse_int(")
				g.emitExpression(e.Arguments[0])
				g.write(")")
			}
			return
		case "parse_float":
			if len(e.Arguments) >= 1 {
				g.write("xpp_parse_float(")
				g.emitExpression(e.Arguments[0])
				g.write(")")
			}
			return
		case "int_to_float":
			if len(e.Arguments) >= 1 {
				g.write("xpp_int_to_float(")
				g.emitExpression(e.Arguments[0])
				g.write(")")
			}
			return
		case "float_to_int":
			if len(e.Arguments) >= 1 {
				g.write("xpp_float_to_int(")
				g.emitExpression(e.Arguments[0])
				g.write(")")
			}
			return
		case "to_string":
			if len(e.Arguments) >= 1 {
				arg := e.Arguments[0]
				if ident2, ok := arg.(*parser.Identifier); ok {
					if t, exists := g.varTypes[ident2.Value]; exists {
						switch t {
						case "double":
							g.write("xpp_string_from_float(")
							g.emitExpression(arg)
							g.write(")")
							return
						case "bool":
							g.write("xpp_string_from_bool(")
							g.emitExpression(arg)
							g.write(")")
							return
						case "char":
							g.write("xpp_string_from_char(")
							g.emitExpression(arg)
							g.write(")")
							return
						}
					}
				}
				g.write("xpp_string_from_int(")
				g.emitExpression(arg)
				g.write(")")
			}
			return
		case "sqrt":
			if len(e.Arguments) >= 1 {
				g.write("xpp_math_sqrt(")
				g.emitExpression(e.Arguments[0])
				g.write(")")
			}
			return
		case "pow":
			if len(e.Arguments) >= 2 {
				g.write("xpp_math_pow(")
				g.emitExpression(e.Arguments[0])
				g.write(", ")
				g.emitExpression(e.Arguments[1])
				g.write(")")
			}
			return
		case "floor":
			if len(e.Arguments) >= 1 {
				g.write("xpp_math_floor(")
				g.emitExpression(e.Arguments[0])
				g.write(")")
			}
			return
		case "ceil":
			if len(e.Arguments) >= 1 {
				g.write("xpp_math_ceil(")
				g.emitExpression(e.Arguments[0])
				g.write(")")
			}
			return
		case "abs":
			if len(e.Arguments) >= 1 {
				g.write("xpp_math_abs(")
				g.emitExpression(e.Arguments[0])
				g.write(")")
			}
			return
		case "min":
			if len(e.Arguments) >= 2 {
				g.write("xpp_math_min(")
				g.emitExpression(e.Arguments[0])
				g.write(", ")
				g.emitExpression(e.Arguments[1])
				g.write(")")
			}
			return
		case "max":
			if len(e.Arguments) >= 2 {
				g.write("xpp_math_max(")
				g.emitExpression(e.Arguments[0])
				g.write(", ")
				g.emitExpression(e.Arguments[1])
				g.write(")")
			}
			return
		case "map_new":
			g.write("xpp_map_new()")
			return
		case "map_set":
			if len(e.Arguments) >= 3 {
				g.write("xpp_map_set(")
				g.emitExpression(e.Arguments[0])
				g.write(", ")
				// Key as C string
				if sl, ok := e.Arguments[1].(*parser.StringLiteral); ok {
					g.write(fmt.Sprintf("%q", sl.Value))
				} else {
					g.emitExpression(e.Arguments[1])
					g.write("->data")
				}
				g.write(", xpp_box_int(")
				g.emitExpression(e.Arguments[2])
				g.write("))")
			}
			return
		case "map_get":
			if len(e.Arguments) >= 2 {
				g.write("xpp_unbox_int(xpp_map_get(")
				g.emitExpression(e.Arguments[0])
				g.write(", ")
				if sl, ok := e.Arguments[1].(*parser.StringLiteral); ok {
					g.write(fmt.Sprintf("%q", sl.Value))
				} else {
					g.emitExpression(e.Arguments[1])
					g.write("->data")
				}
				g.write("))")
			}
			return
		case "map_has":
			if len(e.Arguments) >= 2 {
				g.write("xpp_map_has(")
				g.emitExpression(e.Arguments[0])
				g.write(", ")
				if sl, ok := e.Arguments[1].(*parser.StringLiteral); ok {
					g.write(fmt.Sprintf("%q", sl.Value))
				} else {
					g.emitExpression(e.Arguments[1])
					g.write("->data")
				}
				g.write(")")
			}
			return
		case "channel_close":
			if len(e.Arguments) >= 1 {
				g.write("xpp_channel_close(")
				g.emitExpression(e.Arguments[0])
				g.write(")")
			}
			return
		}

		// User functions get xpp_ prefix
		g.write("xpp_" + ident.Value)
	} else {
		g.emitExpression(e.Function)
	}

	g.write("(")
	for i, arg := range e.Arguments {
		if i > 0 {
			g.write(", ")
		}
		g.emitExpression(arg)
	}
	g.write(")")
}

func (g *CCodegen) emitMethodCall(member *parser.MemberExpression, args []parser.Expression) {
	// Look up the struct type from varTypes for the object
	if ident, ok := member.Object.(*parser.Identifier); ok {
		structType := ""
		if t, exists := g.varTypes[ident.Value]; exists {
			// Strip "struct " prefix if present to get the name
			structType = strings.TrimPrefix(t, "struct ")
		}
		if structType != "" && g.implDefs[structType] != nil {
			// Known struct with methods: emit xpp_StructName_method(&obj, ...)
			g.write(fmt.Sprintf("xpp_%s_%s(&%s", structType, member.Member, ident.Value))
			for _, arg := range args {
				g.write(", ")
				g.emitExpression(arg)
			}
			g.write(")")
			return
		}
	}

	// Fallback: emit as a plain method-like call xpp_method(object, ...)
	g.write(fmt.Sprintf("xpp_%s(", member.Member))
	g.emitExpression(member.Object)
	for _, arg := range args {
		g.write(", ")
		g.emitExpression(arg)
	}
	g.write(")")
}

func (g *CCodegen) emitPrintCall(args []parser.Expression) {
	if len(args) == 0 {
		g.write(`printf("\n")`)
		return
	}

	// Single argument: use xpp_print_* runtime functions
	if len(args) == 1 {
		arg := args[0]
		switch e := arg.(type) {
		case *parser.StringLiteral:
			g.write(fmt.Sprintf("xpp_print_string(xpp_string_new(%q))", e.Value))
		case *parser.IntegerLiteral:
			g.write(fmt.Sprintf("xpp_print_int(%dLL)", e.Value))
		case *parser.FloatLiteral:
			g.write(fmt.Sprintf("xpp_print_float(%s)", e.Raw))
		case *parser.BoolLiteral:
			if e.Value {
				g.write("xpp_print_bool(true)")
			} else {
				g.write("xpp_print_bool(false)")
			}
		case *parser.Identifier:
			if t, ok := g.varTypes[e.Value]; ok {
				switch t {
				case "XppString*":
					g.write(fmt.Sprintf("xpp_print_string(%s)", e.Value))
				case "double":
					g.write(fmt.Sprintf("xpp_print_float(%s)", e.Value))
				case "bool":
					g.write(fmt.Sprintf("xpp_print_bool(%s)", e.Value))
				case "char":
					g.write(fmt.Sprintf("xpp_print_char(%s)", e.Value))
				case "XppArray*":
					g.write(fmt.Sprintf("xpp_print_string(xpp_string_from_int(xpp_array_len(%s)))", e.Value))
				case "XppMap*":
					g.write(fmt.Sprintf("xpp_print_string(xpp_string_from_int(xpp_map_len(%s)))", e.Value))
				case "XppChannel*":
					g.write("xpp_print_cstr(\"<channel>\")")
				default:
					g.write(fmt.Sprintf("xpp_print_int(%s)", e.Value))
				}
			} else {
				g.write(fmt.Sprintf("xpp_print_int(%s)", e.Value))
			}
		case *parser.InterpolatedString:
			g.write("xpp_print_string(")
			g.emitInterpolatedString(e)
			g.write(")")
		case *parser.CallExpression:
			// Check if the call returns a string type
			if g.isStringTypedExpr(arg) {
				g.write("xpp_print_string(")
				g.emitExpression(arg)
				g.write(")")
			} else {
				g.write("xpp_print_int(")
				g.emitExpression(arg)
				g.write(")")
			}
		case *parser.InfixExpression:
			if g.isStringTypedExpr(arg) {
				g.write("xpp_print_string(")
				g.emitExpression(arg)
				g.write(")")
			} else {
				g.write("xpp_print_int(")
				g.emitExpression(arg)
				g.write(")")
			}
		default:
			// Default: check if string typed, otherwise int
			if g.isStringTypedExpr(arg) {
				g.write("xpp_print_string(")
				g.emitExpression(arg)
				g.write(")")
			} else {
				g.write("xpp_print_int(")
				g.emitExpression(arg)
				g.write(")")
			}
		}
		return
	}

	// Multiple args: print space-separated using nonl helpers, then newline
	for i, arg := range args {
		if i > 0 {
			g.write("; printf(\" \"); ")
		}
		g.emitPrintSingleNoNl(arg)
	}
	g.write("; printf(\"\\n\")")
}

// emitPrintSingleNoNl emits a single print call WITHOUT a trailing newline
func (g *CCodegen) emitPrintSingleNoNl(arg parser.Expression) {
	switch e := arg.(type) {
	case *parser.StringLiteral:
		g.write(fmt.Sprintf("xpp_print_string_nonl(xpp_string_new(%q))", e.Value))
	case *parser.IntegerLiteral:
		g.write(fmt.Sprintf("xpp_print_int_nonl(%dLL)", e.Value))
	case *parser.FloatLiteral:
		g.write(fmt.Sprintf("xpp_print_float_nonl(%s)", e.Raw))
	case *parser.BoolLiteral:
		if e.Value {
			g.write("xpp_print_bool_nonl(true)")
		} else {
			g.write("xpp_print_bool_nonl(false)")
		}
	case *parser.Identifier:
		if t, ok := g.varTypes[e.Value]; ok {
			switch t {
			case "XppString*":
				g.write(fmt.Sprintf("xpp_print_string_nonl(%s)", e.Value))
			case "double":
				g.write(fmt.Sprintf("xpp_print_float_nonl(%s)", e.Value))
			case "bool":
				g.write(fmt.Sprintf("xpp_print_bool_nonl(%s)", e.Value))
			default:
				g.write(fmt.Sprintf("xpp_print_int_nonl(%s)", e.Value))
			}
		} else {
			g.write(fmt.Sprintf("xpp_print_int_nonl(%s)", e.Value))
		}
	default:
		if g.isStringTypedExpr(arg) {
			g.write("xpp_print_string_nonl(")
			g.emitExpression(arg)
			g.write(")")
		} else {
			g.write("xpp_print_int_nonl(")
			g.emitExpression(arg)
			g.write(")")
		}
	}
}

func (g *CCodegen) isStringTypedExpr(expr parser.Expression) bool {
	if isStringExpr(expr) {
		return true
	}
	if ident, ok := expr.(*parser.Identifier); ok {
		if t, ok := g.varTypes[ident.Value]; ok {
			return t == "XppString*"
		}
	}
	if infix, ok := expr.(*parser.InfixExpression); ok {
		if infix.Operator == "+" && (g.isStringTypedExpr(infix.Left) || g.isStringTypedExpr(infix.Right)) {
			return true
		}
	}
	if _, ok := expr.(*parser.InterpolatedString); ok {
		return true
	}
	return false
}

func (g *CCodegen) emitMember(e *parser.MemberExpression) {
	g.emitExpression(e.Object)
	g.write(".")
	g.write(e.Member)
}

func (g *CCodegen) emitIndex(e *parser.IndexExpression) {
	// Check if the indexed object is a known array type
	if ident, ok := e.Left.(*parser.Identifier); ok {
		if t, exists := g.varTypes[ident.Value]; exists {
			if t == "XppArray*" {
				g.write("xpp_unbox_int(xpp_array_get(")
				g.emitExpression(e.Left)
				g.write(", ")
				g.emitExpression(e.Index)
				g.write("))")
				return
			}
			if t == "XppString*" {
				g.write("xpp_string_char_at(")
				g.emitExpression(e.Left)
				g.write(", ")
				g.emitExpression(e.Index)
				g.write(")")
				return
			}
		}
	}
	// Fallback: C-style index
	g.emitExpression(e.Left)
	g.write("[")
	g.emitExpression(e.Index)
	g.write("]")
}

func (g *CCodegen) emitArrayLiteral(e *parser.ArrayLiteral) {
	if len(e.Elements) == 0 {
		g.write("xpp_array_new(0)")
		return
	}
	// Use GCC statement expression ({...}) for inline array construction
	tmp := g.newTemp()
	g.write("({")
	g.write(fmt.Sprintf(" XppArray* %s = xpp_array_new(%d);", tmp, len(e.Elements)))
	for _, elem := range e.Elements {
		g.write(fmt.Sprintf(" xpp_array_push(%s, xpp_box_int(", tmp))
		g.emitExpression(elem)
		g.write("));")
	}
	g.write(fmt.Sprintf(" %s; })", tmp))
}

func (g *CCodegen) emitMapLiteral(e *parser.MapLiteral) {
	if len(e.Pairs) == 0 {
		g.write("xpp_map_new()")
		return
	}
	// Use GCC statement expression ({...}) for inline map construction
	tmp := g.newTemp()
	g.write("({")
	g.write(fmt.Sprintf(" XppMap* %s = xpp_map_new();", tmp))
	for _, pair := range e.Pairs {
		g.write(fmt.Sprintf(" xpp_map_set(%s, ", tmp))
		// Key must be a C string (const char*)
		if sl, ok := pair.Key.(*parser.StringLiteral); ok {
			g.write(fmt.Sprintf("%q", sl.Value))
		} else {
			// Evaluate the key expression and get ->data
			g.write("(")
			g.emitExpression(pair.Key)
			g.write(")->data")
		}
		g.write(", xpp_box_int(")
		g.emitExpression(pair.Value)
		g.write("));")
	}
	g.write(fmt.Sprintf(" %s; })", tmp))
}

func (g *CCodegen) emitInterpolatedString(e *parser.InterpolatedString) {
	if len(e.Parts) == 0 {
		g.write(`xpp_string_new("")`)
		return
	}
	if len(e.Parts) == 1 {
		g.emitStringPart(e.Parts[0])
		return
	}
	// Build nested xpp_string_concat calls from left to right:
	// For 3 parts: xpp_string_concat(xpp_string_concat(p0, p1), p2)
	for i := 1; i < len(e.Parts); i++ {
		g.write("xpp_string_concat(")
	}
	g.emitStringPart(e.Parts[0])
	for i := 1; i < len(e.Parts); i++ {
		g.write(", ")
		g.emitStringPart(e.Parts[i])
		g.write(")")
	}
}

// emitStringPart emits a single part of an interpolated string as an XppString*
func (g *CCodegen) emitStringPart(expr parser.Expression) {
	switch e := expr.(type) {
	case *parser.StringLiteral:
		g.write(fmt.Sprintf("xpp_string_new(%q)", e.Value))
	case *parser.IntegerLiteral:
		g.write(fmt.Sprintf("xpp_string_from_int(%dLL)", e.Value))
	case *parser.FloatLiteral:
		g.write(fmt.Sprintf("xpp_string_from_float(%s)", e.Raw))
	case *parser.BoolLiteral:
		if e.Value {
			g.write("xpp_string_from_bool(true)")
		} else {
			g.write("xpp_string_from_bool(false)")
		}
	case *parser.Identifier:
		if typ, ok := g.varTypes[e.Value]; ok {
			switch typ {
			case "XppString*":
				g.write(e.Value)
			case "double":
				g.write(fmt.Sprintf("xpp_string_from_float(%s)", e.Value))
			case "bool":
				g.write(fmt.Sprintf("xpp_string_from_bool(%s)", e.Value))
			case "char":
				g.write(fmt.Sprintf("xpp_string_from_char(%s)", e.Value))
			default:
				g.write(fmt.Sprintf("xpp_string_from_int(%s)", e.Value))
			}
		} else {
			g.write(fmt.Sprintf("xpp_string_from_int(%s)", e.Value))
		}
	default:
		// Wrap expression in string conversion
		if g.isStringTypedExpr(expr) {
			g.emitExpression(expr)
		} else {
			g.write("xpp_string_from_int((int64_t)(")
			g.emitExpression(expr)
			g.write("))")
		}
	}
}

func (g *CCodegen) emitLambda(e *parser.LambdaExpression) {
	// C doesn't support closures natively. For GCC we could use nested
	// functions, but portability is poor. Emit as a documented placeholder.
	g.write("/* lambda: requires closure support */ NULL")
}

func (g *CCodegen) emitThrow(e *parser.ThrowExpression) {
	// Set the global error flag and message.
	// If the value is a string literal, extract its data for the C string.
	// Otherwise use the raw C string expression.
	g.write("(xpp_throw(")
	if sl, ok := e.Value.(*parser.StringLiteral); ok {
		g.write(fmt.Sprintf("%q", sl.Value))
	} else {
		g.write("(")
		g.emitExpression(e.Value)
		g.write(")->data")
	}
	g.write("), (void)0)")
}

// --- Helpers ---

func (g *CCodegen) inferCType(expr parser.Expression, declaredType string) string {
	if declaredType != "" {
		return g.mapType(declaredType)
	}
	switch e := expr.(type) {
	case *parser.IntegerLiteral:
		return "int64_t"
	case *parser.FloatLiteral:
		return "double"
	case *parser.StringLiteral:
		return "XppString*"
	case *parser.BoolLiteral:
		return "bool"
	case *parser.CharLiteral:
		return "char"
	case *parser.ArrayLiteral:
		return "XppArray*"
	case *parser.MapLiteral:
		return "XppMap*"
	case *parser.StructLiteral:
		return "struct " + e.Name
	case *parser.InfixExpression:
		if e.Operator == "+" && (g.isStringTypedExpr(e.Left) || g.isStringTypedExpr(e.Right)) {
			return "XppString*"
		}
		// Check if either side is a float
		if g.isFloatTypedExpr(e.Left) || g.isFloatTypedExpr(e.Right) {
			return "double"
		}
		return "int64_t"
	case *parser.InterpolatedString:
		return "XppString*"
	case *parser.CallExpression:
		// Infer type from known built-in functions
		if ident, ok := e.Function.(*parser.Identifier); ok {
			switch ident.Value {
			case "channel":
				return "XppChannel*"
			case "recv":
				return "int64_t"
			case "len":
				return "int64_t"
			case "read_line", "to_string":
				return "XppString*"
			case "parse_int", "float_to_int", "abs", "min", "max":
				return "int64_t"
			case "parse_float", "int_to_float", "sqrt", "pow", "floor", "ceil":
				return "double"
			case "map_new":
				return "XppMap*"
			case "map_has":
				return "bool"
			}
		}
		return "int64_t"
	case *parser.Identifier:
		if t, ok := g.varTypes[e.Value]; ok {
			return t
		}
		return "int64_t"
	case *parser.NullLiteral:
		return "void*"
	}
	return "int64_t" // default
}

func (g *CCodegen) isFloatTypedExpr(expr parser.Expression) bool {
	if _, ok := expr.(*parser.FloatLiteral); ok {
		return true
	}
	if ident, ok := expr.(*parser.Identifier); ok {
		if t, ok := g.varTypes[ident.Value]; ok {
			return t == "double"
		}
	}
	return false
}

func isStringExpr(expr parser.Expression) bool {
	_, ok := expr.(*parser.StringLiteral)
	return ok
}
