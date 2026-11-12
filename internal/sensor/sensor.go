// Package sensor executa baterias de sensores de qualidade (fmt, vet, test) com saída estruturada e salvaguarda contra falsos positivos.
package sensor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config"
)

// DefaultMaxOutputBytes define o limite padrão de bytes por saída de sensor (8 KB).
const DefaultMaxOutputBytes = 8192

// Options configura limites de saída e nível de verbosidade da execução dos sensores.
type Options struct {
	MaxOutputBytes int  `json:"max_output_bytes,omitempty"`
	VerboseOutput  bool `json:"verbose_output,omitempty"`
}

// SensorStatus representa o status de execução de um sensor.
type SensorStatus string

const (
	StatusPass SensorStatus = "PASS"
	StatusFail SensorStatus = "FAIL"
)

// CheckResult descreve o resultado da execução de um sensor individual.
type CheckResult struct {
	Name         string       `json:"name"`
	Status       SensorStatus `json:"status"`
	Message      string       `json:"message"`
	Output       string       `json:"output,omitempty"`
	Truncated    bool         `json:"truncated,omitempty"`
	TotalBytes   int          `json:"total_bytes,omitempty"`
	OmittedBytes int          `json:"omitted_bytes,omitempty"`
	FailedTests  []string     `json:"failed_tests,omitempty"`
}

// Result representa o resultado consolidado e estruturado de todos os sensores.
type Result struct {
	Overall  SensorStatus  `json:"overall"`
	OK       bool          `json:"ok"`
	TestsRun int           `json:"tests_run"`
	Sensors  []CheckResult `json:"sensors"`
	Message  string        `json:"message,omitempty"`
}

func findBinary(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	candidates := []string{
		filepath.Join("/usr/local/go/bin", name),
		filepath.Join("/usr/bin", name),
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, "go", "bin", name))
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c
		}
	}
	return name
}

// BoundOutput limita o tamanho da saída em bytes garantindo corte seguro em fronteira de runa UTF-8.
func BoundOutput(raw string, maxBytes int) (bounded string, truncated bool, totalBytes int, omittedBytes int) {
	totalBytes = len(raw)
	if maxBytes <= 0 {
		maxBytes = DefaultMaxOutputBytes
	}
	if totalBytes <= maxBytes {
		return raw, false, totalBytes, 0
	}

	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(raw[cut]) {
		cut--
	}
	return raw[:cut], true, totalBytes, totalBytes - cut
}

// BoundTestOutput extrai os nomes dos testes falhos, filtra blocos de testes aprovados (quando !verbose)
// e limita o tamanho em bytes preservando fronteiras UTF-8 válidas.
func BoundTestOutput(raw string, maxBytes int, verbose bool) (bounded string, failedTests []string, truncated bool, totalBytes int, omittedBytes int) {
	totalBytes = len(raw)
	if maxBytes <= 0 {
		maxBytes = DefaultMaxOutputBytes
	}

	lines := strings.Split(raw, "\n")
	seenFail := make(map[string]bool)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--- FAIL: ") {
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "--- FAIL: "))
			name := rest
			if idx := strings.IndexAny(rest, " ("); idx >= 0 {
				name = rest[:idx]
			}
			if name != "" && !seenFail[name] {
				seenFail[name] = true
				failedTests = append(failedTests, name)
			}
		}
	}

	candidate := raw
	if !verbose && totalBytes > 0 {
		var kept []string
		var block []string
		inBlock := false

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "=== RUN ") {
				if inBlock && len(block) > 0 {
					kept = append(kept, block...)
				}
				block = []string{line}
				inBlock = true
				continue
			}
			if strings.HasPrefix(trimmed, "=== PAUSE ") || strings.HasPrefix(trimmed, "=== CONT ") {
				continue
			}
			if inBlock {
				block = append(block, line)
				if strings.HasPrefix(trimmed, "--- PASS:") || strings.HasPrefix(trimmed, "--- SKIP:") {
					block = nil
					inBlock = false
				} else if strings.HasPrefix(trimmed, "--- FAIL:") {
					kept = append(kept, block...)
					block = nil
					inBlock = false
				}
				continue
			}
			if trimmed == "PASS" {
				continue
			}
			kept = append(kept, line)
		}
		if inBlock && len(block) > 0 {
			kept = append(kept, block...)
		}
		candidate = strings.Join(kept, "\n")
	}

	if len(candidate) > maxBytes {
		cut := maxBytes
		for cut > 0 && !utf8.RuneStart(candidate[cut]) {
			cut--
		}
		bounded = candidate[:cut]
	} else {
		bounded = candidate
	}

	if len(bounded) < totalBytes {
		truncated = true
		omittedBytes = totalBytes - len(bounded)
	}
	return bounded, failedTests, truncated, totalBytes, omittedBytes
}

// Run executa os sensores de qualidade no diretório indicado com as opções padrão.
func Run(dir string) (*Result, error) {
	return RunWithOptions(dir, Options{MaxOutputBytes: DefaultMaxOutputBytes})
}

