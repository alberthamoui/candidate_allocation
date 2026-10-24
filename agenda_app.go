package main

import (
	"candidate_alocator/back/allocation"
	"candidate_alocator/back/operations"
	types "candidate_alocator/back/type"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const currentAgenda = "current"
const agendaIdle = time.Minute * 15

type agendaSnapshot struct {
	Run    allocation.ConfiguredAllocationResult `json:"run"`
	Result UIAllocationResult                    `json:"result"`
}
type MoveRequest struct {
	Kind          string `json:"kind"`
	CandidateID   int    `json:"candidateId"`
	GroupID       int    `json:"groupId"`
	DestinationID int    `json:"destinationId"`
	Schedule      string `json:"schedule"`
	From          string `json:"from"`
	To            string `json:"to"`
	Notify        bool   `json:"notify"`
}
type MoveSuggestion struct {
	GroupID     int      `json:"groupId"`
	Description string   `json:"description"`
	Schedule    string   `json:"schedule"`
	Vacancies   int      `json:"vacancies"`
	Penalty     int      `json:"penalty"`
	Alerts      []string `json:"alerts"`
}
type AgendaOperation struct {
	Result  UIAllocationResult `json:"result"`
	BatchID int64              `json:"batchId"`
	Warning string             `json:"warning"`
}

func (a *App) store() (*operations.Store, error) {
	a.operationsMu.Lock()
	defer a.operationsMu.Unlock()
	if a.operationsStore != nil {
		return a.operationsStore, nil
	}
	path := a.operationsPath
	if path == "" {
		dir := os.Getenv("CANDIDATE_ALLOCATOR_DATA_DIR")
		if dir == "" {
			var err error
			dir, err = os.UserConfigDir()
			if err != nil {
				return nil, err
			}
			dir = filepath.Join(dir, "candidate-allocation")
		}
		path = filepath.Join(dir, "operations.db")
	}
	s, err := operations.New(path, agendaIdle, 8)
	if err != nil {
		return nil, err
	}
	a.operationsStore = s
	return s, nil
}

func (a *App) startOperations() error {
	s, err := a.store()
	if err != nil {
		return err
	}
	if err := s.RecoverInterrupted(); err != nil {
		return err
	}
	pending, err := s.PendingBatches(currentAgenda)
	if err != nil {
		return err
	}
	for _, id := range pending {
		go a.executeNoticeBatch(id)
	}
	a.allocationMu.Lock()
	a.restoreAgendaLocked()
	a.allocationMu.Unlock()
	a.operationsMu.Lock()
	a.operationsStop = make(chan struct{})
	stop := a.operationsStop
	a.operationsMu.Unlock()
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case now := <-ticker.C:
				a.evictIdleAgenda(now)
			case <-stop:
				return
			}
		}
	}()
	return nil
}

func (a *App) evictIdleAgenda(now time.Time) {
	a.allocationMu.Lock()
	defer a.allocationMu.Unlock()
	if !a.allocationRunning && !a.allocationAccess.IsZero() && now.Sub(a.allocationAccess) >= agendaIdle {
		a.allocationResult = UIAllocationResult{}
	}
}

func (a *App) restoreAgendaLocked() {
	if a.allocationRunning {
		return
	}
	s, err := a.store()
	if err != nil {
		a.allocationError = err.Error()
		return
	}
	data, err := s.Load(currentAgenda)
	if errors.Is(err, sql.ErrNoRows) {
		return
	}
	if err != nil {
		a.allocationError = err.Error()
		return
	}
	var snapshot agendaSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		a.allocationError = err.Error()
		return
	}
	a.allocationResult = snapshot.Result
	a.allocationHasResult = true
	a.allocationAccess = time.Now()
}

func (a *App) saveConfiguredAgenda(run allocation.ConfiguredAllocationResult, result UIAllocationResult) error {
	s, err := a.store()
	if err != nil {
		return err
	}
	data, err := json.Marshal(agendaSnapshot{run, result})
	if err != nil {
		return err
	}
	return s.Save(currentAgenda, data)
}

