import React from "react";
import { SectionCard, SecondaryButton } from "./workflowShell";

export type ScheduleMapping = {
	valorPreferencia: string;
	dia: string;
	hora: string;
};

export default function ScheduleReview({
	mappings,
	onChange,
	free,
}: {
	mappings: ScheduleMapping[];
	onChange: (value: ScheduleMapping[]) => void;
	free: boolean;
}) {
	return (
		<SectionCard
			title="Revisar horários"
			description={
				free
					? "Informe os dias e intervalos disponíveis para esta rodada. Todos terão a mesma prioridade."
					: "Confira o dia e o intervalo de cada preferência. Valores ambíguos precisam ser preenchidos antes da simulação."
			}
		>
			<div className="space-y-4" data-testid="schedule-review">
				{mappings.map((mapping, i) => (
					<div
						key={i}
						className="grid gap-3 rounded-xl border p-4 sm:grid-cols-2"
						data-testid="schedule-mapping-row"
					>
						{!free && (
							<p className="font-medium sm:col-span-2">
								Preferência: {mapping.valorPreferencia}
							</p>
						)}
						<label className="block text-sm">
							Dia ou data {i + 1}
							<input
								className="minimal-input mt-1"
								aria-label={`Dia ou data ${i + 1}`}
								value={mapping.dia}
								placeholder="quarta ou 2026-11-04"
								onChange={(e) =>
									onChange(
										mappings.map((m, j) =>
											j === i ? { ...m, dia: e.target.value } : m,
										),
									)
								}
							/>
						</label>
						<label className="block text-sm">
							Intervalo {i + 1}
							<input
								className="minimal-input mt-1"
								aria-label={`Intervalo ${i + 1}`}
								value={mapping.hora}
								placeholder="14:00-16:00"
								onChange={(e) =>
									onChange(
										mappings.map((m, j) =>
											j === i ? { ...m, hora: e.target.value } : m,
										),
									)
								}
							/>
						</label>
						{free && (
							<SecondaryButton
								aria-label={`Remover horário ${i + 1}`}
								onClick={() => onChange(mappings.filter((_, j) => j !== i))}
							>
								Remover horário
							</SecondaryButton>
						)}
					</div>
				))}
				{mappings.length === 0 && (
					<p className="text-sm text-slate-600">
						Nenhum horário disponível. Revise a importação ou adicione os
						horários da rodada.
					</p>
				)}
				{free && (
					<SecondaryButton
						onClick={() =>
							onChange([
								...mappings,
								{ valorPreferencia: "", dia: "", hora: "" },
							])
						}
					>
						Adicionar horário
					</SecondaryButton>
				)}
			</div>
		</SectionCard>
	);
}
