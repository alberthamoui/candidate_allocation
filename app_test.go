package main

import (
	"bytes"
	"database/sql"
	"reflect"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/xuri/excelize/v2"
)

// setupSession cria uma sessão de teste com banco em memória.
func setupSession() *Session {
	db, _ := sql.Open("sqlite3", ":memory:")
	db.SetMaxOpenConns(1)
	setupConn(db)
	return &Session{db: db, emailDomain: "@al.insper.edu.br"}
}

// createMockExcelFile cria um arquivo Excel em memória para testes.
func createMockExcelFile(data [][]interface{}) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	// Adiciona uma planilha
	index, err := f.NewSheet("Sheet1")
	if err != nil {
		return nil, err
	}
	// Define a planilha ativa
	f.SetActiveSheet(index)

	// Preenche a planilha com dados
	for i, row := range data {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		f.SetSheetRow("Sheet1", cell, &row)
	}

	// Salva o arquivo em um buffer
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func TestSuggestMapping(t *testing.T) {
	a := setupSession()

	// Colunas fora de ordem: a sugestão deve seguir o nome, não a posição
	excelData := [][]interface{}{
		{"CPF", "Opção 1", "Nome"},
		{"123", "Entidade A", "João"},
	}
	buf, err := createMockExcelFile(excelData)
	if err != nil {
		t.Fatalf("Erro ao criar mock do Excel: %v", err)
	}

	mappings, err := a.SuggestMapping(buf.Bytes(), 1, "@al.insper.edu.br")
	if err != nil {
		t.Fatalf("SuggestMapping retornou um erro inesperado: %v", err)
	}

	// Um item por campo de Candidato (opcoes vira "opcao 1"; extras não é coluna)
	expected := reflect.TypeOf(Candidato{}).NumField() - 1
	if len(mappings) != expected {
		t.Errorf("Esperado %d mapeamentos, mas obteve %d: %+v", expected, len(mappings), mappings)
	}

	want := map[string]MappingItem{
		"nome":    {NomeColuna: "Nome", Indice: 2, Variavel: "nome"},
		"cpf":     {NomeColuna: "CPF", Indice: 0, Variavel: "cpf"},
		"opcao 1": {NomeColuna: "Opção 1", Indice: 1, Variavel: "opcao 1"},
	}
	for _, m := range mappings {
		if w, ok := want[m.Variavel]; ok && m != w {
			t.Errorf("campo %q: esperado %+v, obteve %+v", m.Variavel, w, m)
		}
	}
}

func TestBuildUsuariosWithMapping(t *testing.T) {
	a := setupSession()

	// Cria um arquivo Excel de mock
	excelData := [][]interface{}{
		{"Nome", "CPF", "Opção 1"},
		{"Maria", "456", "Entidade B"},
	}
	buf, err := createMockExcelFile(excelData)
	if err != nil {
		t.Fatalf("Erro ao criar mock do Excel: %v", err)
	}
	a.excelData = buf.Bytes()
	a.nOpcoes = 1

	// Define o mapeamento
	mapping := []MappingItem{
		{NomeColuna: "Nome", Indice: 0, Variavel: "nome"},
		{NomeColuna: "CPF", Indice: 1, Variavel: "cpf"},
		{NomeColuna: "Opção 1", Indice: 2, Variavel: "opcao 1"},
	}

	// Chama a função
	response, err := a.BuildUsuariosWithMapping(mapping)
	if err != nil {
		t.Fatalf("BuildUsuariosWithMapping retornou um erro: %v", err)
	}

	// Verifica se o usuário foi criado corretamente
	if len(response.Usuarios) != 1 {
		t.Fatalf("Esperado 1 usuário, mas obteve %d", len(response.Usuarios))
	}

	// Acessa o usuário (assumindo que a chave é o índice + 1)
	userResult, ok := response.Usuarios[1]
	if !ok {
		t.Fatalf("Usuário com chave 1 não encontrado no mapa")
	}
	user := userResult.Usuario

	if user.Nome != "Maria" || user.CPF != "456" || user.Opcoes[0] != "Entidade B" {
		t.Errorf("Dados do usuário incorretos: %+v", user)
	}
}
