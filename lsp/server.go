package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/00000kkkkk/xusesosplusplus/lexer"
	"github.com/00000kkkkk/xusesosplusplus/parser"
	"github.com/00000kkkkk/xusesosplusplus/typechecker"
)

// Server implements a minimal LSP server for Xuesos++.
type Server struct {
	reader *bufio.Reader
	writer io.Writer
	mu     sync.Mutex
	docs   map[string]string // uri -> content
}

// NewServer creates a new LSP server reading from stdin, writing to stdout.
func NewServer() *Server {
	return &Server{
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
		docs:   make(map[string]string),
	}
}

// Run starts the LSP server main loop.
func (s *Server) Run() error {
	for {
		msg, err := s.readMessage()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		s.handleMessage(msg)
	}
}

func (s *Server) readMessage() (json.RawMessage, error) {
	// Read headers
	contentLength := 0
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			contentLength, _ = strconv.Atoi(val)
		}
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("no content length")
	}

	body := make([]byte, contentLength)
	_, err := io.ReadFull(s.reader, body)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(body), nil
}

func (s *Server) sendMessage(msg interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	body, _ := json.Marshal(msg)
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	s.writer.Write([]byte(header))
	s.writer.Write(body)
}

type jsonrpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func (s *Server) handleMessage(raw json.RawMessage) {
	var msg jsonrpcMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}

	switch msg.Method {
	case "initialize":
		s.handleInitialize(msg)
	case "initialized":
		// no-op
	case "textDocument/didOpen":
		s.handleDidOpen(msg)
	case "textDocument/didChange":
		s.handleDidChange(msg)
	case "textDocument/didClose":
		s.handleDidClose(msg)
	case "textDocument/completion":
		s.handleCompletion(msg)
	case "textDocument/hover":
		s.handleHover(msg)
	case "shutdown":
		s.sendMessage(jsonrpcResponse{JSONRPC: "2.0", ID: msg.ID, Result: nil})
	case "exit":
		os.Exit(0)
	}
}

func (s *Server) handleInitialize(msg jsonrpcMessage) {
	result := map[string]interface{}{
		"capabilities": map[string]interface{}{
			"textDocumentSync": 1, // Full sync
			"diagnosticProvider": map[string]interface{}{
				"interFileDependencies": false,
				"workspaceDiagnostics":  false,
			},
			"completionProvider": map[string]interface{}{
				"triggerCharacters": []string{".", "("},
			},
			"hoverProvider": true,
		},
		"serverInfo": map[string]interface{}{
			"name":    "xuesos-lsp",
			"version": "0.1.0",
		},
	}
	s.sendMessage(jsonrpcResponse{JSONRPC: "2.0", ID: msg.ID, Result: result})
}

func (s *Server) handleDidOpen(msg jsonrpcMessage) {
	var params struct {
		TextDocument struct {
			URI  string `json:"uri"`
			Text string `json:"text"`
		} `json:"textDocument"`
	}
	json.Unmarshal(msg.Params, &params)

	s.docs[params.TextDocument.URI] = params.TextDocument.Text
	s.publishDiagnostics(params.TextDocument.URI, params.TextDocument.Text)
}

func (s *Server) handleDidChange(msg jsonrpcMessage) {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		ContentChanges []struct {
			Text string `json:"text"`
		} `json:"contentChanges"`
	}
	json.Unmarshal(msg.Params, &params)

	if len(params.ContentChanges) > 0 {
		text := params.ContentChanges[len(params.ContentChanges)-1].Text
		s.docs[params.TextDocument.URI] = text
		s.publishDiagnostics(params.TextDocument.URI, text)
	}
}

func (s *Server) handleDidClose(msg jsonrpcMessage) {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	json.Unmarshal(msg.Params, &params)

	delete(s.docs, params.TextDocument.URI)
	// Clear diagnostics
	s.sendMessage(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "textDocument/publishDiagnostics",
		"params": map[string]interface{}{
			"uri":         params.TextDocument.URI,
			"diagnostics": []interface{}{},
		},
	})
}

func (s *Server) publishDiagnostics(uri, text string) {
	var diagnostics []interface{}

	// Lex
	l := lexer.New(uri, text)
	_, lexErrs := l.ScanAll()
	for _, e := range lexErrs {
		diagnostics = append(diagnostics, makeDiagnostic(e.Pos.Line, e.Pos.Column, e.Message, 1))
	}

	// Parse
	l2 := lexer.New(uri, text)
	tokens, _ := l2.ScanAll()
	p := parser.New(tokens)
	program, parseErrs := p.Parse()
	for _, e := range parseErrs {
		diagnostics = append(diagnostics, makeDiagnostic(e.Pos.Line, e.Pos.Column, e.Message, 1))
	}

	// Type check
	if program != nil && len(lexErrs) == 0 && len(parseErrs) == 0 {
		tc := typechecker.New()
		typeErrs := tc.Check(program)
		for _, e := range typeErrs {
			diagnostics = append(diagnostics, makeDiagnostic(e.Pos.Line, e.Pos.Column, e.Message, 2))
		}
	}

	s.sendMessage(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "textDocument/publishDiagnostics",
		"params": map[string]interface{}{
			"uri":         uri,
			"diagnostics": diagnostics,
		},
	})
}

