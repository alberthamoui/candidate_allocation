package logic

import (
	types "candidate_alocator/back/type"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var scheduleText = regexp.MustCompile(`(?i)^(.+?)\s+(\d{1,2}(?:[:h]\d{0,2})?(?:\s*(?:-|–|—|às|as|a)\s*\d{1,2}(?:[:h]\d{0,2})?)?)$`)
var scheduleClock = regexp.MustCompile(`^(\d{1,2})(?:[:h](\d{0,2}))?$`)
var scheduleRange = regexp.MustCompile(`\s*(?:-|–|—|às|as|a)\s*`)

// CanonicalSchedule validates a weekday/date and a clock time or interval.
func CanonicalSchedule(day, hour string) (string, string, error) {
	day = strings.TrimSpace(day)
	canonicalDay := ""
	for _, layout := range []string{"2006-01-02", "02/01/2006"} {
		if date, err := time.Parse(layout, day); err == nil {
			canonicalDay = date.Format("2006-01-02")
			break
		}
	}
	if canonicalDay == "" {
		key := strings.NewReplacer("á", "a", "ã", "a", "ç", "c", "é", "e", "-feira", "", " feira", "").Replace(strings.ToLower(day))
		canonicalDay = map[string]string{"seg": "segunda", "segunda": "segunda", "ter": "terça", "terca": "terça", "qua": "quarta", "quarta": "quarta", "qui": "quinta", "quinta": "quinta", "sex": "sexta", "sexta": "sexta", "sab": "sábado", "sabado": "sábado", "dom": "domingo", "domingo": "domingo"}[key]
	}
	if canonicalDay == "" {
		return "", "", fmt.Errorf("informe um dia da semana ou uma data válida (AAAA-MM-DD ou DD/MM/AAAA)")
	}
	parts := scheduleRange.Split(strings.ToLower(strings.TrimSpace(hour)), -1)
	if len(parts) < 1 || len(parts) > 2 {
		return "", "", fmt.Errorf("informe uma hora ou intervalo, por exemplo 14:00-16:00")
	}
	times := make([]int, len(parts))
	clocks := make([]string, len(parts))
	for i, part := range parts {
		matches := scheduleClock.FindStringSubmatch(strings.TrimSpace(part))
		if matches == nil {
			return "", "", fmt.Errorf("horário inválido; use HH:MM ou HH:MM-HH:MM")
		}
		h, _ := strconv.Atoi(matches[1])
		minute := 0
		if matches[2] != "" {
			minute, _ = strconv.Atoi(matches[2])
		}
		if minute > 59 || h > 24 || (h == 24 && (i == 0 || minute != 0)) {
			return "", "", fmt.Errorf("hora fora do intervalo permitido")
		}
		times[i] = h*60 + minute
		clocks[i] = fmt.Sprintf("%02d:%02d", h, minute)
	}
	if len(times) == 2 && times[1] <= times[0] {
		return "", "", fmt.Errorf("o fim do intervalo precisa ser posterior ao início no mesmo dia")
	}
	return canonicalDay, strings.Join(clocks, "-"), nil
}

// SuggestScheduleMappings leaves ambiguous spreadsheet values for review.
func SuggestScheduleMappings(detections []types.UniqueValueDetection) []types.PreferenceScheduleMapping {
	mappings := make([]types.PreferenceScheduleMapping, 0, len(detections))
	for _, d := range detections {
		mapping := types.PreferenceScheduleMapping{ValorPreferencia: d.ValorOriginal}
		text := strings.ReplaceAll(strings.ToLower(strings.Join(strings.Fields(d.ValorOriginal), " ")), " das ", " ")
		if match := scheduleText.FindStringSubmatch(text); match != nil {
			day, hour, err := CanonicalSchedule(strings.TrimRight(match[1], ","), match[2])
			if err == nil {
				mapping.Dia, mapping.Hora = day, hour
			}
		}
		mappings = append(mappings, mapping)
	}
	return mappings
}

func ValidateScheduleCoverage(detections []types.UniqueValueDetection, mappings []types.PreferenceScheduleMapping) error {
	if len(detections) == 0 {
		return fmt.Errorf("nenhum horário disponível; revise as preferências ou informe os horários da rodada")
	}
	available, mapped := map[string]bool{}, map[string]bool{}
	for _, d := range detections {
		available[normalizeDetectedValue(d.ValorOriginal)] = true
	}
	for _, m := range mappings {
		key := normalizeDetectedValue(m.ValorPreferencia)
		if !available[key] {
			return fmt.Errorf("a preferência %q não consta nos dados atuais; recarregue os horários", m.ValorPreferencia)
		}
		mapped[key] = true
	}
	for _, d := range detections {
		if !mapped[normalizeDetectedValue(d.ValorOriginal)] {
			return fmt.Errorf("confirme dia e intervalo da preferência %q", d.ValorOriginal)
		}
	}
	return nil
}
