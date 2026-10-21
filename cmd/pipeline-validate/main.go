// Command pipeline-validate valida um relatório de delivery antes do commit.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate"
)

const (
	exitOK       = validate.ExitOK
	exitFail     = validate.ExitFail
	exitBadUsage = validate.ExitBadUsage
)

func run(reportPath, baseRef string, noGit bool, stdout, stderr io.Writer) int {
	return validate.Run(reportPath, baseRef, noGit, stdout, stderr)
}

func main() {
	var (
		baseRef = flag.String("base", "", "referência git para o diff (default: base_ref do config)")
		noGit   = flag.Bool("no-git", false, "pula as checagens que dependem do git")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "uso: pipeline-validate [flags] <caminho-do-relatorio.md>\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(exitBadUsage)
	}

	os.Exit(run(flag.Arg(0), *baseRef, *noGit, os.Stdout, os.Stderr))
}
