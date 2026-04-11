package logic

import (
	"reflect"
	"testing"

	types "candidate_alocator/back/type"
)

func TestCalculatePossibleAllocationQuantitiesReturnsUniqueDistribution(t *testing.T) {
	params := types.AllocationParams{
		GruposPorHorario:   2,
		MinPessoasPorGrupo: 5,
		MaxPessoasPorGrupo: 5,
	}

	got := CalculatePossibleAllocationQuantities(params, 10)
	want := [][]int{{5, 5}}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected allocation quantities: got %#v want %#v", got, want)
	}
	if CountPossibleAllocationQuantities(params, 10) != 126 {
		t.Fatalf("expected 126 possible allocations, got %d", CountPossibleAllocationQuantities(params, 10))
	}
}

func TestCountPossibleAllocationQuantitiesReturnsZeroForInvalidParameters(t *testing.T) {
	params := types.AllocationParams{
		GruposPorHorario:   2,
		MinPessoasPorGrupo: 4,
		MaxPessoasPorGrupo: 8,
	}

	if got := CountPossibleAllocationQuantities(params, 3); got != 0 {
		t.Fatalf("expected zero allocations, got %d", got)
	}
}

func TestCountPossibleAllocationQuantitiesAllowsLeavingCandidatesOut(t *testing.T) {
	params := types.AllocationParams{
		GruposPorHorario:   1,
		MinPessoasPorGrupo: 1,
		MaxPessoasPorGrupo: 1,
	}

	if got := CountPossibleAllocationQuantities(params, 3); got != 3 {
		t.Fatalf("expected 3 allocations when leaving candidates out, got %d", got)
	}
}

func TestCountPossibleAllocationQuantitiesAcrossSchedulesCountsDistinctSlots(t *testing.T) {
	params := types.AllocationParams{
		GruposPorHorario:   2,
		MinPessoasPorGrupo: 2,
		MaxPessoasPorGrupo: 2,
	}

	if got := CountPossibleAllocationQuantitiesAcrossSchedules(params, 4, 2); got != 12 {
		t.Fatalf("expected 12 allocations across schedules, got %d", got)
	}
}

func TestCountPossibleAllocationQuantitiesAcrossSchedulesReusesBaseForSingleSchedule(t *testing.T) {
	params := types.AllocationParams{
		GruposPorHorario:   2,
		MinPessoasPorGrupo: 2,
		MaxPessoasPorGrupo: 2,
	}

	if got := CountPossibleAllocationQuantitiesAcrossSchedules(params, 4, 1); got != CountPossibleAllocationQuantities(params, 4) {
		t.Fatalf("expected single schedule count to match base count, got %d", got)
	}
}
