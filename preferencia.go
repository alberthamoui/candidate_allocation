package main

import (
	"sort"
	"strings"
)

// normalizarHorarios limpa a lista digitada pelo administrador: tira espaços,
// aceita vírgula, ponto e vírgula ou quebra de linha, e remove repetidos.
func normalizarHorarios(lista []string) []string {
	var out []string
	visto := map[string]bool{}
	for _, h := range lista {
		partes := strings.FieldsFunc(h, func(r rune) bool {
			return r == ',' || r == ';' || r == '\n' || r == '\r'
		})
		for _, parte := range partes {
			n := normalizaOpcao(parte)
			if n == "" || visto[n] {
				continue
			}
			visto[n] = true
			out = append(out, n)
		}
	}
	return out
}

// aplicarSemPreferencia troca os horários da instância pelos que o
// administrador definiu e marca todo candidato como disponível em todos eles.
// O custo de cada horário fica zero em montarProblema quando SemPreferencia.
func aplicarSemPreferencia(d dadosAlocacao, horarios []string) dadosAlocacao {
	horarios = normalizarHorarios(horarios)
	pessoas := make([]int, 0, len(d.atributos))
	for id := range d.atributos {
		pessoas = append(pessoas, id)
	}
	if len(pessoas) == 0 {
		for id := range d.prefs {
			pessoas = append(pessoas, id)
		}
	}
	sort.Ints(pessoas)

	novo := make(map[int]*Horario, len(horarios))
	ids := make([]int, 0, len(horarios))
	for i, desc := range horarios {
		id := i + 1
		ids = append(ids, id)
		novo[id] = &Horario{ID: id, Descricao: desc, Candidatos: append([]int{}, pessoas...)}
	}
	prefs := make(map[int][]int, len(pessoas))
	for _, pid := range pessoas {
		prefs[pid] = append([]int{}, ids...)
	}
	d.horarios = novo
	d.prefs = prefs
	return d
}

// comPreferenciaDaSessao liga SemPreferencia quando o administrador desligou
// o ranking na entrada. Os horários continuam vindo dos parâmetros.
func (s *Session) comPreferenciaDaSessao(param ParametrosAlocacao) ParametrosAlocacao {
	if s != nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
	}
	if s != nil && s.preferenciaDefinida && !s.usarPreferencia {
		param.SemPreferencia = true
	}
	return param
}
