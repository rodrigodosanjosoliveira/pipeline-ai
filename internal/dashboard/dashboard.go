// Package dashboard coleta métricas, histórico de entregas e governança do pipeline-ai
// para gerar um dashboard HTML interativo, autocontido e responsivo.
package dashboard

import (
	"bytes"
	"errors"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics"
)

var (
	ErrNotImplemented = errors.New("não implementado")
	ErrNilData        = errors.New("dados do dashboard não podem ser nulos")
)

// DeliverySummary representa uma linha da tabela de entregas no dashboard.
type DeliverySummary struct {
	ID     string `json:"id"`
	Task   string `json:"task"`
	Title  string `json:"title"`
	Type   string `json:"type"`
	Date   string `json:"date"`
	Status string `json:"status"`
}

// PillarInfo resume um dos 4 pilares de governança da arquitetura pipeline-ai.
type PillarInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// SkillInfo resume uma skill da esteira do pipeline.
type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Optional    bool   `json:"optional"`
}

// DashboardData agrega todas as informações necessárias para renderizar o dashboard HTML.
type DashboardData struct {
	ProjectName  string              `json:"project_name"`
	ProjectType  string              `json:"project_type"`
	GeneratedAt  time.Time           `json:"generated_at"`
	DORA         metrics.DORAMetrics `json:"dora"`
	Pillars      []PillarInfo        `json:"pillars"`
	Skills       []SkillInfo         `json:"skills"`
	Deliveries   []DeliverySummary   `json:"deliveries"`
	TotalDeploys int                 `json:"total_deploys"`
}

// CollectDashboardData coleta dados de métricas, entregas e configuração a partir de rootDir.
func CollectDashboardData(rootDir string) (*DashboardData, error) {
	if rootDir == "" {
		rootDir = "."
	}

	projectName := resolveProjectName(rootDir)
	projectType := "api"

	cfg, _ := config.Load(rootDir)
	if cfg != nil && cfg.ProjectType != "" {
		projectType = cfg.ProjectType
	}

	// 1. Métricas DORA
	dora := metrics.DORAMetrics{}
	if d, err := metrics.GetDORA(rootDir); err == nil && d != nil {
		dora = *d
	}

	// 2. Entregas registradas em INDEX.md
	deliveries := loadDeliveries(rootDir, cfg)

	// 3. Pilares de governança
	pillars := defaultPillars()

	// 4. Catálogo de skills da esteira
	skills := defaultSkills()

	return &DashboardData{
		ProjectName:  projectName,
		ProjectType:  projectType,
		GeneratedAt:  time.Now(),
		DORA:         dora,
		Pillars:      pillars,
		Skills:       skills,
		Deliveries:   deliveries,
		TotalDeploys: dora.DeploymentsTotal,
	}, nil
}

func resolveProjectName(rootDir string) string {
	goModPath := filepath.Join(rootDir, "go.mod")
	if data, err := os.ReadFile(goModPath); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, l := range lines {
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "module ") {
				mod := strings.TrimSpace(strings.TrimPrefix(l, "module"))
				if mod != "" {
					return filepath.Base(mod)
				}
			}
		}
	}
	base := filepath.Base(rootDir)
	if base == "." || base == "/" || base == "" {
		return "pipeline-ai"
	}
	return base
}

func defaultPillars() []PillarInfo {
	return []PillarInfo{
		{
			Name:        "PreToolUse Fail-Closed",
			Description: "Bloqueio determinístico de edição direta sem agente autorizado",
			Status:      "Ativo",
		},
		{
			Name:        "Invariantes Globais",
			Description: "Go 1.24+, stdlib testing, zero CGO desktop e governança estrita",
			Status:      "Ativo",
		},
		{
			Name:        "Gates Determinísticos",
			Description: "Suíte integrada de validação doctor, sensor e pipeline runner",
			Status:      "Ativo",
		},
		{
			Name:        "Rastreabilidade e Segurança",
			Description: "Contratos de handoff rastreáveis e shift-left security no PRE e POST",
			Status:      "Ativo",
		},
	}
}

