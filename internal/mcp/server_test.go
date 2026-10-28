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
	if serverInfo["version"] != "v0.6.0" {
		t.Errorf("serverInfo[\"version\"] = %v, esperado %q", serverInfo["version"], "v0.6.0")
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

func TestMCPScaffoldOpensActiveDelivery(t *testing.T) {
	tmpDir := t.TempDir()
	delivDir := filepath.Join(tmpDir, "docs-ai", "deliveries")
	if err := os.MkdirAll(delivDir, 0755); err != nil {
		t.Fatal(err)
	}
	initialIndex := "# Índice de Deliveries\n\n| ID | Task | Título | Tipo | Data | Status |\n|---|---|---|---|---|---|\n| — | — | nenhuma delivery registrada | — | — | — |\n"
	if err := os.WriteFile(filepath.Join(delivDir, "INDEX.md"), []byte(initialIndex), 0644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(tmpDir)

	req := `{"jsonrpc":"2.0","id":70,"method":"tools/call","params":{"name":"pipeline_scaffold_delivery","arguments":{"title":"Scaffold Ativo Teste","type":"feat","task_code":"TASK-100"}}}`
	res := runRPC(t, s, req)
	result, ok := res["result"].(map[string]any)
	if !ok || result["isError"] == true {
		t.Fatalf("pipeline_scaffold_delivery falhou: %v", res)
	}

	// Verifica se .pipeline-state/active-delivery.json foi criado com DEL-001
	stateFile := filepath.Join(tmpDir, ".pipeline-state", "active-delivery.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("arquivo %s não foi criado pelo scaffold: %v", stateFile, err)
	}
	stateStr := string(data)
	if !strings.Contains(stateStr, "DEL-001") {
		t.Errorf("active-delivery.json não contém DEL-001: %s", stateStr)
	}
	if !strings.Contains(stateStr, "TASK-100") {
		t.Errorf("active-delivery.json não contém TASK-100: %s", stateStr)
	}
}

func TestMCPCloseDelivery(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".pipeline-state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Cria uma delivery ativa prévia
	activeContent := `{"delivery_id":"DEL-050","task_code":"TASK-050","opened_at":"2026-10-26T12:00:00Z","expires_at":"2026-10-26T14:00:00Z"}`
	stateFile := filepath.Join(stateDir, "active-delivery.json")
	if err := os.WriteFile(stateFile, []byte(activeContent), 0644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(tmpDir)

	// 1. Verifica se pipeline_close_delivery está listado em tools/list
	reqList := `{"jsonrpc":"2.0","id":80,"method":"tools/list","params":{}}`
	resList := runRPC(t, s, reqList)
	resultList, _ := resList["result"].(map[string]any)
	tools, _ := resultList["tools"].([]any)
	hasClose := false
	for _, raw := range tools {
		tool, ok := raw.(map[string]any)
		if ok && tool["name"] == "pipeline_close_delivery" {
			hasClose = true
			break
		}
	}
	if !hasClose {
		t.Errorf("ferramenta 'pipeline_close_delivery' não encontrada em tools/list")
	}

	// 2. Chama pipeline_close_delivery
	reqCall := `{"jsonrpc":"2.0","id":81,"method":"tools/call","params":{"name":"pipeline_close_delivery","arguments":{}}}`
	resCall := runRPC(t, s, reqCall)
	resultCall, ok := resCall["result"].(map[string]any)
	if !ok || resultCall["isError"] == true {
		t.Fatalf("pipeline_close_delivery retornou erro: %v", resCall)
	}

	// 3. Verifica se active-delivery.json foi removido ou limpo
	if data, err := os.ReadFile(stateFile); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		var parsed map[string]any
		if err := json.Unmarshal(data, &parsed); err == nil && parsed["delivery_id"] != nil && parsed["delivery_id"] != "" {
			t.Errorf("active-delivery.json ainda contém delivery ativa após close: %s", string(data))
		}
	}
}

func TestMCPResourceDeliveryActive(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewServer(tmpDir)

	// 1. Leitura sem delivery ativa
	reqEmpty := `{"jsonrpc":"2.0","id":90,"method":"resources/read","params":{"uri":"pipeline://delivery/active"}}`
	resEmpty := runRPC(t, s, reqEmpty)
	resultEmpty, ok := resEmpty["result"].(map[string]any)
	if !ok {
		t.Fatalf("result ausente em resources/read pipeline://delivery/active: %v", resEmpty)
	}
	contentsEmpty, _ := resultEmpty["contents"].([]any)
	if len(contentsEmpty) == 0 {
		t.Fatalf("contents vazio para pipeline://delivery/active")
	}
	firstEmpty, _ := contentsEmpty[0].(map[string]any)
	textEmpty, _ := firstEmpty["text"].(string)
	if !strings.Contains(strings.ToLower(textEmpty), "nenhuma") && !strings.Contains(strings.ToLower(textEmpty), "inativa") && !strings.Contains(strings.ToLower(textEmpty), "fechada") {
		t.Errorf("texto de delivery inativa inesperado: %s", textEmpty)
	}

	// 2. Leitura com delivery ativa
	stateDir := filepath.Join(tmpDir, ".pipeline-state")
	_ = os.MkdirAll(stateDir, 0755)
	activeContent := `{"delivery_id":"DEL-099","task_code":"TASK-099","opened_at":"2026-10-26T12:00:00Z","expires_at":"2026-10-26T14:00:00Z"}`
	_ = os.WriteFile(filepath.Join(stateDir, "active-delivery.json"), []byte(activeContent), 0644)

	reqActive := `{"jsonrpc":"2.0","id":91,"method":"resources/read","params":{"uri":"pipeline://delivery/active"}}`
	resActive := runRPC(t, s, reqActive)
	resultActive, ok := resActive["result"].(map[string]any)
	if !ok {
		t.Fatalf("result ausente em resources/read com delivery ativa: %v", resActive)
	}
	contentsActive, _ := resultActive["contents"].([]any)
	if len(contentsActive) == 0 {
		t.Fatalf("contents vazio")
	}
	firstActive, _ := contentsActive[0].(map[string]any)
	textActive, _ := firstActive["text"].(string)
	if !strings.Contains(textActive, "DEL-099") {
		t.Errorf("texto não contém DEL-099: %s", textActive)
	}
}

func TestScaffoldDeliveryWithoutIndex(t *testing.T) {
	tmpDir := t.TempDir()

	cfgBody := `{
		"version": 1,
		"project_type": "desktop",
		"protected_paths": ["internal/"],
		"require_index": false
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, ".pipeline-ai.json"), []byte(cfgBody), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// 1. Invocação direta da função ScaffoldDelivery em projeto com require_index: false e sem INDEX.md
	delivID, reportPath, err := ScaffoldDelivery(tmpDir, "Delivery Flexível", "feat", "TASK-001")
	if err != nil {
		t.Fatalf("ScaffoldDelivery falhou em projeto com require_index: false: %v", err)
	}
	if delivID == "" {
		t.Errorf("deliveryID retornado está vazio")
	}
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("report.md não foi criado em %s: %v", reportPath, err)
	}

	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("leitura de report.md falhou: %v", err)
	}
	if !strings.Contains(string(content), "Delivery Flexível") {
		t.Errorf("report.md não contém o título esperado: %s", string(content))
	}

	// 2. Invocação via servidor MCP (tool pipeline_scaffold_delivery)
	s := NewServer(tmpDir)
	req := `{"jsonrpc":"2.0","id":100,"method":"tools/call","params":{"name":"pipeline_scaffold_delivery","arguments":{"title":"Segunda Delivery Flexível","type":"feat","task_code":"TASK-002"}}}`
	res := runRPC(t, s, req)
	result, ok := res["result"].(map[string]any)
	if !ok {
		t.Fatalf("result ausente em pipeline_scaffold_delivery: %v", res)
	}
	if result["isError"] == true {
		t.Fatalf("pipeline_scaffold_delivery retornou isError=true: %v", result)
	}
}

func TestMCPToolsRunSensors(t *testing.T) {
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module testmcp\n\ngo 1.24\n"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "calc.go"), []byte("package calc\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "calc_test.go"), []byte("package calc\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(1, 2) != 3 {\n\t\tt.Fatal(\"err\")\n\t}\n}\n"), 0644)

	s := NewServer(tmpDir)

	// 1. Verifica se a ferramenta pipeline_run_sensors está listada em tools/list
	reqList := `{"jsonrpc":"2.0","id":110,"method":"tools/list","params":{}}`
	resList := runRPC(t, s, reqList)
	resultList, ok := resList["result"].(map[string]any)
	if !ok {
		t.Fatalf("result ausente em tools/list: %v", resList)
	}

	tools, ok := resultList["tools"].([]any)
	if !ok {
		t.Fatalf("campo tools não é lista: %v", resultList)
	}

	hasSensors := false
	for _, raw := range tools {
		tool, ok := raw.(map[string]any)
		if ok && tool["name"] == "pipeline_run_sensors" {
			hasSensors = true
			break
		}
	}
	if !hasSensors {
		t.Errorf("ferramenta 'pipeline_run_sensors' não encontrada em tools/list")
	}

	// 2. Invoca pipeline_run_sensors via tools/call
	reqCall := `{"jsonrpc":"2.0","id":111,"method":"tools/call","params":{"name":"pipeline_run_sensors","arguments":{"dir":"."}}}`
	resCall := runRPC(t, s, reqCall)
	resultCall, ok := resCall["result"].(map[string]any)
	if !ok {
		t.Fatalf("result ausente em tools/call pipeline_run_sensors: %v", resCall)
	}

	content, ok := resultCall["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("content vazio no tools/call pipeline_run_sensors: %v", resultCall)
	}

	first, ok := content[0].(map[string]any)
	if !ok || first["type"] != "text" {
		t.Fatalf("formato de content inválido: %v", content)
	}

	text, ok := first["text"].(string)
	if !ok {
		t.Fatalf("campo text não é string")
	}

	var sensorOutput map[string]any
	if err := json.Unmarshal([]byte(text), &sensorOutput); err != nil {
		t.Errorf("saída de pipeline_run_sensors não é JSON válido: %v (raw: %s)", err, text)
	}

	if _, ok := sensorOutput["overall"]; !ok {
		t.Errorf("JSON de retorno dos sensores não contém campo 'overall': %s", text)
	}
	if _, ok := sensorOutput["ok"]; !ok {
		t.Errorf("JSON de retorno dos sensores não contém campo 'ok': %s", text)
	}
}

func TestMCPToolsManageModules(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := `{
		"version": 1,
		"project_type": "desktop",
		"protected_paths": ["internal/"],
		"editor_agents": ["dev-implementation"],
		"enabled_modules": ["release-ops"]
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, ".pipeline-ai.json"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(tmpDir)

	// 1. Presença de pipeline_manage_modules em tools/list
	t.Run("presenca em tools/list", func(t *testing.T) {
		reqList := `{"jsonrpc":"2.0","id":120,"method":"tools/list","params":{}}`
		resList := runRPC(t, s, reqList)
		resultList, ok := resList["result"].(map[string]any)
		if !ok {
			t.Fatalf("result ausente em tools/list: %v", resList)
		}

		tools, ok := resultList["tools"].([]any)
		if !ok {
			t.Fatalf("campo tools não é lista: %v", resultList)
		}

		var toolDef map[string]any
		for _, raw := range tools {
			tool, ok := raw.(map[string]any)
			if ok && tool["name"] == "pipeline_manage_modules" {
				toolDef = tool
				break
			}
		}
		if toolDef == nil {
			t.Fatalf("ferramenta 'pipeline_manage_modules' não encontrada em tools/list")
		}

		schema, ok := toolDef["inputSchema"].(map[string]any)
		if !ok {
			t.Fatalf("inputSchema inválido para pipeline_manage_modules")
		}
		props, ok := schema["properties"].(map[string]any)
		if !ok || props["action"] == nil {
			t.Errorf("inputSchema deve conter propriedade 'action'")
		}
	})

	// 2. tools/call com action: "list"
	t.Run("action list", func(t *testing.T) {
		reqCall := fmt.Sprintf(`{"jsonrpc":"2.0","id":121,"method":"tools/call","params":{"name":"pipeline_manage_modules","arguments":{"action":"list","dir":%q}}}`, tmpDir)
		resCall := runRPC(t, s, reqCall)
		resultCall, ok := resCall["result"].(map[string]any)
		if !ok {
			t.Fatalf("result ausente em tools/call list: %v", resCall)
		}
		if resultCall["isError"] == true {
			t.Fatalf("action list retornou isError=true: %v", resultCall)
		}

		content, ok := resultCall["content"].([]any)
		if !ok || len(content) == 0 {
			t.Fatalf("content vazio")
		}
		first := content[0].(map[string]any)
		text := first["text"].(string)

		if !strings.Contains(text, "release-ops") || !strings.Contains(text, "data-model") {
			t.Errorf("listagem de módulos não contém módulos conhecidos: %s", text)
		}
	})

	// 3. tools/call com action: "enable"
	t.Run("action enable", func(t *testing.T) {
		reqCall := fmt.Sprintf(`{"jsonrpc":"2.0","id":122,"method":"tools/call","params":{"name":"pipeline_manage_modules","arguments":{"action":"enable","module":"data-model","dir":%q}}}`, tmpDir)
		resCall := runRPC(t, s, reqCall)
		resultCall, ok := resCall["result"].(map[string]any)
		if !ok {
			t.Fatalf("result ausente em tools/call enable: %v", resCall)
		}
		if resultCall["isError"] == true {
			t.Fatalf("action enable retornou isError=true: %v", resultCall)
		}

		data, err := os.ReadFile(filepath.Join(tmpDir, ".pipeline-ai.json"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "data-model") {
			t.Errorf("data-model não persistido após enable: %s", string(data))
		}
	})

	// 4. tools/call com action: "disable"
	t.Run("action disable", func(t *testing.T) {
		reqCall := fmt.Sprintf(`{"jsonrpc":"2.0","id":123,"method":"tools/call","params":{"name":"pipeline_manage_modules","arguments":{"action":"disable","module":"release-ops","dir":%q}}}`, tmpDir)
		resCall := runRPC(t, s, reqCall)
		resultCall, ok := resCall["result"].(map[string]any)
		if !ok {
			t.Fatalf("result ausente em tools/call disable: %v", resCall)
		}
		if resultCall["isError"] == true {
			t.Fatalf("action disable retornou isError=true: %v", resultCall)
		}

		data, err := os.ReadFile(filepath.Join(tmpDir, ".pipeline-ai.json"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), `"release-ops"`) {
			t.Errorf("release-ops ainda presente após disable: %s", string(data))
		}
	})

	// 5. Chamadas de erro
	t.Run("erros modulo invalido e acao invalida", func(t *testing.T) {
		reqCallInvMod := fmt.Sprintf(`{"jsonrpc":"2.0","id":124,"method":"tools/call","params":{"name":"pipeline_manage_modules","arguments":{"action":"enable","module":"invalido","dir":%q}}}`, tmpDir)
		resCallInvMod := runRPC(t, s, reqCallInvMod)
		res1, _ := resCallInvMod["result"].(map[string]any)
		if res1 == nil || res1["isError"] != true {
			t.Errorf("esperava isError=true ao ativar módulo inválido")
		}

		reqCallInvAction := fmt.Sprintf(`{"jsonrpc":"2.0","id":125,"method":"tools/call","params":{"name":"pipeline_manage_modules","arguments":{"action":"invalida","dir":%q}}}`, tmpDir)
		resCallInvAction := runRPC(t, s, reqCallInvAction)
		res2, _ := resCallInvAction["result"].(map[string]any)
		if res2 == nil || res2["isError"] != true {
			t.Errorf("esperava isError=true com action desconhecida")
		}
	})
}

func TestMCPToolsRunPipeline(t *testing.T) {
	tmpDir := t.TempDir()

	writeMCPFile := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir falhou: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write falhou: %v", err)
		}
	}

	cfgContent := `{
  "version": 1,
  "project_type": "api",
  "protected_paths": ["internal/"],
  "editor_agents": ["dev-implementation", "test-writer", "docs-update"]
}`
	writeMCPFile(filepath.Join(tmpDir, ".pipeline-ai.json"), cfgContent)
	writeMCPFile(filepath.Join(tmpDir, "go.mod"), "module mcp_run_test\n\ngo 1.24\n")
	writeMCPFile(filepath.Join(tmpDir, "app.go"), "package app\n\nfunc OK() bool {\n\treturn true\n}\n")
	writeMCPFile(filepath.Join(tmpDir, "app_test.go"), "package app\n\nimport \"testing\"\n\nfunc TestOK(t *testing.T) {\n\tif !OK() {\n\t\tt.Fatal(\"err\")\n\t}\n}\n")
	writeMCPFile(filepath.Join(tmpDir, ".githooks", "pre-push"), "#!/bin/sh\nexit 0\n")
	writeMCPFile(filepath.Join(tmpDir, ".git", "HEAD"), "ref: refs/heads/main\n")
	writeMCPFile(filepath.Join(tmpDir, ".agents", "hooks.json"), "{}")
	writeMCPFile(filepath.Join(tmpDir, ".agents", "skills", "test-writer", "SKILL.md"), "# Test Writer")

	s := NewServer(tmpDir)

	// 1. tools/list contém pipeline_run_pipeline
	t.Run("tools/list contem pipeline_run_pipeline", func(t *testing.T) {
		req := `{"jsonrpc":"2.0","id":100,"method":"tools/list","params":{}}`
		res := runRPC(t, s, req)

		result, ok := res["result"].(map[string]any)
		if !ok {
			t.Fatalf("result ausente: %v", res)
		}
		tools, ok := result["tools"].([]any)
		if !ok {
			t.Fatalf("tools ausente: %v", result)
		}

		found := false
		for _, raw := range tools {
			tool, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if tool["name"] == "pipeline_run_pipeline" {
				found = true
				desc, _ := tool["description"].(string)
				if desc == "" {
					t.Errorf("ferramenta pipeline_run_pipeline sem descrição")
				}
				break
			}
		}
		if !found {
			t.Fatalf("ferramenta pipeline_run_pipeline não encontrada em tools/list")
		}
	})

	// 2. tools/call executa pipeline_run_pipeline retornando JSON estruturado
	t.Run("tools/call pipeline_run_pipeline", func(t *testing.T) {
		callReq := fmt.Sprintf(`{"jsonrpc":"2.0","id":101,"method":"tools/call","params":{"name":"pipeline_run_pipeline","arguments":{"dir":%q,"no_git":true}}}`, tmpDir)
		res := runRPC(t, s, callReq)

		result, ok := res["result"].(map[string]any)
		if !ok {
			t.Fatalf("result ausente no tools/call: %v", res)
		}
		content, ok := result["content"].([]any)
		if !ok || len(content) == 0 {
			t.Fatalf("content ausente ou vazio: %v", result)
		}
		first, ok := content[0].(map[string]any)
		if !ok {
			t.Fatalf("primeiro item de content inválido: %v", content)
		}
		text, _ := first["text"].(string)
		if text == "" {
			t.Fatalf("text vazio em content[0]: %v", first)
		}

		var parsed struct {
			Overall string `json:"overall"`
			OK      bool   `json:"ok"`
			Steps   []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			} `json:"steps"`
		}
		if err := json.Unmarshal([]byte(text), &parsed); err != nil {
			t.Fatalf("text retornado não é JSON válido: %v, raw:\n%s", err, text)
		}
		if parsed.Overall != "PASS" || !parsed.OK {
			t.Errorf("overall = %q, ok = %v, esperado PASS/true", parsed.Overall, parsed.OK)
		}
	})
}

func TestMCPToolsValidateHandoff(t *testing.T) {
	s := NewServer(".")

	// 1. tools/list contém pipeline_validate_handoff
	t.Run("tools/list contem pipeline_validate_handoff", func(t *testing.T) {
		req := `{"jsonrpc":"2.0","id":110,"method":"tools/list","params":{}}`
		res := runRPC(t, s, req)

		result, ok := res["result"].(map[string]any)
		if !ok {
			t.Fatalf("result ausente: %v", res)
		}
		tools, ok := result["tools"].([]any)
		if !ok {
			t.Fatalf("tools ausente: %v", result)
		}

		found := false
		for _, raw := range tools {
			tool, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if tool["name"] == "pipeline_validate_handoff" {
				found = true
				desc, _ := tool["description"].(string)
				if desc == "" {
					t.Errorf("ferramenta pipeline_validate_handoff sem descricao")
				}
				break
			}
		}
		if !found {
			t.Fatalf("ferramenta pipeline_validate_handoff nao encontrada em tools/list")
		}
	})

	validHandoff := "DE: architecture-guardrails\n" +
		"PARA: test-writer\n" +
		"DELIVERY_ID: DEL-018\n" +
		"STATUS: ok\n" +
		"ESCOPO: internal/handoff\n" +
		"ARQUIVOS: internal/handoff/handoff_test.go\n" +
		"CONFIRMADO: go test ./... PASS\n" +
		"ASSUNCOES: nenhuma\n" +
		"RISCOS: nenhum\n" +
		"SINAIS:\n" +
		"  SECURITY: nao\n" +
		"  SUPPLY_CHAIN: nao\n" +
		"  MIGRATION: nao\n" +
		"  PERFORMANCE: nao\n" +
		"PROXIMO: dev-implementation\n" +
		"MOTIVO: implementacao TDD"

	invalidHandoff := "DE: architecture-guardrails\n" +
		"PARA: test-writer\n" +
		"DELIVERY_ID: DEL-018\n" +
		"STATUS: status-invalido\n" +
		"CONFIRMADO: tudo certo\n" +
		"PROXIMO: dev-implementation\n" +
		"MOTIVO: teste"

	// 2. tools/call com handoff valido
	t.Run("tools/call com handoff valido", func(t *testing.T) {
		callPayload := map[string]any{
			"jsonrpc": "2.0",
			"id":      111,
			"method":  "tools/call",
			"params": map[string]any{
				"name": "pipeline_validate_handoff",
				"arguments": map[string]any{
					"content": validHandoff,
				},
			},
		}
		reqBytes, _ := json.Marshal(callPayload)
		res := runRPC(t, s, string(reqBytes))

		result, ok := res["result"].(map[string]any)
		if !ok {
			t.Fatalf("result ausente no tools/call: %v", res)
		}
		if result["isError"] == true {
			t.Errorf("esperava isError=false para handoff valido, obteve true: %v", result)
		}
		content, ok := result["content"].([]any)
		if !ok || len(content) == 0 {
			t.Fatalf("content ausente ou vazio: %v", result)
		}
		first, _ := content[0].(map[string]any)
		text, _ := first["text"].(string)

		var parsed struct {
			Valid  bool     `json:"valid"`
			Errors []string `json:"errors"`
		}
		if err := json.Unmarshal([]byte(text), &parsed); err != nil {
			t.Fatalf("resposta nao e JSON de validacao valido: %v; raw:\n%s", err, text)
		}
		if !parsed.Valid {
			t.Errorf("parsed.Valid = false para handoff valido, erros: %v", parsed.Errors)
		}
	})

	// 3. tools/call com handoff invalido
	t.Run("tools/call com handoff invalido", func(t *testing.T) {
		callPayload := map[string]any{
			"jsonrpc": "2.0",
			"id":      112,
			"method":  "tools/call",
			"params": map[string]any{
				"name": "pipeline_validate_handoff",
				"arguments": map[string]any{
					"content": invalidHandoff,
				},
			},
		}
		reqBytes, _ := json.Marshal(callPayload)
		res := runRPC(t, s, string(reqBytes))

		result, ok := res["result"].(map[string]any)
		if !ok {
			t.Fatalf("result ausente no tools/call: %v", res)
		}
		if result["isError"] != true {
			t.Errorf("esperava isError=true para handoff invalido: %v", result)
		}
	})
}

func TestMCPToolsGetMetrics(t *testing.T) {
	dir := t.TempDir()
	s := NewServer(dir)

	// 1. tools/list contém pipeline_get_metrics
	listPayload := `{"jsonrpc":"2.0","id":120,"method":"tools/list"}`
	res := runRPC(t, s, listPayload)
	result := res["result"].(map[string]any)
	tools := result["tools"].([]any)
	var found bool
	for _, raw := range tools {
		tool := raw.(map[string]any)
		if tool["name"] == "pipeline_get_metrics" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("pipeline_get_metrics não encontrado em tools/list")
	}

	// 2. tools/call pipeline_get_metrics em diretório vazio
	callPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      121,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "pipeline_get_metrics",
			"arguments": map[string]any{
				"dir": dir,
			},
		},
	}
	reqBytes, _ := json.Marshal(callPayload)
	callRes := runRPC(t, s, string(reqBytes))
	callResult := callRes["result"].(map[string]any)
	if callResult["isError"] == true {
		t.Fatalf("esperava isError=false para pipeline_get_metrics: %v", callResult)
	}
}
