import React, { useEffect, useState } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import {
	BuildAllocationConfigurationFromDatabase,
	RunAllocation,
	CountPossibleAllocationQuantitiesAcrossSchedules,
} from "../wailsjs/go/main/App";
import { MetricPill, SectionCard, StatusBadge } from "./workflowShell";
import { waitForWailsBindings } from "./wailsReady";

export default function AllocationLoadingPage() {
	const navigate = useNavigate();
	const location = useLocation();
	const [status, setStatus] = useState("Iniciando alocação...");
	const [combinations, setCombinations] = useState<number | null>(null);

	useEffect(() => {
		const run = async () => {
			try {
				await waitForWailsBindings();

				const params = location.state?.params || {
					GruposPorHorario: 5,
					MinPessoasPorGrupo: 5,
					MaxPessoasPorGrupo: 8,
					AvaliadoresPorGrupo: 5,
					SoftCriteria: [],
				};

				setStatus("Montando configuração normalizada da alocação...");
				const config = await BuildAllocationConfigurationFromDatabase(params);
				const totalPeople = 50;
				const scheduleCount = config?.normalized?.preferenceMappings?.length || 1;
				const combos = await CountPossibleAllocationQuantitiesAcrossSchedules(
					config.normalized.params,
					totalPeople,
					scheduleCount
				);
				setCombinations(combos);
				setStatus(`Calculando entre ${combos.toLocaleString()} jeitos possíveis de alocação...`);

				const result = await RunAllocation(config);
				navigate("/allocation-result", { state: { result } });
			} catch (err) {
				setStatus("Erro na alocação: " + err);
			}
		};
		run();
	}, [navigate, location]);

	return (
		<div className="space-y-6">
			<SectionCard
				title="Processamento em andamento"
				description="O sistema está analisando a configuração escolhida e calculando a melhor distribuição possível dentro das regras atuais."
				aside={<StatusBadge tone="accent">Execução ativa</StatusBadge>}
			>
				<div className="grid gap-4 md:grid-cols-3">
					<MetricPill label="Status" value="Em processamento" tone="accent" />
					<MetricPill
						label="Combinações"
						value={combinations ? combinations.toLocaleString() : "Calculando..."}
						tone="success"
					/>
					<MetricPill label="Ação do usuário" value="Aguardar" />
				</div>
			</SectionCard>

			<div className="grid gap-6 lg:grid-cols-[1fr_0.9fr]">
				<SectionCard
					title="Leitura atual"
					description="O algoritmo primeiro estima o espaço combinatório e depois executa a alocação final."
				>
					<div className="rounded-[30px] border border-[rgba(18,48,71,0.12)] bg-[linear-gradient(145deg,rgba(255,251,245,0.92),rgba(236,228,216,0.92))] p-8">
						<div className="flex flex-col items-center text-center">
							<div className="relative flex h-32 w-32 items-center justify-center">
								<div className="absolute inset-0 animate-spin rounded-full border-4 border-[rgba(178,122,68,0.18)] border-t-[var(--accent-strong)]" />
								<div className="absolute inset-4 animate-pulse rounded-full border border-[rgba(18,48,71,0.14)]" />
								<div className="rounded-full border border-[rgba(18,48,71,0.14)] bg-white/80 px-4 py-3 text-xs font-semibold uppercase tracking-[0.18em] text-[var(--accent-strong)]">
									Executando
								</div>
							</div>
							<h2 className="mt-8 text-3xl text-[var(--text)]">{status}</h2>
							<p className="mt-4 max-w-2xl text-sm leading-7 text-[var(--muted)]">
								Por favor aguarde. Quando a execução terminar, a interface abrirá automaticamente o painel final de resultados.
							</p>
						</div>
					</div>
				</SectionCard>

				<SectionCard
					title="O que esperar"
					description="A próxima tela consolida a leitura executiva da rodada."
				>
					<div className="space-y-4">
						<div className="rounded-[24px] border border-[var(--line)] bg-white/70 p-5">
							<div className="text-xs uppercase tracking-[0.18em] text-[var(--accent-strong)]">
								Resultado principal
							</div>
							<p className="mt-3 text-sm leading-6 text-[var(--muted)]">
								Mesas organizadas por horário, avaliadores por grupo e candidatos destacados por filtros.
							</p>
						</div>
						<div className="rounded-[24px] border border-[var(--line)] bg-white/70 p-5">
							<div className="text-xs uppercase tracking-[0.18em] text-[var(--accent-strong)]">
								Exceções
							</div>
							<p className="mt-3 text-sm leading-6 text-[var(--muted)]">
								O painel de não alocados mostrará quem ficou fora da distribuição, ajudando a revisar a capacidade configurada.
							</p>
						</div>
					</div>
				</SectionCard>
			</div>
		</div>
	);
}
