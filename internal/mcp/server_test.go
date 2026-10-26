package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runRPC(t *testing.T, s *Server, req string) map[string]any {
	t.Helper()
	in := bytes.NewBufferString(req + "\n")
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	if err := s.ServeStream(in, out, errOut); err != nil && err != io.EOF {
		t.Fatalf("ServeStream falhou: %v", err)
	}

	scanner := bufio.NewScanner(out)
	if !scanner.Scan() {
		t.Fatalf("nenhuma resposta emitida no stdout")
	}

	var res map[string]any
	if err := json.Unmarshal(scanner.Bytes(), &res); err != nil {
		t.Fatalf("resposta não é JSON válido: %v, raw: %s", err, scanner.Text())
	}
	return res
}

func TestMCPInitializeAndPing(t *testing.T) {
	s := NewServer(".")

	// 1. Initialize
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0"}}}`
	res := runRPC(t, s, initReq)

	if res["jsonrpc"] != "2.0" {
		t.Errorf("esperava jsonrpc 2.0, obteve %v", res["jsonrpc"])
	}
	if res["id"] != float64(1) {
		t.Errorf("esperava id 1, obteve %v", res["id"])
	}

	result, ok := res["result"].(map[string]any)
	if !ok {
		t.Fatalf("campo result ausente ou inválido: %v", res)
	}
	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("protocolVersion inesperado: %v", result["protocolVersion"])
	}

	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok || serverInfo["name"] != "pipeline-ai" {
		t.Errorf("serverInfo inválido: %v", serverInfo)
	}

	// 2. Ping
	pingReq := `{"jsonrpc":"2.0","id":2,"method":"ping","params":{}}`
	res = runRPC(t, s, pingReq)
	if res["id"] != float64(2) {
		t.Errorf("esperava id 2 no ping, obteve %v", res["id"])
	}
}

func TestMCPToolsList(t *testing.T) {
	s := NewServer(".")

	req := `{"jsonrpc":"2.0","id":10,"method":"tools/list","params":{}}`
	res := runRPC(t, s, req)

	result, ok := res["result"].(map[string]any)
	if !ok {
		t.Fatalf("result ausente no tools/list: %v", res)
	}

	tools, ok := result["tools"].([]any)
	if !ok || len(tools) < 2 {
		t.Fatalf("esperava ao menos 2 ferramentas em tools/list, obteve: %v", tools)
	}

	hasDoctor := false
	hasValidate := false
	for _, raw := range tools {
		tool, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if tool["name"] == "pipeline_doctor" {
			hasDoctor = true
		}
		if tool["name"] == "pipeline_validate" {
			hasValidate = true
		}
	}

	if !hasDoctor {
		t.Errorf("ferramenta 'pipeline_doctor' não encontrada em tools/list")
	}
	if !hasValidate {
		t.Errorf("ferramenta 'pipeline_validate' não encontrada em tools/list")
	}
}

func TestMCPToolsCallDoctor(t *testing.T) {
	s := NewServer("../..") // raiz do projeto

	req := `{"jsonrpc":"2.0","id":20,"method":"tools/call","params":{"name":"pipeline_doctor","arguments":{"dir":"."}}}`
	res := runRPC(t, s, req)

	result, ok := res["result"].(map[string]any)
	if !ok {
		t.Fatalf("result ausente no tools/call: %v", res)
	}

	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("content vazio no tools/call: %v", result)
	}

	first, ok := content[0].(map[string]any)
	if !ok || first["type"] != "text" {
		t.Fatalf("formato de content inválido: %v", content)
	}

	text, ok := first["text"].(string)
	if !ok || !strings.Contains(text, "Go Runtime") {
		t.Errorf("texto do resultado do doctor não contém diagnóstico esperado: %s", text)
	}
}

func TestMCPToolsCallValidate(t *testing.T) {
	s := NewServer("../..")

	// 1. Chamada sem parâmetro report_path
	reqMissing := `{"jsonrpc":"2.0","id":30,"method":"tools/call","params":{"name":"pipeline_validate","arguments":{}}}`
	res := runRPC(t, s, reqMissing)
	result, _ := res["result"].(map[string]any)
	if result["isError"] != true {
		t.Errorf("esperava isError true quando report_path está ausente")
	}

	// 2. Chamada em relatório válido
	reqValid := `{"jsonrpc":"2.0","id":31,"method":"tools/call","params":{"name":"pipeline_validate","arguments":{"report_path":"docs-ai/deliveries/DEL-001/report.md","no_git":true}}}`
	res = runRPC(t, s, reqValid)
	result, _ = res["result"].(map[string]any)
	if result["isError"] == true {
		t.Errorf("esperava isError false para DEL-001 report.md")
	}
}

