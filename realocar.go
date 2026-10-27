package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ResultadoOperacao é o que a tela e a CLI recebem depois de realocar,
// avaliar ou disparar avisos.
type ResultadoOperacao struct {
	LoteID    string            `json:"lote_id,omitempty"`
	Envio     *LoteAvisos       `json:"envio,omitempty"`
	Resumo    string            `json:"resumo"`
	Avisos    []AvisoGerado     `json:"avisos"`
	Resultado *AlocacaoResponse `json:"resultado,omitempty"`
}

func (s *Session) exigirResultado() error {
	if s == nil || s.lastResult == nil {
		return fmt.Errorf("nenhuma alocação disponível")
	}
	return nil
}

// RealocarDia move todas as mesas de um horário para outro. Se avisar for
// verdadeiro, monta (e envia, se estiver configurado) a mensagem de cada pessoa.
func (s *Session) realocarDiaLocked(de, para string, avisar bool) (ResultadoOperacao, error) {
	if err := s.exigirResultado(); err != nil {
		return ResultadoOperacao{}, err
	}
	deN, paraN := normalizaOpcao(de), normalizaOpcao(para)
	if deN == "" || paraN == "" {
		return ResultadoOperacao{}, fmt.Errorf("informe o horário de origem e o novo horário")
	}
	if deN == paraN {
		return ResultadoOperacao{}, fmt.Errorf("o horário novo é igual ao atual")
	}
	var ids []int
	n := 0
	for i := range s.lastResult.Mesas {
		atual := normalizaOpcao(s.lastResult.Mesas[i].DiaNome)
		data, hora := dividirDataHorario(atual)
		if atual != deN && data != deN {
			continue
		}
		destino := paraN
		if atual != deN {
			_, horaDestino := dividirDataHorario(paraN)
			if horaDestino != "" {
				return ResultadoOperacao{}, fmt.Errorf("para mover o dia inteiro, informe somente o dia de destino; os intervalos serão preservados")
			}
			destino = strings.TrimSpace(paraN + " " + hora)
		}
		s.lastResult.Mesas[i].DiaNome = destino
		s.lastResult.Mesas[i].Descricao = trocarPrefixoHorario(s.lastResult.Mesas[i].Descricao, destino)
		n++
		for _, c := range s.lastResult.Mesas[i].Candidatos {
			ids = append(ids, c.ID)
		}
	}
	if n == 0 {
		return ResultadoOperacao{}, fmt.Errorf("nenhuma mesa no horário %q", deN)
	}
	return s.finalizar(ids, avisar, fmt.Sprintf("%d mesa(s) de %s passaram para %s.", n, deN, paraN)), nil
}

// RealocarMesa move uma mesa inteira para outro horário.
func (s *Session) realocarMesaLocked(mesaID int, para string, avisar bool) (ResultadoOperacao, error) {
	if err := s.exigirResultado(); err != nil {
		return ResultadoOperacao{}, err
	}
	paraN := normalizaOpcao(para)
	if paraN == "" {
		return ResultadoOperacao{}, fmt.Errorf("informe o novo horário")
	}
	i := indiceMesa(s.lastResult, mesaID)
	if i < 0 {
		return ResultadoOperacao{}, fmt.Errorf("mesa %d não encontrada", mesaID)
	}
	antes := s.lastResult.Mesas[i].DiaNome
	s.lastResult.Mesas[i].DiaNome = paraN
	s.lastResult.Mesas[i].Descricao = trocarPrefixoHorario(s.lastResult.Mesas[i].Descricao, paraN)
	var ids []int
	for _, c := range s.lastResult.Mesas[i].Candidatos {
		ids = append(ids, c.ID)
	}
	return s.finalizar(ids, avisar, fmt.Sprintf("%s saiu de %s e foi para %s.", s.lastResult.Mesas[i].Descricao, antes, paraN)), nil
}

