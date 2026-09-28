package main

import (
	"fmt"
	"strings"
)

// ==================================================
// ============= CRITÉRIOS ADICIONAIS ===============
// ==================================================
//
// Regras opcionais sobre uma coluna dos candidatos (curso ou semestre),
// escolhidas na tela de parâmetros. Cada uma conta "unidades" de desvio em
// cada mesa e custa Peso pontos por unidade:
//
//	misturar    cada par de candidatos com o mesmo valor na mesa
//	agrupar     cada par de candidatos com valores diferentes na mesa
//	maximo      cada candidato de um dos Valores acima de Limite
//	minimo      o que falta para Limite quando um dos Valores aparece
//	            (entre 1 e Limite-1 candidatos dele na mesa)
//	um_de_cada  cada um dos Valores que não aparece na mesa
//
// Para maximo e minimo, Valores vazio vale para todos os valores.
// O custo só depende de quem está em cada mesa, como as preferências de
// horário, e entra no custo da busca e na pontuação.

const (
	CRITERIO_MISTURAR   = "misturar"
	CRITERIO_AGRUPAR    = "agrupar"
	CRITERIO_MAXIMO     = "maximo"
	CRITERIO_MINIMO     = "minimo"
	CRITERIO_UM_DE_CADA = "um_de_cada"

	LIMITE_CRITERIOS = 5
	LIMITE_PESO      = 100
	LIMITE_POR_VALOR = LIMITE_PESSOAS_POR_MESA
	COLUNA_CURSO     = "curso"
	COLUNA_SEMESTRE  = "semestre"
)

// CriterioAlocacao é um critério escolhido na tela.
type CriterioAlocacao struct {
	Tipo    string   `json:"tipo"`
	Coluna  string   `json:"coluna"`
	Valores []string `json:"valores"`
	Limite  int      `json:"limite"`
	Peso    int      `json:"peso"` // pontos por unidade de desvio
}

var nomeColuna = map[string]string{COLUNA_CURSO: "Curso", COLUNA_SEMESTRE: "Semestre"}

func (c CriterioAlocacao) validar() error {
	switch c.Tipo {
	case CRITERIO_MISTURAR, CRITERIO_AGRUPAR, CRITERIO_MAXIMO, CRITERIO_MINIMO, CRITERIO_UM_DE_CADA:
	default:
		return fmt.Errorf("tipo de critério desconhecido: %q", c.Tipo)
	}
	if _, ok := nomeColuna[c.Coluna]; !ok {
		return fmt.Errorf("critério %q: coluna %q não disponível (use curso ou semestre)", c.Tipo, c.Coluna)
	}
	if c.Peso < 1 || c.Peso > LIMITE_PESO {
		return fmt.Errorf("critério %q: peso deve estar entre 1 e %d", c.Tipo, LIMITE_PESO)
	}
	if (c.Tipo == CRITERIO_MAXIMO || c.Tipo == CRITERIO_MINIMO) && (c.Limite < 1 || c.Limite > LIMITE_POR_VALOR) {
		return fmt.Errorf("critério %q: o número por mesa deve estar entre 1 e %d", c.Tipo, LIMITE_POR_VALOR)
	}
	if c.Tipo == CRITERIO_MINIMO && c.Limite < 2 {
		return fmt.Errorf("critério \"pelo menos N\": N precisa ser 2 ou mais (1 sempre é atendido)")
	}
	if c.Tipo == CRITERIO_UM_DE_CADA && len(c.Valores) == 0 {
		return fmt.Errorf("critério \"pelo menos um de cada\": escolha os valores de %s", strings.ToLower(nomeColuna[c.Coluna]))
	}
	return nil
}

// chaveValor normaliza um valor de coluna para comparar (caixa, acento, espaços).
func chaveValor(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(semAcento.Replace(s))), " ")
}