func TestMCPResources(t *testing.T) {
	s := NewServer("../..")

	// 1. resources/list
	reqList := `{"jsonrpc":"2.0","id":40,"method":"resources/list","params":{}}`
	res := runRPC(t, s, reqList)
	result, ok := res["result"].(map[string]any)
	if !ok {
		t.Fatalf("result ausente em resources/list: %v", res)
	}

	resources, ok := result["resources"].([]any)
	if !ok || len(resources) < 3 {
		t.Fatalf("esperava ao menos 3 resources, obteve: %v", resources)
	}

	// 2. resources/read pipeline://invariants
	reqRead := `{"jsonrpc":"2.0","id":41,"method":"resources/read","params":{"uri":"pipeline://invariants"}}`
	res = runRPC(t, s, reqRead)
	result, ok = res["result"].(map[string]any)
	if !ok {
		t.Fatalf("result ausente em resources/read: %v", res)
	}
	contents, ok := result["contents"].([]any)
	if !ok || len(contents) == 0 {
		t.Fatalf("contents vazio em resources/read: %v", result)
	}
	first, _ := contents[0].(map[string]any)
	if text, ok := first["text"].(string); !ok || len(text) == 0 {
		t.Errorf("texto de pipeline://invariants vazio ou inválido")
	}

	// 3. resources/read pipeline://active-delivery
	reqActive := `{"jsonrpc":"2.0","id":42,"method":"resources/read","params":{"uri":"pipeline://active-delivery"}}`
	res = runRPC(t, s, reqActive)
	result, _ = res["result"].(map[string]any)
	contents, _ = result["contents"].([]any)
	if len(contents) == 0 {
		t.Fatalf("contents vazio em pipeline://active-delivery")
	}
}

