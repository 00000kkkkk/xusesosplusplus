package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/00000kkkkk/xusesosplusplus/interpreter"
	"github.com/00000kkkkk/xusesosplusplus/lexer"
	"github.com/00000kkkkk/xusesosplusplus/parser"
)

type RunRequest struct {
	Code string `json:"code"`
}

type RunResponse struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
	Time   string `json:"time"`
}

func runHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(200)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(RunResponse{Error: "invalid request"})
		return
	}

	start := time.Now()
	output, execErr := executeCode(req.Code)
	elapsed := time.Since(start)

	resp := RunResponse{
		Output: output,
		Time:   elapsed.Round(time.Microsecond).String(),
	}
	if execErr != "" {
		resp.Error = execErr
	}

	json.NewEncoder(w).Encode(resp)
}

func executeCode(code string) (string, string) {
	// Lex
	l := lexer.New("playground.xpp", code)
	tokens, lexErrs := l.ScanAll()
	if len(lexErrs) > 0 {
		var errs []string
		for _, e := range lexErrs {
			errs = append(errs, e.Error())
		}
		return "", strings.Join(errs, "\n")
	}

	// Parse
	p := parser.New(tokens)
	program, parseErrs := p.Parse()
	if len(parseErrs) > 0 {
		var errs []string
		for _, e := range parseErrs {
			errs = append(errs, e.Error())
		}
		return "", strings.Join(errs, "\n")
	}

	// Interpret with timeout
	interp := interpreter.New()

	done := make(chan error, 1)
	go func() {
		done <- interp.Run(program)
	}()

	select {
	case err := <-done:
		output := strings.Join(interp.Output(), "\n")
		if err != nil {
			return output, err.Error()
		}
		return output, ""
	case <-time.After(5 * time.Second):
		return "", "execution timeout (5s limit)"
	}
}

