package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// HorarioCapacidade é a procura por um horário na prévia de capacidade.
type HorarioCapacidade struct {
	Descricao     string `json:"descricao"`
	Interessados  int    `json:"interessados"`   // escolheram o horário em qualquer opção
	PrimeiraOpcao int    `json:"primeira_opcao"` // escolheram como 1ª opção
}

// CapacidadeResponse resume o que cabe com os parâmetros dados, antes de rodar
// a alocação.
type CapacidadeResponse struct {
	Parametros           ParametrosAlocacao  `json:"parametros"`
	Candidatos           int                 `json:"candidatos"`
	Avaliadores          int                 `json:"avaliadores"`
	MesasPorHorario      int                 `json:"mesas_por_horario"` // as que dá para formar com os avaliadores
	CapacidadePorHorario int                 `json:"capacidade_por_horario"`
	CapacidadeTotal      int                 `json:"capacidade_total"`
	MaxAlocaveis         int                 `json:"max_alocaveis"` // limite considerando os horários que cada um escolheu
	Horarios             []HorarioCapacidade `json:"horarios"`
	Avisos               []string            `json:"avisos"`
	// ValoresColunas são os valores de curso e semestre, para a tela oferecer
	// nos critérios adicionais.
	ValoresColunas map[string][]ValorColuna `json:"valores_colunas"`
}

// ValorColuna é um valor de uma coluna e quantos candidatos o têm.
type ValorColuna struct {
	Valor      string `json:"valor"`
	Quantidade int    `json:"quantidade"`
}

// PreviaCapacidade calcula quantas mesas e vagas os parâmetros permitem com os
// dados da sessão. Os parâmetros já devem ter sido validados.
func (s *Session) PreviaCapacidade(param ParametrosAlocacao) (CapacidadeResponse, error) {
	d, err := s.carregarDados()
	if err != nil {
		return CapacidadeResponse{}, err
	}
	p := montarProblema(param, d.horarios, d.avals, d.prefs, d.hard, d.soft)

	r := CapacidadeResponse{
		Parametros:           param,
		Candidatos:           p.nCand,
		Avaliadores:          p.nAval,
		MesasPorHorario:      p.nMesas,
		CapacidadePorHorario: p.nMesas * p.maxPessoas,
		CapacidadeTotal:      p.nSlot * p.nMesas * p.maxPessoas,
		MaxAlocaveis:         p.maxAlocaveis(),
		Horarios:             make([]HorarioCapacidade, p.nSlot),
		Avisos:               []string{},
	}
	for s := range r.Horarios {
		r.Horarios[s].Descricao = p.slotDesc[s]
	}
	for _, ps := range p.prefs {
		for nivel, s := range ps {
			r.Horarios[s].Interessados++
			if nivel == 0 {
				r.Horarios[s].PrimeiraOpcao++
			}
		}
	}

	sort.SliceStable(r.Horarios, func(i, j int) bool {
		return ordemHorario(r.Horarios[i].Descricao) < ordemHorario(r.Horarios[j].Descricao)
	})

	switch {
	case p.nMesas == 0:
		r.Avisos = append(r.Avisos, fmt.Sprintf("Com %d avaliadores não dá para formar nenhuma mesa de %d avaliadores.", p.nAval, p.avPorMesa))
	case p.nMesas < param.MesasPorHorario:
		r.Avisos = append(r.Avisos, fmt.Sprintf("Com %d avaliadores cabem só %d mesas por horário (cada avaliador fica em uma mesa por horário).", p.nAval, p.nMesas))
	}
	if p.nMesas > 0 && r.MaxAlocaveis < p.nCand {
		r.Avisos = append(r.Avisos, fmt.Sprintf("Pelos horários que escolheram, no máximo %d dos %d candidatos cabem nas mesas.", r.MaxAlocaveis, p.nCand))
	}
	for _, h := range r.Horarios {
		if h.Interessados > 0 && h.Interessados < p.minPessoas {
			r.Avisos = append(r.Avisos, fmt.Sprintf("%s: só %d candidato(s) escolheram este horário, menos que o mínimo de %d por mesa.", h.Descricao, h.Interessados, p.minPessoas))
		}
	}

	r.ValoresColunas = valoresColunas(d.atributos)
	r.Avisos = append(r.Avisos, avisosCriterios(param, r.ValoresColunas, p)...)
	return r, nil
}