// RunWithOptions executa os sensores de qualidade no diretório indicado respeitando Options.
func RunWithOptions(dir string, opts Options) (*Result, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("caminho inválido: %w", err)
	}

	cfg, _ := config.Load(absDir)
	var cmdEnv []string
	if cfg != nil && cfg.ProjectType == "desktop" {
		cmdEnv = append(os.Environ(), "CGO_ENABLED=0", "GOOS=windows", "GOARCH=amd64")
	}

	var checks []CheckResult
	overallOK := true

	// 1. Sensor FMT: gofmt -l .
	gofmtBin := findBinary("gofmt")
	fmtCmd := exec.Command(gofmtBin, "-l", ".")
	fmtCmd.Dir = absDir
	fmtOut, fmtErr := fmtCmd.CombinedOutput()
	trimmedFmt := strings.TrimSpace(string(fmtOut))

	var fmtCheck CheckResult
	fmtCheck.Name = "fmt"
	if fmtErr != nil || len(trimmedFmt) > 0 {
		fmtCheck.Status = StatusFail
		overallOK = false
		if fmtErr != nil && len(trimmedFmt) == 0 {
			fmtCheck.Message = fmt.Sprintf("Erro ao executar gofmt: %v", fmtErr)
		} else {
			lines := strings.Split(trimmedFmt, "\n")
			fmtCheck.Message = fmt.Sprintf("Arquivos fora do padrão de formatação (%d): %s", len(lines), strings.Join(lines, ", "))
			fmtCheck.Output, fmtCheck.Truncated, fmtCheck.TotalBytes, fmtCheck.OmittedBytes = BoundOutput(trimmedFmt, opts.MaxOutputBytes)
		}
	} else {
		fmtCheck.Status = StatusPass
		fmtCheck.Message = "Todos os arquivos estão formatados segundo o padrão gofmt."
	}
	checks = append(checks, fmtCheck)

	// 2. Sensor VET: go vet ./...
	goBin := findBinary("go")
	vetCmd := exec.Command(goBin, "vet", "./...")
	vetCmd.Dir = absDir
	if cmdEnv != nil {
		vetCmd.Env = cmdEnv
	}
	vetOut, vetErr := vetCmd.CombinedOutput()
	trimmedVet := strings.TrimSpace(string(vetOut))

	var vetCheck CheckResult
	vetCheck.Name = "vet"
	if vetErr != nil {
		vetCheck.Status = StatusFail
		vetCheck.Message = "go vet encontrou violações de conformidade estática."
		vetCheck.Output, vetCheck.Truncated, vetCheck.TotalBytes, vetCheck.OmittedBytes = BoundOutput(trimmedVet, opts.MaxOutputBytes)
		overallOK = false
	} else {
		vetCheck.Status = StatusPass
		vetCheck.Message = "go vet executado com sucesso sem avisos ou erros."
	}
	checks = append(checks, vetCheck)

	// 3. Sensor TEST: go test -v ./...
	testCmd := exec.Command(goBin, "test", "-v", "./...")
	testCmd.Dir = absDir
	if cmdEnv != nil {
		testCmd.Env = cmdEnv
	}
	testOut, testErr := testCmd.CombinedOutput()
	testStr := string(testOut)

	var testsRun, testsPass, testsFail, testsSkip int
	for _, line := range strings.Split(testStr, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--- PASS:") {
			testsRun++
			testsPass++
		} else if strings.HasPrefix(trimmed, "--- FAIL:") {
			testsRun++
			testsFail++
		} else if strings.HasPrefix(trimmed, "--- SKIP:") {
			testsRun++
			testsSkip++
		}
	}

	var testCheck CheckResult
	testCheck.Name = "test"
	trimmedTest := strings.TrimSpace(testStr)
	testCheck.Output, testCheck.FailedTests, testCheck.Truncated, testCheck.TotalBytes, testCheck.OmittedBytes = BoundTestOutput(trimmedTest, opts.MaxOutputBytes, opts.VerboseOutput)

	// Salvaguarda contra falso positivo: zero testes executados reprova
	if testsRun == 0 {
		testCheck.Status = StatusFail
		testCheck.Message = "Salvaguarda ativada: zero testes executados (falso positivo evitado / no tests run)."
		overallOK = false
	} else if testErr != nil || testsFail > 0 {
		testCheck.Status = StatusFail
		testCheck.Message = fmt.Sprintf("Falha na execução dos testes (%d falhas, %d pulados, %d passaram de %d no total).", testsFail, testsSkip, testsPass, testsRun)
		overallOK = false
	} else {
		testCheck.Status = StatusPass
		testCheck.Message = fmt.Sprintf("Todos os testes passaram (%d testes executados, %d pulados).", testsRun, testsSkip)
	}
	checks = append(checks, testCheck)

	var overallStatus SensorStatus = StatusPass
	var message string
	if !overallOK {
		overallStatus = StatusFail
		if testsRun == 0 {
			message = "Salvaguarda ativada: zero testes executados (falso positivo evitado)."
		} else {
			message = "Um ou mais sensores de qualidade reprovaram a verificação."
		}
	} else {
		message = fmt.Sprintf("Todos os sensores passaram com sucesso (%d testes executados).", testsRun)
	}

	return &Result{
		Overall:  overallStatus,
		OK:       overallOK,
		TestsRun: testsRun,
		Sensors:  checks,
		Message:  message,
	}, nil
}
