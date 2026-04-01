.PHONY: build run test clean install repl

# Build the xuesos compiler
build:
	go build -o xuesos .

# Run a .xpp file (usage: make run FILE=examples/hello.xpp)
run:
	go run . run $(FILE)

# Build a .xpp file to binary (usage: make compile FILE=examples/fibonacci.xpp)
compile:
	go run . build $(FILE)

# Start the REPL
repl:
	go run . repl

# Run all tests
test:
	go test ./... -v

# Run tests with coverage
coverage:
	go test ./... -cover

# Clean build artifacts
clean:
	rm -f xuesos *.c *.exe
	rm -f examples/*.c examples/*.exe

# Install globally
install:
	go install .

# Show version
version:
	go run . version
