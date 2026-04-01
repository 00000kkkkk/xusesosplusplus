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
