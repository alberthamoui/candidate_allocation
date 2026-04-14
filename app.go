package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	wailsrt "github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/xuri/excelize/v2"
)

// App struct
type App struct {
	ctx        context.Context
	excelData  []byte
	nOpcoes    int
	lastResult *AlocacaoResponse
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called at application startup
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	setupIfNeeded()
}

// domReady is called after front-end resources have been loaded
func (a App) domReady(ctx context.Context) {
	// Add your action here
}

// beforeClose is called when the application is about to quit,
// either by clicking the window close button or calling runtime.Quit.
// Returning true will cause the application to continue, false will continue shutdown as normal.
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	return false
}

// shutdown is called at application termination
func (a *App) shutdown(ctx context.Context) {
	// Perform your teardown here
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

type Usuario struct {
	Timestamp    string   `json:"timestamp"`
	Nome         string   `json:"nome"`
	CPF          string   `json:"cpf"`
	Numero       string   `json:"numero"`
	Semestre     string   `json:"semestre"`
	Curso        string   `json:"curso"`
	EmailInsper  string   `json:"email_insper"`
	EmailPessoal string   `json:"email_pessoal"`
	Opcoes       []string `json:"opcoes"`
}

type AvaliadorInfo struct {
	Nome  string `json:"nome"`
	Email string `json:"email"`
	Sigla string `json:"sigla"`
}

type Restricao struct {
	Candidato  string `json:"candidato"`
	NaoPosso   string `json:"naoPosso"`
	PrefiroNao string `json:"prefiroNao"`
}

type MappingItem struct {
	NomeColuna string `json:"nomeColuna"`
	Indice     int    `json:"indice"`
	Variavel   string `json:"variavel"`
}

func (a *App) SuggestMapping(data []byte, quantidade_opcoes int) ([]MappingItem, error) {
	a.excelData = data
	a.nOpcoes = quantidade_opcoes
	readerData := bytes.NewReader(data)
	file, err := excelize.OpenReader(readerData)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sheet := file.GetSheetName(0)
	rows, err := file.GetRows(sheet)

	if err != nil {
		return nil, err
	}
	if len(rows) < 1 {
		return nil, fmt.Errorf("arquivo sem dados")
	}
	header := rows[0]
	// fmt.Println("header : ", header)

	// Lista de possíveis variáveis da struct Usuario (em minúsculo)
	variaveisUsuario := getUsuarioFields(quantidade_opcoes)
	fmt.Println("variaveis usuario: ", variaveisUsuario)

	// Alocação aleatória das variáveis para cada coluna
	mappingList := make([]string, len(variaveisUsuario))
	for i, usuarioVar := range variaveisUsuario {
		if i < len(header) {
			mappingList[i] = fmt.Sprintf("[[%q, %d], %q]", header[i], i, usuarioVar)
		} else {
			mappingList[i] = fmt.Sprintf("[[null, %d], %q]", i, usuarioVar)
		}
	}
	fmt.Println(mappingList)
	mapping_json, err := ProcessMapping(mappingList)
	if err != nil {
		return nil, err
	}
	return mapping_json, nil
}
func (a *App) SuggestMappingAvaliador() ([]MappingItem, error) {
	readerData := bytes.NewReader(a.excelData)
	file, err := excelize.OpenReader(readerData)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sheet := file.GetSheetName(1) // Avaliador
	rows, err := file.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	if len(rows) < 1 {
		return nil, fmt.Errorf("arquivo sem dados")
	}
	header := rows[0]

	// gerar dinamicamente as variáveis de avaliador
	variaveisAvaliador := getAvaliadorFields()

	fmt.Println("variaveis avaliador : ", variaveisAvaliador)
	mappingList := make([]string, len(variaveisAvaliador))
	for i, v := range variaveisAvaliador {
		if i < len(header) {
			mappingList[i] = fmt.Sprintf("[[%q, %d], %q]", header[i], i, v)
		} else {
			mappingList[i] = fmt.Sprintf("[[null, %d], %q]", i, v)
		}
	}
	return ProcessMapping(mappingList)
}

func (a *App) SuggestMappingRestricao() ([]MappingItem, error) {
	readerData := bytes.NewReader(a.excelData)
	file, err := excelize.OpenReader(readerData)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sheet := file.GetSheetName(2) // Restrição
	rows, err := file.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	if len(rows) < 1 {
		return nil, fmt.Errorf("arquivo sem dados")
	}
	header := rows[0]

	// gerar dinamicamente as variáveis de restrição
	variaveisRestricao := getRestricaoFields()
	mappingList := make([]string, len(variaveisRestricao))
	for i, v := range variaveisRestricao {
		if i < len(header) {
			mappingList[i] = fmt.Sprintf("[[%q, %d], %q]", header[i], i, v)
		} else {
			mappingList[i] = fmt.Sprintf("[[null, %d], %q]", i, v)
		}
	}
	return ProcessMapping(mappingList)
}

// ProcessMapping converte cada string JSON "[[nomeColuna,indice],variavel]"
// em um Mapping. Retorna erro se algum item não for JSON válido.
func ProcessMapping(items []string) ([]MappingItem, error) {
	var result []MappingItem

	for _, item := range items {
		// decodifica o JSON em um slice genérico
		var arr []interface{}
		if err := json.Unmarshal([]byte(item), &arr); err != nil {
			return nil, fmt.Errorf("invalid JSON '%s': %w", item, err)
		}
		if len(arr) != 2 {
			continue
		}

		// arr[0] => [nomeColuna, indice]
		info, ok := arr[0].([]interface{})
		if !ok || len(info) != 2 {
			continue
		}
		var nomeColuna string
		if info[0] != nil {
			s, ok := info[0].(string)
			if !ok {
				continue
			}
			nomeColuna = s
		}

		indiceF, ok2 := info[1].(float64)
		variavel, ok3 := arr[1].(string)
		if !ok2 || !ok3 {
			continue
		}

		result = append(result, MappingItem{
			NomeColuna: nomeColuna,
			Indice:     int(indiceF),
			Variavel:   variavel,
		})
	}

	return result, nil
}

type UsuariosResponse struct {
	Usuarios   map[int]ValidationResult `json:"usuarios"`
	Duplicates [][]int                  `json:"duplicates"`
}

func (a *App) BuildUsuariosWithMapping(mappingItems []MappingItem) (UsuariosResponse, error) {
	// Abre o arquivo Excel a partir dos dados em []byte
	data := a.excelData

	nOpcoes := a.nOpcoes
	readerData := bytes.NewReader(data)
	file, err := excelize.OpenReader(readerData)
	if err != nil {
		return UsuariosResponse{}, err
	}
	defer file.Close()

	sheet := file.GetSheetName(0)
	rows, err := file.GetRows(sheet)
	if err != nil {
		return UsuariosResponse{}, fmt.Errorf("erro ao ler excel : %w", err)
	}
	if len(rows) < 2 {
		return UsuariosResponse{}, fmt.Errorf("arquivo sem dados além do header")
	}

	var users []Usuario

	for _, row := range rows[1:] {
		u := Usuario{
			Opcoes: make([]string, nOpcoes),
		}
		// Para cada mapping, pega o conteúdo da coluna correspondente e atribui
		for _, mItem := range mappingItems {
			if mItem.Indice >= len(row) {
				continue
			}
			cell := row[mItem.Indice]
			switch mItem.Variavel {
			case "timestamp":
				u.Timestamp = cell
			case "nome":
				u.Nome = cell
			case "cpf":
				u.CPF = cell
			case "numero":
				u.Numero = cell
			case "semestre":
				u.Semestre = cell
			case "curso":
				u.Curso = cell
			case "email_insper":
				u.EmailInsper = cell
			case "email_pessoal":
				u.EmailPessoal = cell
			default:
				// Se for uma opção, o mItem.UserField deve estar no formato "opcao X"
				if strings.HasPrefix(mItem.Variavel, "opcao") {
					parts := strings.Split(mItem.Variavel, " ")
					if len(parts) == 2 {
						optionNum, err := strconv.Atoi(parts[1])
						if err == nil && optionNum > 0 && optionNum <= nOpcoes {
							u.Opcoes[optionNum-1] = cell
						}
					}
				}
			}
		}
		users = append(users, u)
	}
	users_limpo, duplicatedIndices := processData(users)

	return UsuariosResponse{Usuarios: users_limpo, Duplicates: duplicatedIndices}, nil
}

func (a *App) BuildAvaliadoresWithMapping(mappingItems []MappingItem) ([]AvaliadorInfo, error) {
	if a.excelData == nil {
		return nil, fmt.Errorf("dados do Excel ainda não carregados")
	}

	// abre planilha a partir do []byte salvo em a.excelData
	reader := bytes.NewReader(a.excelData)
	file, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("erro abrindo excel: %w", err)
	}
	defer file.Close()

	// 2ª aba (índice 1) onde estão os avaliadores
	sheet := file.GetSheetName(1)
	if sheet == "" {
		return nil, fmt.Errorf("arquivo não possui uma segunda aba com avaliadores")
	}

	rows, err := file.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("erro lendo aba de avaliadores: %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("aba de avaliadores não contém dados além do cabeçalho")
	}

	var avaliadores []AvaliadorInfo

	// percorre linhas (ignorando cabeçalho)
	for _, row := range rows[1:] {
		av := AvaliadorInfo{}
		for _, m := range mappingItems {
			if m.Indice >= len(row) {
				continue // coluna vazia nesta linha
			}
			val := strings.TrimSpace(row[m.Indice])

			switch strings.ToLower(m.Variavel) {
			case "nome":
				av.Nome = val
			case "email":
				av.Email = val
			case "sigla":
				av.Sigla = val
			}
		}

		// ignora linhas totalmente vazias
		if av.Nome == "" && av.Email == "" && av.Sigla == "" {
			continue
		}
		avaliadores = append(avaliadores, av)
	}

	return avaliadores, nil
}

