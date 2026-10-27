// Package sensor executa baterias de sensores de qualidade (fmt, vet, test) com saída estruturada e salvaguarda contra falsos positivos.
package sensor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config"
)

// SensorStatus representa o status de execução de um sensor.
type SensorStatus string

const (
	StatusPass SensorStatus = "PASS"
	StatusFail SensorStatus = "FAIL"
)

// CheckResult descreve o resultado da execução de um sensor individual.
type CheckResult struct {
	Name    string       `json:"name"`
	Status  SensorStatus `json:"status"`
	Message string       `json:"message"`
	Output  string       `json:"output,omitempty"`
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

// Run executa os sensores de qualidade no diretório indicado.
func Run(dir string) (*Result, error) {
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
			fmtCheck.Output = trimmedFmt
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
		vetCheck.Output = trimmedVet
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
	testCheck.Output = strings.TrimSpace(testStr)

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
