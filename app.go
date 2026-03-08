package main

import (
	"candidate_alocator/back/logic"
	types "candidate_alocator/back/type"
	"context"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type UsuariosResponse = logic.UsuariosResponse

// App struct
type App struct {
	ctx       context.Context
	excelData []byte
	nOpcoes   int
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called at application startup
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	SetUp()
	writeWailsSmokeSentinel()
}

// domReady is called after front-end resources have been loaded
func (a App) domReady(ctx context.Context) {
}

// beforeClose is called when the application is about to quit,
// either by clicking the window close button or calling runtime.Quit.
// Returning true will cause the application to continue, false will continue shutdown as normal.
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	return false
}

// shutdown is called at application termination
func (a *App) shutdown(ctx context.Context) {
}

// O fluxo equivalente via terminal foi movido para cli.go.
// Use: go run . cli -file arquivo.xlsx

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) SuggestMapping(data []byte, quantidadeOpcoes int) ([]types.MappingItem, error) {
	a.excelData = data
	a.nOpcoes = quantidadeOpcoes
	return logic.SuggestMapping(data, quantidadeOpcoes)
}

func (a *App) SuggestMappingAvaliador() ([]types.MappingItem, error) {
	return logic.SuggestMappingAvaliador(a.excelData)
}

func (a *App) SuggestMappingRestricao() ([]types.MappingItem, error) {
	return logic.SuggestMappingRestricao(a.excelData)
}

func ProcessMapping(items []string) ([]types.MappingItem, error) {
	return logic.ProcessMapping(items)
}

func (a *App) BuildUsuariosWithMapping(mappingItems []types.MappingItem) (UsuariosResponse, error) {
	return logic.BuildUsuariosWithMapping(a.excelData, a.nOpcoes, mappingItems)
}

func (a *App) BuildAvaliadoresWithMapping(mappingItems []types.MappingItem) ([]types.AvaliadorInfo, error) {
	return logic.BuildAvaliadoresWithMapping(a.excelData, mappingItems)
}

func (a *App) BuildRestricoesWithMapping(mappingItems []types.MappingItem) ([]types.Restricao, error) {
	return logic.BuildRestricoesWithMapping(a.excelData, mappingItems)
}

func (a *App) SaveRestricoesFromMaps(restricaoMaps []map[string]interface{}) error {
	return logic.SaveRestricoesFromMaps(restricaoMaps)
}

func (a *App) SaveUsuariosFromMaps(candidatoMaps []map[string]interface{}) error {
	return logic.SaveUsuariosFromMaps(candidatoMaps)
}

func writeWailsSmokeSentinel() {
	path := os.Getenv("CANDIDATE_ALLOCATOR_WAILS_SMOKE_FILE")
	if path == "" {
		return
	}

	if err := os.WriteFile(path, []byte("ok"), 0o644); err != nil {
		fmt.Println("Erro ao escrever sentinel do smoke test do Wails:", err)
	}
}