// criterioProb é um critério indexado para a busca.
type criterioProb struct {
	tipo     string
	nValores int
	valorDe  []int  // candidato → índice do valor (-1: sem valor na coluna)
	alvo     []bool // valor → conta no critério (maximo, minimo, um_de_cada)
	limite   int
	peso     int
}

// aplicarCriterios indexa os critérios para os candidatos do problema.
// atributos[pessoa_id][coluna] é o valor da coluna para o candidato.
func (p *problema) aplicarCriterios(criterios []CriterioAlocacao, atributos map[int]map[string]string) {
	p.criterios = nil
	for _, c := range criterios {
		cp := criterioProb{tipo: c.Tipo, limite: c.Limite, peso: c.Peso, valorDe: make([]int, p.nCand)}
		indice := map[string]int{}
		for i, pid := range p.candID {
			k := chaveValor(atributos[pid][c.Coluna])
			if k == "" {
				cp.valorDe[i] = -1
				continue
			}
			v, ok := indice[k]
			if !ok {
				v = len(indice)
				indice[k] = v
			}
			cp.valorDe[i] = v
		}
		cp.nValores = len(indice)
		cp.alvo = make([]bool, cp.nValores)
		escolhidos := map[string]bool{}
		for _, v := range c.Valores {
			escolhidos[chaveValor(v)] = true
		}
		for k, v := range indice {
			cp.alvo[v] = len(escolhidos) == 0 || escolhidos[k]
		}
		// "um de cada" com um valor que nenhum candidato tem: sempre faltaria;
		// o valor ganha um índice extra, sem candidatos, para ser cobrado igual
		if c.Tipo == CRITERIO_UM_DE_CADA {
			for k := range escolhidos {
				if _, ok := indice[k]; !ok {
					indice[k] = cp.nValores
					cp.nValores++
					cp.alvo = append(cp.alvo, true)
				}
			}
		}
		p.criterios = append(p.criterios, cp)
	}
}

// unidades conta os desvios de uma mesa com n candidatos e cont[v]
// candidatos de cada valor.
func (cp *criterioProb) unidades(cont []int) int {
	pares := func(x int) int { return x * (x - 1) / 2 }
	u := 0
	switch cp.tipo {
	case CRITERIO_MISTURAR:
		for _, x := range cont {
			u += pares(x)
		}
	case CRITERIO_AGRUPAR:
		total := 0
		for _, x := range cont {
			total += x
			u -= pares(x)
		}
		u += pares(total)
	case CRITERIO_MAXIMO:
		for v, x := range cont {
			if cp.alvo[v] && x > cp.limite {
				u += x - cp.limite
			}
		}
	case CRITERIO_MINIMO:
		for v, x := range cont {
			if cp.alvo[v] && x > 0 && x < cp.limite {
				u += cp.limite - x
			}
		}
	case CRITERIO_UM_DE_CADA:
		for v, x := range cont {
			if cp.alvo[v] && x == 0 {
				u++
			}
		}
	}
	return u
}

// custoCriteriosMembros é o custo dos critérios para uma mesa com esses
// candidatos (índices do problema), contando do zero.
func (p *problema) custoCriteriosMembros(membros []int) int {
	if len(membros) == 0 {
		return 0
	}
	custo := 0
	for k := range p.criterios {
		cp := &p.criterios[k]
		cont := make([]int, cp.nValores)
		for _, c := range membros {
			if v := cp.valorDe[c]; v >= 0 {
				cont[v]++
			}
		}
		custo += cp.peso * cp.unidades(cont)
	}
	return custo
}

// custoCriteriosMesa é o mesmo que custoCriteriosMembros, mas usa as
// contagens que o estado mantém a cada movimento.
func (e *estado) custoCriteriosMesa(m int) int {
	if len(e.membros[m]) == 0 {
		return 0
	}
	custo := 0
	for k := range e.p.criterios {
		cp := &e.p.criterios[k]
		custo += cp.peso * cp.unidades(e.contCrit[k][m*cp.nValores:(m+1)*cp.nValores])
	}
	return custo
}