// RealocarPessoa tira uma pessoa da mesa atual e põe no horário pedido.
// mesaID, se for diferente de zero, escolhe a mesa de destino. Senão, entra
// na mesa daquele horário com menos gente, ou numa mesa nova se o horário
// ainda não existir.
func (s *Session) realocarPessoaLocked(busca, horario string, mesaID int, avisar bool) (ResultadoOperacao, error) {
	if err := s.exigirResultado(); err != nil {
		return ResultadoOperacao{}, err
	}
	cand, mesaIdx, ok, err := s.localizarCandidato(busca)
	if err != nil {
		return ResultadoOperacao{}, err
	}
	if !ok {
		return ResultadoOperacao{}, fmt.Errorf("candidato %q não encontrado", strings.TrimSpace(busca))
	}
	horario = normalizaOpcao(horario)
	if horario == "" && mesaID == 0 {
		return ResultadoOperacao{}, fmt.Errorf("informe o novo horário ou a mesa de destino")
	}

	dest := -1
	if mesaID != 0 {
		dest = indiceMesa(s.lastResult, mesaID)
		if dest < 0 {
			return ResultadoOperacao{}, fmt.Errorf("mesa %d não encontrada", mesaID)
		}
	} else {
		dest = mesaComMenosGente(s.lastResult, horario)
		if dest < 0 {
			s.lastResult.Mesas = append(s.lastResult.Mesas, MesaResult{
				ID:          proximoIDMesa(s.lastResult),
				DiaNome:     horario,
				Descricao:   horario + " - mesa 1",
				Candidatos:  []CandidatoResultado{},
				Avaliadores: []AvaliadorResultado{},
			})
			dest = len(s.lastResult.Mesas) - 1
		}
	}
	if mesaIdx == dest {
		return s.finalizar(nil, false, fmt.Sprintf("%s já está nesse horário.", cand.Nome)), nil
	}

	if mesaIdx >= 0 {
		s.lastResult.Mesas[mesaIdx].Candidatos = removerCandidato(s.lastResult.Mesas[mesaIdx].Candidatos, cand.ID)
	} else {
		s.lastResult.NaoAlocadosInfo = removerCandidato(s.lastResult.NaoAlocadosInfo, cand.ID)
		s.lastResult.TotalAlocados++
	}
	s.lastResult.Mesas[dest].Candidatos = append(s.lastResult.Mesas[dest].Candidatos, cand)
	sort.Slice(s.lastResult.Mesas[dest].Candidatos, func(i, j int) bool {
		return menorNome(s.lastResult.Mesas[dest].Candidatos[i].Nome, s.lastResult.Mesas[dest].Candidatos[j].Nome)
	})
	// mesa sem candidatos e sem avaliadores não aparece; mesa de entrevista
	// que ficou vazia de gente continua, com os avaliadores dela
	if mesaIdx >= 0 && mesaIdx != dest &&
		len(s.lastResult.Mesas[mesaIdx].Candidatos) == 0 &&
		len(s.lastResult.Mesas[mesaIdx].Avaliadores) == 0 {
		s.lastResult.Mesas = append(s.lastResult.Mesas[:mesaIdx], s.lastResult.Mesas[mesaIdx+1:]...)
	}
	onde := horario
	if mesaID != 0 {
		if i := indiceMesa(s.lastResult, mesaID); i >= 0 {
			onde = s.lastResult.Mesas[i].DiaNome
		}
	}
	return s.finalizar([]int{cand.ID}, avisar, fmt.Sprintf("%s foi para %s.", cand.Nome, onde)), nil
}

// Avaliar marca quem passou e quem não passou. resultado vazio volta para pendente.
func (s *Session) avaliarLocked(busca string, id int, resultado string) (ResultadoOperacao, error) {
	if err := s.exigirResultado(); err != nil {
		return ResultadoOperacao{}, err
	}
	res, err := normalizarResultadoAvaliacao(resultado)
	if err != nil {
		return ResultadoOperacao{}, err
	}
	var cand CandidatoResultado
	var ok bool
	if id != 0 {
		cand, _, ok, err = s.localizarCandidato(strconv.Itoa(id))
	} else {
		cand, _, ok, err = s.localizarCandidato(busca)
	}
	if err != nil {
		return ResultadoOperacao{}, err
	}
	if !ok {
		quem := busca
		if id != 0 {
			quem = strconv.Itoa(id)
		}
		return ResultadoOperacao{}, fmt.Errorf("candidato %q não encontrado", quem)
	}
	s.definirAvaliacao(cand.ID, res)
	texto := "pendente"
	if res != "" {
		texto = textoAvaliacao(res)
	}
	return s.finalizar(nil, false, fmt.Sprintf("%s: %s.", cand.Nome, texto)), nil
}

// EnviarAvisosTodos manda a mensagem configurada para cada pessoa alocada.
// O administrador precisa ter aceitado (o botão ou a flag da CLI).
func (s *Session) enviarAvisosTodosLocked() (ResultadoOperacao, error) {
	if err := s.exigirResultado(); err != nil {
		return ResultadoOperacao{}, err
	}
	var ids []int
	for _, m := range s.lastResult.Mesas {
		for _, c := range m.Candidatos {
			ids = append(ids, c.ID)
		}
	}
	if len(ids) == 0 {
		return ResultadoOperacao{}, fmt.Errorf("não há candidatos alocados para avisar")
	}
	op := s.finalizar(ids, true, fmt.Sprintf("Avisos preparados para %d candidato(s).", len(ids)))
	return op, nil
}

