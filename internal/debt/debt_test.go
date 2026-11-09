package debt_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/debt"
)

func TestParseComment(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantLimit   string
		wantTrigger string
		wantOK      bool
	}{
		{
			name:        "formato pipe com upgrade explicito",
			line:        "// shortcut: armazenamento em memória | upgrade: quando exceder 10k registros",
			wantLimit:   "armazenamento em memória",
			wantTrigger: "quando exceder 10k registros",
			wantOK:      true,
		},
		{
			name:        "formato virgula simples",
			line:        "// shortcut: buffer fixo de 512KB, migrar para streaming acima de 1MB",
			wantLimit:   "buffer fixo de 512KB",
			wantTrigger: "migrar para streaming acima de 1MB",
			wantOK:      true,
		},
		{
			name:        "formato com espacos extras e tabulacao",
			line:        "\t//   shortcut:   locks locais em processo   |   upgrade:   distribuir via Redis em multi-node  ",
			wantLimit:   "locks locais em processo",
			wantTrigger: "distribuir via Redis em multi-node",
			wantOK:      true,
		},
		{
			name:        "formato case-insensitive SHORTCUT e UPGRADE",
			line:        "// SHORTCUT: polling a cada 5s | UPGRADE: usar webhook quando latência < 1s for exigida",
			wantLimit:   "polling a cada 5s",
			wantTrigger: "usar webhook quando latência < 1s for exigida",
			wantOK:      true,
		},
		{
			name:        "comentario inline apos codigo Go",
			line:        "var cache = map[string]string{} // shortcut: mapa sem expiração | upgrade: adicionar TTL quando memória > 100MB",
			wantLimit:   "mapa sem expiração",
			wantTrigger: "adicionar TTL quando memória > 100MB",
			wantOK:      true,
		},
		{
			name:        "comentario comum sem marcador shortcut",
			line:        "// Carrega configuração do disco",
			wantLimit:   "",
			wantTrigger: "",
			wantOK:      false,
		},
		{
			name:        "comentario TODO tradicional ignorado",
			line:        "// TODO: refatorar isso depois",
			wantLimit:   "",
			wantTrigger: "",
			wantOK:      false,
		},
		{
			name:        "shortcut sem gatilho de evolucao (incompleto)",
			line:        "// shortcut: apenas um limite sem gatilho",
			wantLimit:   "",
			wantTrigger: "",
			wantOK:      false,
		},
		{
			name:        "shortcut vazio",
			line:        "// shortcut: | upgrade: ",
			wantLimit:   "",
			wantTrigger: "",
			wantOK:      false,
		},
		{
			name:        "linha vazia",
			line:        "",
			wantLimit:   "",
			wantTrigger: "",
			wantOK:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLimit, gotTrigger, gotOK := debt.ParseComment(tt.line)
			if gotOK != tt.wantOK {
				t.Fatalf("ParseComment(%q) ok = %v, esperado %v", tt.line, gotOK, tt.wantOK)
			}
			if gotLimit != tt.wantLimit {
				t.Errorf("ParseComment(%q) limit = %q, esperado %q", tt.line, gotLimit, tt.wantLimit)
			}
			if gotTrigger != tt.wantTrigger {
				t.Errorf("ParseComment(%q) trigger = %q, esperado %q", tt.line, gotTrigger, tt.wantTrigger)
			}
		})
	}
}

