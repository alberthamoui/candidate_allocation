package main

import (
	"sort"
	"strings"
	"unicode"
)

// ==================================================
// ========= SUGESTÃO DE MAPEAMENTO POR NOME ========
// ==================================================
//
// Compara o nome de cada campo esperado com o cabeçalho de cada coluna e
// sugere o melhor par, sem repetir coluna. A comparação ignora caixa, acentos,
// espaços, pontuação, "_" e "-", separa camelCase e letras de números e
// entende ordinais ("Primeira Opção" = "opcao 1"). Campos sem par por nome
// ficam sem coluna; em candidatos e avaliadores as colunas que sobram viram
// campos extras (acrescentarExtras), e o usuário arrasta a coluna certa para o
// campo vazio. Nas restrições, que não têm extras, os campos sem par recebem
// as colunas que sobraram, na ordem da planilha (porPosicao).

// aliasesCampo lista outros nomes comuns de coluna para cada campo.
var aliasesCampo = map[string][]string{
	"email_insper": {"email institucional", "email corporativo", "email da faculdade"},
	"numero":       {"telefone", "celular", "whatsapp"},
	"semestre":     {"periodo"},
	"nome":         {"avaliador", "candidato"},
	"candidato":    {"nome"},
	"naoPosso":     {"nao pode", "impedido"},
	"prefiroNao":   {"prefere nao", "evitar"},
}

var semAcento = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n", "º", "o", "ª", "a",
	"Á", "A", "À", "A", "Â", "A", "Ã", "A", "Ä", "A",
	"É", "E", "È", "E", "Ê", "E", "Ë", "E",
	"Í", "I", "Ì", "I", "Î", "I", "Ï", "I",
	"Ó", "O", "Ò", "O", "Ô", "O", "Õ", "O", "Ö", "O",
	"Ú", "U", "Ù", "U", "Û", "U", "Ü", "U",
	"Ç", "C", "Ñ", "N",
)

var numeroPorExtenso = map[string]string{
	"um": "1", "uma": "1", "primeira": "1", "primeiro": "1", "1a": "1", "1o": "1",
	"dois": "2", "duas": "2", "segunda": "2", "segundo": "2", "2a": "2", "2o": "2",
	"tres": "3", "terceira": "3", "terceiro": "3", "3a": "3", "3o": "3",
	"quatro": "4", "quarta": "4", "quarto": "4", "4a": "4", "4o": "4",
	"cinco": "5", "quinta": "5", "quinto": "5", "5a": "5", "5o": "5",
	"seis": "6", "sexta": "6", "sexto": "6", "6a": "6", "6o": "6",
	"sete": "7", "setima": "7", "setimo": "7", "7a": "7", "7o": "7",
	"oito": "8", "oitava": "8", "oitavo": "8", "8a": "8", "8o": "8",
	"nove": "9", "nona": "9", "nono": "9", "9a": "9", "9o": "9",
	"dez": "10", "decima": "10", "decimo": "10", "10a": "10", "10o": "10",
}