func defaultSkills() []SkillInfo {
	return []SkillInfo{
		{Name: "pm-triage", Description: "Ponto de entrada, triagem e delimitação do escopo", Optional: false},
		{Name: "architecture-guardrails", Description: "Limites de arquitetura e requisitos de segurança", Optional: false},
		{Name: "data-model", Description: "Modelagem de dados e esquemas relacionais/NoSQL", Optional: true},
		{Name: "test-writer", Description: "Testes Red phase automatizados antes da implementação", Optional: false},
		{Name: "dev-implementation", Description: "Implementação Green phase estritamente dentro do escopo", Optional: false},
		{Name: "code-reviewer", Description: "Revisão estática de código e contratos de qualidade", Optional: false},
		{Name: "migration-reviewer", Description: "Revisão de scripts DDL de migração de banco", Optional: true},
		{Name: "query-performance", Description: "Análise de performance de queries e índices", Optional: true},
		{Name: "security-review", Description: "Auditoria de segurança shift-left da superfície", Optional: false},
		{Name: "supply-chain-review", Description: "Auditoria de dependências externas e go.mod", Optional: false},
		{Name: "qa-validation", Description: "Validação comportamental e execução de gates", Optional: false},
		{Name: "release-ops", Description: "Versionamento semântico e empacotamento operacional", Optional: true},
		{Name: "agent-auditor", Description: "Auditoria de qualidade e conformidade de handoffs", Optional: true},
		{Name: "context-loader", Description: "Carga contextual e síntese de documentações", Optional: true},
		{Name: "docs-update", Description: "Atualização documental e relatório de delivery", Optional: false},
		{Name: "git-delivery", Description: "Formalização de branch, commit e Pull Request", Optional: false},
	}
}

func loadDeliveries(rootDir string, cfg *config.Config) []DeliverySummary {
	deliveries := []DeliverySummary{}

	indexPath := filepath.Join(rootDir, "docs-ai", "deliveries", "INDEX.md")
	if cfg != nil && cfg.IndexFile != "" {
		cand := filepath.Join(rootDir, filepath.FromSlash(cfg.IndexFile))
		if _, err := os.Stat(cand); err == nil {
			indexPath = cand
		}
	}

	data, err := os.ReadFile(indexPath)
	if err != nil {
		return deliveries
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		rawParts := strings.Split(line, "|")
		var parts []string
		for _, p := range rawParts {
			p = strings.TrimSpace(p)
			if p != "" {
				parts = append(parts, p)
			}
		}
		if len(parts) < 3 {
			continue
		}
		if strings.EqualFold(parts[0], "ID") || strings.Contains(parts[0], "---") || strings.Contains(parts[1], "---") {
			continue
		}
		if !strings.HasPrefix(parts[0], "DEL-") {
			continue
		}

		item := DeliverySummary{
			ID:    parts[0],
			Task:  parts[1],
			Title: parts[2],
		}
		if len(parts) == 4 {
			item.Status = parts[3]
		} else if len(parts) == 5 {
			item.Type = parts[3]
			item.Status = parts[4]
		} else if len(parts) >= 6 {
			item.Type = parts[3]
			item.Date = parts[4]
			item.Status = parts[5]
		}

		deliveries = append(deliveries, item)
	}

	return deliveries
}

