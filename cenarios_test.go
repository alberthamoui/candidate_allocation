package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// ==================================================
// ============ CENÁRIOS COM PLANILHAS ==============
// ==================================================
//
// Cada cenário monta uma planilha, grava em Excels/cenarios/ (fora do git,
// para dar para abrir e testar à mão pelo app), passa pelo mesmo caminho da
// tela (sugestão de mapeamento, revisão, gravação das três abas, prévia e
// alocação) e confere o que deveria acontecer. As planilhas são recriadas a
// cada execução, então sempre batem com o que o teste espera.

const pastaCenarios = "Excels/cenarios"

// --------------------------------------------------
// Tabelas (abas) das planilhas
// --------------------------------------------------

// tabela é uma aba: colunas com título e chave, e linhas indexadas pela chave
// (linha nil = linha vazia na planilha).
type tabela struct {
	titulos []string
	chaves  []string
	linhas  []map[string]any
}

func novaTabela(colunas ...string) *tabela {
	return &tabela{titulos: slices.Clone(colunas), chaves: slices.Clone(colunas)}
}

func (t *tabela) linha(valores map[string]any) *tabela {
	t.linhas = append(t.linhas, valores)
	return t
}

// renomear troca o título da coluna (a chave continua a mesma).
func (t *tabela) renomear(chave, titulo string) *tabela {
	t.titulos[slices.Index(t.chaves, chave)] = titulo
	return t
}

// remover tira a coluna da planilha.
func (t *tabela) remover(chave string) *tabela {
	i := slices.Index(t.chaves, chave)
	t.chaves = slices.Delete(t.chaves, i, i+1)
	t.titulos = slices.Delete(t.titulos, i, i+1)
	return t
}

// adicionar põe uma coluna no fim, com o valor de cada linha.
func (t *tabela) adicionar(titulo string, valor func(i int) any) *tabela {
	t.chaves = append(t.chaves, titulo)
	t.titulos = append(t.titulos, titulo)
	for i, l := range t.linhas {
		if l != nil {
			l[titulo] = valor(i)
		}
	}
	return t
}

// reordenar põe as colunas na ordem das chaves dadas.
func (t *tabela) reordenar(chaves ...string) *tabela {
	titulos := make([]string, len(chaves))
	for i, c := range chaves {
		titulos[i] = t.titulos[slices.Index(t.chaves, c)]
	}
	t.chaves, t.titulos = chaves, titulos
	return t
}

// vazias insere linhas vazias antes das posições dadas.
func (t *tabela) vazias(posicoes ...int) *tabela {
	slices.Sort(posicoes)
	for k := len(posicoes) - 1; k >= 0; k-- {
		t.linhas = slices.Insert(t.linhas, posicoes[k], map[string]any(nil))
	}
	return t
}

func (t *tabela) linhasPlanilha() [][]any {
	saida := [][]any{toAny(t.titulos)}
	for _, l := range t.linhas {
		if l == nil {
			saida = append(saida, nil)
			continue
		}
		linha := make([]any, len(t.chaves))
		for i, c := range t.chaves {
			linha[i] = l[c]
		}
		saida = append(saida, linha)
	}
	return saida
}

func toAny(s []string) []any {
	r := make([]any, len(s))
	for i, v := range s {
		r[i] = v
	}
	return r
}

// gravarPlanilha grava as abas (na ordem: candidatos, avaliadores, restrições).
func gravarPlanilha(caminho string, abas ...*tabela) error {
	f := excelize.NewFile()
	defer f.Close()
	nomes := []string{"Candidatos", "Avaliadores", "Restricoes"}
	for i, aba := range abas {
		if i == 0 {
			if err := f.SetSheetName("Sheet1", nomes[0]); err != nil {
				return err
			}
		} else if _, err := f.NewSheet(nomes[i]); err != nil {
			return err
		}
		for j, l := range aba.linhasPlanilha() {
			if l == nil {
				continue // linha vazia
			}
			cel, _ := excelize.CoordinatesToCellName(1, j+1)
			if err := f.SetSheetRow(nomes[i], cel, &l); err != nil {
				return err
			}
		}
	}
	return f.SaveAs(caminho)
}

// --------------------------------------------------
// Geradores de dados
// --------------------------------------------------

var (
	diasCenario   = []string{"segunda", "terca", "quarta", "quinta", "sexta"}
	faixasCenario = []string{"8-10", "10-12", "14-16", "16-18"}
	cursosCenario = []string{"Engenharia de Computacao", "Engenharia Civil", "Administracao", "Economia", "Direito"}
)

// horariosCenario devolve n horários distintos (até 20).
func horariosCenario(n int) []string {
	var hs []string
	for _, f := range faixasCenario {
		for _, d := range diasCenario {
			hs = append(hs, d+" "+f)
		}
	}
	return hs[:n]
}

func nomeOpcao(k int) string { return fmt.Sprintf("Opcao %d", k) }

// candidatos gera n candidatos válidos com nOpcoes colunas de opção; opcoes
// diz os horários de cada um (pode devolver menos que nOpcoes).
func candidatos(n, nOpcoes int, opcoes func(i int) []string) *tabela {
	cols := []string{"Timestamp", "Nome", "CPF", "Numero", "Semestre", "Curso", "Email Institucional", "Email Pessoal"}
	for k := 1; k <= nOpcoes; k++ {
		cols = append(cols, nomeOpcao(k))
	}
	t := novaTabela(cols...)
	for i := range n {
		login := fmt.Sprintf("cand%03d", i+1)
		l := map[string]any{
			"Timestamp":           "2025/03/10 09:00:00",
			"Nome":                fmt.Sprintf("Candidato %03d", i+1),
			"CPF":                 fmt.Sprintf("%011d", 20000000000+i),
			"Numero":              fmt.Sprintf("%09d", 202300000+i),
			"Semestre":            fmt.Sprint(i%10 + 1),
			"Curso":               cursosCenario[i%len(cursosCenario)],
			"Email Institucional": login + "@al.insper.edu.br",
			"Email Pessoal":       login + "@gmail.com",
		}
		for k, h := range opcoes(i) {
			if k < nOpcoes {
				l[nomeOpcao(k+1)] = h
			}
		}
		t.linha(l)
	}
	return t
}

