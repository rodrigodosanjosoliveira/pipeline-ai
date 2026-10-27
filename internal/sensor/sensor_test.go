package sensor

import (
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