func (a *App) BuildRestricoesWithMapping(mappingItems []MappingItem) ([]Restricao, error) {
	readerData := bytes.NewReader(a.excelData)
	file, err := excelize.OpenReader(readerData)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sheet := file.GetSheetName(2) // aba de restrição
	rows, err := file.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler excel: %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("arquivo sem dados além do header")
	}

	var restricoes []Restricao
	for _, row := range rows[1:] {
		r := Restricao{}
		for _, m := range mappingItems {
			if m.Indice >= len(row) {
				continue
			}
			cell := row[m.Indice]
			switch m.Variavel {
			case "candidato":
				r.Candidato = cell
			case "naoPosso":
				r.NaoPosso = cell
			case "prefiroNao":
				r.PrefiroNao = cell
			}
		}
		restricoes = append(restricoes, r)
	}
	return restricoes, nil
}

func openDB() (*sql.DB, error) {
	return sql.Open("sqlite3", "./insper.db")
}

// SetupDB resets and recreates the database (wipes all data).
func (a *App) SetupDB() error {
	SetUp()
	return nil
}

// SaveUsuarios persists candidates to the database.
func (a *App) SaveUsuarios(data []Usuario) error {
	conn, err := openDB()
	if err != nil {
		return fmt.Errorf("erro ao abrir banco: %w", err)
	}
	defer conn.Close()
	fillDb(conn, data)
	return nil
}

