package main

import (
	"candidate_alocator/back/logic"
	types "candidate_alocator/back/type"
	"context"
	"flag"
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

func main() {
	SetUp()

	path := flag.String("file", "", "caminho para o arquivo .xlsx")
	flag.Parse()
	if *path == "" {
		fmt.Println("Uso: go run main.go -file seu_arquivo.xlsx")
		os.Exit(1)
	}

	data, err := os.ReadFile(*path)
	if err != nil {
		fmt.Println("Erro ao ler o arquivo:", err)
		os.Exit(1)
	}

	app := NewApp()
	mapping, err := app.SuggestMapping(data, 5)
	if err != nil {
		fmt.Println("Erro ao sugerir mapeamento:", err)
		os.Exit(1)
	}

	fmt.Println("mapping candidatos : ", mapping)
	fmt.Println()

	mappingAvaliador, err := app.SuggestMappingAvaliador()
	fmt.Println("mapping avaliadores : ", mappingAvaliador)
	fmt.Println()

	// mappingRestricao, err := app.SuggestMappingRestricao()

	// fmt.Println("mapping restricao : ", mappingRestricao)
	// fmt.Println("\n")

	// usuarios, err := app.BuildUsuariosWithMapping(mapping)
	// if err != nil {
	// 	fmt.Println("Erro ao ler o arquivo:", err)
	// 	os.Exit(1)
	// }
	// usuarios_filtrados := FilterUniqueUsers(usuarios)

	// avaliadores, err := app.BuildAvaliadoresWithMapping(mappingAvaliador)
	// if err != nil {
	// 	fmt.Println("Erro ao ler o arquivo:", err)
	// 	os.Exit(1)
	// }
	// restricao, err := app.BuildRestricoesWithMapping(mappingRestricao)
	// if err != nil {
	// 	fmt.Println("Erro ao ler o arquivo:", err)
	// 	os.Exit(1)
	// }
	// fmt.Println("\n")
	// fmt.Println("usuarios: ", usuarios)
	// fmt.Println("\n\n\n")
	// fmt.Println("avaliadores: ", avaliadores)
	// fmt.Println("\n")
	// fmt.Println("REstricao: ", restricao)
	// fmt.Println("\n")
	// logic.Save(usuarios_filtrados)
	// logic.Save(avaliadores)
	// logic.Save(restricao)

	// // Alocacao
	// conn, err := sql.Open("sqlite3", "./insper.db")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer conn.Close()
	// Alocar(conn)

	// out1, _ := json.MarshalIndent(mapping, "", " ")
	// out, _ := json.MarshalIndent(usuarios_filtrados, "", "  ")
	// out2, _ := json.MarshalIndent(duplicatedIndices, "", "  ")
	// fmt.Println("usuarios : ", usuarios_filtrados)
	// fmt.Println("\n")
	// fmt.Println(string(out2))

}

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