func (a *App) readAgenda() (agendaSnapshot, error) {
	s, err := a.store()
	if err != nil {
		return agendaSnapshot{}, err
	}
	data, err := s.Load(currentAgenda)
	if err != nil {
		return agendaSnapshot{}, fmt.Errorf("nenhuma agenda salva; execute uma alocação: %w", err)
	}
	var snap agendaSnapshot
	err = json.Unmarshal(data, &snap)
	return snap, err
}

func (a *App) GetAgenda() (UIAllocationResult, error) {
	a.allocationMu.Lock()
	defer a.allocationMu.Unlock()
	a.restoreAgendaLocked()
	if !a.allocationHasResult {
		return UIAllocationResult{}, errors.New("nenhuma agenda salva")
	}
	a.allocationAccess = time.Now()
	data, err := json.Marshal(a.allocationResult)
	if err != nil {
		return UIAllocationResult{}, err
	}
	var result UIAllocationResult
	err = json.Unmarshal(data, &result)
	return result, err
}

func (a *App) EvaluateCandidates(ids []int, value string) (UIAllocationResult, error) {
	a.allocationMu.Lock()
	defer a.allocationMu.Unlock()
	if a.allocationRunning {
		return UIAllocationResult{}, errors.New("pare a verificação antes de avaliar")
	}
	if value != "" && value != "aprovado" && value != "reprovado" {
		return UIAllocationResult{}, errors.New("avaliação inválida")
	}
	snap, err := a.readAgenda()
	if err != nil {
		return UIAllocationResult{}, err
	}
	wanted := map[int]bool{}
	for _, id := range ids {
		wanted[id] = true
	}
	for i := range snap.Result.Mesas {
		for j := range snap.Result.Mesas[i].Candidatos {
			c := &snap.Result.Mesas[i].Candidatos[j]
			if wanted[c.ID] {
				c.Avaliacao = value
				delete(wanted, c.ID)
			}
		}
	}
	for i := range snap.Result.NaoAlocados {
		c := &snap.Result.NaoAlocados[i]
		if wanted[c.ID] {
			c.Avaliacao = value
			delete(wanted, c.ID)
		}
	}
	if len(wanted) > 0 {
		return UIAllocationResult{}, errors.New("candidato não encontrado")
	}
	if err := a.saveConfiguredAgenda(snap.Run, snap.Result); err != nil {
		return UIAllocationResult{}, err
	}
	a.allocationResult = snap.Result
	a.allocationAccess = time.Now()
	return snap.Result, nil
}

func (a *App) GetMoveSuggestions(candidateID int) ([]MoveSuggestion, error) {
	snap, err := a.readAgenda()
	if err != nil {
		return nil, err
	}
	out := []MoveSuggestion{}
	source := snap.Run.Result.Assignments[candidateID]
	found := false
	for _, c := range snap.Run.Problem.Candidates {
		if c.ID == candidateID {
			found = true
		}
	}
	if !found {
		return nil, errors.New("candidato não encontrado")
	}
	for _, group := range snap.Run.Problem.Groups {
		if group.ID == source {
			continue
		}
		assignments := map[int]int{}
		occupied := 0
		for c, g := range snap.Run.Result.Assignments {
			assignments[c] = g
			if g == group.ID {
				occupied++
			}
		}
		assignments[candidateID] = group.ID
		review := allocation.ReviewManualAllocation(snap.Run.Problem, assignments, snap.Run.Config.Normalized.Params.AvaliadoresPorGrupo)
		s := MoveSuggestion{GroupID: group.ID, Description: group.Label, Schedule: group.ScheduleKey, Vacancies: group.MaxCandidates - occupied, Penalty: review.Score.TotalPenalty, Alerts: []string{}}
		for _, v := range review.HardViolations {
			if v.GroupID == group.ID || v.CandidateID == candidateID || v.GroupID == source {
				s.Alerts = append(s.Alerts, v.Message)
			}
		}
		out = append(out, s)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if len(out[i].Alerts) != len(out[j].Alerts) {
			return len(out[i].Alerts) < len(out[j].Alerts)
		}
		if out[i].Penalty != out[j].Penalty {
			return out[i].Penalty < out[j].Penalty
		}
		return out[i].GroupID < out[j].GroupID
	})
	return out, nil
}