func (s *Server) handleCompletion(msg jsonrpcMessage) {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Position struct {
			Line      int `json:"line"`
			Character int `json:"character"`
		} `json:"position"`
	}
	json.Unmarshal(msg.Params, &params)

	var items []interface{}

	// Get current document text
	text := s.docs[params.TextDocument.URI]

	// Get the current line and word being typed
	lines := strings.Split(text, "\n")
	currentLine := ""
	if params.Position.Line < len(lines) {
		currentLine = lines[params.Position.Line]
	}

	// Get the prefix (word being typed)
	prefix := ""
	col := params.Position.Character
	if col <= len(currentLine) {
		start := col
		for start > 0 && isAlphaNum(currentLine[start-1]) {
			start--
		}
		prefix = currentLine[start:col]
	}

	// Keywords
	keywords := []struct{ label, detail string }{
		{"xuen", "Function declaration"},
		{"xuet", "Immutable variable (let)"},
		{"xuiar", "Mutable variable (var)"},
		{"xuif", "If statement"},
		{"xuelse", "Else branch"},
		{"xuior", "For loop"},
		{"xuile", "While loop"},
		{"xuin", "In (for iteration)"},
		{"xueturn", "Return from function"},
		{"xuieak", "Break loop"},
		{"xuitinue", "Continue loop"},
		{"xuiruct", "Struct declaration"},
		{"xuimpl", "Impl block (methods)"},
		{"xuenum", "Enum declaration"},
		{"xuiatch", "Match/switch statement"},
		{"xuimport", "Import module"},
		{"xuinterface", "Interface declaration"},
		{"xudefer", "Defer statement"},
		{"xutry", "Try block"},
		{"xucatch", "Catch block"},
		{"xuthrow", "Throw error"},
		{"xuselect", "Select on channels"},
		{"xuitru", "Boolean true"},
		{"xuinia", "Boolean false"},
		{"xuinull", "Null value"},
	}

	// Type keywords
	types := []struct{ label, detail string }{
		{"int", "64-bit integer"},
		{"float", "64-bit float"},
		{"str", "UTF-8 string"},
		{"bool", "Boolean"},
		{"char", "Character"},
		{"byte", "Byte (uint8)"},
	}

	// Built-in functions (most common ones)
	builtins := []struct{ label, detail string }{
		{"print", "print(args...) — Print to stdout"},
		{"println", "println(args...) — Print with newline"},
		{"len", "len(x) — Length of string/array/map"},
		{"type", "type(x) — Type name as string"},
		{"append", "append(arr, val) — Append to array"},
		{"push", "push(arr, val) — Push to array in-place"},
		{"split", "split(str, sep) — Split string"},
		{"join", "join(arr, sep) — Join array"},
		{"contains", "contains(str, sub) — Check substring"},
		{"trim", "trim(str) — Trim whitespace"},
		{"upper", "upper(str) — Uppercase"},
		{"lower", "lower(str) — Lowercase"},
		{"replace", "replace(str, old, new) — Replace"},
		{"sort_arr", "sort_arr(arr) — Sort array"},
		{"sort_by", "sort_by(arr, cmp) — Custom sort"},
		{"reverse", "reverse(x) — Reverse string/array"},
		{"keys", "keys(map) — Get map keys"},
		{"values", "values(map) — Get map values"},
		{"has_key", "has_key(map, key) — Check key"},
		{"json_parse", "json_parse(str) — Parse JSON"},
		{"json_stringify", "json_stringify(val) — To JSON"},
		{"http_get", "http_get(url) — HTTP GET"},
		{"http_post", "http_post(url, body) — HTTP POST"},
		{"http_serve", "http_serve(addr, handler) — HTTP server"},
		{"spawn", "spawn(func) — Run goroutine"},
		{"channel", "channel(size?) — Create channel"},
		{"send", "send(ch, val) — Send to channel"},
		{"recv", "recv(ch) — Receive from channel"},
		{"sleep", "sleep(ms) — Sleep milliseconds"},
		{"wg_new", "wg_new() — Create WaitGroup"},
		{"mutex_new", "mutex_new() — Create Mutex"},
		{"tuple", "tuple(args...) — Create tuple"},
		{"first", "first(tuple) — First element"},
		{"second", "second(tuple) — Second element"},
		{"assert_eq", "assert_eq(a, b) — Assert equality"},
		{"benchmark", "benchmark(name, fn, n) — Run benchmark"},
		{"regex_match", "regex_match(pat, str) — Regex test"},
		{"sha256", "sha256(str) — SHA-256 hash"},
		{"base64_encode", "base64_encode(str) — Encode"},
		{"template", "template(tmpl, data) — Render template"},
		{"io_read_file", "io_read_file(path) — Read file"},
		{"io_write_file", "io_write_file(path, data) — Write file"},
		{"file_exists", "file_exists(path) — Check file"},
		{"format", "format(tmpl, args...) — Format string"},
		{"abs", "abs(x) — Absolute value"},
		{"max", "max(a, b) — Maximum"},
		{"min", "min(a, b) — Minimum"},
		{"sqrt", "sqrt(x) — Square root"},
		{"error_new", "error_new(msg) — Create error"},
		{"is_err", "is_err(val) — Check if error"},
		{"fields", "fields(struct) — Field names"},
		{"type_name", "type_name(val) — Type name"},
		{"exit", "exit(code) — Exit program"},
	}

	// Add matching keywords
	for _, kw := range keywords {
		if prefix == "" || strings.HasPrefix(kw.label, prefix) {
			items = append(items, map[string]interface{}{
				"label":  kw.label,
				"kind":   14, // Keyword
				"detail": kw.detail,
			})
		}
	}

	// Add matching types
	for _, tp := range types {
		if prefix == "" || strings.HasPrefix(tp.label, prefix) {
			items = append(items, map[string]interface{}{
				"label":  tp.label,
				"kind":   25, // TypeParameter
				"detail": tp.detail,
			})
		}
	}

	// Add matching builtins
	for _, fn := range builtins {
		if prefix == "" || strings.HasPrefix(fn.label, prefix) {
			items = append(items, map[string]interface{}{
				"label":            fn.label,
				"kind":             3, // Function
				"detail":           fn.detail,
				"insertText":       fn.label + "($0)",
				"insertTextFormat": 2, // Snippet
			})
		}
	}

	s.sendMessage(jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"isIncomplete": false,
			"items":        items,
		},
	})
}

