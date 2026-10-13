package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func gravarArquivoAtomico(path string, b []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".estado-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer func() { _ = os.Remove(name) }()
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
func cmdInspecionar(args []string) error {
	fs := novoFlagSet(args[0])
	dir := fs.String("workspace", "alocacao-cli", "pasta local")
	pessoa := fs.String("pessoa", "", "nome ou id")
	lote := fs.String("lote", "", "id do envio")
	if _, err := parseFlags(fs, args[1:]); err != nil {
		return err
	}
	s, err := carregarEstado(*dir)
	if err != nil {
		return err
	}
	var value any
	switch args[0] {
	case "sugestoes":
		value, err = s.Sugestoes(*pessoa)
	case "historico":
		value = s.HistoricoAvisos()
	case "repetir-falhas":
		op, e := s.RepetirFalhas(*lote, false)
		if e != nil {
			return e
		}
		if err := salvarEstado(*dir, s); err != nil {
			return err
		}
		imprimirOperacao(op)
		return erroEnvio(op)
	case "mesas", "resultado":
		value, err = s.ResultadoAtual()
	default:
		return fmt.Errorf("comando inválido")
	}
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
