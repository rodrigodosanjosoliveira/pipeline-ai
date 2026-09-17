package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const cfgBody = `{
  "version": 1,
  "project_type": "api",
  "protected_paths": ["internal/"],
  "editor_agents": ["dev-implementation"],
  "security_surface_paths": ["internal/auth/", "go.mod"],
  "deliveries_dir": "docs-ai/deliveries",
  "index_file": "docs-ai/deliveries/INDEX.md",
  "base_ref": "main"
}`

const relatorioOK = `---
delivery_id: DEL-001
task_code: TASK-42
title: Ajuste no parser
status: done
require_security: false
---

# Contexto
Parser rejeitava tokens com padding.

# Escopo
internal/parser.go

# Implementação
Normalização antes do decode.

# Testes
Table-driven cobrindo 6 casos.

# Code Review
Aprovado sem ressalvas.

# Final Checklist
- [x] build passa
- [x] testes passam
`

// projeto monta um diretório com config, índice e relatório.
func projeto(t *testing.T, indexBody, reportBody string) (root, reportPath string) {
	t.Helper()
	root = t.TempDir()

	write(t, filepath.Join(root, ".pipeline-ai.json"), cfgBody)

	delivDir := filepath.Join(root, "docs-ai", "deliveries", "DEL-001")
	if err := os.MkdirAll(delivDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	write(t, filepath.Join(root, "docs-ai", "deliveries", "INDEX.md"), indexBody)

	reportPath = filepath.Join(delivDir, "report.md")
	write(t, reportPath, reportBody)
	return root, reportPath
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func runValidate(t *testing.T, reportPath string, noGit bool) (int, string) {
	t.Helper()
	var out, errBuf bytes.Buffer
	code := run(reportPath, "", noGit, &out, &errBuf)
	return code, out.String() + errBuf.String()
}

func TestRelatorioAprovado(t *testing.T) {
	_, rp := projeto(t, "# Índice\n\n- DEL-001 — Ajuste no parser\n", relatorioOK)

	code, out := runValidate(t, rp, true)
	if code != exitOK {
		t.Errorf("exit = %d, quer %d\n%s", code, exitOK, out)
	}
	if !strings.Contains(out, "APROVADO") {
		t.Errorf("saída não diz APROVADO:\n%s", out)
	}
}

func TestDeliveryNaoRegistradaNoIndice(t *testing.T) {
	_, rp := projeto(t, "# Índice\n\n- DEL-999 — outra coisa\n", relatorioOK)

	code, out := runValidate(t, rp, true)
	if code != exitFail {
		t.Errorf("exit = %d, quer %d\n%s", code, exitFail, out)
	}
	if !strings.Contains(out, "DEL-001") {
		t.Errorf("saída não cita a delivery ausente:\n%s", out)
	}
}

func TestIndiceInexistente(t *testing.T) {
	root, rp := projeto(t, "x", relatorioOK)
	if err := os.Remove(filepath.Join(root, "docs-ai", "deliveries", "INDEX.md")); err != nil {
		t.Fatalf("remove: %v", err)
	}

	code, out := runValidate(t, rp, true)
	if code != exitFail {
		t.Errorf("exit = %d, quer %d\n%s", code, exitFail, out)
	}
}

func TestRelatorioInexistente(t *testing.T) {
	code, _ := runValidate(t, filepath.Join(t.TempDir(), "nao-existe.md"), true)
	if code != exitBadUsage {
		t.Errorf("exit = %d, quer %d", code, exitBadUsage)
	}
}

func TestSemConfigNoProjeto(t *testing.T) {
	dir := t.TempDir()
	rp := filepath.Join(dir, "report.md")
	write(t, rp, relatorioOK)

	code, _ := runValidate(t, rp, true)
	if code != exitBadUsage {
		t.Skipf("exit = %d: provável .pipeline-ai.json acima de %s", code, dir)
	}
}

func TestChecklistIncompletoReprova(t *testing.T) {
	body := strings.Replace(relatorioOK, "- [x] testes passam", "- [ ] testes passam", 1)
	_, rp := projeto(t, "- DEL-001\n", body)

	code, out := runValidate(t, rp, true)
	if code != exitFail {
		t.Errorf("exit = %d, quer %d\n%s", code, exitFail, out)
	}
}

// --- salvaguarda D-14, com repositório git real ---

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// repoComDiff cria um repo com um commit base em main e um segundo commit
// numa branch, tocando os arquivos indicados.
func repoComDiff(t *testing.T, root string, arquivos ...string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git não disponível")
	}

	git(t, root, "init", "-b", "main")
	write(t, filepath.Join(root, "README.md"), "base\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "base")

	git(t, root, "checkout", "-b", "task/TASK-42")
	for _, f := range arquivos {
		write(t, filepath.Join(root, filepath.FromSlash(f)), "conteúdo\n")
	}
	git(t, root, "add", ".")
	git(t, root, "commit", "-m", "mudança")
}

func TestSalvaguardaSegurancaReprova(t *testing.T) {
	root, rp := projeto(t, "- DEL-001\n", relatorioOK) // require_security: false
	repoComDiff(t, root, "internal/auth/jwt.go")

	code, out := runValidate(t, rp, false)
	if code != exitFail {
		t.Errorf("exit = %d, quer %d\n%s", code, exitFail, out)
	}
	if !strings.Contains(out, "require_security") {
		t.Errorf("saída não explica a flag:\n%s", out)
	}
	if !strings.Contains(out, "internal/auth/jwt.go") {
		t.Errorf("saída não cita o arquivo sensível:\n%s", out)
	}
}

func TestSalvaguardaSegurancaAprovaComFlagLigada(t *testing.T) {
	body := strings.Replace(relatorioOK, "require_security: false", "require_security: true", 1)
	body += "\n# Security\ngovulncheck limpo; token validado contra chave rotacionada.\n"

	root, rp := projeto(t, "- DEL-001\n", body)
	repoComDiff(t, root, "internal/auth/jwt.go")

	code, out := runValidate(t, rp, false)
	if code != exitOK {
		t.Errorf("exit = %d, quer %d\n%s", code, exitOK, out)
	}
}

func TestSalvaguardaIgnoraDiffNaoSensivel(t *testing.T) {
	root, rp := projeto(t, "- DEL-001\n", relatorioOK)
	repoComDiff(t, root, "internal/ui/render.go")

	code, out := runValidate(t, rp, false)
	if code != exitOK {
		t.Errorf("exit = %d, quer %d\n%s", code, exitOK, out)
	}
}

func TestSalvaguardaCobreGoMod(t *testing.T) {
	root, rp := projeto(t, "- DEL-001\n", relatorioOK)
	repoComDiff(t, root, "go.mod")

	code, out := runValidate(t, rp, false)
	if code != exitFail {
		t.Errorf("exit = %d, quer %d: go.mod é superfície de segurança\n%s", code, exitFail, out)
	}
}

// TestGitIndisponivelVirarAviso fixa a decisão em aberto: sem git, a
// salvaguarda não roda e a delivery passa com aviso, em vez de reprovar.
func TestGitIndisponivelViraAviso(t *testing.T) {
	_, rp := projeto(t, "- DEL-001\n", relatorioOK) // sem git init

	code, out := runValidate(t, rp, false)
	if code != exitOK {
		t.Errorf("exit = %d, quer %d\n%s", code, exitOK, out)
	}
	if !strings.Contains(out, "AVISO") {
		t.Errorf("saída não traz AVISO:\n%s", out)
	}
}
