package main

import (
	"encoding/base64"
	"github.com/xuri/excelize/v2"
	"strings"
)

func phoneFromExtras(extras map[string]string) string {
	for key, value := range extras {
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "telefone", "whatsapp", "celular", "telefone/whatsapp", "telefone whatsapp":
			return value
		}
	}
	return ""
}

// ExportAgendaExcel exports the same persisted agenda used by UI and CLI.
func (a *App) ExportAgendaExcel() (string, error) {
	snapshot, err := a.readAgenda()
	if err != nil {
		return "", err
	}
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := f.GetSheetName(0)
	headers := []interface{}{"Mesa", "Horário", "Candidato", "E-mail", "Matrícula", "Telefone WhatsApp", "Avaliação", "Avaliadores"}
	if err := f.SetSheetRow(sheet, "A1", &headers); err != nil {
		return "", err
	}
	row := 2
	for _, m := range snapshot.Result.Mesas {
		names := []string{}
		for _, e := range m.Avaliadores {
			names = append(names, e.Nome)
		}
		for _, c := range m.Candidatos {
			values := []interface{}{m.Descricao, m.Horario, c.Nome, emailDe(c), c.Numero, c.Telefone, textoAvaliacao(c.Avaliacao), strings.Join(names, ", ")}
			cell, _ := excelize.CoordinatesToCellName(1, row)
			if err := f.SetSheetRow(sheet, cell, &values); err != nil {
				return "", err
			}
			row++
		}
	}
	if _, err := f.NewSheet("Sem mesa"); err != nil {
		return "", err
	}
	headers = []interface{}{"Candidato", "E-mail", "Avaliação"}
	if err := f.SetSheetRow("Sem mesa", "A1", &headers); err != nil {
		return "", err
	}
	for i, c := range snapshot.Result.NaoAlocados {
		values := []interface{}{c.Nome, emailDe(c), textoAvaliacao(c.Avaliacao)}
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		if err := f.SetSheetRow("Sem mesa", cell, &values); err != nil {
			return "", err
		}
	}
	data, err := f.WriteToBuffer()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data.Bytes()), nil
}