// sorteio escolhe k horários distintos para cada candidato (semente fixa).
func sorteio(semente int64, horarios []string, k int) func(i int) []string {
	rng := rand.New(rand.NewSource(semente))
	cache := map[int][]string{}
	return func(i int) []string {
		if o, ok := cache[i]; ok {
			return o
		}
		perm := rng.Perm(len(horarios))
		var o []string
		for _, p := range perm[:min(k, len(horarios))] {
			o = append(o, horarios[p])
		}
		cache[i] = o
		return o
	}
}

// rodizio dá a cada candidato um horário só, em rodízio (mesma quantidade por horário).
func rodizio(horarios []string) func(i int) []string {
	return func(i int) []string { return []string{horarios[i%len(horarios)]} }
}

func avaliadores(n int) *tabela {
	t := novaTabela("Nome", "Email", "Sigla")
	for i := range n {
		t.linha(map[string]any{
			"Nome":  fmt.Sprintf("Avaliador %02d", i+1),
			"Email": fmt.Sprintf("aval%02d@insper.edu.br", i+1),
			"Sigla": siglaCenario(i),
		})
	}
	return t
}

func siglaCenario(i int) string { return fmt.Sprintf("A%02d", i+1) }

// restricoes sorteia, para a fração dada dos candidatos, nNaoPosso siglas de
// "não posso" e nPrefiro de "prefiro não".
func restricoes(semente int64, nCand, nAval int, fracao float64, nNaoPosso, nPrefiro int) *tabela {
	rng := rand.New(rand.NewSource(semente))
	t := novaTabela("Candidato", "NaoPosso", "PrefiroNao")
	for i := range nCand {
		if rng.Float64() >= fracao {
			continue
		}
		perm := rng.Perm(nAval)
		var np, pn []string
		for _, a := range perm[:nNaoPosso] {
			np = append(np, siglaCenario(a))
		}
		for _, a := range perm[nNaoPosso : nNaoPosso+nPrefiro] {
			pn = append(pn, siglaCenario(a))
		}
		t.linha(map[string]any{
			"Candidato":  fmt.Sprintf("Candidato %03d", i+1),
			"NaoPosso":   strings.Join(np, ", "),
			"PrefiroNao": strings.Join(pn, ", "),
		})
	}
	return t
}

// semRestricoes é a aba de restrições só com o cabeçalho (ninguém tem restrição).
func semRestricoes() *tabela { return novaTabela("Candidato", "NaoPosso", "PrefiroNao") }

// --------------------------------------------------
// Rodada: o caminho da tela com uma planilha
// --------------------------------------------------

type rodada struct {
	s        *Session
	param    ParametrosAlocacao
	mapping  []MappingItem
	usuarios UsuariosResponse
	salvos   int // candidatos gravados (depois de resolver duplicados)

	// erro da primeira etapa que falhou ("" = chegou ao resultado)
	etapa string
	erro  error

	previa    CapacidadeResponse
	resultado AlocacaoResponse
	d         dadosAlocacao
}

// rodar faz o que a tela faz. Duplicados são resolvidos como "aceitar um"
// (fica o primeiro de cada grupo).
func rodar(t *testing.T, caminho string, horariosLivres []string, param ParametrosAlocacao) *rodada {
	t.Helper()
	dados, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	r := &rodada{s: setupSession(), param: param}
	r.s.horariosLivres = horariosLivres
	falhou := func(etapa string, err error) bool {
		if err != nil {
			r.etapa, r.erro = etapa, err
		}
		return err != nil
	}

	nOpcoes := OPCOES_AUTOMATICO
	if horariosLivres != nil {
		nOpcoes = 0
	}
	if r.mapping, err = r.s.SuggestMapping(dados, nOpcoes, r.s.emailDomain); falhou("upload", err) {
		return r
	}
	if r.usuarios, err = r.s.BuildUsuariosWithMapping(r.mapping); falhou("candidatos", err) {
		return r
	}
	descartar := map[int]bool{}
	for _, g := range r.usuarios.Duplicates {
		for _, id := range g {
			if id != slices.Min(g) {
				descartar[id] = true
			}
		}
	}
	var cands []Usuario
	for i := 1; i <= len(r.usuarios.Usuarios); i++ {
		if !descartar[i] {
			cands = append(cands, r.usuarios.Usuarios[i].Usuario)
		}
	}
	r.salvos = len(cands)
	if falhou("candidatos", r.s.SaveUsuarios(cands)) {
		return r
	}

	mAv, err := r.s.SuggestMappingAvaliador()
	if falhou("avaliadores", err) {
		return r
	}
	avs, err := r.s.BuildAvaliadoresWithMapping(mAv)
	if falhou("avaliadores", err) || falhou("avaliadores", r.s.SaveAvaliadores(avs)) {
		return r
	}

	mRe, err := r.s.SuggestMappingRestricao()
	if falhou("restricoes", err) {
		return r
	}
	res, err := r.s.BuildRestricoesWithMapping(mRe)
	if falhou("restricoes", err) || falhou("restricoes", r.s.SaveRestricoes(res)) {
		return r
	}

	if r.previa, err = r.s.PreviaCapacidade(param); falhou("previa", err) {
		return r
	}
	if r.resultado, err = r.s.RunAlocacao(context.Background(), param, func(any) {}); falhou("alocacao", err) {
		return r
	}
	if r.d, err = r.s.carregarDados(); falhou("alocacao", err) {
		return r
	}
	return r
}

