package main

import (
	"fmt"
	"sort"
	"strings"
)

type SugestaoRealocacao struct {
	MesaID    int      `json:"mesa_id"`
	Descricao string   `json:"descricao"`
	Horario   string   `json:"horario"`
	Pontos    int      `json:"pontos"`
	Vagas     int      `json:"vagas"`
	Motivos   []string `json:"motivos"`
	Alertas   []string `json:"alertas"`
}

func contemNome(lista []string, nome string) bool {
	for _, s := range lista {
		if strings.EqualFold(s, nome) {
			return true
		}
	}
	return false
}
func (s *Session) Sugestoes(busca string) ([]SugestaoRealocacao, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.exigirResultado(); err != nil {
		return nil, err
	}
	c, origem, ok, err := s.localizarCandidato(busca)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("candidato não encontrado")
	}
	p := s.parametrosAtuais()
	out := []SugestaoRealocacao{}
	for i, m := range s.lastResult.Mesas {
		if i == origem {
			continue
		}
		sg := SugestaoRealocacao{MesaID: m.ID, Descricao: m.Descricao, Horario: m.DiaNome, Vagas: p.MaxPessoasPorMesa - len(m.Candidatos), Motivos: []string{}, Alertas: []string{}}
		if sg.Vagas > 0 {
			sg.Pontos += 20
			sg.Motivos = append(sg.Motivos, fmt.Sprintf("%d vaga(s) disponíveis", sg.Vagas))
		} else {
			sg.Pontos -= 100
			sg.Alertas = append(sg.Alertas, "Mesa no limite ou acima da capacidade")
		}
		if len(m.Avaliadores) >= p.AvaliadoresPorMesa {
			sg.Pontos += 30
			sg.Motivos = append(sg.Motivos, "Equipe de avaliadores completa")
		} else {
			sg.Pontos -= 200
			sg.Alertas = append(sg.Alertas, "Equipe de avaliadores incompleta")
		}
		for _, a := range m.Avaliadores {
			if contemNome(c.NaoPosso, a.Nome) {
				sg.Pontos -= 1000
				sg.Alertas = append(sg.Alertas, "Restrição não posso: "+a.Nome)
			}
			if contemNome(c.PrefiroNao, a.Nome) {
				sg.Pontos -= 25
				sg.Alertas = append(sg.Alertas, "Prefiro não: "+a.Nome)
			}
			for j, outra := range s.lastResult.Mesas {
				if j != i && outra.DiaNome == m.DiaNome {
					for _, b := range outra.Avaliadores {
						if a.ID == b.ID {
							sg.Pontos -= 100
							sg.Alertas = append(sg.Alertas, "Avaliador em outra mesa neste horário: "+a.Nome)
						}
					}
				}
			}
		}
		if !s.lastResult.SemPreferencia {
			opcao := 0
			for n, h := range c.Opcoes {
				if normalizaOpcao(h) == normalizaOpcao(m.DiaNome) {
					opcao = n + 1
					break
				}
			}
			if opcao > 0 {
				sg.Pontos += 50 - 5*opcao
				sg.Motivos = append(sg.Motivos, fmt.Sprintf("%dª preferência de horário", opcao))
			} else {
				sg.Pontos -= 20
				sg.Alertas = append(sg.Alertas, "Fora das preferências informadas")
			}
		}
		// Regras opcionais contribuem para ordenar sem proibir escolhas manuais.
		for _, crit := range p.Criterios {
			antes := itemCriterio(AlocacaoResponse{Mesas: []MesaResult{m}}, 0, crit).Valor
			m.Candidatos = append(append([]CandidatoResultado{}, m.Candidatos...), c)
			depois := itemCriterio(AlocacaoResponse{Mesas: []MesaResult{m}}, 0, crit).Valor
			m.Candidatos = m.Candidatos[:len(m.Candidatos)-1]
			if depois > antes {
				sg.Pontos -= crit.Peso * 3
				sg.Alertas = append(sg.Alertas, "Pode piorar critério de "+crit.Coluna)
			}
		}
		out = append(out, sg)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Pontos != out[j].Pontos {
			return out[i].Pontos > out[j].Pontos
		}
		if out[i].Vagas != out[j].Vagas {
			return out[i].Vagas > out[j].Vagas
		}
		return out[i].MesaID < out[j].MesaID
	})
	return out, nil
}

