// Command sharur-sandbox is a standalone gRPC extension that restricts sharur
// file-system tools to the directory it is started in.
//
// Build:
//
//	cd examples/sandbox && go build -o sharur-sandbox .
//
// Use:
//
//	shr --extension /path/to/sharur-sandbox "What files are here?"
package main

import (
	"os"

	"github.com/goppydae/sharur/extensions"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	ext, err := newSandbox(root)
	if err != nil {
		panic(err)
	}
	extensions.Serve(ext)
}
