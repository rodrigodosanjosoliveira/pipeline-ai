package doctor

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config"
)

type CheckStatus string

const (
	StatusPass CheckStatus = "PASS"
	StatusWarn CheckStatus = "WARN"
	StatusFail CheckStatus = "FAIL"
)

type CheckResult struct {
	Name    string      `json:"name"`
	Status  CheckStatus `json:"status"`
	Message string      `json:"message"`
}

// Report representa o diagnóstico estruturado do ambiente para serialização JSON.
type Report struct {
	OK      bool          `json:"ok"`
	Results []CheckResult `json:"results"`
}

var goVersionRegex = regexp.MustCompile(`go version go(\d+)\.(\d+)`)

// ParseGoVersion extrai as versões major e minor da saída de "go version".
func ParseGoVersion(output string) (int, int, bool) {
	matches := goVersionRegex.FindStringSubmatch(output)
	if len(matches) < 3 {
		return 0, 0, false
	}
	major, err1 := strconv.Atoi(matches[1])
	minor, err2 := strconv.Atoi(matches[2])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return major, minor, true
}

// CheckGo valida se a versão instalada do Go é >= 1.24.
func CheckGo() CheckResult {
	cmd := exec.Command("go", "version")
	out, err := cmd.Output()
	if err != nil {
		return CheckResult{
			Name:    "Go Runtime",
			Status:  StatusFail,
			Message: "comando 'go' não encontrado no PATH",
		}
	}
	raw := strings.TrimSpace(string(out))
	major, minor, ok := ParseGoVersion(raw)
	if !ok {
		return CheckResult{
			Name:    "Go Runtime",
			Status:  StatusFail,
			Message: fmt.Sprintf("versão do Go não reconhecida: %s", raw),
		}
	}
	if major < 1 || (major == 1 && minor < 24) {
		return CheckResult{
			Name:    "Go Runtime",
			Status:  StatusFail,
			Message: fmt.Sprintf("versão %d.%d incompatível: requer Go 1.24+ para diretiva tool (instalada: %s)", major, minor, raw),
		}
	}
	return CheckResult{
		Name:    "Go Runtime",
		Status:  StatusPass,
		Message: raw,
	}
}

// CheckToolchain valida utilitários essenciais da toolchain.
func CheckToolchain() []CheckResult {
	var results []CheckResult
	tools := []struct {
		bin      string
		required bool
		purpose  string
	}{
		{"gofmt", true, "formatação determinística (PostToolUse hook)"},
		{"git", true, "controle de versão e pre-push hook"},
		{"gh", false, "abertura de Pull Requests via CLI"},
	}

	for _, t := range tools {
		path, err := exec.LookPath(t.bin)
		if err != nil {
			status := StatusWarn
			if t.required {
				status = StatusFail
			}
			results = append(results, CheckResult{
				Name:    fmt.Sprintf("Toolchain: %s", t.bin),
				Status:  status,
				Message: fmt.Sprintf("não encontrado no PATH (%s)", t.purpose),
			})
		} else {
			results = append(results, CheckResult{
				Name:    fmt.Sprintf("Toolchain: %s", t.bin),
				Status:  StatusPass,
				Message: path,
			})
		}
	}
	return results
}

// CheckConfig valida a existência e integridade do .pipeline-ai.json.
func CheckConfig(rootDir string) CheckResult {
	cfg, err := config.Load(rootDir)
	if err != nil {
		return CheckResult{
			Name:    "Configuração (.pipeline-ai.json)",
			Status:  StatusFail,
			Message: err.Error(),
		}
	}

	details := fmt.Sprintf("tipo=%s, %d caminhos protegidos, %d agentes editores",
		cfg.ProjectType, len(cfg.ProtectedPaths), len(cfg.EditorAgents))
	return CheckResult{
		Name:    "Configuração (.pipeline-ai.json)",
		Status:  StatusPass,
		Message: details,
	}
}