func (s *Session) finalizar(ids []int, avisar bool, resumo string) ResultadoOperacao {
	s.atualizarAgendaLocked()
	op := ResultadoOperacao{Resumo: resumo, Avisos: []AvisoGerado{}, Resultado: clonarResultado(s.lastResult)}
	if avisar && len(ids) > 0 {
		lote := s.prepararLoteLocked(ids)
		s.novoLote = &lote
		op.LoteID = lote.ID
		op.Envio = &lote
	}
	return op
}

func (s *Session) aplicarAvaliacoes(r *AlocacaoResponse) {
	if r == nil || len(s.avaliacoes) == 0 {
		return
	}
	por := func(c *CandidatoResultado) {
		if v, ok := s.avaliacoes[c.ID]; ok {
			c.Avaliacao = v
		}
	}
	for i := range r.Mesas {
		for j := range r.Mesas[i].Candidatos {
			por(&r.Mesas[i].Candidatos[j])
		}
	}
	for i := range r.NaoAlocadosInfo {
		por(&r.NaoAlocadosInfo[i])
	}
}

func (s *Session) definirAvaliacao(id int, resultado string) {
	if s.avaliacoes == nil {
		s.avaliacoes = map[int]string{}
	}
	if resultado == "" {
		delete(s.avaliacoes, id)
	} else {
		s.avaliacoes[id] = resultado
	}
	if s.lastResult == nil {
		return
	}
	por := func(c *CandidatoResultado) {
		if c.ID == id {
			c.Avaliacao = resultado
		}
	}
	for i := range s.lastResult.Mesas {
		for j := range s.lastResult.Mesas[i].Candidatos {
			por(&s.lastResult.Mesas[i].Candidatos[j])
		}
	}
	for i := range s.lastResult.NaoAlocadosInfo {
		por(&s.lastResult.NaoAlocadosInfo[i])
	}
}

func normalizarResultadoAvaliacao(s string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "aprovado", "passou", "sim":
		return "aprovado", nil
	case "reprovado", "não passou", "nao passou", "não", "nao":
		return "reprovado", nil
	case "", "pendente", "limpar":
		return "", nil
	default:
		return "", fmt.Errorf("resultado %q inválido: use aprovado ou reprovado", s)
	}
}

// localizarCandidato acha a pessoa pelo id ou pelo nome. mesaIdx é -1 se ela
// está na lista de não alocados.
func (s *Session) localizarCandidato(busca string) (CandidatoResultado, int, bool, error) {
	busca = strings.TrimSpace(busca)
	if busca == "" {
		return CandidatoResultado{}, 0, false, fmt.Errorf("informe o candidato")
	}
	id, errID := strconv.Atoi(busca)
	type achado struct {
		c    CandidatoResultado
		mesa int
	}
	var exatos []achado
	var parciais []achado
	considerar := func(c CandidatoResultado, mesa int) {
		if errID == nil && c.ID == id {
			exatos = append(exatos, achado{c, mesa})
			return
		}
		if strings.EqualFold(strings.TrimSpace(c.Nome), busca) {
			exatos = append(exatos, achado{c, mesa})
			return
		}
		if strings.Contains(strings.ToLower(c.Nome), strings.ToLower(busca)) {
			parciais = append(parciais, achado{c, mesa})
		}
	}
	for i, m := range s.lastResult.Mesas {
		for _, c := range m.Candidatos {
			considerar(c, i)
		}
	}
	for _, c := range s.lastResult.NaoAlocadosInfo {
		considerar(c, -1)
	}
	lista := exatos
	if len(lista) == 0 {
		lista = parciais
	}
	switch len(lista) {
	case 0:
		return CandidatoResultado{}, 0, false, nil
	case 1:
		return lista[0].c, lista[0].mesa, true, nil
	default:
		return CandidatoResultado{}, 0, false, fmt.Errorf("há mais de um candidato para %q; use o id", busca)
	}
}

func indiceMesa(r *AlocacaoResponse, id int) int {
	for i, m := range r.Mesas {
		if m.ID == id {
			return i
		}
	}
	return -1
}

