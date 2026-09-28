package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// ==================================================
// ============ RECURSOS DO SERVIDOR ================
// ==================================================
//
// Em produção (ex.: Render gratuito, 0,1 CPU) o container tem bem menos CPU
// que a máquina. O Go 1.23 não enxerga esse limite, então lemos a cota do
// cgroup e ajustamos: quantas buscas a alocação roda em paralelo, quantas
// alocações rodam ao mesmo tempo e o GOMAXPROCS. Cada valor pode ser fixado
// por variável de ambiente.

// Recursos são os limites em uso pelo servidor.
type Recursos struct {
	CPUs                 float64 // CPU disponível (cota do container ou núcleos da máquina)
	Execucoes            int     // buscas em paralelo por alocação (ALOCACAO_EXECUCOES)
	Iteracoes            int     // movimentos por busca (ALOCACAO_ITERACOES)
	AlocacoesSimultaneas int     // alocações ao mesmo tempo; as outras esperam (ALOCACOES_SIMULTANEAS)
	MaxSessoes           int     // sessões abertas ao mesmo tempo (MAX_SESSOES)
	MaxUploadMB          int     // tamanho máximo da planilha (MAX_UPLOAD_MB)
}

// recursos começa com os valores de uma máquina sem limite; main chama
// configurarRecursos para ajustar ao ambiente.
var recursos = Recursos{
	CPUs:                 float64(runtime.NumCPU()),
	Execucoes:            SA_EXECUCOES,
	Iteracoes:            SA_ITERACOES,
	AlocacoesSimultaneas: 2,
	MaxSessoes:           200,
	MaxUploadMB:          10,
}

// configurarRecursos decide os limites a partir da CPU disponível e das
// variáveis de ambiente, e ajusta o GOMAXPROCS.
func configurarRecursos(getenv func(string) string, cotaCgroup float64) (Recursos, error) {
	r := recursos
	if cotaCgroup > 0 && cotaCgroup < r.CPUs {
		r.CPUs = cotaCgroup
	}
	nucleos := max(1, int(math.Ceil(r.CPUs)))
	// uma busca por núcleo, até SA_EXECUCOES; com menos de 1 CPU, só uma
	r.Execucoes = min(SA_EXECUCOES, nucleos)
	r.AlocacoesSimultaneas = max(1, nucleos/r.Execucoes)

	for _, v := range []struct {
		nome     string
		dst      *int
		min, max int
	}{
		{"ALOCACAO_EXECUCOES", &r.Execucoes, 1, 64},
		{"ALOCACAO_ITERACOES", &r.Iteracoes, 1_000, 10_000_000},
		{"ALOCACOES_SIMULTANEAS", &r.AlocacoesSimultaneas, 1, 64},
		{"MAX_SESSOES", &r.MaxSessoes, 1, 100_000},
		{"MAX_UPLOAD_MB", &r.MaxUploadMB, 1, 100},
	} {
		s := strings.TrimSpace(getenv(v.nome))
		if s == "" {
			continue
		}
		n, err := strconv.Atoi(s)
		if err != nil || n < v.min || n > v.max {
			return r, fmt.Errorf("variável %s inválida: %q (esperado de %d a %d)", v.nome, s, v.min, v.max)
		}
		*v.dst = n
	}
	return r, nil
}

// cotaCPU lê a cota de CPU do container (cgroup v2 ou v1); 0 se não houver.
func cotaCPU() float64 {
	if b, err := os.ReadFile("/sys/fs/cgroup/cpu.max"); err == nil {
		return lerCpuMax(string(b))
	}
	quota, err1 := os.ReadFile("/sys/fs/cgroup/cpu/cpu.cfs_quota_us")
	periodo, err2 := os.ReadFile("/sys/fs/cgroup/cpu/cpu.cfs_period_us")
	if err1 == nil && err2 == nil {
		return lerCpuMax(strings.TrimSpace(string(quota)) + " " + strings.TrimSpace(string(periodo)))
	}
	return 0
}

// lerCpuMax interpreta "cota período" (ex.: "10000 100000" = 0,1 CPU);
// "max" ou valor inválido = sem cota (0).
func lerCpuMax(s string) float64 {
	campos := strings.Fields(s)
	if len(campos) != 2 || campos[0] == "max" {
		return 0
	}
	cota, err1 := strconv.ParseFloat(campos[0], 64)
	periodo, err2 := strconv.ParseFloat(campos[1], 64)
	if err1 != nil || err2 != nil || cota <= 0 || periodo <= 0 {
		return 0
	}
	return cota / periodo
}

// aplicarRecursos configura o processo com os limites escolhidos.
func aplicarRecursos(r Recursos) {
	recursos = r
	runtime.GOMAXPROCS(max(1, int(math.Ceil(r.CPUs))))
	log.Printf("Recursos: %.2f CPU | alocação com %d busca(s) × %d iterações | %d alocação(ões) por vez | até %d sessões | upload até %d MB",
		r.CPUs, r.Execucoes, r.Iteracoes, r.AlocacoesSimultaneas, r.MaxSessoes, r.MaxUploadMB)
}
