package main

import (
	"os"

	swcli "scriptweaver/internal/cli/sw"
)

func main() {
	os.Exit(swcli.Main(os.Args[1:], os.Stdout, os.Stderr))
}
