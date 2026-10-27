package sensor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir falhou: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFile falhou: %v", err)
	}
}

func TestRunSensors_Success(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "go.mod"), "module testmod\n\ngo 1.24\n")
	writeFile(t, filepath.Join(dir, "calc.go"), "package calc\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n")
	writeFile(t, filepath.Join(dir, "calc_test.go"), "package calc\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(2, 3) != 5 {\n\t\tt.Fatal(\"soma incorreta\")\n\t}\n}\n")

	res, err := Run(dir)
	if err != nil {
		t.Fatalf("Run retornou erro inesperado: %v", err)
	}
	if res == nil {
		t.Fatal("res retornado é nil")
	}

	if res.Overall != StatusPass {
		t.Errorf("res.Overall = %q, quer %q", res.Overall, StatusPass)
	}
	if !res.OK {
		t.Errorf("res.OK = false, quer true")
	}
	if res.TestsRun < 1 {
		t.Errorf("res.TestsRun = %d, quer >= 1", res.TestsRun)
	}
}

func TestRunSensors_ZeroTestsNegativeCheck(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "go.mod"), "module testmod\n\ngo 1.24\n")
	writeFile(t, filepath.Join(dir, "service.go"), "package service\n\nfunc DoSomething() string {\n\treturn \"done\"\n}\n")

	res, err := Run(dir)
	if err != nil {
		t.Fatalf("Run retornou erro inesperado: %v", err)
	}
	if res == nil {
		t.Fatal("res retornado é nil")
	}

	if res.Overall != StatusFail {
		t.Errorf("res.Overall = %q, quer %q (autoteste negativo deve falhar quando zero testes são executados)", res.Overall, StatusFail)
	}
	if res.OK {
		t.Errorf("res.OK = true, quer false quando zero testes são executados")
	}

	// Verifica se a mensagem de erro ou dos sensores explicita o falso positivo evitado / zero testes
	allMessages := strings.ToLower(res.Message)
	for _, s := range res.Sensors {
		allMessages += " " + strings.ToLower(s.Name) + " " + strings.ToLower(s.Message) + " " + strings.ToLower(s.Output)
	}

	hasWarning := strings.Contains(allMessages, "zero") ||
		strings.Contains(allMessages, "nenhum teste") ||
		strings.Contains(allMessages, "falso positivo") ||
		strings.Contains(allMessages, "no test")

	if !hasWarning {
		t.Errorf("mensagem de erro não explicitou zero testes / falso positivo evitado: %s", allMessages)
	}
}

func TestRunSensors_FmtFailure(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "go.mod"), "module testmod\n\ngo 1.24\n")
	// Código fora do padrão do gofmt (espaços extras)
	writeFile(t, filepath.Join(dir, "bad_fmt.go"), "package   bad\n\nfunc  Bad( )  string   {\nreturn \"bad\"\n}\n")
	writeFile(t, filepath.Join(dir, "bad_fmt_test.go"), "package bad\n\nimport \"testing\"\n\nfunc TestBad(t *testing.T) {}\n")

	res, err := Run(dir)
	if err != nil {
		t.Fatalf("Run retornou erro inesperado: %v", err)
	}
	if res == nil {
		t.Fatal("res retornado é nil")
	}

	if res.Overall != StatusFail {
		t.Errorf("res.Overall = %q, quer %q para código desformatado", res.Overall, StatusFail)
	}
	if res.OK {
		t.Errorf("res.OK = true, quer false para falha de gofmt")
	}

	foundFmtFail := false
	for _, s := range res.Sensors {
		if strings.Contains(strings.ToLower(s.Name), "fmt") && s.Status == StatusFail {
			foundFmtFail = true
			break
		}
	}
	if !foundFmtFail {
		t.Errorf("nenhum sensor 'fmt' reportou falha (sensors: %+v)", res.Sensors)
	}
}

func TestRunSensors_VetFailure(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "go.mod"), "module testmod\n\ngo 1.24\n")
	// Código com erro detectável pelo go vet (formato Printf incompatível)
	writeFile(t, filepath.Join(dir, "bad_vet.go"), "package bad\n\nimport \"fmt\"\n\nfunc PrintWrong() {\n\tfmt.Printf(\"%d\", \"not an int\")\n}\n")
	writeFile(t, filepath.Join(dir, "bad_vet_test.go"), "package bad\n\nimport \"testing\"\n\nfunc TestPrintWrong(t *testing.T) {}\n")

	res, err := Run(dir)
	if err != nil {
		t.Fatalf("Run retornou erro inesperado: %v", err)
	}
	if res == nil {
		t.Fatal("res retornado é nil")
	}

	if res.Overall != StatusFail {
		t.Errorf("res.Overall = %q, quer %q para código com erro de go vet", res.Overall, StatusFail)
	}
	if res.OK {
		t.Errorf("res.OK = true, quer false para falha de go vet")
	}

	foundVetFail := false
	for _, s := range res.Sensors {
		if strings.Contains(strings.ToLower(s.Name), "vet") && s.Status == StatusFail {
			foundVetFail = true
			break
		}
	}
	if !foundVetFail {
		t.Errorf("nenhum sensor 'vet' reportou falha (sensors: %+v)", res.Sensors)
	}
}