// CheckGit valida o repositório git e a instalação de hooks locais.
func CheckGit(rootDir string) CheckResult {
	gitDir := filepath.Join(rootDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return CheckResult{
			Name:    "Git e Hooks",
			Status:  StatusWarn,
			Message: "diretório .git não encontrado",
		}
	}

	hasPrePush := false
	for _, p := range []string{
		filepath.Join(rootDir, ".githooks", "pre-push"),
		filepath.Join(gitDir, "hooks", "pre-push"),
	} {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			hasPrePush = true
			break
		}
	}

	if !hasPrePush {
		return CheckResult{
			Name:    "Git e Hooks",
			Status:  StatusWarn,
			Message: "hook pre-push não encontrado em .githooks/pre-push ou .git/hooks/pre-push",
		}
	}

	return CheckResult{
		Name:    "Git e Hooks",
		Status:  StatusPass,
		Message: "repositório git inicializado e hook pre-push configurado",
	}
}

// CheckAgents valida a integridade do diretório .agents.
func CheckAgents(rootDir string) CheckResult {
	agentsDir := filepath.Join(rootDir, ".agents")
	if _, err := os.Stat(agentsDir); os.IsNotExist(err) {
		return CheckResult{
			Name:    "Antigravity (.agents)",
			Status:  StatusWarn,
			Message: "diretório .agents não encontrado (execute 'pipeline-ai sync' para instalar)",
		}
	}

	hooksFile := filepath.Join(agentsDir, "hooks.json")
	if _, err := os.Stat(hooksFile); os.IsNotExist(err) {
		return CheckResult{
			Name:    "Antigravity (.agents)",
			Status:  StatusWarn,
			Message: ".agents/hooks.json não encontrado",
		}
	}

	skillsDir := filepath.Join(agentsDir, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil || len(entries) == 0 {
		return CheckResult{
			Name:    "Antigravity (.agents)",
			Status:  StatusWarn,
			Message: ".agents/skills está vazio ou inacessível",
		}
	}

	return CheckResult{
		Name:    "Antigravity (.agents)",
		Status:  StatusPass,
		Message: fmt.Sprintf(".agents/hooks.json presente, %d skills instaladas", len(entries)),
	}
}

// Collect executa todas as verificações do doctor e retorna os resultados e se o ambiente está aprovado.
func Collect(rootDir string) ([]CheckResult, bool) {
	var results []CheckResult
	results = append(results, CheckGo())
	results = append(results, CheckToolchain()...)
	results = append(results, CheckConfig(rootDir))
	results = append(results, CheckGit(rootDir))
	results = append(results, CheckAgents(rootDir))

	hasFail := false
	for _, res := range results {
		if res.Status == StatusFail {
			hasFail = true
		}
	}
	return results, !hasFail
}

// Run executa todas as verificações do doctor e imprime o relatório em stdout.
// Retorna 0 se tudo passar (ou avisos), 1 se houver falhas críticas.
func Run(rootDir string, stdout, stderr io.Writer) int {
	fmt.Fprintln(stdout, "=== pipeline-ai doctor ===")
	fmt.Fprintln(stdout)

	results, ok := Collect(rootDir)
	for _, res := range results {
		var icon string
		switch res.Status {
		case StatusPass:
			icon = "✓"
		case StatusWarn:
			icon = "!"
		case StatusFail:
			icon = "✗"
		}
		fmt.Fprintf(stdout, " [%s] %-32s : %s\n", icon, res.Name, res.Message)
	}

	fmt.Fprintln(stdout)
	if !ok {
		fmt.Fprintln(stdout, "Resultado: Uma ou mais verificações críticas falharam.")
		return 1
	}
	fmt.Fprintln(stdout, "Resultado: Ambiente validado com sucesso!")
	return 0
}

// RunJSON executa todas as verificações do doctor e emite o diagnóstico estruturado em formato JSON no stdout.
// Retorna 0 se aprovado (ou avisos), 1 se houver falhas críticas.
func RunJSON(rootDir string, stdout io.Writer) int {
	results, ok := Collect(rootDir)
	rep := Report{
		OK:      ok,
		Results: results,
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rep); err != nil {
		return 1
	}
	if !ok {
		return 1
	}
	return 0
}
