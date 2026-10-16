package main

import (
	"fmt"
	"strings"
)

// ==================================================
// ========== HORÁRIOS SEM PREFERÊNCIA ==============
// ==================================================
//
// No modo sem preferência, os candidatos não escolhem horário: quem organiza
// digita os horários na tela inicial e todo candidato pode ir em qualquer um
// deles, com o mesmo custo. Só as restrições dos avaliadores (e os critérios
// adicionais) pesam na alocação.
//
// Para o resto do sistema continuar igual, ao salvar os candidatos cada um
// recebe todos esses horários como "opções" (SaveUsuarios). O que muda é que
// a ordem não vale nada: custo zero em montarProblema, nenhum ponto por opção
// em pontuarResultado e nada de "1ª opção" na prévia e no resultado.

// LIMITE_HORARIOS_LIVRES é o máximo de horários digitados.
const LIMITE_HORARIOS_LIVRES = 50

// lerHorariosLivres separa os horários digitados (um por linha, ou separados
// por vírgula ou ponto e vírgula), normaliza como as opções da planilha e
// tira os repetidos, mantendo a ordem.
func lerHorariosLivres(texto string) ([]string, error) {
	partes := strings.FieldsFunc(texto, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == ';'
	})
	horarios := []string{}
	visto := map[string]bool{}
	for _, p := range partes {
		h := normalizaOpcao(p)
		if h == "" || visto[h] {
			continue
		}
		visto[h] = true
		horarios = append(horarios, h)
	}
	switch {
	case len(horarios) == 0:
		return nil, fmt.Errorf("informe pelo menos um horário")
	case len(horarios) > LIMITE_HORARIOS_LIVRES:
		return nil, fmt.Errorf("no máximo %d horários (foram %d)", LIMITE_HORARIOS_LIVRES, len(horarios))
	}
	return horarios, nil
}

// semPreferencia diz se a sessão está no modo sem preferência de horário.
func (s *Session) semPreferencia() bool {
	return s.horariosLivres != nil
}
