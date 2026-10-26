// Package validate implementa a validação de relatórios de delivery e conformidade do diff.
package validate

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report"
)

const (
	ExitOK       = 0
	ExitFail     = 1
	ExitBadUsage = 2
)

// Run executa a validação do relatório de delivery.
func Run(reportPath, baseRef string, noGit bool, stdout, stderr io.Writer) int {
	absReport, err := filepath.Abs(reportPath)
	if err != nil {
		fmt.Fprintf(stderr, "caminho inválido: %v\n", err)
		return ExitBadUsage
	}

	content, err := os.ReadFile(absReport)
	if err != nil {
		fmt.Fprintf(stderr, "falha ao ler o relatório: %v\n", err)
		return ExitBadUsage
	}

	cfg, err := config.Load(filepath.Dir(absReport))
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return ExitBadUsage
	}

	r, err := report.Parse(string(content))
	if err != nil {
		fmt.Fprintf(stderr, "ERRO: %v\n", err)
		return ExitFail
	}

	findings := r.ValidateWithConfig(cfg)
	findings = append(findings, checkIndex(cfg, r)...)

	if !noGit {
		if baseRef == "" {
			baseRef = cfg.BaseRef
		}
		findings = append(findings, checkSecuritySurface(cfg, r, baseRef)...)
	}

	return printResult(stdout, filepath.Base(absReport), findings)
}

// ValidationResult representa o resultado estruturado da validação para serialização JSON.
type ValidationResult struct {
	Status   string           `json:"status"`
	OK       bool             `json:"ok"`
	Report   string           `json:"report"`
	Errors   int              `json:"errors"`
	Warnings int              `json:"warnings"`
	Findings []report.Finding `json:"findings"`
}

// RunJSON executa a validação emitindo o resultado estruturado em JSON no stdout.
func RunJSON(reportPath, baseRef string, noGit bool, stdout, stderr io.Writer) int {
	absReport, err := filepath.Abs(reportPath)
	if err != nil {
		fmt.Fprintf(stderr, "caminho inválido: %v\n", err)
		return ExitBadUsage
	}

	content, err := os.ReadFile(absReport)
	if err != nil {
		fmt.Fprintf(stderr, "falha ao ler o relatório: %v\n", err)
		return ExitBadUsage
	}

	cfg, err := config.Load(filepath.Dir(absReport))
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return ExitBadUsage
	}

	r, err := report.Parse(string(content))
	if err != nil {
		fmt.Fprintf(stderr, "ERRO: %v\n", err)
		return ExitFail
	}

	findings := r.ValidateWithConfig(cfg)
	findings = append(findings, checkIndex(cfg, r)...)

	if !noGit {
		if baseRef == "" {
			baseRef = cfg.BaseRef
		}
		findings = append(findings, checkSecuritySurface(cfg, r, baseRef)...)
	}

	var nErros, nAvisos int
	for _, f := range findings {
		switch f.Level {
		case "ERRO":
			nErros++
		case "AVISO":
			nAvisos++
		}
	}

	status := "PASS"
	ok := true
	if nErros > 0 {
		status = "FAIL"
		ok = false
	}
	if findings == nil {
		findings = []report.Finding{}
	}

	res := ValidationResult{
		Status:   status,
		OK:       ok,
		Report:   filepath.Base(absReport),
		Errors:   nErros,
		Warnings: nAvisos,
		Findings: findings,
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(res); err != nil {
		fmt.Fprintf(stderr, "erro ao serializar JSON: %v\n", err)
		return ExitBadUsage
	}

	if ok {
		return ExitOK
	}
	return ExitFail
}

func checkIndex(cfg *config.Config, r *report.Report) []report.Finding {
	if cfg != nil && !cfg.IsIndexRequired() {
		return nil
	}
	id := strings.TrimSpace(r.Front["delivery_id"])
	if id == "" {
		return nil
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

func checkSecuritySurface(cfg *config.Config, r *report.Report, baseRef string) []report.Finding {
	changed, err := changedFiles(cfg.Root(), baseRef)
	if err != nil {
		return []report.Finding{{
			Level:   "ERRO",
			Message: fmt.Sprintf("não foi possível calcular o diff contra %s (%v); use --no-git se quiser ignorar a checagem", baseRef, err),
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
	commands := [][]string{
		{"diff", "--name-only", baseRef + "...HEAD"},
		{"diff", "--name-only", "HEAD"},
		{"diff", "--cached", "--name-only"},
		{"ls-files", "--others", "--exclude-standard"},
	}

	seen := make(map[string]bool)
	var files []string

	for _, args := range commands {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.Output()
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !seen[line] {
				seen[line] = true
				files = append(files, line)
			}
		}
	}
	return files, nil
}

func printResult(stdout io.Writer, name string, findings []report.Finding) int {
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
		return ExitOK
	}

	for _, f := range findings {
		fmt.Fprintf(stdout, "  %s\n", f)
	}
	fmt.Fprintf(stdout, "\n%d erro(s), %d aviso(s)\n", erros, avisos)

	if erros > 0 {
		fmt.Fprintln(stdout, "REPROVADO")
		return ExitFail
	}
	fmt.Fprintln(stdout, "APROVADO com avisos")
	return ExitOK
}