// SaveAvaliadores persists evaluators to the database.
func (a *App) SaveAvaliadores(data []AvaliadorInfo) error {
	conn, err := openDB()
	if err != nil {
		return fmt.Errorf("erro ao abrir banco: %w", err)
	}
	defer conn.Close()
	fillDb(conn, data)
	return nil
}

// SaveRestricoes persists restrictions to the database.
func (a *App) SaveRestricoes(data []Restricao) error {
	conn, err := openDB()
	if err != nil {
		return fmt.Errorf("erro ao abrir banco: %w", err)
	}
	defer conn.Close()
	fillDb(conn, data)
	return nil
}

// MesaResult is the serialisable form of a Mesa with human-readable names.
type MesaResult struct {
	ID          int      `json:"id"`
	Descricao   string   `json:"descricao"`
	Candidatos  []string `json:"candidatos"`
	Avaliadores []string `json:"avaliadores"`
}

// PessoaInfo carries the fields shown for non-allocated candidates.
type PessoaInfo struct {
	ID          int    `json:"id"`
	Nome        string `json:"nome"`
	EmailInsper string `json:"email_insper"`
	Curso       string `json:"curso"`
	Semestre    int    `json:"semestre"`
}

// AlocacaoResponse is what RunAlocacao returns to the frontend.
type AlocacaoResponse struct {
	Mesas           []MesaResult `json:"mesas"`
	TotalAlocados   int          `json:"total_alocados"`
	NaoAlocadosInfo []PessoaInfo `json:"nao_alocados_info"`
	Pontuacao       int          `json:"pontuacao"`
}

