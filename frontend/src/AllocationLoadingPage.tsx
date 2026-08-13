import React, { useEffect, useState } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import {
	BuildAllocationConfigurationFromDatabase,
	RunAllocation,
	GetWorkflowDefinition,
} from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";
import { MetricPill, SectionCard, StatusBadge } from "./workflowShell";
import { waitForWailsBindings } from "./wailsReady";

type SolverProgress = {
	percent: number;
	branchesResolved: string;
	totalBranches: string;
	branchesPruned: string;
	nodesVisited: number;
	prunedSubtrees: number;
};

const initialProgress: SolverProgress = {
	percent: 0,
	branchesResolved: "0",
	totalBranches: "0",
	branchesPruned: "0",
	nodesVisited: 0,
	prunedSubtrees: 0,
};

function formatBranchCount(value: string) {
	try {
		return BigInt(value).toLocaleString("pt-BR");
	} catch {
		return value;
	}
}

export default function AllocationLoadingPage() {
	const navigate = useNavigate();
	const location = useLocation();
	const [status, setStatus] = useState("Iniciando alocação...");
	const [progress, setProgress] = useState<SolverProgress>(initialProgress);

	useEffect(() => {
		let active = true;
		let stopListening = () => {};
		const run = async () => {
			try {
				await waitForWailsBindings();
				if (!active) return;
				stopListening = EventsOn("allocation:progress", (snapshot: SolverProgress) => {
					if (!active) return;
					setProgress(snapshot);
					setStatus(snapshot.prunedSubtrees > 0
						? "Analisando e descartando branches sem solução..."
						: "Analisando branches de alocação...");
				});

				let params = location.state?.params;
				if (!params) {
					const workflow = await GetWorkflowDefinition();
					params = workflow.defaultAllocationParams;
				}

				setStatus("Montando configuração normalizada da alocação...");
				const config = await BuildAllocationConfigurationFromDatabase(params);
				setStatus("Preparando a árvore de branches da alocação...");

				const result = await RunAllocation(config);
				if (active) navigate("/allocation-result", { state: { result } });
			} catch (err) {
				if (active) setStatus("Erro na alocação: " + err);
			}
		};
		run();
		return () => {
			active = false;
			stopListening();
		};
	}, [navigate, location]);

	const percentage = Math.min(100, Math.max(0, progress.percent));

	return (
		<div className="space-y-6" data-testid="allocation-loading-page">
			<SectionCard
				title="Processamento em andamento"
				description="O sistema está analisando a configuração escolhida e calculando a melhor distribuição possível dentro das regras atuais."
				aside={<StatusBadge tone="accent">Execução ativa</StatusBadge>}
			>
				<div className="grid gap-4 md:grid-cols-3">
					<MetricPill label="Status" value="Em processamento" tone="accent" />
					<MetricPill label="Branches checadas" value={progress.nodesVisited.toLocaleString("pt-BR")} tone="success" />
					<MetricPill label="Ação do usuário" value="Aguardar" />
				</div>
			</SectionCard>

			<div className="grid gap-6 lg:grid-cols-[1fr_0.9fr]">
				<SectionCard
					title="Leitura atual"
					description="A barra avança conforme branches são verificadas ou eliminadas com segurança pelas podas do solver."
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
							<div className="mt-8 w-full max-w-2xl" data-testid="allocation-progress">
								<div className="mb-3 flex items-end justify-between gap-4">
									<div className="text-left">
										<div className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--muted)]">Espaço de busca resolvido</div>
										<div className="mt-1 text-sm text-[var(--muted)]" data-testid="allocation-progress-count">
											{formatBranchCount(progress.branchesResolved)} de {formatBranchCount(progress.totalBranches)} branches
										</div>
									</div>
									<div className="text-3xl font-semibold tabular-nums text-[var(--accent)]" data-testid="allocation-progress-percent">
										{percentage.toFixed(1)}%
									</div>
								</div>
								<div
									className="h-4 overflow-hidden rounded-full border border-[rgba(94,106,210,0.2)] bg-white/80 shadow-inner"
									role="progressbar"
									aria-label="Progresso da alocação"
									aria-valuemin={0}
									aria-valuemax={100}
									aria-valuenow={percentage}
								>
									<div
										className="h-full rounded-full bg-[linear-gradient(90deg,var(--accent),#8b94ed)] transition-[width] duration-300 ease-out"
										style={{ width: `${percentage}%` }}
									/>
								</div>
								<div className="mt-4 grid gap-3 text-left sm:grid-cols-2">
									<div className="rounded-xl border border-[var(--line)] bg-white/70 px-4 py-3">
										<div className="text-xs text-[var(--muted)]">Branches eliminadas por poda</div>
										<div className="mt-1 font-semibold tabular-nums" data-testid="allocation-pruned-branches">{formatBranchCount(progress.branchesPruned)}</div>
									</div>
									<div className="rounded-xl border border-[var(--line)] bg-white/70 px-4 py-3">
										<div className="text-xs text-[var(--muted)]">Subárvores podadas</div>
										<div className="mt-1 font-semibold tabular-nums">{progress.prunedSubtrees.toLocaleString("pt-BR")}</div>
									</div>
								</div>
							</div>
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