func TestRunSensors_DesktopEnvironment(t *testing.T) {
	tests := []struct {
		name             string
		projectType      string
		expectDesktopEnv bool
	}{
		{
			name:             "desktop project injects CGO_ENABLED=0 GOOS=windows GOARCH=amd64",
			projectType:      "desktop",
			expectDesktopEnv: true,
		},
		{
			name:             "api project keeps system default environment",
			projectType:      "api",
			expectDesktopEnv: false,
		},
		{
			name:             "cli project keeps system default environment",
			projectType:      "cli",
			expectDesktopEnv: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()

			// 1. Arquivo de configuração .pipeline-ai.json
			cfgContent := fmt.Sprintf(`{
  "version": 1,
  "project_type": %q,
  "protected_paths": ["internal/"]
}
`, tc.projectType)
			writeFile(t, filepath.Join(dir, ".pipeline-ai.json"), cfgContent)

			// 2. Arquivos Go válidos com teste que valida ausência de CGO / variáveis de compilação
			writeFile(t, filepath.Join(dir, "go.mod"), "module desktopmod\n\ngo 1.24\n")
			writeFile(t, filepath.Join(dir, "app.go"), "package desktopmod\n\nfunc AppName() string {\n\treturn \"demo\"\n}\n")
			writeFile(t, filepath.Join(dir, "app_test.go"), `package desktopmod

import (
	"os"
	"testing"
)

func TestApp(t *testing.T) {
	if AppName() != "demo" {
		t.Fatal("AppName inválido")
	}
	if os.Getenv("CGO_ENABLED") == "1" {
		t.Fatal("CGO não pode estar habilitado em desktop")
	}
}
`)

			// 3. Mock wrapper do toolchain Go para inspecionar as chamadas e variáveis injetadas
			binDir := t.TempDir()
			logFile := filepath.Join(binDir, "calls.log")

			goScript := fmt.Sprintf(`#!/bin/sh
echo "CMD=$1|GOOS=$GOOS|GOARCH=$GOARCH|CGO_ENABLED=$CGO_ENABLED" >> %q
if [ "$1" = "test" ]; then
  echo "=== RUN   TestApp"
  echo "--- PASS: TestApp (0.00s)"
  echo "PASS"
  exit 0
fi
exit 0
`, logFile)

			gofmtScript := `#!/bin/sh
exit 0
`
			goBinPath := filepath.Join(binDir, "go")
			if err := os.WriteFile(goBinPath, []byte(goScript), 0o755); err != nil {
				t.Fatalf("falha ao criar mock de go: %v", err)
			}
			gofmtBinPath := filepath.Join(binDir, "gofmt")
			if err := os.WriteFile(gofmtBinPath, []byte(gofmtScript), 0o755); err != nil {
				t.Fatalf("falha ao criar mock de gofmt: %v", err)
			}

			t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

			res, err := Run(dir)
			if err != nil {
				t.Fatalf("Run retornou erro inesperado: %v", err)
			}
			if res == nil {
				t.Fatal("res é nil")
			}
			if !res.OK {
				t.Fatalf("esperava res.OK=true, obteve false: %s", res.Message)
			}

			// 4. Analisa o log de chamadas
			logData, err := os.ReadFile(logFile)
			if err != nil {
				t.Fatalf("falha ao ler log de chamadas: %v", err)
			}
			logLines := strings.Split(strings.TrimSpace(string(logData)), "\n")

			type cmdEnv struct {
				cmd        string
				goos       string
				goarch     string
				cgoEnabled string
			}
			recorded := make(map[string]cmdEnv)

			for _, line := range logLines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				parts := strings.Split(line, "|")
				var entry cmdEnv
				for _, part := range parts {
					kv := strings.SplitN(part, "=", 2)
					if len(kv) != 2 {
						continue
					}
					switch kv[0] {
					case "CMD":
						entry.cmd = kv[1]
					case "GOOS":
						entry.goos = kv[1]
					case "GOARCH":
						entry.goarch = kv[1]
					case "CGO_ENABLED":
						entry.cgoEnabled = kv[1]
					}
				}
				if entry.cmd != "" {
					recorded[entry.cmd] = entry
				}
			}

			for _, requiredCmd := range []string{"vet", "test"} {
				entry, ok := recorded[requiredCmd]
				if !ok {
					t.Fatalf("comando %s não foi executado (chamadas registradas: %v)", requiredCmd, logLines)
				}

				if tc.expectDesktopEnv {
					if entry.goos != "windows" {
						t.Errorf("comando %s: GOOS esperado \"windows\", obtido %q", requiredCmd, entry.goos)
					}
					if entry.goarch != "amd64" {
						t.Errorf("comando %s: GOARCH esperado \"amd64\", obtido %q", requiredCmd, entry.goarch)
					}
					if entry.cgoEnabled != "0" {
						t.Errorf("comando %s: CGO_ENABLED esperado \"0\", obtido %q", requiredCmd, entry.cgoEnabled)
					}
				} else {
					if entry.goos == "windows" {
						t.Errorf("comando %s: não deveria ter GOOS=windows em projeto %s", requiredCmd, tc.projectType)
					}
					if entry.cgoEnabled == "0" {
						t.Errorf("comando %s: não deveria forçar CGO_ENABLED=0 em projeto %s", requiredCmd, tc.projectType)
					}
				}
			}
		})
	}
}
