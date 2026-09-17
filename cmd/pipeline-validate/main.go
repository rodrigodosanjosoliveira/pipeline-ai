// Command pipeline-validate valida um relatório de delivery antes do commit.
//
// Além das checagens internas ao relatório (pacote report), faz duas checagens
// cruzadas que exigem acesso ao repositório:
//
//  1. registro no INDEX;
//  2. salvaguarda do require_security (D-14): se o diff tocar superfície de
//     segurança e a flag estiver desligada, a delivery é reprovada. Isso impede
//     que um esquecimento do triage deixe passar uma entrega sem revisão.
//
// Saída:
//
//	exit 0 → aprovado
//	exit 1 → reprovado
//	exit 2 → erro de uso ou ambiente
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report"
)

const (
	exitOK       = 0
	exitFail     = 1
	exitBadUsage = 2
)

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

func run(reportPath, baseRef string, noGit bool, stdout, stderr *os.File) int {
	absReport, err := filepath.Abs(reportPath)
	if err != nil {
		fmt.Fprintf(stderr, "caminho inválido: %v\n", err)
		return exitBadUsage
	}

	content, err := os.ReadFile(absReport)
	if err != nil {
		fmt.Fprintf(stderr, "falha ao ler o relatório: %v\n", err)
		return exitBadUsage
	}

	cfg, err := config.Load(filepath.Dir(absReport))
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return exitBadUsage
	}

	r, err := report.Parse(string(content))
	if err != nil {
		fmt.Fprintf(stderr, "ERRO: %v\n", err)
		return exitFail
	}

	findings := r.Validate()
	findings = append(findings, checkIndex(cfg, r)...)

	if !noGit {
		if baseRef == "" {
			baseRef = cfg.BaseRef
		}
		findings = append(findings, checkSecuritySurface(cfg, r, baseRef)...)
	}

	return printResult(stdout, filepath.Base(absReport), findings)
}

// checkIndex confirma que a delivery está registrada no índice.
func checkIndex(cfg *config.Config, r *report.Report) []report.Finding {
	id := strings.TrimSpace(r.Front["delivery_id"])
	if id == "" {
		return nil // já reportado pela validação de front matter
	}

	indexPath := filepath.Join(cfg.Root(), filepath.FromSlash(cfg.IndexFile))
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return []report.Finding{{
			Level:   "ERRO",
			Message: fmt.Sprintf("índice não encontrado em %s: %v", cfg.IndexFile, err),
		}}
	}
	if !strings.Contains(string(data), id) {
		return []report.Finding{{
			Level:   "ERRO",
			Message: fmt.Sprintf("delivery %s não está registrada em %s", id, cfg.IndexFile),
		}}
	}
	return nil
}

// checkSecuritySurface é a salvaguarda do D-14.
func checkSecuritySurface(cfg *config.Config, r *report.Report, baseRef string) []report.Finding {
	changed, err := changedFiles(cfg.Root(), baseRef)
	if err != nil {
		// Sem git não dá para afirmar nada. Vira aviso, não erro, para não
		// reprovar uma delivery por causa do ambiente.
		return []report.Finding{{
			Level:   "AVISO",
			Message: fmt.Sprintf("não foi possível calcular o diff contra %s (%v); a salvaguarda de segurança não rodou", baseRef, err),
		}}
	}

	var touched []string
	for _, f := range changed {
		if cfg.IsSecuritySurface(f) {
			touched = append(touched, f)
		}
	}
	if len(touched) == 0 {
		return nil
	}
	if r.Flag("require_security") {
		return nil
	}

	return []report.Finding{{
		Level: "ERRO",
		Message: fmt.Sprintf(
			"o diff toca superfície de segurança (%s) mas require_security está desligado; ligue a flag e preencha a seção Security",
			strings.Join(touched, ", "),
		),
	}}
}

func changedFiles(root, baseRef string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", baseRef+"...HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

func printResult(stdout *os.File, name string, findings []report.Finding) int {
	var erros, avisos int
	for _, f := range findings {
		if f.Level == "ERRO" {
			erros++
		} else {
			avisos++
		}
	}

	fmt.Fprintf(stdout, "pipeline-validate: %s\n", name)
	if len(findings) == 0 {
		fmt.Fprintln(stdout, "APROVADO: nenhum problema encontrado.")
		return exitOK
	}

	for _, f := range findings {
		fmt.Fprintf(stdout, "  %s\n", f)
	}
	fmt.Fprintf(stdout, "\n%d erro(s), %d aviso(s)\n", erros, avisos)

	if erros > 0 {
		fmt.Fprintln(stdout, "REPROVADO")
		return exitFail
	}
	fmt.Fprintln(stdout, "APROVADO com avisos")
	return exitOK
}