func isAlphaNum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func (s *Server) handleHover(msg jsonrpcMessage) {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Position struct {
			Line      int `json:"line"`
			Character int `json:"character"`
		} `json:"position"`
	}
	json.Unmarshal(msg.Params, &params)

	text := s.docs[params.TextDocument.URI]
	lines := strings.Split(text, "\n")

	word := ""
	if params.Position.Line < len(lines) {
		line := lines[params.Position.Line]
		col := params.Position.Character
		start, end := col, col
		for start > 0 && isAlphaNum(line[start-1]) {
			start--
		}
		for end < len(line) && isAlphaNum(line[end]) {
			end++
		}
		if start < end {
			word = line[start:end]
		}
	}

	// Look up hover info
	hoverInfo := getHoverInfo(word)
	if hoverInfo == "" {
		s.sendMessage(jsonrpcResponse{JSONRPC: "2.0", ID: msg.ID, Result: nil})
		return
	}

	s.sendMessage(jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"contents": map[string]string{
				"kind":  "markdown",
				"value": hoverInfo,
			},
		},
	})
}

func getHoverInfo(word string) string {
	hovers := map[string]string{
		"xuen":    "**xuen** — Function declaration\n```xuesos\nxuen name(params) returnType { body }\n```",
		"xuet":    "**xuet** — Immutable variable declaration\n```xuesos\nxuet name = value\n```",
		"xuiar":   "**xuiar** — Mutable variable declaration\n```xuesos\nxuiar name = value\nname = newValue  // OK\n```",
		"xuif":    "**xuif** — Conditional statement\n```xuesos\nxuif (condition) { body }\n```",
		"xuior":   "**xuior** — For loop\n```xuesos\nxuior (i xuin 0..10) { body }\nxuior (xuiar i = 0 : i < 10 : i = i + 1) { body }\n```",
		"xuile":   "**xuile** — While loop\n```xuesos\nxuile (condition) { body }\n```",
		"xuiatch": "**xuiatch** — Pattern matching\n```xuesos\nxuiatch (value) {\n    pattern => body\n    _ => default\n}\n```",
		"print":   "**print(args...)** — Print values to stdout with newline",
		"len":     "**len(x)** — Returns length of string, array, or map",
		"spawn":   "**spawn(func)** — Run function in a goroutine",
		"channel": "**channel(size?)** — Create a channel for goroutine communication",
		"xudefer": "**xudefer** — Execute statement when function/program exits (LIFO order)",
		"xutry":   "**xutry** — Try block for error handling\n```xuesos\nxutry { body } xucatch (e) { handler }\n```",
	}
	if info, ok := hovers[word]; ok {
		return info
	}
	return ""
}

func makeDiagnostic(line, col int, message string, severity int) map[string]interface{} {
	if line < 1 {
		line = 1
	}
	if col < 1 {
		col = 1
	}
	return map[string]interface{}{
		"range": map[string]interface{}{
			"start": map[string]int{"line": line - 1, "character": col - 1},
			"end":   map[string]int{"line": line - 1, "character": col + 10},
		},
		"severity": severity,
		"source":   "xuesos",
		"message":  message,
	}
}
