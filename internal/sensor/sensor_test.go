package sensor

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
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

func TestBoundOutput(t *testing.T) {
	tests := []struct {
		name             string
		raw              string
		maxBytes         int
		wantTruncated    bool
		wantTotalBytes   int
		wantOmittedMin   int
		wantContains     string
		checkValidUTF8   bool
		expectUnmodified bool
	}{
		{
			name:             "texto menor que maxBytes nao trunca",
			raw:              "linha 1\nlinha 2\nlinha 3",
			maxBytes:         256,
			wantTruncated:    false,
			wantTotalBytes:   len("linha 1\nlinha 2\nlinha 3"),
			wantOmittedMin:   0,
			expectUnmodified: true,
			checkValidUTF8:   true,
		},
		{
			name:           "texto maior que maxBytes trunca e reporta metadados",
			raw:            strings.Repeat("linha de log extensa para teste de truncamento\n", 40),
			maxBytes:       120,
			wantTruncated:  true,
			wantTotalBytes: len(strings.Repeat("linha de log extensa para teste de truncamento\n", 40)),
			wantOmittedMin: 1,
			checkValidUTF8: true,
		},
		{
			name:           "preserva UTF-8 valido quando corte cai no meio de caractere multibyte",
			raw:            strings.Repeat("ação coração 🚀 ", 30),
			maxBytes:       55, // cai em fronteira arbitrária com caracteres de 2 e 4 bytes ("ç", "ã", "🚀")
			wantTruncated:  true,
			wantTotalBytes: len(strings.Repeat("ação coração 🚀 ", 30)),
			wantOmittedMin: 1,
			checkValidUTF8: true,
		},
		{
			name:             "maxBytes zero ou negativo usa DefaultMaxOutputBytes",
			raw:              "curto",
			maxBytes:         0,
			wantTruncated:    false,
			wantTotalBytes:   5,
			wantOmittedMin:   0,
			expectUnmodified: true,
			checkValidUTF8:   true,
		},
	}

	if DefaultMaxOutputBytes != 8192 {
		t.Errorf("DefaultMaxOutputBytes = %d, esperado 8192", DefaultMaxOutputBytes)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bounded, truncated, totalBytes, omittedBytes := BoundOutput(tt.raw, tt.maxBytes)
			if truncated != tt.wantTruncated {
				t.Errorf("truncated = %v, esperado %v", truncated, tt.wantTruncated)
			}
			if totalBytes != tt.wantTotalBytes {
				t.Errorf("totalBytes = %d, esperado %d", totalBytes, tt.wantTotalBytes)
			}
			if tt.wantTruncated {
				if omittedBytes < tt.wantOmittedMin {
					t.Errorf("omittedBytes = %d, esperado >= %d", omittedBytes, tt.wantOmittedMin)
				}
			} else if omittedBytes != 0 {
				t.Errorf("omittedBytes = %d, esperado 0 quando não truncado", omittedBytes)
			}
			if tt.expectUnmodified && bounded != tt.raw {
				t.Errorf("bounded = %q, esperado %q", bounded, tt.raw)
			}
			if tt.checkValidUTF8 && !utf8.ValidString(bounded) {
				t.Errorf("bounded contém UTF-8 inválido: %q", bounded)
			}
		})
	}
}

