// Command pipeline-hook é o hook PreToolUse que impede a sessão principal de
// editar código diretamente, forçando a delegação aos agents do pipeline.
package main

import (
	"io"
	"os"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/hook"
)

const (
	exitAllow = hook.ExitAllow
	exitBlock = hook.ExitBlock
)

func run(stdin io.Reader, stdout, stderr io.Writer) int {
	return hook.Run(stdin, stdout, stderr)
}

func main() {
	os.Exit(run(os.Stdin, os.Stdout, os.Stderr))
}