func (a *App) MoveAgenda(p MoveRequest) (AgendaOperation, error) {
	a.allocationMu.Lock()
	if a.allocationRunning {
		a.allocationMu.Unlock()
		return AgendaOperation{}, errors.New("pare a verificação antes de alterar a agenda")
	}
	snap, err := a.readAgenda()
	if err != nil {
		a.allocationMu.Unlock()
		return AgendaOperation{}, err
	}
	ids, err := moveSnapshot(&snap, p)
	if err == nil {
		refreshManualPreferences(&snap)
		snap.Run.Result = allocation.ReviewManualAllocation(snap.Run.Problem, snap.Run.Result.Assignments, snap.Run.Config.Normalized.Params.AvaliadoresPorGrupo)
		snap.Result.Score = snap.Run.Result.Score
		snap.Result.HardViolations = snap.Run.Result.HardViolations
		snap.Result.Quality = allocation.BuildAllocationQualityReport(snap.Run.Problem, snap.Run.Result)
		snap.Result.SolverStatus = "manual"
		snap.Result.Status = "Agenda ajustada manualmente"
		snap.Result.Editado = true
		err = a.saveConfiguredAgenda(snap.Run, snap.Result)
	}
	if err == nil {
		a.allocationResult = snap.Result
		a.allocationHasResult = true
		a.allocationAccess = time.Now()
	}
	a.allocationMu.Unlock()
	if err != nil {
		return AgendaOperation{}, err
	}
	op := AgendaOperation{Result: snap.Result}
	if p.Notify {
		op.BatchID, err = a.SendAgendaNotices(ids, 0)
		if err != nil {
			op.Warning = "Agenda alterada; avisos não iniciados: " + err.Error()
			err = nil
		}
	}
	return op, err
}

func refreshManualPreferences(s *agendaSnapshot) {
	for i := range s.Run.Problem.Candidates {
		c := &s.Run.Problem.Candidates[i]
		c.PreferredGroupIDs = []int{}
		c.PreferenceRankByGroupID = map[int]int{}
		var original types.Candidato
		for n, item := range s.Run.Candidatos {
			if n+1 == c.ID {
				original = item
				break
			}
		}
		for _, g := range s.Run.Problem.Groups {
			if s.Run.Config.Normalized.Params.SemPreferencia {
				c.PreferredGroupIDs = append(c.PreferredGroupIDs, g.ID)
				c.PreferenceRankByGroupID[g.ID] = 0
				continue
			}
			for rank, option := range original.Opcoes {
				matches := strings.EqualFold(strings.TrimSpace(option), strings.TrimSpace(g.ScheduleKey))
				for _, mapping := range s.Run.Config.Normalized.PreferenceMappings {
					if strings.EqualFold(mapping.ValorPreferencia, option) && strings.EqualFold(strings.TrimSpace(mapping.Dia+" "+mapping.Hora), strings.TrimSpace(g.ScheduleKey)) {
						matches = true
					}
				}
				if matches {
					c.PreferredGroupIDs = append(c.PreferredGroupIDs, g.ID)
					c.PreferenceRankByGroupID[g.ID] = rank
					break
				}
			}
		}
	}
}

