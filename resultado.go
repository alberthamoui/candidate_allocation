package main

import (
	"fmt"
	"sort"
	"strings"
)

// ==================================================
// ============ RESULTADO PARA A TELA ===============
// ==================================================
//
// Depois da alocação, junta tudo o que a tela de resultado mostra: as mesas
// com os dados de cada pessoa, os não alocados e um relatório de qualidade em
// que cada item traz os ids dos candidatos envolvidos, para a tela destacá-los.

// CandidatoResultado é um candidato como aparece no resultado.
type CandidatoResultado struct {
	ID          int      `json:"id"`
	Nome        string   `json:"nome"`
	EmailInsper string   `json:"email_insper"`
	Curso       string   `json:"curso"`
	Semestre    int      `json:"semestre"`
	Opcoes      []string `json:"opcoes"`      // horários na ordem de preferência
	Opcao       int      `json:"opcao"`       // em qual opção ficou (1 = 1ª); 0 se não alocado
	NaoPosso    []string `json:"nao_posso"`   // avaliadores que não podem avaliá-lo
	PrefiroNao  []string `json:"prefiro_nao"` // avaliadores que preferem não avaliá-lo
	// Conflitos são os avaliadores "prefiro não" que ficaram na mesa dele.
	Conflitos []string `json:"conflitos"`
}

// AvaliadorResultado é um avaliador como aparece no resultado.
type AvaliadorResultado struct {
	ID         int      `json:"id"`
	Nome       string   `json:"nome"`
	Email      string   `json:"email"`
	Sigla      string   `json:"sigla"`
	NaoPosso   []string `json:"nao_posso"`   // candidatos que ele não pode avaliar
	PrefiroNao []string `json:"prefiro_nao"` // candidatos que prefere não avaliar
}

// ItemQualidade é uma característica do resultado; clicar nela na tela
// destaca os candidatos listados.
type ItemQualidade struct {
	Codigo     string `json:"codigo"`
	Titulo     string `json:"titulo"`
	Descricao  string `json:"descricao"`
	Valor      int    `json:"valor"`
	Tom        string `json:"tom"` // "bom", "neutro", "atencao" ou "ruim"
	Candidatos []int  `json:"candidatos"`
}

type candidatoDB struct {
	ID          int
	Nome        string
	EmailInsper string
	Curso       string
	Semestre    int
}

