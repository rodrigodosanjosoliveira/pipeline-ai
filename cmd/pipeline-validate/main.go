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

func runCLI(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("pipeline-validate", flag.ContinueOnError)
	fs.SetOutput(stderr)

	baseRef := fs.String("base", "", "referência git para o diff (default: base_ref do config)")
	noGit := fs.Bool("no-git", false, "pula as checagens que dependem do git")
	asJSON := fs.Bool("json", false, "emite o resultado estruturado em JSON")

	fs.Usage = func() {
		fmt.Fprintf(stderr, "uso: pipeline-validate [flags] <caminho-do-relatorio.md>\n\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return exitBadUsage
	}

	if fs.NArg() != 1 {
		fs.Usage()
		return exitBadUsage
	}

	if *asJSON {
		return validate.RunJSON(fs.Arg(0), *baseRef, *noGit, stdout, stderr)
	}
	return run(fs.Arg(0), *baseRef, *noGit, stdout, stderr)
}

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}
