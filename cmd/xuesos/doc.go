package xuesos

import (
	"fmt"
	"os"
	"strings"

	"github.com/00000kkkkk/xusesosplusplus/lexer"
	"github.com/00000kkkkk/xusesosplusplus/parser"
)

func runDoc(args []string) error {
	if len(args) == 0 {
		return printBuiltinDocs()
	}

	filename := args[0]
	src, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", filename, err)
	}

	l := lexer.New(filename, string(src))
	tokens, _ := l.ScanAll()
	p := parser.New(tokens)
	program, _ := p.Parse()

	if program == nil {
		return fmt.Errorf("cannot parse %s", filename)
	}

	fmt.Printf("Documentation for %s\n", filename)
	fmt.Println(strings.Repeat("=", 40))
	fmt.Println()

	for _, stmt := range program.Statements {
		switch s := stmt.(type) {
		case *parser.XuenStatement:
			params := formatDocParams(s.Params)
			ret := s.ReturnType
			if ret == "" {
				ret = "void"
			}
			fmt.Printf("xuen %s(%s) %s\n", s.Name, params, ret)
			// Look for comment above (heuristic: check if previous line in source is a comment)
			if s.Pos.Line > 1 {
				lines := strings.Split(string(src), "\n")
				if s.Pos.Line-2 < len(lines) {
					prev := strings.TrimSpace(lines[s.Pos.Line-2])
					if strings.HasPrefix(prev, "//") {
						fmt.Printf("  %s\n", strings.TrimPrefix(prev, "// "))
					}
				}
			}
			fmt.Println()

		case *parser.XuiructStatement:
			fmt.Printf("xuiruct %s\n", s.Name)
			for _, f := range s.Fields {
				fmt.Printf("  %s %s\n", f.Name, f.TypeName)
			}
			fmt.Println()

		case *parser.XuimplStatement:
			fmt.Printf("xuimpl %s\n", s.Name)
			for _, m := range s.Methods {
				params := formatDocParams(m.Params)
				ret := m.ReturnType
				if ret == "" {
					ret = "void"
				}
				fmt.Printf("  xuen %s(%s) %s\n", m.Name, params, ret)
			}
			fmt.Println()

		case *parser.XuenumStatement:
			fmt.Printf("xuenum %s\n", s.Name)
			for _, v := range s.Variants {
				fmt.Printf("  %s\n", v)
			}
			fmt.Println()

		case *parser.XuinterfaceStatement:
			fmt.Printf("xuinterface %s\n", s.Name)
			for _, m := range s.Methods {
				ret := m.ReturnType
				if ret == "" {
					ret = "void"
				}
				fmt.Printf("  %s(%s) %s\n", m.Name, strings.Join(m.ParamTypes, ", "), ret)
			}
			fmt.Println()
		}
	}

	return nil
}

func formatDocParams(params []parser.Parameter) string {
	var parts []string
	for _, p := range params {
		if p.TypeName == "" {
			parts = append(parts, p.Name)
		} else {
			parts = append(parts, p.Name+" "+p.TypeName)
		}
	}
	return strings.Join(parts, ", ")
}

