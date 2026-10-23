package validate

import (
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
