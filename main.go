package main

import (
	"fmt"
	"os"

	xuesos "github.com/00000kkkkk/xusesosplusplus/cmd/xuesos"
)

func main() {
	if err := xuesos.Execute(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