func TestBoundTestOutput(t *testing.T) {
	// Constrói saída com 50 testes que passam
	var passBuilder strings.Builder
	for i := 1; i <= 50; i++ {
		fmt.Fprintf(&passBuilder, "=== RUN   TestCase%02d\n", i)
		fmt.Fprintf(&passBuilder, "--- PASS: TestCase%02d (0.00s)\n", i)
	}
	passBuilder.WriteString("PASS\n")
	passBuilder.WriteString("ok  \tpkg\t0.1s\n")
	fiftyPassingRaw := passBuilder.String()

	// Constrói saída mista com vários PASS e 2 FAIL (TestAlpha e TestBeta)
	var mixedBuilder strings.Builder
	for i := 1; i <= 10; i++ {
		fmt.Fprintf(&mixedBuilder, "=== RUN   TestPass%02d\n", i)
		fmt.Fprintf(&mixedBuilder, "--- PASS: TestPass%02d (0.00s)\n", i)
	}
	mixedBuilder.WriteString("=== RUN   TestAlpha\n")
	mixedBuilder.WriteString("    alpha_test.go:15: erro detalhado em TestAlpha\n")
	mixedBuilder.WriteString("--- FAIL: TestAlpha (0.01s)\n")
	for i := 11; i <= 20; i++ {
		fmt.Fprintf(&mixedBuilder, "=== RUN   TestPass%02d\n", i)
		fmt.Fprintf(&mixedBuilder, "--- PASS: TestPass%02d (0.00s)\n", i)
	}
	mixedBuilder.WriteString("=== RUN   TestBeta\n")
	mixedBuilder.WriteString("    beta_test.go:42: erro detalhado em TestBeta\n")
	mixedBuilder.WriteString("--- FAIL: TestBeta (0.02s)\n")
	mixedBuilder.WriteString("FAIL\n")
	mixedBuilder.WriteString("FAIL\tpkg\t0.15s\n")
	mixedRaw := mixedBuilder.String()

	tests := []struct {
		name               string
		raw                string
		maxBytes           int
		verbose            bool
		wantFailedTests    []string
		wantTruncated      bool
		wantContains       []string
		wantNotContains    []string
		wantMaxBoundedSize int
	}{
		{
			name:            "50 testes passando com verbose=false filtra RUN e PASS e preserva resumo ok",
			raw:             fiftyPassingRaw,
			maxBytes:        8192,
			verbose:         false,
			wantFailedTests: nil,
			wantTruncated:   true,
			wantContains:    []string{"ok  \tpkg\t0.1s"},
			wantNotContains: []string{"=== RUN   TestCase01", "--- PASS: TestCase01"},
		},
		{
			name:            "saida mista com verbose=false preserva blocos de falha TestAlpha e TestBeta e omite PASS",
			raw:             mixedRaw,
			maxBytes:        8192,
			verbose:         false,
			wantFailedTests: []string{"TestAlpha", "TestBeta"},
			wantTruncated:   true,
			wantContains: []string{
				"--- FAIL: TestAlpha",
				"alpha_test.go:15: erro detalhado em TestAlpha",
				"--- FAIL: TestBeta",
				"beta_test.go:42: erro detalhado em TestBeta",
				"FAIL\tpkg\t0.15s",
			},
			wantNotContains: []string{
				"--- PASS: TestPass01",
				"--- PASS: TestPass15",
			},
		},
		{
			name:            "verbose=true preserva linhas PASS ate o limite de maxBytes",
			raw:             fiftyPassingRaw,
			maxBytes:        300,
			verbose:         true,
			wantFailedTests: nil,
			wantTruncated:   true,
			wantContains:    []string{"--- PASS: TestCase01"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bounded, failedTests, truncated, totalBytes, omittedBytes := BoundTestOutput(tt.raw, tt.maxBytes, tt.verbose)

			if !utf8.ValidString(bounded) {
				t.Errorf("bounded não é UTF-8 válido")
			}
			if totalBytes != len(tt.raw) {
				t.Errorf("totalBytes = %d, esperado %d", totalBytes, len(tt.raw))
			}
			if truncated != tt.wantTruncated {
				t.Errorf("truncated = %v, esperado %v", truncated, tt.wantTruncated)
			}
			if tt.wantTruncated && omittedBytes <= 0 {
				t.Errorf("omittedBytes = %d, esperado > 0 quando truncated=true", omittedBytes)
			}
			if len(tt.wantFailedTests) == 0 {
				if len(failedTests) != 0 {
					t.Errorf("failedTests = %v, esperado vazio", failedTests)
				}
			} else if !reflect.DeepEqual(failedTests, tt.wantFailedTests) {
				t.Errorf("failedTests = %v, esperado %v", failedTests, tt.wantFailedTests)
			}

			for _, sub := range tt.wantContains {
				if !strings.Contains(bounded, sub) {
					t.Errorf("bounded não contém %q; saída:\n%s", sub, bounded)
				}
			}
			for _, notSub := range tt.wantNotContains {
				if strings.Contains(bounded, notSub) {
					t.Errorf("bounded não deveria conter %q; saída:\n%s", notSub, bounded)
				}
			}
		})
	}
}