// valoresColunas agrupa os valores de curso e semestre (sem diferenciar caixa
// e acento), com a grafia da primeira ocorrência.
func valoresColunas(atributos map[int]map[string]string) map[string][]ValorColuna {
	res := map[string][]ValorColuna{}
	for _, coluna := range []string{COLUNA_CURSO, COLUNA_SEMESTRE} {
		indice := map[string]int{}
		vals := []ValorColuna{}
		ids := make([]int, 0, len(atributos))
		for id := range atributos {
			ids = append(ids, id)
		}
		sort.Ints(ids) // grafia estável: a do candidato de menor id
		for _, id := range ids {
			v := strings.TrimSpace(atributos[id][coluna])
			k := chaveValor(v)
			if k == "" {
				continue
			}
			if i, ok := indice[k]; ok {
				vals[i].Quantidade++
				continue
			}
			indice[k] = len(vals)
			vals = append(vals, ValorColuna{Valor: v, Quantidade: 1})
		}
		sort.Slice(vals, func(i, j int) bool {
			a, errA := strconv.Atoi(vals[i].Valor)
			b, errB := strconv.Atoi(vals[j].Valor)
			if errA == nil && errB == nil {
				return a < b
			}
			return menorNome(vals[i].Valor, vals[j].Valor)
		})
		res[coluna] = vals
	}
	return res
}

// avisosCriterios aponta critérios que citam valores inexistentes ou que não
// têm como ser atendidos em todas as mesas.
func avisosCriterios(param ParametrosAlocacao, valores map[string][]ValorColuna, p *problema) []string {
	var avisos []string
	minMesas := 0 // mesas necessárias para caber todos, no mínimo
	if p.maxPessoas > 0 {
		minMesas = (p.nCand + p.maxPessoas - 1) / p.maxPessoas
	}
	for _, c := range param.Criterios {
		qtd := map[string]ValorColuna{}
		for _, v := range valores[c.Coluna] {
			qtd[chaveValor(v.Valor)] = v
		}
		coluna := strings.ToLower(nomeColuna[c.Coluna])
		for _, v := range c.Valores {
			vc, ok := qtd[chaveValor(v)]
			switch {
			case !ok:
				avisos = append(avisos, fmt.Sprintf("Nenhum candidato tem %s %q.", coluna, v))
			case c.Tipo == CRITERIO_UM_DE_CADA && vc.Quantidade < minMesas:
				avisos = append(avisos, fmt.Sprintf("Pelo menos um de cada: só %d candidato(s) têm %s %q, e serão pelo menos %d mesas; nem todas terão um.", vc.Quantidade, coluna, vc.Valor, minMesas))
			}
		}
	}
	return avisos
}

var diasDaSemana = map[string]int{
	"segunda": 1, "terca": 2, "quarta": 3, "quinta": 4, "sexta": 5, "sabado": 6, "domingo": 7,
}

// ordemHorario ordena descrições como "terça 10-12" por dia da semana e depois
// pela primeira hora; o que não reconhece vai para o fim, em ordem alfabética.
func ordemHorario(desc string) string {
	dia, hora := 9, 99
	// não usa tokensNome: ele converteria "segunda", "quarta"... em ordinais
	palavras := strings.FieldsFunc(strings.ToLower(semAcento.Replace(desc)), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for _, t := range palavras {
		if d, ok := diasDaSemana[t]; ok && dia == 9 {
			dia = d
		} else if n, err := strconv.Atoi(t); err == nil && hora == 99 {
			hora = n
		}
	}
	return fmt.Sprintf("%d %02d %s", dia, hora, desc)
}
