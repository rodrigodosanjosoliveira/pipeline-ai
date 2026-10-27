package validate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s falhou: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s falhou: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s falhou: %v", path, err)
	}
}

func TestChangedFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git não encontrado no PATH")
	}

	root := t.TempDir()
	gitCmd(t, root, "init", "-b", "main")
	gitCmd(t, root, "config", "user.name", "test")
	gitCmd(t, root, "config", "user.email", "test@example.com")

	writeFile(t, filepath.Join(root, "base.txt"), "base content\n")
	writeFile(t, filepath.Join(root, "modified_unstaged.txt"), "versao 1\n")
	gitCmd(t, root, "add", ".")
	gitCmd(t, root, "commit", "-m", "commit inicial em main")

	gitCmd(t, root, "checkout", "-b", "task/TASK-008")

	// 1. Arquivo commitado na branch de trabalho
	writeFile(t, filepath.Join(root, "committed_branch.txt"), "conteudo da branch\n")
	gitCmd(t, root, "add", "committed_branch.txt")
	gitCmd(t, root, "commit", "-m", "commit na branch")

	// 2. Modificação no working tree (não commitada / unstaged)
	writeFile(t, filepath.Join(root, "modified_unstaged.txt"), "versao 2 unstaged\n")

	// 3. Arquivo na staging area (git add)
	writeFile(t, filepath.Join(root, "staged_file.txt"), "conteudo staged\n")
	gitCmd(t, root, "add", "staged_file.txt")

	// 4. Arquivo untracked novo
	writeFile(t, filepath.Join(root, "untracked_file.txt"), "conteudo untracked\n")

	files, err := changedFiles(root, "main")
	if err != nil {
		t.Fatalf("changedFiles falhou: %v", err)
	}

	expected := []struct {
		name string
		path string
	}{
		{name: "arquivo commitado na branch", path: "committed_branch.txt"},
		{name: "modificado no working tree (unstaged)", path: "modified_unstaged.txt"},
		{name: "arquivo staged (git add)", path: "staged_file.txt"},
		{name: "arquivo untracked novo", path: "untracked_file.txt"},
	}

	fileMap := make(map[string]bool)
	for _, f := range files {
		fileMap[filepath.ToSlash(f)] = true
	}

	for _, tc := range expected {
		t.Run(tc.name, func(t *testing.T) {
			if !fileMap[tc.path] {
				t.Errorf("changedFiles não detectou %s (%s). Arquivos detectados: %v", tc.name, tc.path, files)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}

const validReportContent = `---
delivery_id: DEL-001
task_code: TASK-42
title: Validação flexível
status: done
require_security: false
---

# Contexto
Contexto explicativo.

# Escopo
Arquivos no escopo.

# Implementação
Detalhes da implementação.

# Testes
Testes executados.

# Code Review
Aprovado sem ressalvas.

# Final Checklist
- [x] itens concluídos
`

func TestValidateRequireIndex(t *testing.T) {
	tests := []struct {
		name         string
		requireIndex *bool
		hasIndex     bool
		indexHasID   bool
		wantExitCode int
		wantOutput   string
	}{
		{
			name:         "require_index false sem INDEX.md aprova",
			requireIndex: boolPtr(false),
			hasIndex:     false,
			wantExitCode: ExitOK,
			wantOutput:   "APROVADO",
		},
		{
			name:         "require_index false com INDEX.md sem o ID aprova",
			requireIndex: boolPtr(false),
			hasIndex:     true,
			indexHasID:   false,
			wantExitCode: ExitOK,
			wantOutput:   "APROVADO",
		},
		{
			name:         "require_index true sem INDEX.md falha",
			requireIndex: boolPtr(true),
			hasIndex:     false,
			wantExitCode: ExitFail,
			wantOutput:   "REPROVADO",
		},
		{
			name:         "require_index true com INDEX.md sem o ID falha",
			requireIndex: boolPtr(true),
			hasIndex:     true,
			indexHasID:   false,
			wantExitCode: ExitFail,
			wantOutput:   "REPROVADO",
		},
		{
			name:         "require_index omitido sem INDEX.md falha",
			requireIndex: nil,
			hasIndex:     false,
			wantExitCode: ExitFail,
			wantOutput:   "REPROVADO",
		},
		{
			name:         "require_index omitido com INDEX.md contendo ID aprova",
			requireIndex: nil,
			hasIndex:     true,
			indexHasID:   true,
			wantExitCode: ExitOK,
			wantOutput:   "APROVADO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()

			var cfgBody string
			if tt.requireIndex == nil {
				cfgBody = `{"version": 1, "project_type": "desktop", "protected_paths": ["internal/"]}`
			} else {
				cfgBody = fmt.Sprintf(`{"version": 1, "project_type": "desktop", "protected_paths": ["internal/"], "require_index": %t}`, *tt.requireIndex)
			}
			writeFile(t, filepath.Join(root, ".pipeline-ai.json"), cfgBody)

			reportPath := filepath.Join(root, "docs-ai", "deliveries", "DEL-001", "report.md")
			writeFile(t, reportPath, validReportContent)

			if tt.hasIndex {
				indexPath := filepath.Join(root, "docs-ai", "deliveries", "INDEX.md")
				var indexBody string
				if tt.indexHasID {
					indexBody = "# Índice\n\n| DEL-001 | TASK-42 | Validação flexível | feat | 2026-10-26 | done |\n"
				} else {
					indexBody = "# Índice\n\n| DEL-999 | TASK-99 | Outra | feat | 2026-10-26 | done |\n"
				}
				writeFile(t, indexPath, indexBody)
			}

			var stdout, stderr bytes.Buffer
			code := Run(reportPath, "", true, &stdout, &stderr)

			if code != tt.wantExitCode {
				t.Errorf("Run exit code = %d, quer %d (stdout: %q, stderr: %q)", code, tt.wantExitCode, stdout.String(), stderr.String())
			}

			fullOutput := stdout.String() + stderr.String()
			if !strings.Contains(fullOutput, tt.wantOutput) {
				t.Errorf("saída esperada contendo %q não encontrada. Saída obtida:\n%s", tt.wantOutput, fullOutput)
			}
		})
	}
}

func TestValidateJSONOutput(t *testing.T) {
	tests := []struct {
		name         string
		report       string
		wantExitCode int
		wantStatus   string
		wantOK       bool
		wantMinErr   int
	}{
		{
			name:         "relatório válido emite JSON com status PASS e ok true",
			report:       validReportContent,
			wantExitCode: ExitOK,
			wantStatus:   "PASS",
			wantOK:       true,
			wantMinErr:   0,
		},
		{
			name:         "relatório inválido com checklist incompleto emite FAIL",
			report:       strings.Replace(validReportContent, "- [x] itens concluídos", "- [ ] itens concluídos", 1),
			wantExitCode: ExitFail,
			wantStatus:   "FAIL",
			wantOK:       false,
			wantMinErr:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			cfgBody := `{"version": 1, "project_type": "desktop", "protected_paths": ["internal/"], "require_index": false}`
			writeFile(t, filepath.Join(root, ".pipeline-ai.json"), cfgBody)

			reportPath := filepath.Join(root, "docs-ai", "deliveries", "DEL-001", "report.md")
			writeFile(t, reportPath, tt.report)

			var stdout, stderr bytes.Buffer
			code := RunJSON(reportPath, "", true, &stdout, &stderr)

			if code != tt.wantExitCode {
				t.Errorf("RunJSON exit code = %d, quer %d (stderr: %q)", code, tt.wantExitCode, stderr.String())
			}

			var res ValidationResult
			if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
				t.Fatalf("stdout não é JSON válido de ValidationResult: %v (raw: %q)", err, stdout.String())
			}

			if res.Status != tt.wantStatus {
				t.Errorf("res.Status = %q, quer %q", res.Status, tt.wantStatus)
			}
			if res.OK != tt.wantOK {
				t.Errorf("res.OK = %v, quer %v", res.OK, tt.wantOK)
			}
			if res.Errors < tt.wantMinErr {
				t.Errorf("res.Errors = %d, quer ao menos %d", res.Errors, tt.wantMinErr)
			}
			if res.Report == "" {
				t.Errorf("res.Report veio vazio no JSON")
			}
			if tt.wantMinErr > 0 && len(res.Findings) == 0 {
				t.Errorf("res.Findings veio vazio para relatório com erro")
			}
		})
	}
}