// RunAlocacao runs the allocation algorithm and returns a serialisable result.
func (a *App) RunAlocacao() (AlocacaoResponse, error) {
	conn, err := openDB()
	if err != nil {
		return AlocacaoResponse{}, fmt.Errorf("erro ao abrir banco: %w", err)
	}
	defer conn.Close()

	// Run same steps as Alocar() in alocate.go (unchanged).
	avals := carregarAvaliadores(conn)
	hard, soft := carregarRestricoes(conn)
	horarios := carregarHorarios(conn)
	prefs := carregarDisponibilidades(conn, horarios)
	mesas, porDia := gerarMesas(horarios, avals)
	res := fazerAlocacaoMesas(mesas, porDia, prefs, hard, soft)

	// Evaluator name index from already-loaded slice.
	avalNames := make(map[int]string, len(avals))
	for _, av := range avals {
		avalNames[av.ID] = av.Nome
	}

	// Query all candidates once for both name lookup and non-allocated list.
	type pessoaRow struct {
		ID          int
		Nome        string
		EmailInsper string
		Curso       string
		Semestre    int
	}
	var todasPessoas []pessoaRow
	pessoaNames := make(map[int]string)

	pRows, err := conn.Query(`SELECT id, nome, email_insper, curso, semestre FROM pessoa`)
	if err != nil {
		return AlocacaoResponse{}, fmt.Errorf("erro ao carregar candidatos: %w", err)
	}
	for pRows.Next() {
		var p pessoaRow
		if scanErr := pRows.Scan(&p.ID, &p.Nome, &p.EmailInsper, &p.Curso, &p.Semestre); scanErr == nil {
			pessoaNames[p.ID] = p.Nome
			todasPessoas = append(todasPessoas, p)
		}
	}
	pRows.Close()

	// Assemble mesa results (skip empty mesas).
	var mesaResults []MesaResult
	for _, m := range mesas {
		if len(m.Candidatos) == 0 {
			continue
		}
		mr := MesaResult{ID: m.ID, Descricao: m.Descricao}
		for _, pid := range m.Candidatos {
			name := pessoaNames[pid]
			if name == "" {
				name = fmt.Sprintf("ID %d", pid)
			}
			mr.Candidatos = append(mr.Candidatos, name)
		}
		for _, aid := range m.Avaliadores {
			name := avalNames[aid]
			if name == "" {
				name = fmt.Sprintf("ID %d", aid)
			}
			mr.Avaliadores = append(mr.Avaliadores, name)
		}
		mesaResults = append(mesaResults, mr)
	}
	sort.Slice(mesaResults, func(i, j int) bool {
		return mesaResults[i].ID < mesaResults[j].ID
	})

	// Build non-allocated list.
	alocadosSet := make(map[int]bool, len(res.Alocacao))
	for pid := range res.Alocacao {
		alocadosSet[pid] = true
	}
	var naoAlocados []PessoaInfo
	for _, p := range todasPessoas {
		if !alocadosSet[p.ID] {
			naoAlocados = append(naoAlocados, PessoaInfo{
				ID:          p.ID,
				Nome:        p.Nome,
				EmailInsper: p.EmailInsper,
				Curso:       p.Curso,
				Semestre:    p.Semestre,
			})
		}
	}

	result := AlocacaoResponse{
		Mesas:           mesaResults,
		TotalAlocados:   res.Alocados,
		NaoAlocadosInfo: naoAlocados,
		Pontuacao:       res.Pontuacao,
	}
	a.lastResult = &result
	return result, nil
}

// ResetApp wipes the database and clears in-memory state so the user can start over.
func (a *App) ResetApp() error {
	SetUp()
	a.excelData = nil
	a.lastResult = nil
	a.nOpcoes = 0
	return nil
}

// ExportResultado opens a save-file dialog and writes the last allocation to an xlsx file.
func (a *App) ExportResultado() error {
	if a.lastResult == nil {
		return fmt.Errorf("nenhuma alocação disponível para exportar")
	}

	path, err := wailsrt.SaveFileDialog(a.ctx, wailsrt.SaveDialogOptions{
		Title:           "Salvar Resultado da Alocação",
		DefaultFilename: "alocacao.xlsx",
		Filters: []wailsrt.FileFilter{
			{DisplayName: "Excel (*.xlsx)", Pattern: "*.xlsx"},
		},
	})
	if err != nil {
		return fmt.Errorf("erro ao abrir diálogo: %w", err)
	}
	if path == "" {
		return nil // user cancelled
	}

	f := excelize.NewFile()
	defer f.Close()

	// ── Sheet 1: Alocação ──────────────────────────────────────────────────
	sheet1 := "Alocação"
	f.SetSheetName("Sheet1", sheet1)
	for i, h := range []string{"Mesa", "Candidato", "Avaliadores"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet1, cell, h)
	}
	row := 2
	for _, mesa := range a.lastResult.Mesas {
		avStr := strings.Join(mesa.Avaliadores, ", ")
		for _, cand := range mesa.Candidatos {
			f.SetCellValue(sheet1, fmt.Sprintf("A%d", row), mesa.Descricao)
			f.SetCellValue(sheet1, fmt.Sprintf("B%d", row), cand)
			f.SetCellValue(sheet1, fmt.Sprintf("C%d", row), avStr)
			row++
		}
	}

	// ── Sheet 2: Não Alocados ──────────────────────────────────────────────
	sheet2 := "Não Alocados"
	f.NewSheet(sheet2)
	for i, h := range []string{"Nome", "Email Insper", "Curso", "Semestre"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet2, cell, h)
	}
	for i, p := range a.lastResult.NaoAlocadosInfo {
		r := i + 2
		f.SetCellValue(sheet2, fmt.Sprintf("A%d", r), p.Nome)
		f.SetCellValue(sheet2, fmt.Sprintf("B%d", r), p.EmailInsper)
		f.SetCellValue(sheet2, fmt.Sprintf("C%d", r), p.Curso)
		f.SetCellValue(sheet2, fmt.Sprintf("D%d", r), p.Semestre)
	}

	return f.SaveAs(path)
}