// tokensNome quebra um nome em palavras minúsculas, sem acento, com ordinais
// e números por extenso convertidos em dígitos.
// Ex.: "Primeira Opção" → [1 opcao]; "naoPosso" → [nao posso].
func tokensNome(nome string) []string {
	var b strings.Builder
	var ant rune
	for _, r := range semAcento.Replace(nome) {
		switch {
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			b.WriteRune(' ')
		case unicode.IsUpper(r) && (unicode.IsLower(ant) || unicode.IsDigit(ant)),
			unicode.IsDigit(r) && unicode.IsLetter(ant):
			b.WriteRune(' ')
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
		ant = r
	}
	tokens := strings.Fields(b.String())
	for i, t := range tokens {
		if n, ok := numeroPorExtenso[t]; ok {
			tokens[i] = n
		}
	}
	return tokens
}

func ehNumero(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return s != ""
}

// similaridadeNomes pontua o quanto o cabeçalho parece com o nome do campo
// (0 = nada a ver). Mesmas palavras em qualquer ordem valem 10.000; senão,
// conta palavras em comum e se um nome contém o outro. Quando os dois
// terminam em número, números iguais somam e diferentes derrubam o par
// (para "opcao 1" nunca cair em "Opção 2").
func similaridadeNomes(campo, cabecalho string) int {
	tc, th := tokensNome(campo), tokensNome(cabecalho)
	if len(tc) == 0 || len(th) == 0 {
		return 0
	}
	jc, jh := strings.Join(tc, ""), strings.Join(th, "")
	comuns := palavrasEmComum(tc, th)

	pontos := 0
	switch {
	case jc == jh || (comuns == len(tc) && comuns == len(th)):
		pontos = 10_000
	default:
		if strings.Contains(jh, jc) || strings.Contains(jc, jh) {
			pontos += 3_000
		}
		if comuns > 0 {
			pontos += 200 + 700*comuns
		}
	}
	if pontos == 0 {
		return 0
	}

	nc, temNC := ultimoNumero(tc)
	nh, temNH := ultimoNumero(th)
	if temNC && temNH {
		if nc == nh {
			pontos += 2_500
		} else {
			pontos -= 2_500
		}
	}
	// desempate: prefere cabeçalhos com a mesma quantidade de palavras
	if d := len(tc) - len(th); d == 0 {
		pontos += 100
	} else {
		pontos -= 10 * max(d, -d)
	}
	return max(pontos, 0)
}

func palavrasEmComum(a, b []string) int {
	restantes := make(map[string]int, len(b))
	for _, t := range b {
		restantes[t]++
	}
	n := 0
	for _, t := range a {
		if restantes[t] > 0 {
			restantes[t]--
			n++
		}
	}
	return n
}

func ultimoNumero(tokens []string) (string, bool) {
	ult := tokens[len(tokens)-1]
	return ult, ehNumero(ult)
}

// sugerirMapeamento devolve um MappingItem por campo, na ordem de campos.
// Com porPosicao, campos sem par por nome recebem as colunas que sobraram.
// Campo sem coluna fica com NomeColuna vazio e Indice fora do cabeçalho, que
// os Build*WithMapping ignoram.
func sugerirMapeamento(cabecalho, campos []string, porPosicao bool) []MappingItem {
	type par struct{ campo, coluna, pontos int }
	var pares []par
	for i, campo := range campos {
		nomes := append([]string{campo}, aliasesCampo[campo]...)
		for j, col := range cabecalho {
			melhor := 0
			for _, nome := range nomes {
				melhor = max(melhor, similaridadeNomes(nome, col))
			}
			if melhor > 0 {
				pares = append(pares, par{i, j, melhor})
			}
		}
	}
	// guloso: os pares mais parecidos primeiro; empate fica com a ordem da planilha
	sort.SliceStable(pares, func(a, b int) bool {
		if pares[a].pontos != pares[b].pontos {
			return pares[a].pontos > pares[b].pontos
		}
		if pares[a].coluna != pares[b].coluna {
			return pares[a].coluna < pares[b].coluna
		}
		return pares[a].campo < pares[b].campo
	})

	colunaDe := make([]int, len(campos))
	for i := range colunaDe {
		colunaDe[i] = -1
	}
	usada := make([]bool, len(cabecalho))
	for _, p := range pares {
		if colunaDe[p.campo] < 0 && !usada[p.coluna] {
			colunaDe[p.campo] = p.coluna
			usada[p.coluna] = true
		}
	}

	// campos sem par por nome: colunas que sobraram, na ordem da planilha
	livre := 0
	for i := range campos {
		if colunaDe[i] >= 0 || !porPosicao {
			continue
		}
		for livre < len(cabecalho) && usada[livre] {
			livre++
		}
		if livre < len(cabecalho) {
			colunaDe[i] = livre
			usada[livre] = true
		}
	}

	itens := make([]MappingItem, len(campos))
	for i, campo := range campos {
		if j := colunaDe[i]; j >= 0 {
			itens[i] = MappingItem{NomeColuna: cabecalho[j], Indice: j, Variavel: campo}
		} else {
			itens[i] = MappingItem{Indice: len(cabecalho) + i, Variavel: campo}
		}
	}
	return itens
}
