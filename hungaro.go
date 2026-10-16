package main

import "math"

// hungaro resolve a atribuição de custo mínimo de n linhas a colunas
// distintas de m ≥ n (algoritmo húngaro com potenciais, O(n²·m)).
// Os buffers são reaproveitados entre chamadas.
type hungaro struct {
	custo         []int // [linha*m+coluna], a partir de 0
	u, v, minv, p []int
	way           []int
	usado         []bool
}

func novoHungaro(maxLinhas, maxColunas int) hungaro {
	return hungaro{
		custo: make([]int, maxLinhas*maxColunas),
		u:     make([]int, maxLinhas+1),
		v:     make([]int, maxColunas+1),
		minv:  make([]int, maxColunas+1),
		p:     make([]int, maxColunas+1),
		way:   make([]int, maxColunas+1),
		usado: make([]bool, maxColunas+1),
	}
}

// resolver devolve o custo mínimo; depois, h.p[j] (j = 1..m) é a linha
// (a partir de 1) atribuída à coluna j-1, ou 0 se a coluna ficou livre.
//
// Depois de processar i linhas, -v[0] é o custo ótimo só dessas linhas, que
// nunca diminui. Se passar de corte, para e devolve completo = false (o
// custo devolvido é então só um limite inferior).
func (h *hungaro) resolver(n, m, corte int) (custo int, completo bool) {
	const inf = math.MaxInt / 4
	u, v, p, way, minv, usado := h.u[:n+1], h.v[:m+1], h.p[:m+1], h.way[:m+1], h.minv[:m+1], h.usado[:m+1]
	clear(u)
	clear(v)
	clear(p)
	for i := 1; i <= n; i++ {
		p[0] = i
		j0 := 0
		for j := range minv {
			minv[j], usado[j] = inf, false
		}
		for {
			usado[j0] = true
			i0, delta, j1 := p[j0], inf, 0
			linha := h.custo[(i0-1)*m : i0*m]
			for j := 1; j <= m; j++ {
				if usado[j] {
					continue
				}
				if cur := linha[j-1] - u[i0] - v[j]; cur < minv[j] {
					minv[j], way[j] = cur, j0
				}
				if minv[j] < delta {
					delta, j1 = minv[j], j
				}
			}
			for j := 0; j <= m; j++ {
				if usado[j] {
					u[p[j]] += delta
					v[j] -= delta
				} else {
					minv[j] -= delta
				}
			}
			j0 = j1
			if p[j0] == 0 {
				break
			}
		}
		for j0 != 0 {
			j1 := way[j0]
			p[j0] = p[j1]
			j0 = j1
		}
		if -v[0] > corte && i < n {
			return -v[0], false
		}
	}
	return -v[0], true
}
