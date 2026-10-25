package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
)

func runAgendaCLI(args []string) error {
	if len(args) == 0 {
		return errors.New("uso: cli agenda resultado|avaliar|sugestoes|realocar-pessoa|realocar-mesa|realocar-dia|canais|testar-canais|previa|enviar|historico|detalhes|repetir-falhas|exportar")
	}
	f := flag.NewFlagSet("agenda "+args[0], flag.ContinueOnError)
	id := f.Int("pessoa", 0, "ID da pessoa")
	group := f.Int("mesa", 0, "ID da mesa")
	destination := f.Int("destino", 0, "ID da mesa de destino")
	schedule := f.String("horario", "", "novo horário")
	from := f.String("de", "", "dia ou intervalo de origem")
	to := f.String("para", "", "dia ou intervalo de destino")
	value := f.String("resultado", "", "aprovado, reprovado ou vazio")
	batch := f.Int64("lote", 0, "ID do envio")
	cursor := f.Int64("cursor", 0, "cursor de paginação")
	file := f.String("arquivo", "", "arquivo de configuração JSON ou saída Excel")
	notify := f.Bool("avisar", false, "autoriza enviar avisos da alteração")
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	a := NewApp()
	a.synchronousNotices = true
	defer a.clearTransientState()
	var result any
	var err error
	switch args[0] {
	case "resultado":
		result, err = a.GetAgenda()
	case "avaliar":
		result, err = a.EvaluateCandidates([]int{*id}, *value)
	case "sugestoes":
		result, err = a.GetMoveSuggestions(*id)
	case "realocar-pessoa", "realocar-mesa", "realocar-dia":
		kind := map[string]string{"realocar-pessoa": "person", "realocar-mesa": "group", "realocar-dia": "day"}[args[0]]
		result, err = a.MoveAgenda(MoveRequest{Kind: kind, CandidateID: *id, GroupID: *group, DestinationID: *destination, Schedule: *schedule, From: *from, To: *to, Notify: *notify})
	case "canais":
		if *file != "" {
			// #nosec G304 G703 -- explicit local input path chosen by the CLI user.
			data, e := os.ReadFile(*file)
			if e != nil {
				return e
			}
			var cfg ConfigAvisos
			if e := json.Unmarshal(data, &cfg); e != nil {
				return e
			}
			if err = a.SaveChannelConfig(cfg); err != nil {
				return err
			}
		}
		result, err = a.GetChannelConfig()
	case "testar-canais":
		result, err = a.TestChannels()
	case "previa":
		ids := []int{}
		if *id != 0 {
			ids = append(ids, *id)
		}
		result, err = a.PreviewAgendaNotices(ids, *group)
	case "enviar":
		ids := []int{}
		if *id != 0 {
			ids = append(ids, *id)
		}
		result, err = a.SendAgendaNotices(ids, *group)
	case "historico":
		result, err = a.GetNoticeHistory(*cursor)
	case "detalhes":
		result, err = a.GetNoticeDetails(*batch, *cursor)
	case "repetir-falhas":
		result, err = a.RetryNoticeFailures(*batch)
	case "exportar":
		if *file == "" {
			return errors.New("informe -arquivo resultado.xlsx")
		}
		data, e := a.ExportAgendaExcel()
		if e != nil {
			return e
		}
		decoded, e := base64.StdEncoding.DecodeString(data)
		if e != nil {
			return e
		}
		// #nosec G703 -- explicit local output path chosen by the CLI user.
		err = os.WriteFile(*file, decoded, 0o600)
		result = map[string]string{"arquivo": *file}
	default:
		return errors.New("comando de agenda desconhecido")
	}
	if err != nil {
		return err
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		return err
	}
	var batchID int64
	switch value := result.(type) {
	case int64:
		batchID = value
	case AgendaOperation:
		batchID = value.BatchID
		if value.Warning != "" {
			return errors.New(value.Warning)
		}
	}
	if batchID > 0 {
		s, e := a.store()
		if e != nil {
			return e
		}
		var after int64
		for {
			page, e := s.Deliveries(currentAgenda, batchID, after, 100)
			if e != nil {
				return e
			}
			for _, d := range page.Items {
				if d.Status != "accepted" {
					return fmt.Errorf("envio %d contém pendências ou falhas; consulte detalhes", batchID)
				}
			}
			if page.Next == 0 {
				break
			}
			after = page.Next
		}
	}
	return nil
}
