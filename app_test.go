package main

import (
	"bytes"
	"context"

	"reflect"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/xuri/excelize/v2"
)

// setupApp inicializa o App para os testes.
func setupApp() *App {
	a := NewApp()
	a.startup(context.Background())
	return a
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
	a := setupApp()

	// Cria um arquivo Excel de mock
	excelData := [][]interface{}{
		{"Nome", "CPF", "Opção 1"},
		{"João", "123", "Entidade A"},
	}
	buf, err := createMockExcelFile(excelData)
	if err != nil {
		t.Fatalf("Erro ao criar mock do Excel: %v", err)
	}

	// Chama a função a ser testada

	mappings, err := a.SuggestMapping(buf.Bytes(), 1)
	if err != nil {
		t.Fatalf("SuggestMapping retornou um erro inesperado: %v", err)
	}

	// Verifica se o número de mapeamentos está correto
	expected := reflect.TypeOf(Candidato{}).NumField()
	if len(mappings) != expected {
		t.Errorf("Esperado %d mapeamentos, mas obteve %d", expected, len(mappings))
	}

	// Verifica um dos mapeamentos para garantir que a lógica está correta
	expectedMapping := MappingItem{NomeColuna: "Nome", Indice: 0, Variavel: "timestamp"}
	found := false
	for _, m := range mappings {
		if m.NomeColuna == expectedMapping.NomeColuna && m.Indice == expectedMapping.Indice {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Mapeamento esperado não encontrado: %+v", expectedMapping)
	}
}

func TestProcessMapping(t *testing.T) {
	// Caso de teste com JSONs válidos
	validItems := []string{
		`[["Nome", 0], "nome"]`,
		`[["CPF", 1], "cpf"]`,
	}
	expected := []MappingItem{
		{NomeColuna: "Nome", Indice: 0, Variavel: "nome"},
		{NomeColuna: "CPF", Indice: 1, Variavel: "cpf"},
	}

	result, err := ProcessMapping(validItems)
	if err != nil {
		t.Fatalf("ProcessMapping retornou um erro inesperado: %v", err)
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Resultado esperado %+v, mas obteve %+v", expected, result)
	}

	// Caso de teste com JSON inválido
	invalidItems := []string{`["Nome", 0], "nome"]`} // JSON malformado
	_, err = ProcessMapping(invalidItems)
	if err == nil {
		t.Errorf("Esperado um erro para JSON inválido, mas não ocorreu")
	}
}

func TestBuildUsuariosWithMapping(t *testing.T) {
	a := setupApp()

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