func TestRunWithOptions_BoundedAndFailedTests(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "go.mod"), "module boundedtestmod\n\ngo 1.24\n")
	writeFile(t, filepath.Join(dir, "svc.go"), "package svc\n\nfunc Value() int {\n\treturn 42\n}\n")

	var testSrc strings.Builder
	testSrc.WriteString("package svc\n\nimport (\n\t\"strings\"\n\t\"testing\"\n)\n\n")
	for i := 1; i <= 15; i++ {
		fmt.Fprintf(&testSrc, "func TestPass%02d(t *testing.T) {\n\tif Value() != 42 {\n\t\tt.Fatal(\"err\")\n\t}\n}\n\n", i)
	}
	testSrc.WriteString("func TestBrokenHeavy(t *testing.T) {\n")
	testSrc.WriteString("\tt.Log(strings.Repeat(\"detalhe extenso de falha 🚀 \", 40))\n")
	testSrc.WriteString("\tt.Fatal(\"falha crítica em TestBrokenHeavy\")\n")
	testSrc.WriteString("}\n")

	writeFile(t, filepath.Join(dir, "svc_test.go"), testSrc.String())

	res, err := RunWithOptions(dir, Options{
		MaxOutputBytes: 256,
		VerboseOutput:  false,
	})
	if err != nil {
		t.Fatalf("RunWithOptions retornou erro inesperado: %v", err)
	}
	if res == nil {
		t.Fatal("res retornado é nil")
	}
	if res.Overall != StatusFail || res.OK {
		t.Errorf("res.Overall = %q, res.OK = %v, esperado FAIL/false", res.Overall, res.OK)
	}
	if res.TestsRun != 16 {
		t.Errorf("res.TestsRun = %d, esperado 16", res.TestsRun)
	}

	var testSensor *CheckResult
	for i := range res.Sensors {
		if res.Sensors[i].Name == "test" {
			testSensor = &res.Sensors[i]
			break
		}
	}
	if testSensor == nil {
		t.Fatal("sensor 'test' não encontrado em res.Sensors")
	}
	if !testSensor.Truncated {
		t.Errorf("testSensor.Truncated = false, esperado true")
	}
	if testSensor.TotalBytes <= 256 {
		t.Errorf("testSensor.TotalBytes = %d, esperado > 256", testSensor.TotalBytes)
	}
	if testSensor.OmittedBytes <= 0 {
		t.Errorf("testSensor.OmittedBytes = %d, esperado > 0", testSensor.OmittedBytes)
	}
	if !reflect.DeepEqual(testSensor.FailedTests, []string{"TestBrokenHeavy"}) {
		t.Errorf("testSensor.FailedTests = %v, esperado [\"TestBrokenHeavy\"]", testSensor.FailedTests)
	}
	if !utf8.ValidString(testSensor.Output) {
		t.Errorf("testSensor.Output não é UTF-8 válido")
	}
}

func FuzzBoundTestOutput(f *testing.F) {
	f.Add("=== RUN   TestA\n--- PASS: TestA (0.00s)\nPASS\nok\tpkg\t0.01s\n", 128, false)
	f.Add("=== RUN   TestFail\n    a_test.go:10: boom 🚀 çã\n--- FAIL: TestFail (0.01s)\nFAIL\n", 64, false)
	f.Add(strings.Repeat("linha com utf8 çãõé 🚀🔥\n", 50), 100, true)
	f.Add("", 0, false)
	f.Add("texto curto", -10, true)

	f.Fuzz(func(t *testing.T, raw string, maxBytes int, verbose bool) {
		bounded, _, truncated, totalBytes, omittedBytes := BoundTestOutput(raw, maxBytes, verbose)
		if utf8.ValidString(raw) && !utf8.ValidString(bounded) {
			t.Errorf("BoundTestOutput produziu UTF-8 inválido para entrada UTF-8 válida (maxBytes=%d, verbose=%v)", maxBytes, verbose)
		}
		if totalBytes != len(raw) {
			t.Errorf("totalBytes = %d, len(raw) = %d", totalBytes, len(raw))
		}
		if !truncated && omittedBytes != 0 {
			t.Errorf("truncated=false mas omittedBytes=%d", omittedBytes)
		}
	})
}
