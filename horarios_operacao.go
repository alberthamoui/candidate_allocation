package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var intervaloAgenda = regexp.MustCompile(`(?i)^(.*?)\s+(\d{1,2})(?::(\d{2}))?\s*(?:h\s*-?|[-–—])\s*(\d{1,2})(?::(\d{2}))?h?$`)

func intervaloHorario(s string) (dia string, inicio, fim int, ok bool) {
	m := intervaloAgenda.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return "", 0, 0, false
	}
	var h1, h2, min1, min2 int
	_, _ = fmt.Sscan(m[2], &h1)
	_, _ = fmt.Sscan(m[4], &h2)
	if m[3] != "" {
		_, _ = fmt.Sscan(m[3], &min1)
	}
	if m[5] != "" {
		_, _ = fmt.Sscan(m[5], &min2)
	}
	inicio, fim = h1*60+min1, h2*60+min2
	return normalizaOpcao(m[1]), inicio, fim, h1 < 24 && h2 <= 24 && min1 < 60 && min2 < 60 && inicio < fim && fim <= 24*60
}
func horariosSobrepostos(a, b string) bool {
	if normalizaOpcao(a) == normalizaOpcao(b) {
		return true
	}
	da, ia, fa, oka := intervaloHorario(a)
	db, ib, fb, okb := intervaloHorario(b)
	return oka && okb && da == db && ia < fb && ib < fa
}
func ordemDataHorario(s string) (string, bool) {
	dia, inicio, _, ok := intervaloHorario(s)
	if !ok {
		return "", false
	}
	for _, layout := range []string{"02/01/2006", "2/1/2006", "2006-01-02"} {
		if d, err := time.Parse(layout, dia); err == nil {
			return fmt.Sprintf("0 %s %04d", d.Format("2006-01-02"), inicio), true
		}
	}
	return "", false
}
