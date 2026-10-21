package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
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