func (s *Session) atualizarAgendaLocked() {
	r := s.lastResult
	if r == nil {
		return
	}
	p := s.parametrosAtuais()
	dias := map[string]int{}
	next := 1
	r.TotalAlocados = 0
	r.Alertas = []string{}
	score := SCORE_BASE + PENALIDADE_NAO_ALOCADO*len(r.NaoAlocadosInfo)
	for i := range r.Mesas {
		m := &r.Mesas[i]
		if _, ok := dias[m.DiaNome]; !ok {
			dias[m.DiaNome] = next
			next++
		}
		m.DiaID = dias[m.DiaNome]
		r.TotalAlocados += len(m.Candidatos)
		if len(m.Candidatos) > p.MaxPessoasPorMesa {
			r.Alertas = append(r.Alertas, fmt.Sprintf("%s: %d candidatos, máximo %d", m.Descricao, len(m.Candidatos), p.MaxPessoasPorMesa))
		}
		if len(m.Candidatos) < p.MinPessoasPorMesa {
			r.Alertas = append(r.Alertas, m.Descricao+": abaixo do mínimo de candidatos")
		}
		if len(m.Avaliadores) < p.AvaliadoresPorMesa {
			r.Alertas = append(r.Alertas, m.Descricao+": equipe de avaliadores incompleta")
		}
		for j := range m.Candidatos {
			c := &m.Candidatos[j]
			c.Opcao = 0
			c.Conflitos = []string{}
			for n, h := range c.Opcoes {
				if normalizaOpcao(h) == normalizaOpcao(m.DiaNome) {
					c.Opcao = n + 1
					break
				}
			}
			if !r.SemPreferencia {
				if c.Opcao > 0 {
					score += pontosOpcao(c.Opcao - 1)
				} else {
					r.Alertas = append(r.Alertas, c.Nome+": horário fora das preferências")
				}
			}
			for _, a := range m.Avaliadores {
				if contemNome(c.NaoPosso, a.Nome) {
					r.Alertas = append(r.Alertas, c.Nome+": restrição não posso com "+a.Nome)
					score += PENALIDADE_HARD
				}
				if contemNome(c.PrefiroNao, a.Nome) {
					c.Conflitos = append(c.Conflitos, a.Nome)
					score -= 5
				}
			}
		}
	}
	agendas := map[string]map[int]bool{}
	for _, m := range r.Mesas {
		if agendas[m.DiaNome] == nil {
			agendas[m.DiaNome] = map[int]bool{}
		}
		for _, a := range m.Avaliadores {
			if agendas[m.DiaNome][a.ID] {
				r.Alertas = append(r.Alertas, a.Nome+": mais de uma mesa em "+m.DiaNome)
			}
			agendas[m.DiaNome][a.ID] = true
		}
	}
	for h := range dias {
		count := 0
		for _, m := range r.Mesas {
			if m.DiaNome == h {
				count++
			}
		}
		if count > p.MesasPorHorario {
			r.Alertas = append(r.Alertas, h+": acima do número de mesas configurado")
		}
	}
	for _, m := range r.Mesas {
		score -= custoCriteriosResultado(m, p.Criterios)
	}
	r.Pontuacao = score
	r.Qualidade = relatorioQualidade(*r, p.Criterios)
	if r.SemPreferencia {
		r.marcarSemPreferencia()
	}
	r.Editado = true
	ordenarMesasResultado(r)
}

func custoCriteriosResultado(m MesaResult, criterios []CriterioAlocacao) int {
	p := problema{nCand: len(m.Candidatos)}
	attrs := map[int]map[string]string{}
	ids := []int{}
	for i, c := range m.Candidatos {
		p.candID = append(p.candID, c.ID)
		attrs[c.ID] = map[string]string{"curso": c.Curso, "semestre": fmt.Sprint(c.Semestre)}
		ids = append(ids, i)
	}
	p.aplicarCriterios(criterios, attrs)
	return p.custoCriteriosMembros(ids)
}