const dashboardHTMLTemplate = `<!DOCTYPE html>
<html lang="pt-BR" class="dark">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Pipeline-AI Engineering Dashboard - {{.ProjectName}}</title>
  <script src="https://www.gstatic.com/antigravity/web/dev/tailwindcss.min.js"></script>
</head>
<body class="bg-slate-950 text-slate-100 min-h-screen font-sans antialiased">
  <header class="border-b border-slate-800 bg-slate-900/60 backdrop-blur sticky top-0 z-50">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4 flex flex-wrap items-center justify-between gap-4">
      <div class="flex items-center gap-3">
        <div class="h-10 w-10 rounded-xl bg-indigo-600 flex items-center justify-center font-black text-xl text-white shadow-lg shadow-indigo-500/30">
          π
        </div>
        <div>
          <div class="flex items-center gap-2">
            <h1 class="text-xl font-bold tracking-tight text-white">{{.ProjectName}}</h1>
            <span class="px-2.5 py-0.5 rounded-full text-xs font-semibold uppercase tracking-wider bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
              {{.ProjectType}}
            </span>
          </div>
          <p class="text-xs text-slate-400">Dashboard de Governança e Métricas de Engenharia</p>
        </div>
      </div>
      <div class="flex items-center gap-4 text-xs text-slate-400">
        <div>
          Total de Deploys: <span class="font-semibold text-slate-200">{{.TotalDeploys}}</span>
        </div>
        <div class="border-l border-slate-800 pl-4">
          Gerado em: <span class="font-mono text-slate-300">{{.GeneratedAt.Format "2006-01-02 15:04:05 UTC"}}</span>
        </div>
      </div>
    </div>
  </header>

  <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8">
    <!-- Navegação em Abas -->
    <div class="border-b border-slate-800 flex gap-2">
      <button id="btn-tab-dora" onclick="showTab('tab-dora')" class="tab-btn px-4 py-2 text-sm font-medium rounded-t-lg bg-indigo-600 text-white transition-colors">
        Métricas DORA
      </button>
      <button id="btn-tab-pipeline" onclick="showTab('tab-pipeline')" class="tab-btn px-4 py-2 text-sm font-medium rounded-t-lg text-slate-400 hover:text-slate-200 hover:bg-slate-900 transition-colors">
        Esteira de Skills
      </button>
      <button id="btn-tab-deliveries" onclick="showTab('tab-deliveries')" class="tab-btn px-4 py-2 text-sm font-medium rounded-t-lg text-slate-400 hover:text-slate-200 hover:bg-slate-900 transition-colors">
        Entregas Recentes ({{len .Deliveries}})
      </button>
      <button id="btn-tab-pillars" onclick="showTab('tab-pillars')" class="tab-btn px-4 py-2 text-sm font-medium rounded-t-lg text-slate-400 hover:text-slate-200 hover:bg-slate-900 transition-colors">
        Pilares de Governança
      </button>
    </div>

    <!-- Aba 1: Métricas DORA -->
    <section id="tab-dora" class="tab-content space-y-6">
      <div>
        <h2 class="text-lg font-bold text-white">Quatro Métricas-Chave DORA (DevOps Research and Assessment)</h2>
        <p class="text-sm text-slate-400">Indicadores objetivos de velocidade de entrega e estabilidade operacional.</p>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <!-- Deployment Frequency -->
        <div class="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm space-y-2">
          <div class="flex items-center justify-between">
            <span class="text-xs font-semibold uppercase tracking-wider text-slate-400">Deployment Frequency</span>
            <span class="px-2 py-0.5 text-xs rounded bg-blue-500/10 text-blue-400 border border-blue-500/20 font-medium">Cadência</span>
          </div>
          <div class="text-2xl font-black text-white">
            {{printf "%.2f" .DORA.DeploymentFrequency}} <span class="text-sm font-normal text-slate-400">deploys/dia</span>
          </div>
          <p class="text-xs text-slate-400">Total de {{.DORA.DeploymentsTotal}} deploys na janela</p>
        </div>

        <!-- Lead Time for Changes -->
        <div class="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm space-y-2">
          <div class="flex items-center justify-between">
            <span class="text-xs font-semibold uppercase tracking-wider text-slate-400">Lead Time for Changes</span>
            <span class="px-2 py-0.5 text-xs rounded bg-indigo-500/10 text-indigo-400 border border-indigo-500/20 font-medium">Velocidade</span>
          </div>
          <div class="text-2xl font-black text-white">
            {{if eq .DORA.LeadTimeForChanges 0}}0s{{else}}{{.DORA.LeadTimeForChanges}}{{end}}
          </div>
          <p class="text-xs text-slate-400">Tempo médio de ciclo por entrega</p>
        </div>

        <!-- Change Failure Rate -->
        <div class="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm space-y-2">
          <div class="flex items-center justify-between">
            <span class="text-xs font-semibold uppercase tracking-wider text-slate-400">Change Failure Rate</span>
            <span class="px-2 py-0.5 text-xs rounded bg-amber-500/10 text-amber-400 border border-amber-500/20 font-medium">Estabilidade</span>
          </div>
          <div class="text-2xl font-black text-white">
            {{printf "%.1f" .DORA.ChangeFailureRate}}%
          </div>
          <p class="text-xs text-slate-400">{{.DORA.FailedDeployments}} falhas registradas</p>
        </div>

        <!-- Time to Restore -->
        <div class="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm space-y-2">
          <div class="flex items-center justify-between">
            <span class="text-xs font-semibold uppercase tracking-wider text-slate-400">Time to Restore</span>
            <span class="px-2 py-0.5 text-xs rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 font-medium">Recuperação</span>
          </div>
          <div class="text-2xl font-black text-white">
            {{if eq .DORA.TimeToRestore 0}}0s{{else}}{{.DORA.TimeToRestore}}{{end}}
          </div>
          <p class="text-xs text-slate-400">{{.DORA.RestoresTotal}} restaurações realizadas</p>
        </div>
      </div>
    </section>

    <!-- Aba 2: Esteira de Skills -->
    <section id="tab-pipeline" class="tab-content hidden space-y-6">
      <div>
        <h2 class="text-lg font-bold text-white">Esteira de Execução e Catálogo de Skills Nativas</h2>
        <p class="text-sm text-slate-400">Fluxo determinístico com gates e transição automatizada entre agentes.</p>
      </div>

      <!-- Steps da Esteira Visual -->
      <div class="bg-slate-900 border border-slate-800 rounded-xl p-6 overflow-x-auto shadow-sm">
        <div class="text-xs font-semibold uppercase tracking-wider text-slate-400 mb-4">Fluxo Principal da Esteira</div>
        <div class="flex items-center gap-2 min-w-[900px]">
          <div class="flex-1 bg-slate-800/80 border border-slate-700/60 rounded-lg p-3 text-center">
            <div class="text-xs font-bold text-indigo-400">1. pm-triage</div>
            <div class="text-xs text-slate-400 mt-1">Triagem &amp; Escopo</div>
          </div>
          <div class="text-slate-500 font-bold">→</div>
          <div class="flex-1 bg-slate-800/80 border border-slate-700/60 rounded-lg p-3 text-center">
            <div class="text-xs font-bold text-indigo-400">2. architecture-guardrails</div>
            <div class="text-xs text-slate-400 mt-1">Guardrails &amp; PRE</div>
          </div>
          <div class="text-slate-500 font-bold">→</div>
          <div class="flex-1 bg-slate-800/80 border border-slate-700/60 rounded-lg p-3 text-center">
            <div class="text-xs font-bold text-indigo-400">3. test-writer</div>
            <div class="text-xs text-slate-400 mt-1">Testes Red Phase</div>
          </div>
          <div class="text-slate-500 font-bold">→</div>
          <div class="flex-1 bg-slate-800/80 border border-slate-700/60 rounded-lg p-3 text-center">
            <div class="text-xs font-bold text-indigo-400">4. dev-implementation</div>
            <div class="text-xs text-slate-400 mt-1">Green Phase</div>
          </div>
          <div class="text-slate-500 font-bold">→</div>
          <div class="flex-1 bg-slate-800/80 border border-slate-700/60 rounded-lg p-3 text-center">
            <div class="text-xs font-bold text-indigo-400">5. code-reviewer</div>
            <div class="text-xs text-slate-400 mt-1">Revisão Estática</div>
          </div>
          <div class="text-slate-500 font-bold">→</div>
          <div class="flex-1 bg-slate-800/80 border border-slate-700/60 rounded-lg p-3 text-center">
            <div class="text-xs font-bold text-indigo-400">6. security-review</div>
            <div class="text-xs text-slate-400 mt-1">Auditoria Shift-Left</div>
          </div>
          <div class="text-slate-500 font-bold">→</div>
          <div class="flex-1 bg-slate-800/80 border border-slate-700/60 rounded-lg p-3 text-center">
            <div class="text-xs font-bold text-indigo-400">7. qa-validation</div>
            <div class="text-xs text-slate-400 mt-1">Validação dos Gates</div>
          </div>
          <div class="text-slate-500 font-bold">→</div>
          <div class="flex-1 bg-slate-800/80 border border-slate-700/60 rounded-lg p-3 text-center">
            <div class="text-xs font-bold text-indigo-400">8. docs-update</div>
            <div class="text-xs text-slate-400 mt-1">Relatório &amp; Doc</div>
          </div>
          <div class="text-slate-500 font-bold">→</div>
          <div class="flex-1 bg-slate-800/80 border border-slate-700/60 rounded-lg p-3 text-center">
            <div class="text-xs font-bold text-indigo-400">9. git-delivery</div>
            <div class="text-xs text-slate-400 mt-1">Branch, Commit &amp; PR</div>
          </div>
        </div>
      </div>

      <!-- Catálogo das 16 Skills -->
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {{range .Skills}}
        <div class="bg-slate-900 border border-slate-800 rounded-xl p-4 shadow-sm space-y-2">
          <div class="flex items-center justify-between">
            <span class="font-mono text-sm font-semibold text-white">{{.Name}}</span>
            {{if .Optional}}
            <span class="px-2 py-0.5 text-xs rounded bg-slate-800 text-slate-400 border border-slate-700">Opcional</span>
            {{else}}
            <span class="px-2 py-0.5 text-xs rounded bg-indigo-500/10 text-indigo-400 border border-indigo-500/20 font-semibold">Obrigatória</span>
            {{end}}
          </div>
          <p class="text-xs text-slate-400 leading-relaxed">{{.Description}}</p>
        </div>
        {{end}}
      </div>
    </section>

    <!-- Aba 3: Entregas Recentes -->
    <section id="tab-deliveries" class="tab-content hidden space-y-6">
      <div class="flex items-center justify-between">
        <div>
          <h2 class="text-lg font-bold text-white">Histórico e Status de Deliveries</h2>
          <p class="text-sm text-slate-400">Entregas formalizadas registradas no arquivo de índice do pipeline.</p>
        </div>
      </div>

      <div class="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-sm">
        <div class="overflow-x-auto">
          <table class="w-full text-left border-collapse text-sm">
            <thead>
              <tr class="border-b border-slate-800 bg-slate-900/80 text-xs font-semibold text-slate-400 uppercase tracking-wider">
                <th class="px-4 py-3">ID</th>
                <th class="px-4 py-3">Task</th>
                <th class="px-4 py-3">Título</th>
                <th class="px-4 py-3">Tipo</th>
                <th class="px-4 py-3">Data</th>
                <th class="px-4 py-3">Status</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-800/60">
              {{range .Deliveries}}
              <tr class="hover:bg-slate-800/40 transition-colors">
                <td class="px-4 py-3 font-mono font-bold text-indigo-400">{{.ID}}</td>
                <td class="px-4 py-3 font-mono text-slate-300">{{.Task}}</td>
                <td class="px-4 py-3 text-slate-200 font-medium">{{.Title}}</td>
                <td class="px-4 py-3">
                  <span class="px-2 py-0.5 rounded text-xs font-medium bg-slate-800 text-slate-300">
                    {{.Type}}
                  </span>
                </td>
                <td class="px-4 py-3 font-mono text-xs text-slate-400">{{.Date}}</td>
                <td class="px-4 py-3">
                  <span class="px-2.5 py-0.5 rounded-full text-xs font-semibold uppercase tracking-wider
                    {{if eq .Status "merged"}}bg-emerald-500/10 text-emerald-400 border border-emerald-500/20
                    {{else if eq .Status "draft"}}bg-amber-500/10 text-amber-400 border border-amber-500/20
                    {{else}}bg-blue-500/10 text-blue-400 border border-blue-500/20{{end}}">
                    {{.Status}}
                  </span>
                </td>
              </tr>
              {{else}}
              <tr>
                <td colspan="6" class="px-4 py-8 text-center text-sm text-slate-500">
                  Nenhuma delivery registrada no momento.
                </td>
              </tr>
              {{end}}
            </tbody>
          </table>
        </div>
      </div>
    </section>

    <!-- Aba 4: Pilares de Governança -->
    <section id="tab-pillars" class="tab-content hidden space-y-6">
      <div>
        <h2 class="text-lg font-bold text-white">Quatro Pilares Fundamentais de Governança</h2>
        <p class="text-sm text-slate-400">Salvaguardas que garantem autonomia sem alucinação e conformidade determinística.</p>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        {{range .Pillars}}
        <div class="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm space-y-2">
          <div class="flex items-center justify-between">
            <span class="font-bold text-white text-base">{{.Name}}</span>
            <span class="px-2.5 py-0.5 rounded-full text-xs font-semibold uppercase tracking-wider bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
              {{.Status}}
            </span>
          </div>
          <p class="text-sm text-slate-400 leading-relaxed">{{.Description}}</p>
        </div>
        {{end}}
      </div>
    </section>
  </main>

  <footer class="border-t border-slate-800 bg-slate-950 py-6 text-center text-xs text-slate-500">
    pipeline-ai • Toolkit de Governança Determinística para Agentes de IA
  </footer>

  <script>
    function showTab(tabId) {
      document.querySelectorAll('.tab-content').forEach(el => el.classList.add('hidden'));
      document.querySelectorAll('.tab-btn').forEach(el => {
        el.classList.remove('bg-indigo-600', 'text-white');
        el.classList.add('text-slate-400', 'hover:text-slate-200', 'hover:bg-slate-900');
      });
      const activeTab = document.getElementById(tabId);
      if (activeTab) activeTab.classList.remove('hidden');
      const activeBtn = document.getElementById('btn-' + tabId);
      if (activeBtn) {
        activeBtn.classList.remove('text-slate-400', 'hover:text-slate-200', 'hover:bg-slate-900');
        activeBtn.classList.add('bg-indigo-600', 'text-white');
      }
    }
  </script>
</body>
</html>
`

var parsedTemplate = template.Must(template.New("dashboard").Parse(dashboardHTMLTemplate))

// RenderHTML renderiza a página HTML interativa autocontida com TailwindCSS a partir dos dados agregados.
func RenderHTML(data *DashboardData) (string, error) {
	if data == nil {
		return "", ErrNilData
	}

	var buf bytes.Buffer
	if err := parsedTemplate.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
