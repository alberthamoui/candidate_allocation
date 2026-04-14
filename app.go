package main

import (
	"context"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// App é a struct principal da aplicação Wails.
type App struct {
	ctx         context.Context
	excelData   []byte
	nOpcoes     int
	emailDomain string
	lastResult  *AlocacaoResponse
}

// NewApp cria uma nova instância da aplicação.
func NewApp() *App {
	return &App{}
}

// startup é chamado na inicialização da aplicação.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	setupIfNeeded()
}

// domReady é chamado após o frontend carregar.
func (a App) domReady(ctx context.Context) {}

// beforeClose é chamado antes do app fechar.
// Retornar true cancela o fechamento.
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	return false
}

// shutdown é chamado ao encerrar a aplicação.
func (a *App) shutdown(ctx context.Context) {}

// Greet retorna uma saudação (método de exemplo do template Wails).
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// SetupDB recria o banco do zero (apaga todos os dados).
func (a *App) SetupDB() error {
	SetUp()
	return nil
}

// SaveUsuarios persiste candidatos no banco de dados.
func (a *App) SaveUsuarios(data []Usuario) error {
	conn, err := openDB()
	if err != nil {
		return fmt.Errorf("erro ao abrir banco: %w", err)
	}
	defer conn.Close()
	fillDb(conn, data)
	return nil
}

// SaveAvaliadores persiste avaliadores no banco de dados.
func (a *App) SaveAvaliadores(data []AvaliadorInfo) error {
	conn, err := openDB()
	if err != nil {
		return fmt.Errorf("erro ao abrir banco: %w", err)
	}
	defer conn.Close()
	fillDb(conn, data)
	return nil
}

// SaveRestricoes persiste restrições no banco de dados.
func (a *App) SaveRestricoes(data []Restricao) error {
	conn, err := openDB()
	if err != nil {
		return fmt.Errorf("erro ao abrir banco: %w", err)
	}
	defer conn.Close()
	fillDb(conn, data)
	return nil
}

// ResetApp limpa o banco e o estado em memória para o usuário recomeçar.
func (a *App) ResetApp() error {
	SetUp()
	a.excelData = nil
	a.lastResult = nil
	a.nOpcoes = 0
	return nil
}