func moveSnapshot(s *agendaSnapshot, p MoveRequest) ([]int, error) {
	if s.Run.Result.Assignments == nil {
		s.Run.Result.Assignments = map[int]int{}
	}
	ids := []int{}
	schedule := strings.TrimSpace(p.Schedule)
	if p.Kind == "person" {
		var person UICandidate
		found := false
		for i := range s.Result.Mesas {
			list := s.Result.Mesas[i].Candidatos
			for j, c := range list {
				if c.ID == p.CandidateID {
					person = c
					found = true
					s.Result.Mesas[i].Candidatos = append(list[:j], list[j+1:]...)
					break
				}
			}
		}
		for j, c := range s.Result.NaoAlocados {
			if c.ID == p.CandidateID {
				person = c
				found = true
				s.Result.NaoAlocados = append(s.Result.NaoAlocados[:j], s.Result.NaoAlocados[j+1:]...)
				break
			}
		}
		if !found {
			return nil, errors.New("candidato não encontrado")
		}
		target := -1
		for i, m := range s.Result.Mesas {
			if m.ID == p.DestinationID || (p.DestinationID == 0 && strings.EqualFold(m.Horario, schedule)) {
				target = i
				break
			}
		}
		if target < 0 {
			var group *types.SolverGroup
			for i := range s.Run.Problem.Groups {
				if s.Run.Problem.Groups[i].ID == p.DestinationID {
					group = &s.Run.Problem.Groups[i]
					break
				}
			}
			if group != nil {
				m := buildUIMesaMap([]types.SolverGroup{*group}, buildUIEvaluatorMap(s.Run.Avaliadores, s.Run.Problem.Candidates))
				s.Result.Mesas = append(s.Result.Mesas, m[group.ID])
			} else {
				if p.DestinationID != 0 {
					return nil, errors.New("mesa de destino não existe; atualize a agenda")
				}
				if schedule == "" {
					return nil, errors.New("informe um horário ou mesa")
				}
				id := 1
				for _, g := range s.Run.Problem.Groups {
					id = max(id, g.ID+1)
				}
				groupValue := types.SolverGroup{ID: id, Label: schedule + " grupo manual", ScheduleKey: schedule, MinCandidates: s.Run.Config.Normalized.Params.MinPessoasPorGrupo, MaxCandidates: s.Run.Config.Normalized.Params.MaxPessoasPorGrupo}
				s.Run.Problem.Groups = append(s.Run.Problem.Groups, groupValue)
				s.Result.Mesas = append(s.Result.Mesas, UIMesa{ID: id, Horario: schedule, Descricao: "Mesa manual", Candidatos: []UICandidate{}, Avaliadores: []UIAvaliador{}})
			}
			target = len(s.Result.Mesas) - 1
		}
		s.Result.Mesas[target].Candidatos = append(s.Result.Mesas[target].Candidatos, person)
		s.Run.Result.Assignments[person.ID] = s.Result.Mesas[target].ID
		return []int{person.ID}, nil
	}
	if p.Kind != "group" && p.Kind != "day" {
		return nil, errors.New("operação inválida")
	}
	to := strings.TrimSpace(p.To)
	if to == "" {
		return nil, errors.New("informe o destino")
	}
	for i := range s.Result.Mesas {
		m := &s.Result.Mesas[i]
		date, hour := dividirDataHorario(m.Horario)
		matches := p.Kind == "group" && m.ID == p.GroupID || p.Kind == "day" && (strings.EqualFold(m.Horario, p.From) || strings.EqualFold(date, p.From))
		if !matches {
			continue
		}
		destination := to
		if p.Kind == "day" && strings.EqualFold(date, p.From) && !strings.EqualFold(m.Horario, p.From) {
			destination = to + " " + hour
		}
		m.Horario = destination
		for j := range s.Run.Problem.Groups {
			if s.Run.Problem.Groups[j].ID == m.ID {
				s.Run.Problem.Groups[j].ScheduleKey = destination
				s.Run.Problem.Groups[j].Label = destination + " grupo " + m.Descricao
			}
		}
		for _, c := range m.Candidatos {
			ids = append(ids, c.ID)
		}
	}
	if len(ids) == 0 {
		return nil, errors.New("nenhuma pessoa afetada")
	}
	return ids, nil
}