// chegouAoFim falha o teste se alguma etapa deu erro.
func (r *rodada) chegouAoFim(t *testing.T) bool {
	t.Helper()
	if r.erro != nil {
		t.Errorf("parou na etapa %q: %v", r.etapa, r.erro)
		return false
	}
	return true
}

// coluna devolve a coluna da planilha ligada à variável no mapeamento.
func (r *rodada) coluna(variavel string) string {
	for _, m := range r.mapping {
		if m.Variavel == variavel {
			return m.NomeColuna
		}
	}
	return "(campo não pedido)"
}

// errosEm conta os candidatos com erro de validação no campo.
func (r *rodada) errosEm(campo string) int {
	n := 0
	for _, v := range r.usuarios.Usuarios {
		for _, e := range v.Erros {
			if e.Field == campo {
				n++
				break
			}
		}
	}
	return n
}

func (r *rodada) totalErros() int {
	n := 0
	for _, v := range r.usuarios.Usuarios {
		n += len(v.Erros)
	}
	return n
}

// esperarAlocados confere quantos ficaram com mesa.
func (r *rodada) esperarAlocados(t *testing.T, n int) {
	t.Helper()
	if r.resultado.TotalAlocados != n {
		t.Errorf("%d alocados; esperado %d (sem mesa: %d)", r.resultado.TotalAlocados, n, len(r.resultado.NaoAlocadosInfo))
	}
}

// conferirRegras confere, no resultado, as regras que toda alocação tem que
// cumprir: tamanho das mesas, avaliadores distintos por horário, "não posso"
// nunca na mesa e candidato só em horário que escolheu.
func (r *rodada) conferirRegras(t *testing.T) {
	t.Helper()
	p := r.param
	vistos := map[int]bool{}
	avPorHorario := map[int]map[int]bool{}
	mesasPorHorario := map[int]int{}
	for _, m := range r.resultado.Mesas {
		if n := len(m.Candidatos); n < p.MinPessoasPorMesa || n > p.MaxPessoasPorMesa {
			t.Errorf("%s: %d candidatos (esperado %d a %d)", m.Descricao, n, p.MinPessoasPorMesa, p.MaxPessoasPorMesa)
		}
		if len(m.Avaliadores) != p.AvaliadoresPorMesa {
			t.Errorf("%s: %d avaliadores (esperado %d)", m.Descricao, len(m.Avaliadores), p.AvaliadoresPorMesa)
		}
		if mesasPorHorario[m.DiaID]++; mesasPorHorario[m.DiaID] > p.MesasPorHorario {
			t.Errorf("%s: mais de %d mesas no horário", m.Descricao, p.MesasPorHorario)
		}
		if avPorHorario[m.DiaID] == nil {
			avPorHorario[m.DiaID] = map[int]bool{}
		}
		for _, a := range m.Avaliadores {
			if avPorHorario[m.DiaID][a.ID] {
				t.Errorf("%s: %s em duas mesas do mesmo horário", m.Descricao, a.Nome)
			}
			avPorHorario[m.DiaID][a.ID] = true
		}
		for _, c := range m.Candidatos {
			if vistos[c.ID] {
				t.Errorf("%s em mais de uma mesa", c.Nome)
			}
			vistos[c.ID] = true
			if !slices.Contains(r.d.prefs[c.ID], m.DiaID) {
				t.Errorf("%s alocado em horário que não escolheu (%s)", c.Nome, m.Descricao)
			}
			for _, a := range m.Avaliadores {
				if r.d.hard[a.ID][c.ID] {
					t.Errorf("%s: %s marcou \"não posso\" para %s", m.Descricao, a.Nome, c.Nome)
				}
			}
		}
	}
	if len(vistos) != r.resultado.TotalAlocados {
		t.Errorf("TotalAlocados = %d, mas há %d candidatos nas mesas", r.resultado.TotalAlocados, len(vistos))
	}
}