func TestScanDir(t *testing.T) {
	t.Run("diretorio com multiplos arquivos Go, ignorando testes e pastas ocultas/vendor", func(t *testing.T) {
		tmpDir := t.TempDir()

		writeFile := func(relPath, content string) {
			t.Helper()
			fullPath := filepath.Join(tmpDir, filepath.FromSlash(relPath))
			if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", fullPath, err)
			}
			if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
				t.Fatalf("write %s: %v", fullPath, err)
			}
		}

		writeFile("internal/store/memory.go", `package store

// TODO: comentário comum não deve entrar
// shortcut: armazenamento JSON em arquivo único | upgrade: migrar para SQLite quando > 50MB
type MemoryStore struct{}

func (m *MemoryStore) Save() {
	// shortcut: lock global por processo, usar lock por chave acima de 100 req/s
}
`)

		writeFile("cmd/app/main.go", `package main

// shortcut: configuração fixa em flags | upgrade: suportar arquivo YAML se passar de 10 opções
func main() {}
`)

		// Arquivos que DEVEM ser ignorados:
		writeFile("internal/store/memory_test.go", `package store
// shortcut: falso positivo em arquivo de teste | upgrade: nunca deve aparecer
`)
		writeFile(".git/hooks/hook.go", `package hooks
// shortcut: falso positivo em .git | upgrade: nunca deve aparecer
`)
		writeFile(".hidden/secret.go", `package hidden
// shortcut: falso positivo em pasta oculta | upgrade: nunca deve aparecer
`)
		writeFile("vendor/pkg/dep.go", `package dep
// shortcut: falso positivo em vendor | upgrade: nunca deve aparecer
`)
		writeFile("node_modules/pkg/index.go", `package idx
// shortcut: falso positivo em node_modules | upgrade: nunca deve aparecer
`)
		writeFile("README.md", `# Docs
// shortcut: falso positivo em markdown | upgrade: nunca deve aparecer
`)

		summary, err := debt.ScanDir(tmpDir)
		if err != nil {
			t.Fatalf("ScanDir retornou erro inesperado: %v", err)
		}
		if summary == nil {
			t.Fatalf("ScanDir retornou summary nil")
		}
		if summary.Total != 3 {
			t.Fatalf("summary.Total = %d, esperado 3; items: %+v", summary.Total, summary.Items)
		}
		if len(summary.Items) != 3 {
			t.Fatalf("len(summary.Items) = %d, esperado 3", len(summary.Items))
		}

		// Verifica ordenação determinística por File e Line
		if summary.Items[0].File != "cmd/app/main.go" || summary.Items[0].Line != 3 {
			t.Errorf("Items[0] = %+v, esperado File=cmd/app/main.go Line=3", summary.Items[0])
		}
		if summary.Items[0].Limit != "configuração fixa em flags" || summary.Items[0].UpgradeTrigger != "suportar arquivo YAML se passar de 10 opções" {
			t.Errorf("Items[0] limit/trigger incorretos: %+v", summary.Items[0])
		}

		if summary.Items[1].File != "internal/store/memory.go" || summary.Items[1].Line != 4 {
			t.Errorf("Items[1] = %+v, esperado File=internal/store/memory.go Line=4", summary.Items[1])
		}
		if summary.Items[2].File != "internal/store/memory.go" || summary.Items[2].Line != 8 {
			t.Errorf("Items[2] = %+v, esperado File=internal/store/memory.go Line=8", summary.Items[2])
		}
	})

	t.Run("diretorio sem atalhos retorna Total 0 e slice vazio nao-nil", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "clean.go"), []byte("package clean\n\n// código limpo\nfunc OK() {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		summary, err := debt.ScanDir(tmpDir)
		if err != nil {
			t.Fatalf("ScanDir falhou: %v", err)
		}
		if summary.Total != 0 {
			t.Errorf("summary.Total = %d, esperado 0", summary.Total)
		}
		if summary.Items == nil || len(summary.Items) != 0 {
			t.Errorf("summary.Items = %v, esperado slice vazio não-nil", summary.Items)
		}
	})
}

func FuzzParseComment(f *testing.F) {
	f.Add("// shortcut: armazenamento em memória | upgrade: quando exceder 10k registros")
	f.Add("// shortcut: buffer fixo de 512KB, migrar para streaming acima de 1MB")
	f.Add("// TODO: ajustar depois")
	f.Add("// shortcut: | upgrade:")
	f.Add("")

	f.Fuzz(func(t *testing.T, line string) {
		limit, trigger, ok := debt.ParseComment(line)
		if ok {
			if limit == "" || trigger == "" {
				t.Errorf("ParseComment(%q) retornou ok=true com limit=%q ou trigger=%q vazios", line, limit, trigger)
			}
		} else {
			if limit != "" || trigger != "" {
				t.Errorf("ParseComment(%q) retornou ok=false mas limit=%q, trigger=%q não vazios", line, limit, trigger)
			}
		}
	})
}
