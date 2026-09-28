package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// lerCabecalho devolve a primeira linha da aba de índice aba do Excel da sessão.
func (s *Session) lerCabecalho(aba int) ([]string, error) {
	file, err := excelize.OpenReader(bytes.NewReader(s.excelData))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	rows, err := file.GetRows(file.GetSheetName(aba))
	if err != nil {
		return nil, err
	}
	if len(rows) < 1 {
		return nil, fmt.Errorf("arquivo sem dados")
	}
	return rows[0], nil
}

// SuggestMapping guarda o Excel na sessão, lê o cabeçalho da primeira aba e
// sugere o mapeamento entre colunas e campos de Usuario pelo nome das colunas.
func (s *Session) SuggestMapping(data []byte, quantidade_opcoes int, emailDomain string) ([]MappingItem, error) {
	s.excelData = data
	s.nOpcoes = quantidade_opcoes
	s.emailDomain = emailDomain
	header, err := s.lerCabecalho(0)
	if err != nil {
		return nil, err
	}
	return acrescentarExtras(sugerirMapeamento(header, getUsuarioFields(quantidade_opcoes), false), header), nil
}

// SuggestMappingAvaliador lê o cabeçalho da segunda aba do Excel e sugere
// o mapeamento para campos de AvaliadorInfo.
func (s *Session) SuggestMappingAvaliador() ([]MappingItem, error) {
	header, err := s.lerCabecalho(1)
	if err != nil {
		return nil, err
	}
	return acrescentarExtras(sugerirMapeamento(header, getAvaliadorFields(), false), header), nil
}

// SuggestMappingRestricao lê o cabeçalho da terceira aba do Excel e sugere
// o mapeamento para campos de Restricao.
func (s *Session) SuggestMappingRestricao() ([]MappingItem, error) {
	header, err := s.lerCabecalho(2)
	if err != nil {
		return nil, err
	}
	return sugerirMapeamento(header, getRestricaoFields(), true), nil
}

// BuildUsuariosWithMapping lê a primeira aba do Excel aplicando o mapeamento
// fornecido e retorna usuários validados e índices de duplicatas.
func (s *Session) BuildUsuariosWithMapping(mappingItems []MappingItem) (UsuariosResponse, error) {
	data := s.excelData
	nOpcoes := s.nOpcoes
	readerData := bytes.NewReader(data)
	file, err := excelize.OpenReader(readerData)
	if err != nil {
		return UsuariosResponse{}, err
	}
	defer file.Close()

	sheet := file.GetSheetName(0)
	rows, err := file.GetRows(sheet)
	if err != nil {
		return UsuariosResponse{}, fmt.Errorf("erro ao ler excel : %w", err)
	}
	if len(rows) < 2 {
		return UsuariosResponse{}, fmt.Errorf("arquivo sem dados além do header")
	}
	nomesExtras, err := validarExtras(mappingItems, getUsuarioFields(nOpcoes))
	if err != nil {
		return UsuariosResponse{}, err
	}
	s.extrasCandidatos = nomesExtras

	var users []Usuario

	for _, row := range rows[1:] {
		u := Usuario{
			Opcoes: make([]string, nOpcoes),
			Extras: novosExtras(nomesExtras),
		}
		for _, mItem := range mappingItems {
			if mItem.Indice >= len(row) {
				continue
			}
			cell := row[mItem.Indice]
			switch mItem.Variavel {
			case "timestamp":
				u.Timestamp = cell
			case "nome":
				u.Nome = cell
			case "cpf":
				u.CPF = cell
			case "numero":
				u.Numero = cell
			case "semestre":
				u.Semestre = cell
			case "curso":
				u.Curso = cell
			case "email_insper":
				u.EmailInsper = cell
			case "email_pessoal":
				u.EmailPessoal = cell
			default:
				if nome, ok := nomeExtra(mItem.Variavel); ok {
					u.Extras[nome] = strings.TrimSpace(cell)
				} else if strings.HasPrefix(mItem.Variavel, "opcao") {
					parts := strings.Split(mItem.Variavel, " ")
					if len(parts) == 2 {
						optionNum, err := strconv.Atoi(parts[1])
						if err == nil && optionNum > 0 && optionNum <= nOpcoes {
							u.Opcoes[optionNum-1] = cell
						}
					}
				}
			}
		}
		users = append(users, u)
	}
	users_limpo, duplicatedIndices := processData(users, s.emailDomain)

	return UsuariosResponse{Usuarios: users_limpo, Duplicates: duplicatedIndices}, nil
}

// BuildAvaliadoresWithMapping lê a segunda aba do Excel aplicando o mapeamento fornecido.
func (s *Session) BuildAvaliadoresWithMapping(mappingItems []MappingItem) ([]AvaliadorInfo, error) {
	if s.excelData == nil {
		return nil, fmt.Errorf("dados do Excel ainda não carregados")
	}

	reader := bytes.NewReader(s.excelData)
	file, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("erro abrindo excel: %w", err)
	}
	defer file.Close()

	sheet := file.GetSheetName(1)
	if sheet == "" {
		return nil, fmt.Errorf("arquivo não possui uma segunda aba com avaliadores")
	}

	rows, err := file.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("erro lendo aba de avaliadores: %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("aba de avaliadores não contém dados além do cabeçalho")
	}
	nomesExtras, err := validarExtras(mappingItems, getAvaliadorFields())
	if err != nil {
		return nil, err
	}

	var avaliadores []AvaliadorInfo

	for _, row := range rows[1:] {
		av := AvaliadorInfo{Extras: novosExtras(nomesExtras)}
		for _, m := range mappingItems {
			if m.Indice >= len(row) {
				continue
			}
			val := strings.TrimSpace(row[m.Indice])
			if nome, ok := nomeExtra(m.Variavel); ok {
				av.Extras[nome] = val
				continue
			}

			switch strings.ToLower(m.Variavel) {
			case "nome":
				av.Nome = val
			case "email":
				av.Email = val
			case "sigla":
				av.Sigla = val
			}
		}

		if av.Nome == "" && av.Email == "" && av.Sigla == "" {
			continue
		}
		avaliadores = append(avaliadores, av)
	}

	return avaliadores, nil
}

// BuildRestricoesWithMapping lê a terceira aba do Excel aplicando o mapeamento fornecido.
func (s *Session) BuildRestricoesWithMapping(mappingItems []MappingItem) ([]Restricao, error) {
	readerData := bytes.NewReader(s.excelData)
	file, err := excelize.OpenReader(readerData)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sheet := file.GetSheetName(2)
	rows, err := file.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler excel: %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("arquivo sem dados além do header")
	}

	var restricoes []Restricao
	for _, row := range rows[1:] {
		r := Restricao{}
		for _, m := range mappingItems {
			if m.Indice >= len(row) {
				continue
			}
			cell := row[m.Indice]
			switch m.Variavel {
			case "candidato":
				r.Candidato = cell
			case "naoPosso":
				r.NaoPosso = cell
			case "prefiroNao":
				r.PrefiroNao = cell
			}
		}
		restricoes = append(restricoes, r)
	}
	return restricoes, nil
}
