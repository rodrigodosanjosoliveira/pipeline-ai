package doctor

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseGoVersion(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		wantMajor int
		wantMinor int
		wantOk    bool
	}{
		{
			name:      "go 1.24.0",
			output:    "go version go1.24.0 linux/amd64",
			wantMajor: 1,
			wantMinor: 24,
			wantOk:    true,
		},
		{
			name:      "go 1.25beta1",
			output:    "go version go1.25beta1 darwin/arm64",
			wantMajor: 1,
			wantMinor: 25,
			wantOk:    true,
		},
		{
			name:      "go 1.23.2 (abaixo do requisito)",
			output:    "go version go1.23.2 linux/amd64",
			wantMajor: 1,
			wantMinor: 23,
			wantOk:    false,
		},
		{
			name:      "go 1.22",
			output:    "go version go1.22 linux/amd64",
			wantMajor: 1,
			wantMinor: 22,
			wantOk:    false,
		},
		{
			name:      "formato desconhecido",
			output:    "not a valid go version output",
			wantMajor: 0,
			wantMinor: 0,
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, ok := ParseGoVersion(tt.output)
			if major != tt.wantMajor || minor != tt.wantMinor {
				t.Errorf("ParseGoVersion(%q) = (%d, %d), esperado (%d, %d)", tt.output, major, minor, tt.wantMajor, tt.wantMinor)
			}
			isSatisfied := ok && (major > 1 || (major == 1 && minor >= 24))
			if isSatisfied != tt.wantOk {
				t.Errorf("isSatisfied = %v, esperado %v", isSatisfied, tt.wantOk)
			}
		})
	}
}

func TestCheckConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Caso 1: sem config
	res := CheckConfig(tmpDir)
	if res.Status != StatusFail {
		t.Errorf("CheckConfig em diretório vazio: status = %v, esperado %v", res.Status, StatusFail)
	}

	// Caso 2: config válida
	cfgValid := `{
		"version": 1,
		"project_type": "desktop",
		"protected_paths": ["internal/"],
		"editor_agents": ["dev-implementation"]
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, ".pipeline-ai.json"), []byte(cfgValid), 0644); err != nil {
		t.Fatal(err)
	}
	res = CheckConfig(tmpDir)
	if res.Status != StatusPass {
		t.Errorf("CheckConfig com config válida: status = %v (msg: %s), esperado %v", res.Status, res.Message, StatusPass)
	}

	// Caso 3: config inválida
	cfgInvalid := `{"version": 1, "project_type": "invalido"}`
	if err := os.WriteFile(filepath.Join(tmpDir, ".pipeline-ai.json"), []byte(cfgInvalid), 0644); err != nil {
		t.Fatal(err)
	}
	res = CheckConfig(tmpDir)
	if res.Status != StatusFail {
		t.Errorf("CheckConfig com tipo inválido: status = %v, esperado %v", res.Status, StatusFail)
	}
}

func TestCheckGit(t *testing.T) {
	tmpDir := t.TempDir()

	// Diretório sem .git
	res := CheckGit(tmpDir)
	if res.Status != StatusWarn && res.Status != StatusFail {
		t.Errorf("CheckGit sem .git: status = %v, esperado aviso ou falha", res.Status)
	}

	// Com .git e sem pre-push
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	res = CheckGit(tmpDir)
	if res.Status != StatusWarn {
		t.Errorf("CheckGit sem pre-push: status = %v, esperado StatusWarn", res.Status)
	}

	// Com .githooks/pre-push
	githooksDir := filepath.Join(tmpDir, ".githooks")
	if err := os.MkdirAll(githooksDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(githooksDir, "pre-push"), []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	res = CheckGit(tmpDir)
	if res.Status != StatusPass {
		t.Errorf("CheckGit com .githooks/pre-push: status = %v (msg: %s), esperado StatusPass", res.Status, res.Message)
	}
}

func TestRunDoctor(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(".", &stdout, &stderr)
	// Deve executar sem panic e gerar output no stdout
	if stdout.Len() == 0 {
		t.Errorf("esperava output no stdout")
	}
	t.Logf("doctor exit code: %d, output:\n%s", code, stdout.String())
}

func TestRunDoctorJSON(t *testing.T) {
	var stdout bytes.Buffer
	code := RunJSON(".", &stdout)
	if code != 0 {
		t.Errorf("esperava código de saída 0 no repositório atual, obteve %d", code)
	}

	raw := stdout.Bytes()
	var rep Report
	if err := json.Unmarshal(raw, &rep); err != nil {
		t.Fatalf("saída não é JSON válido: %v\nOutput:\n%s", err, string(raw))
	}

	if !rep.OK {
		t.Errorf("esperava rep.OK == true no repositório atual")
	}
	if len(rep.Results) == 0 {
		t.Fatalf("esperava ao menos um resultado de verificação")
	}

	for _, r := range rep.Results {
		if r.Name == "" {
			t.Errorf("resultado sem campo 'name'")
		}
		if r.Status == "" {
			t.Errorf("resultado sem campo 'status'")
		}
	}
}