// contarNoBanco conta linhas de uma consulta no banco da sessão.
func (r *rodada) contarNoBanco(t *testing.T, consulta string) int {
	t.Helper()
	var n int
	if err := r.s.db.QueryRow(consulta).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// --------------------------------------------------
// Os cenários
// --------------------------------------------------

type cenario struct {
	nome      string
	descricao string // aparece no LEIAME.md da pasta
	abas      func() []*tabela
	// horariosLivres liga o modo "Eu defino" com esses horários
	horariosLivres []string
	param          *ParametrosAlocacao // nil = parâmetros padrão
	conferir       func(t *testing.T, r *rodada)
}

func cenariosPlanilha() []cenario {
	h8, h12 := horariosCenario(8), horariosCenario(12)
	padrao := func(n, nOpcoes int, hs []string, nAval int) func() []*tabela {
		return func() []*tabela {
			return []*tabela{candidatos(n, nOpcoes, sorteio(1, hs, nOpcoes)), avaliadores(nAval), restricoes(2, n, nAval, 0.3, 1, 1)}
		}
	}

	return []cenario{
		// ---------- colunas de opção ----------
		{
			nome: "01_padrao_5_opcoes", descricao: "60 candidatos, 5 opções, 8 horários, 15 avaliadores, restrições em 30% dos candidatos. Todos devem ter mesa.",
			abas: padrao(60, 5, h8, 15),
			conferir: func(t *testing.T, r *rodada) {
				if r.chegouAoFim(t) {
					if r.s.nOpcoes != 5 {
						t.Errorf("%d opções detectadas; esperado 5", r.s.nOpcoes)
					}
					r.esperarAlocados(t, 60)
					r.conferirRegras(t)
				}
			},
		},
		{
			nome: "02_uma_opcao", descricao: "30 candidatos com uma opção só, 10 por horário em 3 horários. Todos devem ter mesa (2 mesas de 5 por horário).",
			abas: func() []*tabela {
				hs := horariosCenario(3)
				return []*tabela{candidatos(30, 1, rodizio(hs)), avaliadores(10), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if r.chegouAoFim(t) {
					if r.s.nOpcoes != 1 {
						t.Errorf("%d opções detectadas; esperado 1", r.s.nOpcoes)
					}
					r.esperarAlocados(t, 30)
					r.conferirRegras(t)
				}
			},
		},
		{
			nome: "03_dez_opcoes", descricao: "50 candidatos com 10 opções em 12 horários.",
			abas: padrao(50, 10, h12, 15),
			conferir: func(t *testing.T, r *rodada) {
				if r.chegouAoFim(t) {
					if r.s.nOpcoes != 10 {
						t.Errorf("%d opções detectadas; esperado 10", r.s.nOpcoes)
					}
					r.esperarAlocados(t, 50)
					r.conferirRegras(t)
				}
			},
		},
		{
			nome: "04_opcoes_por_extenso", descricao: `Colunas "Primeira Opção" a "Quarta Opção".`,
			abas: func() []*tabela {
				c := candidatos(40, 4, sorteio(3, h8, 4))
				for k, nome := range []string{"Primeira Opção", "Segunda Opção", "Terceira Opção", "Quarta Opção"} {
					c.renomear(nomeOpcao(k+1), nome)
				}
				return []*tabela{c, avaliadores(15), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if !r.chegouAoFim(t) {
					return
				}
				if r.s.nOpcoes != 4 || r.coluna("opcao 1") != "Primeira Opção" || r.coluna("opcao 4") != "Quarta Opção" {
					t.Errorf("%d opções; opcao 1 = %q, opcao 4 = %q", r.s.nOpcoes, r.coluna("opcao 1"), r.coluna("opcao 4"))
				}
				r.esperarAlocados(t, 40)
			},
		},
		{
			nome: "05_horario_numerado", descricao: `Colunas "Horário 1" a "Horário 3" em vez de "Opção".`,
			abas: func() []*tabela {
				c := candidatos(40, 3, sorteio(4, h8, 3))
				for k := 1; k <= 3; k++ {
					c.renomear(nomeOpcao(k), fmt.Sprintf("Horário %d", k))
				}
				return []*tabela{c, avaliadores(15), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if r.chegouAoFim(t) {
					if r.s.nOpcoes != 3 || r.coluna("opcao 2") != "Horário 2" {
						t.Errorf("%d opções; opcao 2 = %q", r.s.nOpcoes, r.coluna("opcao 2"))
					}
					r.esperarAlocados(t, 40)
				}
			},
		},
		{
			nome: "06_opcao_faltando_no_meio", descricao: `Colunas "Opcao 1", "Opcao 2" e "Opcao 4" (sem a 3).`,
			abas: func() []*tabela {
				return []*tabela{candidatos(40, 4, sorteio(5, h8, 4)).remover(nomeOpcao(3)), avaliadores(15), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if r.chegouAoFim(t) {
					if r.s.nOpcoes != 4 || r.coluna("opcao 3") != "" || r.coluna("opcao 4") != "Opcao 4" {
						t.Errorf("%d opções; opcao 3 = %q, opcao 4 = %q", r.s.nOpcoes, r.coluna("opcao 3"), r.coluna("opcao 4"))
					}
					r.esperarAlocados(t, 40)
				}
			},
		},
		{
			nome: "07_coluna_observacoes", descricao: `Coluna extra "Observações" com texto livre: não pode virar horário.`,
			abas: func() []*tabela {
				c := candidatos(40, 3, sorteio(6, h8, 3)).adicionar("Observações", func(i int) any { return "tenho carro" })
				return []*tabela{c, avaliadores(15), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if !r.chegouAoFim(t) {
					return
				}
				for _, h := range r.d.horarios {
					if h.Descricao == "tenho carro" {
						t.Error(`"Observações" virou horário`)
					}
				}
				r.esperarAlocados(t, 40)
			},
		},
		{
			nome: "08_colunas_embaralhadas", descricao: "Mesmas colunas da planilha oficial, em outra ordem.",
			abas: func() []*tabela {
				c := candidatos(40, 3, sorteio(7, h8, 3)).reordenar(
					nomeOpcao(2), "Email Pessoal", "Nome", nomeOpcao(3), "Curso", "CPF", "Timestamp", "Email Institucional", nomeOpcao(1), "Semestre", "Numero")
				return []*tabela{c, avaliadores(15), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if !r.chegouAoFim(t) {
					return
				}
				for v, col := range map[string]string{"nome": "Nome", "email_insper": "Email Institucional", "numero": "Numero", "opcao 1": "Opcao 1", "opcao 3": "Opcao 3"} {
					if r.coluna(v) != col {
						t.Errorf("%s ligado a %q; esperado %q", v, r.coluna(v), col)
					}
				}
				if n := r.totalErros(); n != 0 {
					t.Errorf("%d erros de validação; esperado 0", n)
				}
				r.esperarAlocados(t, 40)
			},
		},
		{
			nome: "09_sem_colunas_de_opcao", descricao: `Nenhuma coluna de opção no modo "Candidatos escolhem": o upload tem que avisar.`,
			abas: func() []*tabela {
				c := candidatos(30, 0, func(int) []string { return nil })
				return []*tabela{c, avaliadores(10), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if r.etapa != "upload" || !strings.Contains(fmt.Sprint(r.erro), "colunas de opção") {
					t.Errorf(`esperado erro no upload sobre "colunas de opção"; etapa %q, erro %v`, r.etapa, r.erro)
				}
			},
		},
		{
			nome: "10_sem_opcoes_eu_defino", descricao: `Mesma planilha do 09, no modo "Eu defino" com 4 horários.`,
			abas: func() []*tabela {
				return []*tabela{candidatos(30, 0, func(int) []string { return nil }), avaliadores(10), semRestricoes()}
			},
			horariosLivres: []string{"sabado 9-11", "sabado 11-13", "sabado 14-16", "sabado 16-18"},
			conferir: func(t *testing.T, r *rodada) {
				if r.chegouAoFim(t) {
					if !r.resultado.SemPreferencia {
						t.Error("resultado deveria estar sem preferência")
					}
					r.esperarAlocados(t, 30)
					r.conferirRegras(t)
				}
			},
		},
		{
			nome: "11_eu_defino_ignora_opcoes", descricao: `Planilha com colunas de opção no modo "Eu defino": as opções são ignoradas.`,
			abas:           padrao(40, 5, h8, 15),
			horariosLivres: []string{"sabado 9-11", "sabado 11-13"},
			conferir: func(t *testing.T, r *rodada) {
				if !r.chegouAoFim(t) {
					return
				}
				for _, m := range r.resultado.Mesas {
					if !strings.HasPrefix(m.DiaNome, "sabado") {
						t.Errorf("mesa em horário da planilha: %s", m.Descricao)
					}
				}
				r.esperarAlocados(t, 40)
				r.conferirRegras(t)
			},
		},
		{
			nome: "12_cabecalhos_variantes", descricao: `Cabeçalhos com outros nomes: "Nome completo", "E-mail institucional", "Celular", "Período"...`,
			abas: func() []*tabela {
				c := candidatos(40, 3, sorteio(8, h8, 3)).
					renomear("Timestamp", "Carimbo de data/hora").renomear("Nome", "Nome completo").
					renomear("Email Institucional", "E-mail institucional").renomear("Email Pessoal", "E-mail pessoal").
					renomear("Numero", "Celular").renomear("Semestre", "Período")
				return []*tabela{c, avaliadores(15), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if !r.chegouAoFim(t) {
					return
				}
				for v, col := range map[string]string{"nome": "Nome completo", "email_insper": "E-mail institucional", "email_pessoal": "E-mail pessoal", "numero": "Celular", "semestre": "Período"} {
					if r.coluna(v) != col {
						t.Errorf("%s ligado a %q; esperado %q", v, r.coluna(v), col)
					}
				}
				if n := r.totalErros(); n != 0 {
					t.Errorf("%d erros de validação; esperado 0", n)
				}
			},
		},

		// ---------- dados dos candidatos ----------
		{
			nome: "13_sem_coluna_cpf", descricao: "Sem a coluna CPF (é opcional).",
			abas: func() []*tabela {
				return []*tabela{candidatos(40, 3, sorteio(9, h8, 3)).remover("CPF"), avaliadores(15), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if !r.chegouAoFim(t) {
					return
				}
				if n := r.errosEm("cpf"); n != 0 {
					t.Errorf("%d erros de CPF; esperado 0", n)
				}
				if n := r.contarNoBanco(t, `SELECT COUNT(*) FROM pessoa WHERE cpf IS NULL`); n != 40 {
					t.Errorf("%d candidatos sem CPF no banco; esperado 40", n)
				}
				r.esperarAlocados(t, 40)
			},
		},
		{
			nome: "14_cpf_e_telefone_formatados", descricao: `CPF com pontuação ("123.456.789-01") e telefone formatado ("(11) 98888-7777").`,
			abas: func() []*tabela {
				c := candidatos(40, 3, sorteio(10, h8, 3))
				for i, l := range c.linhas {
					cpf := fmt.Sprintf("%011d", 30000000000+i)
					l["CPF"] = cpf[:3] + "." + cpf[3:6] + "." + cpf[6:9] + "-" + cpf[9:]
					l["Numero"] = fmt.Sprintf("(11) 9%04d-%04d", 8000+i, 1000+i)
				}
				return []*tabela{c, avaliadores(15), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if !r.chegouAoFim(t) {
					return
				}
				if n := r.totalErros(); n != 0 {
					t.Errorf("%d erros de validação; esperado 0", n)
				}
				if u := r.usuarios.Usuarios[1].Usuario; u.CPF != "30000000000" || u.Numero != "11980001000" {
					t.Errorf("CPF %q e telefone %q; esperado só os dígitos", u.CPF, u.Numero)
				}
			},
		},
		{
			nome: "15_dados_invalidos", descricao: "5 candidatos com um campo inválido cada (email, email pessoal, semestre, CPF, telefone).",
			abas: func() []*tabela {
				c := candidatos(30, 3, sorteio(11, h8, 3))
				c.linhas[0]["Email Institucional"] = "cand001@gmail.com"
				c.linhas[1]["Email Pessoal"] = "sem-arroba"
				c.linhas[2]["Semestre"] = "11"
				c.linhas[3]["CPF"] = "abc"
				c.linhas[4]["Numero"] = "123"
				return []*tabela{c, avaliadores(15), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				for linha, campo := range map[int]string{1: "email_insper", 2: "email_pessoal", 3: "semestre", 4: "cpf", 5: "numero"} {
					erros := r.usuarios.Usuarios[linha].Erros
					if len(erros) != 1 || erros[0].Field != campo {
						t.Errorf("linha %d: erros %v; esperado só %s", linha, erros, campo)
					}
				}
				if r.totalErros() != 5 {
					t.Errorf("%d erros no total; esperado 5", r.totalErros())
				}
				if r.chegouAoFim(t) {
					r.conferirRegras(t)
				}
			},
		},
		{
			nome: "16_duplicados", descricao: "Dois pares de candidatos repetidos (mesmo CPF; mesmo email).",
			abas: func() []*tabela {
				c := candidatos(30, 3, sorteio(12, h8, 3))
				c.linhas[6]["CPF"] = c.linhas[2]["CPF"]
				c.linhas[8]["Email Institucional"] = c.linhas[3]["Email Institucional"]
				return []*tabela{c, avaliadores(15), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if len(r.usuarios.Duplicates) != 2 {
					t.Errorf("grupos de duplicados: %v; esperado 2", r.usuarios.Duplicates)
				}
				if r.chegouAoFim(t) {
					r.esperarAlocados(t, 28)
				}
			},
		},
		{
			nome: "17_linhas_vazias_no_meio", descricao: "30 candidatos com 3 linhas vazias no meio: as vazias não são candidatos.",
			abas: func() []*tabela {
				return []*tabela{candidatos(30, 3, sorteio(13, h8, 3)).vazias(5, 12, 20), avaliadores(15), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if n := len(r.usuarios.Usuarios); n != 30 {
					t.Errorf("%d candidatos lidos; esperado 30 (linhas vazias viraram candidatos)", n)
				}
				if r.chegouAoFim(t) {
					r.esperarAlocados(t, 30)
				}
			},
		},
		{
			nome: "18_horario_escrito_diferente", descricao: `O mesmo horário escrito de jeitos diferentes ("Segunda 14-16", "segunda 14-16 ", "SEGUNDA  14-16").`,
			abas: func() []*tabela {
				jeitos := []string{"segunda 14-16", "Segunda 14-16", "segunda 14-16 ", "SEGUNDA  14-16"}
				c := candidatos(24, 2, func(i int) []string { return []string{jeitos[i%len(jeitos)], "terca 10-12"} })
				return []*tabela{c, avaliadores(10), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if !r.chegouAoFim(t) {
					return
				}
				if len(r.d.horarios) != 2 {
					var hs []string
					for _, h := range r.d.horarios {
						hs = append(hs, fmt.Sprintf("%q", h.Descricao))
					}
					t.Errorf("%d horários (%s); esperado 2", len(hs), strings.Join(hs, ", "))
				}
				r.esperarAlocados(t, 24)
			},
		},
		{
			nome: "19_horario_repetido_no_candidato", descricao: "Candidatos que repetem o mesmo horário em duas opções.",
			abas: func() []*tabela {
				c := candidatos(30, 3, func(i int) []string { h := h8[i%4]; return []string{h, h8[4+i%4], h} })
				return []*tabela{c, avaliadores(15), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if r.chegouAoFim(t) {
					r.esperarAlocados(t, 30)
					r.conferirRegras(t)
				}
			},
		},
		{
			nome: "20_candidatos_sem_opcao", descricao: "30 candidatos normais e 3 sem nenhuma opção preenchida: os 3 ficam sem mesa, sem quebrar.",
			abas: func() []*tabela {
				sorteados := sorteio(14, h8, 3)
				c := candidatos(33, 3, func(i int) []string {
					if i >= 30 {
						return nil
					}
					return sorteados(i)
				})
				return []*tabela{c, avaliadores(15), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if !r.chegouAoFim(t) {
					return
				}
				r.esperarAlocados(t, 30)
				if n := len(r.resultado.NaoAlocadosInfo); n != 3 {
					t.Errorf("%d sem mesa; esperado 3", n)
				}
			},
		},
		{
			nome: "21_nome_diferente_na_restricao", descricao: `Restrições com o nome do candidato em outra caixa e com espaços a mais ("candidato 001 ", "CANDIDATO  002").`,
			abas: func() []*tabela {
				r := semRestricoes().
					linha(map[string]any{"Candidato": "candidato 001 ", "NaoPosso": "A01"}).
					linha(map[string]any{"Candidato": "CANDIDATO  002", "NaoPosso": "A02"}).
					linha(map[string]any{"Candidato": " Candidato 003", "NaoPosso": "A03"})
				return []*tabela{candidatos(30, 3, sorteio(15, h8, 3)), avaliadores(15), r}
			},
			conferir: func(t *testing.T, r *rodada) {
				if r.chegouAoFim(t) {
					if n := r.contarNoBanco(t, `SELECT COUNT(*) FROM restricoesNposso`); n != 3 {
						t.Errorf("%d restrições gravadas; esperado 3 (nome com caixa/espaços diferentes foi ignorado)", n)
					}
				}
			},
		},
		{
			nome: "22_timestamp_numerico", descricao: "Timestamp como número do Excel (45684,82), como sai de formulários.",
			abas: func() []*tabela {
				c := candidatos(30, 3, sorteio(16, h8, 3))
				for i, l := range c.linhas {
					l["Timestamp"] = 45684.82 + float64(i)/100
				}
				return []*tabela{c, avaliadores(15), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if r.chegouAoFim(t) {
					if n := r.totalErros(); n != 0 {
						t.Errorf("%d erros de validação; esperado 0", n)
					}
					r.esperarAlocados(t, 30)
				}
			},
		},

		// ---------- avaliadores ----------
		{
			nome: "23_poucos_avaliadores", descricao: "4 avaliadores com 5 por mesa: nenhuma mesa dá para formar; a prévia avisa.",
			abas: padrao(20, 3, h8, 4),
			conferir: func(t *testing.T, r *rodada) {
				if !r.chegouAoFim(t) {
					return
				}
				if r.previa.MesasPorHorario != 0 || len(r.previa.Avisos) == 0 {
					t.Errorf("prévia: %d mesas por horário, avisos %q", r.previa.MesasPorHorario, r.previa.Avisos)
				}
				r.esperarAlocados(t, 0)
			},
		},
		{
			nome: "24_avaliadores_exatos", descricao: "5 avaliadores (uma mesa por horário) e 5 candidatos em cada um dos 4 horários.",
			abas: func() []*tabela {
				return []*tabela{candidatos(20, 1, rodizio(horariosCenario(4))), avaliadores(5), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if r.chegouAoFim(t) {
					if r.previa.MesasPorHorario != 1 {
						t.Errorf("%d mesas por horário; esperado 1", r.previa.MesasPorHorario)
					}
					r.esperarAlocados(t, 20)
					r.conferirRegras(t)
				}
			},
		},
		{
			nome: "25_muitos_avaliadores", descricao: "60 avaliadores para 20 candidatos.",
			abas: padrao(20, 3, h8, 60),
			conferir: func(t *testing.T, r *rodada) {
				if r.chegouAoFim(t) {
					r.esperarAlocados(t, 20)
					r.conferirRegras(t)
				}
			},
		},
		{
			nome: "26_avaliador_repetido", descricao: "Dois avaliadores com o mesmo nome: a gravação tem que avisar.",
			abas: func() []*tabela {
				a := avaliadores(10)
				a.linhas[5]["Nome"] = a.linhas[2]["Nome"]
				return []*tabela{candidatos(30, 3, sorteio(17, h8, 3)), a, semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if r.etapa != "avaliadores" || !strings.Contains(fmt.Sprint(r.erro), "repetido") {
					t.Errorf("esperado erro de avaliador repetido; etapa %q, erro %v", r.etapa, r.erro)
				}
			},
		},
		{
			nome: "27_avaliadores_linhas_vazias", descricao: "15 avaliadores com linhas vazias no meio.",
			abas: func() []*tabela {
				return []*tabela{candidatos(30, 3, sorteio(18, h8, 3)), avaliadores(15).vazias(3, 9), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if r.chegouAoFim(t) {
					if n := r.contarNoBanco(t, `SELECT COUNT(*) FROM avaliador`); n != 15 {
						t.Errorf("%d avaliadores gravados; esperado 15", n)
					}
				}
			},
		},

		// ---------- restrições ----------
		{
			nome: "28_restricoes_pesadas", descricao: `Todo candidato com "não posso" para 6 dos 15 avaliadores e "prefiro não" para outros 3.`,
			abas: func() []*tabela {
				return []*tabela{candidatos(40, 5, sorteio(19, h8, 5)), avaliadores(15), restricoes(20, 40, 15, 1, 6, 3)}
			},
			conferir: func(t *testing.T, r *rodada) {
				// tão restrito que nem todos cabem: o que importa é nunca violar
				if r.chegouAoFim(t) {
					r.conferirRegras(t)
					t.Logf("%d de 40 alocados", r.resultado.TotalAlocados)
				}
			},
		},
		{
			nome: "29_restricoes_inexistentes", descricao: "Restrições com sigla e candidato que não existem: são ignoradas, as outras valem.",
			abas: func() []*tabela {
				r := semRestricoes().
					linha(map[string]any{"Candidato": "Candidato 001", "NaoPosso": "A01, ZZZ"}).
					linha(map[string]any{"Candidato": "Fulano de Tal", "NaoPosso": "A02"})
				return []*tabela{candidatos(30, 3, sorteio(21, h8, 3)), avaliadores(15), r}
			},
			conferir: func(t *testing.T, r *rodada) {
				if r.chegouAoFim(t) {
					if n := r.contarNoBanco(t, `SELECT COUNT(*) FROM restricoesNposso`); n != 1 {
						t.Errorf("%d restrições gravadas; esperado 1", n)
					}
				}
			},
		},
		{
			nome: "30_restricoes_separadores", descricao: `Siglas separadas por vírgula, espaço, ponto e vírgula, barra e quebra de linha.`,
			abas: func() []*tabela {
				r := semRestricoes()
				for i, s := range []string{"A01,A02", "A01 A02", "A01;A02", "A01 / A02", "A01\nA02"} {
					r.linha(map[string]any{"Candidato": fmt.Sprintf("Candidato %03d", i+1), "NaoPosso": s})
				}
				return []*tabela{candidatos(30, 3, sorteio(22, h8, 3)), avaliadores(15), r}
			},
			conferir: func(t *testing.T, r *rodada) {
				if r.chegouAoFim(t) {
					if n := r.contarNoBanco(t, `SELECT COUNT(*) FROM restricoesNposso`); n != 10 {
						t.Errorf("%d restrições gravadas; esperado 10 (2 por candidato)", n)
					}
				}
			},
		},
		{
			nome: "31_restricao_impossivel", descricao: `33 candidatos, 11 por horário; um com "não posso" para todos os avaliadores: só ele fica sem mesa.`,
			abas: func() []*tabela {
				var todas []string
				for i := range 15 {
					todas = append(todas, siglaCenario(i))
				}
				r := semRestricoes().linha(map[string]any{"Candidato": "Candidato 001", "NaoPosso": strings.Join(todas, ", ")})
				return []*tabela{candidatos(33, 1, rodizio(horariosCenario(3))), avaliadores(15), r}
			},
			conferir: func(t *testing.T, r *rodada) {
				if !r.chegouAoFim(t) {
					return
				}
				r.conferirRegras(t)
				r.esperarAlocados(t, 32)
			},
		},
		{
			nome: "32_aba_restricoes_vazia", descricao: "Aba de restrições só com o cabeçalho (ninguém tem restrição).",
			abas: func() []*tabela {
				return []*tabela{candidatos(30, 3, sorteio(24, h8, 3)), avaliadores(15), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if r.chegouAoFim(t) {
					r.esperarAlocados(t, 30)
				}
			},
		},

		// ---------- estrutura e escala ----------
		{
			nome: "33_so_aba_de_candidatos", descricao: "Planilha com uma aba só: o erro tem que dizer que falta a aba de avaliadores.",
			abas: func() []*tabela { return []*tabela{candidatos(30, 3, sorteio(25, h8, 3))} },
			conferir: func(t *testing.T, r *rodada) {
				if r.etapa != "avaliadores" || !strings.Contains(fmt.Sprint(r.erro), "aba") {
					t.Errorf("esperado erro sobre a aba de avaliadores; etapa %q, erro %v", r.etapa, r.erro)
				}
			},
		},
		{
			nome: "34_um_candidato", descricao: "Um candidato só (menos que o mínimo de 5 por mesa).",
			abas: padrao(1, 3, h8, 15),
			conferir: func(t *testing.T, r *rodada) {
				if r.chegouAoFim(t) {
					r.esperarAlocados(t, 0)
					if len(r.resultado.NaoAlocadosInfo) != 1 {
						t.Errorf("%d sem mesa; esperado 1", len(r.resultado.NaoAlocadosInfo))
					}
				}
			},
		},
		{
			nome: "35_capacidade_insuficiente", descricao: "150 candidatos, 10 avaliadores e 3 horários: cabem no máximo 48.",
			abas: padrao(150, 3, horariosCenario(3), 10),
			conferir: func(t *testing.T, r *rodada) {
				if !r.chegouAoFim(t) {
					return
				}
				if r.previa.MaxAlocaveis != 48 || len(r.previa.Avisos) == 0 {
					t.Errorf("prévia: cabem %d, avisos %q; esperado 48 e um aviso", r.previa.MaxAlocaveis, r.previa.Avisos)
				}
				r.esperarAlocados(t, 48)
				r.conferirRegras(t)
			},
		},
		{
			nome: "36_horario_lotado", descricao: `Todos com "quinta 10-12" como 1ª opção: a maioria vai para a 2ª ou 3ª.`,
			abas: func() []*tabela {
				outros := sorteio(26, slices.DeleteFunc(slices.Clone(h8), func(h string) bool { return h == "quinta 10-12" }), 2)
				c := candidatos(60, 3, func(i int) []string { return append([]string{"quinta 10-12"}, outros(i)...) })
				return []*tabela{c, avaliadores(15), semRestricoes()}
			},
			conferir: func(t *testing.T, r *rodada) {
				if !r.chegouAoFim(t) {
					return
				}
				r.esperarAlocados(t, 60)
				r.conferirRegras(t)
				for _, it := range r.resultado.Qualidade {
					if it.Codigo == "opcao_1" && it.Valor != 24 {
						t.Errorf("%d na 1ª opção; esperado 24 (3 mesas de 8)", it.Valor)
					}
				}
			},
		},
		{
			nome: "37_grande", descricao: "400 candidatos, 40 avaliadores, 12 horários, 5 opções e restrições.",
			abas: func() []*tabela {
				return []*tabela{candidatos(400, 5, sorteio(27, h12, 5)), avaliadores(40), restricoes(28, 400, 40, 0.4, 2, 2)}
			},
			conferir: func(t *testing.T, r *rodada) {
				if r.chegouAoFim(t) {
					r.esperarAlocados(t, 400)
					r.conferirRegras(t)
				}
			},
		},
		{
			nome: "38_parametros_diferentes", descricao: "Mesas de 3 a 4 candidatos com 2 avaliadores (parâmetros mudados na tela).",
			abas:  padrao(40, 4, h8, 12),
			param: &ParametrosAlocacao{MesasPorHorario: 6, MinPessoasPorMesa: 3, MaxPessoasPorMesa: 4, AvaliadoresPorMesa: 2, Criterios: []CriterioAlocacao{}},
			conferir: func(t *testing.T, r *rodada) {
				if r.chegouAoFim(t) {
					r.esperarAlocados(t, 40)
					r.conferirRegras(t)
				}
			},
		},
	}
}

// TestCenariosPlanilha grava as planilhas em Excels/cenarios/ e roda cada uma.
func TestCenariosPlanilha(t *testing.T) {
	if testing.Short() {
		t.Skip("cenários com planilhas são lentos")
	}
	// buscas menores: os cenários são pequenos e o teste fica rápido
	rec := recursos
	rec.Execucoes, rec.Iteracoes = 4, 100_000
	comRecursos(t, rec)

	if err := os.MkdirAll(pastaCenarios, 0o755); err != nil {
		t.Fatal(err)
	}
	var leiame strings.Builder
	leiame.WriteString("# Planilhas de cenários\n\nGeradas por `go test -run TestCenariosPlanilha` (cenarios_test.go). Não edite: são recriadas a cada execução.\n\n| Planilha | Modo | O que testa |\n|---|---|---|\n")

	for _, c := range cenariosPlanilha() {
		caminho := filepath.Join(pastaCenarios, c.nome+".xlsx")
		modo := "Candidatos escolhem"
		if c.horariosLivres != nil {
			modo = "Eu defino: " + strings.Join(c.horariosLivres, ", ")
		}
		fmt.Fprintf(&leiame, "| %s | %s | %s |\n", c.nome, modo, c.descricao)

		t.Run(c.nome, func(t *testing.T) {
			if err := gravarPlanilha(caminho, c.abas()...); err != nil {
				t.Fatal(err)
			}
			param := parametrosAlocacaoPadrao()
			if c.param != nil {
				param = *c.param
			}
			inicio := time.Now()
			r := rodar(t, caminho, c.horariosLivres, param)
			c.conferir(t, r)
			if d := time.Since(inicio); d > 30*time.Second {
				t.Errorf("levou %v", d.Round(time.Millisecond))
			}
		})
	}
	if err := os.WriteFile(filepath.Join(pastaCenarios, "LEIAME.md"), []byte(leiame.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}
