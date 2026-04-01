package xuesos

import "github.com/00000kkkkk/xusesosplusplus/lsp"

func runLsp() error {
	server := lsp.NewServer()
	return server.Run()
}