func TestMCPScaffoldDelivery(t *testing.T) {
	tmpDir := t.TempDir()

	// Configura estrutura básica de docs-ai/deliveries/INDEX.md
	delivDir := filepath.Join(tmpDir, "docs-ai", "deliveries")
	if err := os.MkdirAll(delivDir, 0755); err != nil {
		t.Fatal(err)
	}
	initialIndex := "# Índice de Deliveries\n\n| ID | Task | Título | Tipo | Data | Status |\n|---|---|---|---|---|---|\n| — | — | nenhuma delivery registrada | — | — | — |\n\n<!--\n| DEL-001 | TASK-42 | Exemplo | feat | 2026-09-17 | merged |\n-->"
	if err := os.WriteFile(filepath.Join(delivDir, "INDEX.md"), []byte(initialIndex), 0644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(tmpDir)

	// Executa pipeline_scaffold_delivery
	req := `{"jsonrpc":"2.0","id":50,"method":"tools/call","params":{"name":"pipeline_scaffold_delivery","arguments":{"title":"Nova Feature Teste","type":"feat"}}}`
	res := runRPC(t, s, req)
	result, ok := res["result"].(map[string]any)
	if !ok {
		t.Fatalf("result ausente em pipeline_scaffold_delivery: %v", res)
	}
	if result["isError"] == true {
		t.Fatalf("pipeline_scaffold_delivery retornou erro: %v", result)
	}

	// Verifica se DEL-001 foi criado ignorando o comentário HTML
	expectedReport := filepath.Join(delivDir, "DEL-001", "report.md")
	content, err := os.ReadFile(expectedReport)
	if err != nil {
		t.Fatalf("report.md não foi criado em %s: %v", expectedReport, err)
	}
	reportStr := string(content)
	if !strings.Contains(reportStr, "delivery_id: DEL-001") {
		t.Errorf("report.md não contém delivery_id: DEL-001, obtido: %s", reportStr)
	}
	if !strings.Contains(reportStr, "Nova Feature Teste") {
		t.Errorf("report.md não contém o título fornecido")
	}

	// Verifica se INDEX.md foi atualizado com DEL-001 no topo e removeu a linha placeholder
	indexBytes, err := os.ReadFile(filepath.Join(delivDir, "INDEX.md"))
	if err != nil {
		t.Fatal(err)
	}
	indexStr := string(indexBytes)
	if !strings.Contains(indexStr, "DEL-001") {
		t.Errorf("INDEX.md não contém a nova delivery DEL-001")
	}
	if strings.Contains(indexStr, "nenhuma delivery registrada") {
		t.Errorf("INDEX.md ainda contém a linha placeholder 'nenhuma delivery registrada'")
	}
}

func TestMCPSyncSkills(t *testing.T) {
	tmpDir := t.TempDir()
	// Cria pasta .agents prévia para simular projeto existente
	if err := os.MkdirAll(filepath.Join(tmpDir, ".agents"), 0755); err != nil {
		t.Fatal(err)
	}
	s := NewServer(tmpDir)

	req := `{"jsonrpc":"2.0","id":60,"method":"tools/call","params":{"name":"pipeline_sync_skills","arguments":{"dir":"."}}}`
	res := runRPC(t, s, req)
	result, ok := res["result"].(map[string]any)
	if !ok {
		t.Fatalf("result ausente em pipeline_sync_skills: %v", res)
	}
	if result["isError"] == true {
		t.Fatalf("pipeline_sync_skills retornou erro inesperado: %v", result)
	}

	// Verifica se .agents/skills foi criado
	skillsDir := filepath.Join(tmpDir, ".agents", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil || len(entries) < 10 {
		t.Errorf("esperava skills sincronizadas em %s, obteve %d entradas", skillsDir, len(entries))
	}
}

func TestScaffoldDeliveryCustomPaths(t *testing.T) {
	tmpDir := t.TempDir()

	customDeliveriesDir := "custom-docs/deliveries"
	customIndexFile := "custom-docs/deliveries/INDEX.md"

	cfgBody := fmt.Sprintf(`{
  "version": 1,
  "project_type": "desktop",
  "protected_paths": ["internal/"],
  "deliveries_dir": %q,
  "index_file": %q
}`, customDeliveriesDir, customIndexFile)

	if err := os.WriteFile(filepath.Join(tmpDir, ".pipeline-ai.json"), []byte(cfgBody), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	customDir := filepath.Join(tmpDir, filepath.FromSlash(customDeliveriesDir))
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatalf("mkdir customDir: %v", err)
	}

	initialIndex := "# Índice de Deliveries\n\n| ID | Task | Título | Tipo | Data | Status |\n|---|---|---|---|---|---|\n| — | — | nenhuma delivery registrada | — | — | — |\n"
	if err := os.WriteFile(filepath.Join(tmpDir, filepath.FromSlash(customIndexFile)), []byte(initialIndex), 0o644); err != nil {
		t.Fatalf("write custom index: %v", err)
	}

	deliveryID, reportPath, err := ScaffoldDelivery(tmpDir, "Custom Delivery Title", "feat", "TASK-001")
	if err != nil {
		t.Fatalf("ScaffoldDelivery falhou: %v", err)
	}

	if deliveryID != "DEL-001" {
		t.Errorf("deliveryID = %q, quer DEL-001", deliveryID)
	}

	expectedReport := filepath.Join(customDir, "DEL-001", "report.md")
	if reportPath != expectedReport {
		t.Errorf("reportPath = %q, quer %q", reportPath, expectedReport)
	}

	if _, err := os.Stat(expectedReport); err != nil {
		t.Errorf("relatório não foi criado no caminho customizado %s: %v", expectedReport, err)
	}

	// Garante que não criou em docs-ai/deliveries
	unwantedDir := filepath.Join(tmpDir, "docs-ai", "deliveries")
	if _, err := os.Stat(unwantedDir); !os.IsNotExist(err) {
		t.Errorf("docs-ai/deliveries não deveria ter sido criado")
	}

	// Valida que custom-docs/deliveries/INDEX.md foi atualizado
	indexContent, err := os.ReadFile(filepath.Join(tmpDir, filepath.FromSlash(customIndexFile)))
	if err != nil {
		t.Fatalf("falha ao ler index customizado: %v", err)
	}
	if !strings.Contains(string(indexContent), "DEL-001") {
		t.Errorf("INDEX customizado não contém DEL-001: %s", string(indexContent))
	}
	if strings.Contains(string(indexContent), "nenhuma delivery registrada") {
		t.Errorf("INDEX customizado ainda contém placeholder")
	}
}