const htmlPage = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Xuesos++ Playground</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: 'Segoe UI', system-ui, sans-serif; background: #0d1117; color: #e6edf3; }
.header { background: #161b22; padding: 16px 24px; border-bottom: 1px solid #30363d; display: flex; align-items: center; justify-content: space-between; }
.header h1 { font-size: 20px; font-weight: 600; }
.header h1 span { color: #58a6ff; }
.header .badges { display: flex; gap: 8px; font-size: 12px; }
.badge { background: #21262d; padding: 4px 10px; border-radius: 12px; border: 1px solid #30363d; }
.container { display: grid; grid-template-columns: 1fr 1fr; height: calc(100vh - 56px); }
.panel { display: flex; flex-direction: column; }
.panel-header { background: #161b22; padding: 8px 16px; font-size: 13px; font-weight: 600; color: #8b949e; border-bottom: 1px solid #30363d; display: flex; justify-content: space-between; align-items: center; }
#editor { flex: 1; background: #0d1117; color: #e6edf3; font-family: 'Fira Code', 'Cascadia Code', monospace; font-size: 14px; line-height: 1.6; padding: 16px; border: none; outline: none; resize: none; tab-size: 4; }
#output { flex: 1; background: #0d1117; color: #3fb950; font-family: 'Fira Code', monospace; font-size: 14px; line-height: 1.6; padding: 16px; overflow-y: auto; white-space: pre-wrap; }
.error { color: #f85149; }
.time { color: #8b949e; font-size: 12px; margin-top: 8px; border-top: 1px solid #30363d; padding-top: 8px; }
.btn { background: #238636; color: white; border: none; padding: 6px 16px; border-radius: 6px; cursor: pointer; font-size: 13px; font-weight: 600; }
.btn:hover { background: #2ea043; }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-examples { background: #21262d; border: 1px solid #30363d; }
.btn-examples:hover { background: #30363d; }
.examples-dropdown { position: relative; display: inline-block; }
.examples-menu { display: none; position: absolute; right: 0; top: 100%; background: #161b22; border: 1px solid #30363d; border-radius: 8px; min-width: 200px; z-index: 10; padding: 4px 0; margin-top: 4px; }
.examples-menu.show { display: block; }
.examples-menu a { display: block; padding: 8px 16px; color: #e6edf3; text-decoration: none; font-size: 13px; }
.examples-menu a:hover { background: #21262d; }
.right-panel { border-left: 1px solid #30363d; }
@media (max-width: 768px) { .container { grid-template-columns: 1fr; grid-template-rows: 1fr 1fr; } .right-panel { border-left: none; border-top: 1px solid #30363d; } }
</style>
</head>
<body>
<div class="header">
    <h1>Xuesos<span>++</span> Playground</h1>
    <div style="display:flex;gap:8px;align-items:center">
        <div class="examples-dropdown">
            <button class="btn btn-examples" onclick="toggleExamples()">Examples &#9662;</button>
            <div class="examples-menu" id="examplesMenu">
                <a href="#" onclick="loadExample('hello')">Hello World</a>
                <a href="#" onclick="loadExample('fibonacci')">Fibonacci</a>
                <a href="#" onclick="loadExample('fizzbuzz')">FizzBuzz</a>
                <a href="#" onclick="loadExample('structs')">Structs</a>
                <a href="#" onclick="loadExample('closures')">Closures</a>
                <a href="#" onclick="loadExample('maps')">Maps</a>
                <a href="#" onclick="loadExample('concurrency')">Concurrency</a>
                <a href="#" onclick="loadExample('errors')">Error Handling</a>
            </div>
        </div>
        <button class="btn" id="runBtn" onclick="runCode()">&#9654; Run</button>
    </div>
</div>
<div class="container">
    <div class="panel">
        <div class="panel-header">Code</div>
        <textarea id="editor" spellcheck="false">// Welcome to Xuesos++ Playground!
xuen main() {
    print("Hello from Xuesos++!")

    xuior (i xuin 1..6) {
        print("  {i} squared = {i * i}")
    }

    print("Done!")
}</textarea>
    </div>
    <div class="panel right-panel">
        <div class="panel-header">
            <span>Output</span>
            <span id="timing"></span>
        </div>
        <div id="output">Press "Run" or Ctrl+Enter to execute...</div>
    </div>
</div>
<script>
var examples = {
    hello: 'xuen main() {\n    print("Hello from Xuesos++!")\n}',
    fibonacci: 'xuen fib(n int) int {\n    xuif (n <= 1) {\n        xueturn n\n    }\n    xueturn fib(n - 1) + fib(n - 2)\n}\n\nxuen main() {\n    xuior (i xuin 0..15) {\n        print("fib({i}) = {fib(i)}")\n    }\n}',
    fizzbuzz: 'xuen main() {\n    xuior (i xuin 1..21) {\n        xuif (i % 15 == 0) {\n            print("FizzBuzz")\n        } xuelse xuif (i % 3 == 0) {\n            print("Fizz")\n        } xuelse xuif (i % 5 == 0) {\n            print("Buzz")\n        } xuelse {\n            print(i)\n        }\n    }\n}',
    structs: 'xuiruct Player {\n    name str\n    health int\n}\n\nxuimpl Player {\n    xuen hit(xuiar self, dmg int) {\n        self.health = self.health - dmg\n    }\n    xuen info(self) str {\n        xueturn "{self.name}: {self.health}hp"\n    }\n}\n\nxuen main() {\n    xuiar p = Player { name = "Hero", health = 100 }\n    print(p.info())\n    p.hit(30)\n    print(p.info())\n}',
    closures: 'xuen main() {\n    xuiar count = 0\n    xuen inc() int {\n        count = count + 1\n        xueturn count\n    }\n    print(inc())\n    print(inc())\n    print(inc())\n}',
    maps: 'xuen main() {\n    xuiar scores = {"alice": 90, "bob": 85, "charlie": 95}\n    print("Scores: {scores}")\n    print("Alice: {scores[\\"alice\\"]}")\n    scores["david"] = 88\n    print("Keys: {keys(scores)}")\n    print("Count: {len(scores)}")\n}',
    concurrency: 'xuen main() {\n    xuet ch = channel(5)\n    xuior (i xuin 0..5) {\n        send(ch, i * 10)\n    }\n    xuior (i xuin 0..5) {\n        print("Received: {recv(ch)}")\n    }\n}',
    errors: 'xuen divide(a int, b int) {\n    xuif (b == 0) {\n        xueturn tuple(0, error_new("division by zero"))\n    }\n    xueturn tuple(a / b, xuinull)\n}\n\nxuen main() {\n    xuet r1 = divide(10, 3)\n    print("10/3 = {first(r1)}")\n    \n    xuet r2 = divide(10, 0)\n    xuif (second(r2) != xuinull) {\n        print("Error: {second(r2)[\\"message\\"]}")\n    }\n}'
};

function loadExample(name) {
    document.getElementById('editor').value = examples[name];
    document.getElementById('examplesMenu').classList.remove('show');
}

function toggleExamples() {
    document.getElementById('examplesMenu').classList.toggle('show');
}

document.addEventListener('click', function(e) {
    if (!e.target.closest('.examples-dropdown')) {
        document.getElementById('examplesMenu').classList.remove('show');
    }
});

function runCode() {
    var btn = document.getElementById('runBtn');
    var output = document.getElementById('output');
    var timing = document.getElementById('timing');
    var code = document.getElementById('editor').value;

    btn.disabled = true;
    btn.textContent = 'Running...';
    output.innerHTML = 'Executing...';

    fetch('/api/run', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({code: code})
    })
    .then(function(resp) { return resp.json(); })
    .then(function(data) {
        var html = '';
        if (data.output) html += escapeHtml(data.output);
        if (data.error) html += (html ? '\n' : '') + '<span class="error">' + escapeHtml(data.error) + '</span>';
        if (!html) html = '<span style="color:#8b949e">(no output)</span>';
        output.innerHTML = html;
        timing.textContent = data.time || '';
    })
    .catch(function(e) {
        output.innerHTML = '<span class="error">Network error: ' + e.message + '</span>';
    })
    .finally(function() {
        btn.disabled = false;
        btn.innerHTML = '&#9654; Run';
    });
}

function escapeHtml(s) {
    return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}

document.getElementById('editor').addEventListener('keydown', function(e) {
    if (e.ctrlKey && e.key === 'Enter') { e.preventDefault(); runCode(); }
    if (e.key === 'Tab') {
        e.preventDefault();
        var t = e.target;
        var start = t.selectionStart;
        t.value = t.value.substring(0, start) + '    ' + t.value.substring(t.selectionEnd);
        t.selectionStart = t.selectionEnd = start + 4;
    }
});
</script>
</body>
</html>`

func main() {
	http.HandleFunc("/api/run", runHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, htmlPage)
	})

	addr := ":3000"
	fmt.Printf("Xuesos++ Playground running at http://localhost%s\n", addr)
	http.ListenAndServe(addr, nil)
}