func carregarCandidatos(s *Session) ([]candidatoDB, error) {
	rows, err := s.db.Query(`SELECT id, nome, email_insper, curso, semestre FROM pessoa`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cands []candidatoDB
	for rows.Next() {
		var c candidatoDB
		if err := rows.Scan(&c.ID, &c.Nome, &c.EmailInsper, &c.Curso, &c.Semestre); err != nil {
			return nil, err
		}
		cands = append(cands, c)
	}
	return cands, rows.Err()
}

// montarResultado converte a alocação no que a tela e a exportação usam.
func montarResultado(d dadosAlocacao, cands []candidatoDB, res ResultadoAlocacao, mesas []*Mesa, criterios []CriterioAlocacao) AlocacaoResponse {
	avalPorID := make(map[int]*Avaliador, len(d.avals))
	for _, a := range d.avals {
		avalPorID[a.ID] = a
	}
	nomeCand := make(map[int]string, len(cands))
	for _, c := range cands {
		nomeCand[c.ID] = c.Nome
	}
	nomesAval := func(ids []int) []string {
		nomes := make([]string, 0, len(ids))
		for _, id := range ids {
			if a := avalPorID[id]; a != nil {
				nomes = append(nomes, a.Nome)
			}
		}
		sort.Strings(nomes)
		return nomes
	}
	// avaliadores com restrição a cada candidato
	naoPosso, prefiroNao := map[int][]int{}, map[int][]int{}
	for aid, cs := range d.hard {
		for cid := range cs {
			naoPosso[cid] = append(naoPosso[cid], aid)
		}
	}
	for aid, cs := range d.soft {
		for cid := range cs {
			prefiroNao[cid] = append(prefiroNao[cid], aid)
		}
	}

	mesaDe := make(map[int]*Mesa, len(res.Alocacao))
	for _, m := range mesas {
		for _, pid := range m.Candidatos {
			mesaDe[pid] = m
		}
	}

	candidato := func(c candidatoDB) CandidatoResultado {
		r := CandidatoResultado{
			ID: c.ID, Nome: c.Nome, EmailInsper: c.EmailInsper, Curso: c.Curso, Semestre: c.Semestre,
			Opcoes:     []string{},
			NaoPosso:   nomesAval(naoPosso[c.ID]),
			PrefiroNao: nomesAval(prefiroNao[c.ID]),
			Conflitos:  []string{},
		}
		m := mesaDe[c.ID]
		for i, hid := range d.prefs[c.ID] {
			if h := d.horarios[hid]; h != nil {
				r.Opcoes = append(r.Opcoes, h.Descricao)
			}
			if m != nil && hid == m.DiaID && r.Opcao == 0 {
				r.Opcao = i + 1
			}
		}
		if m != nil {
			var conflitos []int
			for _, aid := range m.Avaliadores {
				if d.soft[aid][c.ID] {
					conflitos = append(conflitos, aid)
				}
			}
			r.Conflitos = nomesAval(conflitos)
		}
		return r
	}
	avaliador := func(aid int) AvaliadorResultado {
		a := avalPorID[aid]
		r := AvaliadorResultado{ID: aid, Nome: fmt.Sprintf("ID %d", aid), NaoPosso: []string{}, PrefiroNao: []string{}}
		if a != nil {
			r.Nome, r.Email, r.Sigla = a.Nome, a.Email, a.Sigla
		}
		for cid := range d.hard[aid] {
			r.NaoPosso = append(r.NaoPosso, nomeCand[cid])
		}
		for cid := range d.soft[aid] {
			r.PrefiroNao = append(r.PrefiroNao, nomeCand[cid])
		}
		sort.Strings(r.NaoPosso)
		sort.Strings(r.PrefiroNao)
		return r
	}

	candPorID := make(map[int]candidatoDB, len(cands))
	for _, c := range cands {
		candPorID[c.ID] = c
	}
	var resultado AlocacaoResponse
	for _, m := range mesas {
		if len(m.Candidatos) == 0 {
			continue
		}
		mr := MesaResult{ID: m.ID, DiaID: m.DiaID, Descricao: m.Descricao}
		if h := d.horarios[m.DiaID]; h != nil {
			mr.DiaNome = h.Descricao
		}
		for _, pid := range m.Candidatos {
			c, ok := candPorID[pid]
			if !ok {
				c = candidatoDB{ID: pid, Nome: fmt.Sprintf("ID %d", pid)}
			}
			mr.Candidatos = append(mr.Candidatos, candidato(c))
		}
		sort.Slice(mr.Candidatos, func(i, j int) bool { return menorNome(mr.Candidatos[i].Nome, mr.Candidatos[j].Nome) })
		for _, aid := range m.Avaliadores {
			mr.Avaliadores = append(mr.Avaliadores, avaliador(aid))
		}
		sort.Slice(mr.Avaliadores, func(i, j int) bool { return menorNome(mr.Avaliadores[i].Nome, mr.Avaliadores[j].Nome) })
		resultado.Mesas = append(resultado.Mesas, mr)
	}
	// mesas por horário (dia da semana e hora) e, no horário, pelo número
	sort.SliceStable(resultado.Mesas, func(i, j int) bool {
		a, b := resultado.Mesas[i], resultado.Mesas[j]
		if oa, ob := ordemHorario(a.DiaNome), ordemHorario(b.DiaNome); oa != ob {
			return oa < ob
		}
		return a.ID < b.ID
	})

	for _, c := range cands {
		if mesaDe[c.ID] == nil {
			resultado.NaoAlocadosInfo = append(resultado.NaoAlocadosInfo, candidato(c))
		}
	}
	sort.Slice(resultado.NaoAlocadosInfo, func(i, j int) bool {
		return menorNome(resultado.NaoAlocadosInfo[i].Nome, resultado.NaoAlocadosInfo[j].Nome)
	})

	resultado.TotalAlocados = res.Alocados
	resultado.Pontuacao = res.Pontuacao
	resultado.Qualidade = relatorioQualidade(resultado, criterios)
	return resultado
}

// relatorioQualidade resume o resultado em itens clicáveis: quantos ficaram
// em cada opção de horário, quantos têm avaliador "prefiro não" na mesa e
// quantos ficaram sem mesa.
func relatorioQualidade(r AlocacaoResponse, criterios []CriterioAlocacao) []ItemQualidade {
	porOpcao := map[int][]int{}
	var conflito []int
	maiorOpcao := 1
	for _, m := range r.Mesas {
		for _, c := range m.Candidatos {
			porOpcao[c.Opcao] = append(porOpcao[c.Opcao], c.ID)
			maiorOpcao = max(maiorOpcao, c.Opcao)
			if len(c.Conflitos) > 0 {
				conflito = append(conflito, c.ID)
			}
		}
	}

	var itens []ItemQualidade
	for k := 1; k <= maiorOpcao; k++ {
		ids := porOpcao[k]
		if k > 1 && len(ids) == 0 {
			continue
		}
		tom := "bom"
		switch {
		case k == 2:
			tom = "neutro"
		case k >= 3:
			tom = "atencao"
		}
		itens = append(itens, ItemQualidade{
			Codigo:     fmt.Sprintf("opcao_%d", k),
			Titulo:     fmt.Sprintf("Na %dª opção", k),
			Descricao:  fmt.Sprintf("Candidatos alocados no horário que marcaram como %dª opção.", k),
			Valor:      len(ids),
			Tom:        tom,
			Candidatos: nuncaNil(ids),
		})
	}

	tomConflito := "bom"
	if len(conflito) > 0 {
		tomConflito = "atencao"
	}
	itens = append(itens, ItemQualidade{
		Codigo:     "prefiro_nao",
		Titulo:     "Com avaliador \"prefiro não\"",
		Descricao:  "Candidatos avaliados por alguém que marcou \"prefiro não\" para eles.",
		Valor:      len(conflito),
		Tom:        tomConflito,
		Candidatos: nuncaNil(conflito),
	})

	var semMesa []int
	for _, c := range r.NaoAlocadosInfo {
		semMesa = append(semMesa, c.ID)
	}
	tomSemMesa := "bom"
	if len(semMesa) > 0 {
		tomSemMesa = "ruim"
	}
	for i, c := range criterios {
		itens = append(itens, itemCriterio(r, i, c))
	}

	itens = append(itens, ItemQualidade{
		Codigo:     "nao_alocados",
		Titulo:     "Sem mesa",
		Descricao:  "Candidatos que não couberam em nenhuma mesa dos horários que escolheram.",
		Valor:      len(semMesa),
		Tom:        tomSemMesa,
		Candidatos: nuncaNil(semMesa),
	})
	return itens
}

// itemCriterio resume quantas mesas não atendem ao critério; clicar destaca
// os candidatos que causam o desvio.
func itemCriterio(r AlocacaoResponse, i int, c CriterioAlocacao) ItemQualidade {
	coluna := strings.ToLower(nomeColuna[c.Coluna])
	valorDe := func(cand CandidatoResultado) string {
		if c.Coluna == COLUNA_SEMESTRE {
			if cand.Semestre == 0 {
				return ""
			}
			return fmt.Sprint(cand.Semestre)
		}
		return chaveValor(cand.Curso)
	}
	alvo := map[string]bool{}
	for _, v := range c.Valores {
		alvo[chaveValor(v)] = true
	}
	conta := func(v string) bool { return len(alvo) == 0 || alvo[v] }
	listaValores := strings.Join(c.Valores, ", ")
	if listaValores == "" {
		listaValores = "qualquer " + coluna
	}

	var titulo, descricao string
	switch c.Tipo {
	case CRITERIO_MISTURAR:
		titulo = fmt.Sprintf("Mesas com %s repetido", coluna)
		descricao = fmt.Sprintf("Critério \"misturar %s\": mesas com dois ou mais candidatos do mesmo %s.", coluna, coluna)
	case CRITERIO_AGRUPAR:
		titulo = fmt.Sprintf("Mesas com mais de um %s", coluna)
		descricao = fmt.Sprintf("Critério \"agrupar %s\": mesas que misturam %ss diferentes.", coluna, coluna)
	case CRITERIO_MAXIMO:
		titulo = fmt.Sprintf("Mesas com mais de %d do mesmo %s", c.Limite, coluna)
		descricao = fmt.Sprintf("Critério \"no máximo %d por mesa\" (%s).", c.Limite, listaValores)
	case CRITERIO_MINIMO:
		titulo = fmt.Sprintf("Mesas com menos de %d do mesmo %s", c.Limite, coluna)
		descricao = fmt.Sprintf("Critério \"se aparecer, pelo menos %d por mesa\" (%s).", c.Limite, listaValores)
	case CRITERIO_UM_DE_CADA:
		titulo = fmt.Sprintf("Mesas sem todos os %ss escolhidos", coluna)
		descricao = fmt.Sprintf("Critério \"pelo menos um de cada\" (%s).", listaValores)
	}

	var mesasFora int
	var ids []int
	for _, m := range r.Mesas {
		porValor := map[string][]int{}
		for _, cand := range m.Candidatos {
			if v := valorDe(cand); v != "" {
				porValor[v] = append(porValor[v], cand.ID)
			}
		}
		var destaque []int
		switch c.Tipo {
		case CRITERIO_MISTURAR:
			for _, g := range porValor {
				if len(g) > 1 {
					destaque = append(destaque, g...)
				}
			}
		case CRITERIO_AGRUPAR:
			if len(porValor) > 1 {
				for _, g := range porValor {
					destaque = append(destaque, g...)
				}
			}
		case CRITERIO_MAXIMO:
			for v, g := range porValor {
				if conta(v) && len(g) > c.Limite {
					destaque = append(destaque, g...)
				}
			}
		case CRITERIO_MINIMO:
			for v, g := range porValor {
				if conta(v) && len(g) < c.Limite {
					destaque = append(destaque, g...)
				}
			}
		case CRITERIO_UM_DE_CADA:
			for v := range alvo {
				if len(porValor[v]) == 0 {
					for _, cand := range m.Candidatos {
						destaque = append(destaque, cand.ID)
					}
					break
				}
			}
		}
		if len(destaque) > 0 {
			mesasFora++
			ids = append(ids, destaque...)
		}
	}
	tom := "bom"
	if mesasFora > 0 {
		tom = "atencao"
	}
	sort.Ints(ids)
	return ItemQualidade{
		Codigo:     fmt.Sprintf("criterio_%d", i+1),
		Titulo:     titulo,
		Descricao:  descricao,
		Valor:      mesasFora,
		Tom:        tom,
		Candidatos: nuncaNil(ids),
	}
}

func nuncaNil(ids []int) []int {
	if ids == nil {
		return []int{}
	}
	return ids
}

// menorNome compara nomes sem diferenciar maiúsculas e acentos.
func menorNome(a, b string) bool {
	return strings.ToLower(semAcento.Replace(a)) < strings.ToLower(semAcento.Replace(b))
}
