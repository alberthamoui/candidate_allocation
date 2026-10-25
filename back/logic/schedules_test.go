package logic

import (
	types "candidate_alocator/back/type"
	"testing"
)

func TestCanonicalSchedule(t *testing.T) {
	for _, c := range []struct{ day, hour, wantDay, wantHour string }{
		{"Segunda-feira", "14h às 16h", "segunda", "14:00-16:00"},
		{"terça", "9:30-12:00", "terça", "09:30-12:00"},
		{"04/11/2026", "22-24", "2026-11-04", "22:00-24:00"},
		{"2026-11-04", "10h", "2026-11-04", "10:00"},
	} {
		day, hour, err := CanonicalSchedule(c.day, c.hour)
		if err != nil || day != c.wantDay || hour != c.wantHour {
			t.Fatal(c, day, hour, err)
		}
	}
	for _, c := range [][2]string{{"31/02/2026", "10-12"}, {"qualquer", "10-12"}, {"quarta", "25-26"}, {"quarta", "14-13"}, {"quarta", "14:60-16:00"}, {"quarta", "24-24"}, {"quarta", "horario importado"}, {"quarta", "10-12-14"}} {
		if _, _, err := CanonicalSchedule(c[0], c[1]); err == nil {
			t.Fatal("invalid schedule accepted", c)
		}
	}
}

func TestScheduleSuggestionsAndCoverage(t *testing.T) {
	d := DetectUniquePreferenceValues([]types.Candidato{{Opcoes: []string{"quarta 14-16", "opção A", "04/11/2026 10h-12h"}}})
	m := SuggestScheduleMappings(d)
	if m[0].Hora != "14:00-16:00" || m[1].Dia != "" || m[2].Dia != "2026-11-04" {
		t.Fatal(m)
	}
	if err := ValidateScheduleCoverage(d, m); err != nil {
		t.Fatal(err)
	}
	if err := ValidateScheduleCoverage(d, m[:1]); err == nil {
		t.Fatal("missing mapping accepted")
	}
	if err := ValidateScheduleCoverage(nil, nil); err == nil {
		t.Fatal("empty schedules accepted")
	}
	m[1].ValorPreferencia = "stale"
	if err := ValidateScheduleCoverage(d, m); err == nil {
		t.Fatal("stale mapping accepted")
	}
}