func printBuiltinDocs() error {
	fmt.Println("Xuesos++ Built-in Functions")
	fmt.Println(strings.Repeat("=", 40))
	fmt.Println()

	categories := []struct {
		name  string
		funcs []string
	}{
		{"I/O", []string{
			"print(args...)           Print values to stdout",
			"println(args...)         Print values with newline",
			"input(prompt)            Read line from stdin",
			"printf(template, args...)  Formatted print",
		}},
		{"Type Conversion", []string{
			"to_int(x)               Convert to int",
			"to_float(x)             Convert to float",
			"to_str(x)               Convert to string",
			"cast_int(x)             Force cast to int (parses strings)",
			"cast_float(x)           Force cast to float",
			"cast_str(x)             Force cast to string",
			"cast_bool(x)            Force cast to bool",
		}},
		{"String", []string{
			"len(s)                  String/array length",
			"contains(s, sub)        Check substring",
			"starts_with(s, pre)     Check prefix",
			"ends_with(s, suf)       Check suffix",
			"split(s, sep)           Split by separator",
			"join(arr, sep)          Join array to string",
			"trim(s)                 Trim whitespace",
			"upper(s)                Uppercase",
			"lower(s)                Lowercase",
			"replace(s, old, new)    Replace all occurrences",
			"repeat(s, n)            Repeat string",
			"reverse(s)              Reverse string/array",
			"count(s, sub)           Count occurrences",
			"index_of(s, sub)        Find substring index",
			"substr(s, start, len)   Get substring",
			"format(tmpl, args...)   Format string with {} placeholders",
		}},
		{"Array", []string{
			"append(arr, val)        Append (returns new array)",
			"push(arr, val)          Append in-place",
			"slice(arr, start, end)  Get sub-array",
			"sort_arr(arr)           Sort array",
			"unique(arr)             Remove duplicates",
			"flatten(arr)            Flatten nested arrays",
			"zip(a, b)              Zip two arrays",
			"enumerate(arr)          Array of (index, value) tuples",
			"range_arr(start, end)   Create array from range",
		}},
		{"Map", []string{
			"has_key(m, key)         Check key exists",
			"keys(m)                 Get all keys",
			"values(m)               Get all values",
			"delete(m, key)          Delete key",
		}},
		{"Math", []string{
			"abs(x)                  Absolute value",
			"max(a, b)               Maximum",
			"min(a, b)               Minimum",
			"sqrt(x)                 Square root",
			"math_pi()               Pi constant",
			"math_e()                Euler's number",
			"math_pow(base, exp)     Power",
			"math_floor(x)           Floor",
			"math_ceil(x)            Ceiling",
			"math_round(x)           Round",
			"math_sin/cos/log(x)     Trig and log",
			"math_random()           Random float [0,1)",
			"math_rand_int(min,max)  Random int [min,max)",
		}},
		{"Concurrency", []string{
			"spawn(func)             Run in goroutine",
			"channel(size?)          Create channel",
			"send(ch, val)           Send to channel",
			"recv(ch)                Receive from channel",
			"sleep(ms)               Sleep milliseconds",
			"wg_new()                Create WaitGroup",
			"wg_add(wg, n?)         Add to WaitGroup",
			"wg_done(wg)             Mark done",
			"wg_wait(wg)             Wait for completion",
			"mutex_new()             Create Mutex",
			"mutex_lock(mu)          Lock",
			"mutex_unlock(mu)        Unlock",
		}},
		{"Tuple", []string{
			"tuple(args...)          Create tuple",
			"first(t)                First element",
			"second(t)               Second element",
			"unpack(t, idx)          Get by index",
		}},
		{"Regex", []string{
			"regex_match(pat, s)     Test regex match",
			"regex_find(pat, s)      Find first match",
			"regex_find_all(pat, s)  Find all matches",
			"regex_replace(pat,s,r)  Replace matches",
		}},
		{"Crypto/Encoding", []string{
			"sha256(s)               SHA-256 hash",
			"md5_hash(s)             MD5 hash",
			"base64_encode(s)        Base64 encode",
			"base64_decode(s)        Base64 decode",
			"url_encode(s)           URL encode",
			"url_decode(s)           URL decode",
		}},
		{"File System", []string{
			"io_read_file(path)      Read file contents",
			"io_write_file(path, s)  Write file",
			"file_exists(path)       Check file exists",
			"list_dir(path)          List directory",
			"mkdir(path)             Create directory",
			"remove(path)            Delete file",
			"path_join(parts...)     Join path parts",
		}},
		{"HTTP", []string{
			"http_get(url)           HTTP GET, returns body",
			"http_status(url)        HTTP GET, returns status code",
		}},
		{"JSON", []string{
			"json_parse(s)           Parse JSON to value",
			"json_stringify(v)       Value to JSON string",
		}},
		{"OS/System", []string{
			"os_args()               Command line arguments",
			"os_getenv(key)          Get env variable",
			"os_setenv(key, val)     Set env variable",
			"os_cwd()                Current directory",
			"os_hostname()           Hostname",
			"os_platform()           OS name",
			"os_arch()               CPU architecture",
			"exit(code)              Exit program",
			"panic(msg)              Panic and exit",
		}},
		{"Testing", []string{
			"assert(cond, msg?)      Assert condition",
			"assert_eq(a, b)         Assert equality",
			"benchmark(name,fn,n)    Run benchmark",
			"time_now()              Current time (ms)",
			"time_since(start)       Elapsed time (ms)",
		}},
		{"Type Info", []string{
			"type(x)                 Type name as string",
			"sizeof(x)              Size info",
			"implements(v, iface)    Check interface",
			"is_error(x)             Check if error value",
		}},
	}

	for _, cat := range categories {
		fmt.Printf("--- %s ---\n", cat.name)
		for _, f := range cat.funcs {
			fmt.Printf("  %s\n", f)
		}
		fmt.Println()
	}

	return nil
}
