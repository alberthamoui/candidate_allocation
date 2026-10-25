//go:build !debug && !dev

package main

import (
	"fmt"
	"io"
)

func runDebugCLI(_ []string, _ io.Writer) error {
	return fmt.Errorf("diagnósticos indisponíveis nesta versão; compile com -tags debug")
}
