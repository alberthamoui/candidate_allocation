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
	return r, nil
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