func mesaComMenosGente(r *AlocacaoResponse, horario string) int {
	melhor, n := -1, 0
	for i, m := range r.Mesas {
		if normalizaOpcao(m.DiaNome) != horario {
			continue
		}
		if melhor < 0 || len(m.Candidatos) < n {
			melhor, n = i, len(m.Candidatos)
		}
	}
	return melhor
}

func proximoIDMesa(r *AlocacaoResponse) int {
	maxID := 0
	for _, m := range r.Mesas {
		if m.ID > maxID {
			maxID = m.ID
		}
	}
	return maxID + 1
}

func removerCandidato(lista []CandidatoResultado, id int) []CandidatoResultado {
	out := lista[:0]
	for _, c := range lista {
		if c.ID != id {
			out = append(out, c)
		}
	}
	return out
}

func trocarPrefixoHorario(desc, horario string) string {
	if i := strings.LastIndex(desc, " - mesa"); i >= 0 {
		return horario + desc[i:]
	}
	return horario
}

func ordenarMesasResultado(r *AlocacaoResponse) {
	if r == nil {
		return
	}
	sort.SliceStable(r.Mesas, func(i, j int) bool {
		a, b := r.Mesas[i], r.Mesas[j]
		if oa, ob := ordemHorario(a.DiaNome), ordemHorario(b.DiaNome); oa != ob {
			return oa < ob
		}
		return a.ID < b.ID
	})
}

func (s *Session) RealocarDia(de, para string, avisar bool) (ResultadoOperacao, error) {
	return s.operar(func() (ResultadoOperacao, error) { return s.realocarDiaLocked(de, para, avisar) }, false)
}
func (s *Session) RealocarMesa(id int, para string, avisar bool) (ResultadoOperacao, error) {
	return s.operar(func() (ResultadoOperacao, error) { return s.realocarMesaLocked(id, para, avisar) }, false)
}
func (s *Session) RealocarPessoa(busca, horario string, id int, avisar bool) (ResultadoOperacao, error) {
	return s.operar(func() (ResultadoOperacao, error) { return s.realocarPessoaLocked(busca, horario, id, avisar) }, false)
}
func (s *Session) Avaliar(busca string, id int, res string) (ResultadoOperacao, error) {
	return s.operar(func() (ResultadoOperacao, error) { return s.avaliarLocked(busca, id, res) }, false)
}
func (s *Session) EnviarAvisosTodos() (ResultadoOperacao, error) {
	return s.operar(s.enviarAvisosTodosLocked, false)
}
func (s *Session) RealocarAsync(p pedidoRealocar) (ResultadoOperacao, error) {
	return s.operar(func() (ResultadoOperacao, error) {
		switch p.Tipo {
		case "dia":
			return s.realocarDiaLocked(p.De, p.Para, p.Avisar)
		case "mesa":
			return s.realocarMesaLocked(p.MesaID, p.Para, p.Avisar)
		case "pessoa":
			if p.Horario == "" {
				p.Horario = p.Para
			}
			return s.realocarPessoaLocked(p.Pessoa, p.Horario, p.MesaID, p.Avisar)
		default:
			return ResultadoOperacao{}, fmt.Errorf("tipo deve ser dia, mesa ou pessoa")
		}
	}, true)
}

type pedidoEnvio struct {
	Pessoa string `json:"pessoa,omitempty"`
	MesaID int    `json:"mesa_id,omitempty"`
}

func (s *Session) EnviarAvisos(p pedidoEnvio, async bool) (ResultadoOperacao, error) {
	return s.operar(func() (ResultadoOperacao, error) {
		if p.Pessoa == "" && p.MesaID == 0 {
			return s.enviarAvisosTodosLocked()
		}
		if err := s.exigirResultado(); err != nil {
			return ResultadoOperacao{}, err
		}
		var ids []int
		if p.Pessoa != "" {
			c, i, ok, err := s.localizarCandidato(p.Pessoa)
			if err != nil {
				return ResultadoOperacao{}, err
			}
			if !ok || i < 0 {
				return ResultadoOperacao{}, fmt.Errorf("pessoa sem alocação")
			}
			ids = append(ids, c.ID)
		} else {
			i := indiceMesa(s.lastResult, p.MesaID)
			if i < 0 {
				return ResultadoOperacao{}, fmt.Errorf("mesa não encontrada")
			}
			for _, c := range s.lastResult.Mesas[i].Candidatos {
				ids = append(ids, c.ID)
			}
		}
		if len(ids) == 0 {
			return ResultadoOperacao{}, fmt.Errorf("nenhuma pessoa alocada para avisar")
		}
		return s.finalizar(ids, true, "Avisos preparados para os participantes selecionados."), nil
	}, async)
}
