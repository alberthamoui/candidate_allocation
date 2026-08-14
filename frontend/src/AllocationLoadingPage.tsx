import React, { useEffect, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { BuildAllocationConfigurationFromDatabase, GetWorkflowDefinition } from "../wailsjs/go/main/App";
import { MetricPill, SectionCard, StatusBadge } from "./workflowShell";
import { waitForWailsBindings } from "./wailsReady";
import { useAllocationRun } from "./AllocationRunContext";

function formatPossibilityCount(value: string) {
	try {
		return BigInt(value).toLocaleString("pt-BR");
	} catch {
		return value;
	}
}

export default function AllocationLoadingPage() {
	const navigate = useNavigate();
	const location = useLocation();
	const { result, progress, error, start } = useAllocationRun();
	const [status, setStatus] = useState("Preparando a busca...");
	const started = useRef(false);

	useEffect(() => {
		let active = true;
		const run = async () => {
			try {
				await waitForWailsBindings();
				if (!active) return;
				let params = location.state?.params;
				if (!params) {
					const workflow = await GetWorkflowDefinition();
					params = workflow.defaultAllocationParams;
				}
				setStatus("Montando configuração da alocação...");
				const config = await BuildAllocationConfigurationFromDatabase(params);
				if (!active) return;
				started.current = true;
				setStatus("Buscando a primeira solução viável...");
				await start(config);
			} catch (runError) {
				if (active) setStatus("Erro na alocação: " + runError);
			}
		};
		run();
		return () => { active = false; };
	}, [location.state, start]);

	useEffect(() => {
		if (started.current && result) navigate("/allocation-result");
	}, [navigate, result]);

	const percentage = Math.min(100, Math.max(0, progress.percent));
	const displayedStatus = error ? "Erro na alocação: " + error : status;

	return (
		<div className="space-y-6" data-testid="allocation-loading-page">
			<SectionCard
				title="Busca em andamento"
				description="A primeira solução válida será exibida assim que encontrada; a verificação de alternativas continuará em segundo plano."
				aside={<StatusBadge tone="accent">Execução ativa</StatusBadge>}
			>
				<div className="grid gap-4 md:grid-cols-3">
					<MetricPill label="Status" value="Procurando solução" tone="accent" />
					<MetricPill label="Possibilidades totais" value={formatPossibilityCount(progress.totalBranches)} />
					<MetricPill label="Próxima etapa" value="Exibir a primeira solução" tone="success" />
				</div>
			</SectionCard>

			<SectionCard title="Progresso da verificação" description="O total é fixo para esta execução. Podas contam como possibilidades analisadas com segurança.">
				<div className="mx-auto max-w-3xl rounded-[30px] border border-[rgba(18,48,71,0.12)] bg-white/75 p-8 text-center">
					<div className="mx-auto h-16 w-16 animate-spin rounded-full border-4 border-[rgba(178,122,68,0.18)] border-t-[var(--accent-strong)]" />
					<h2 className="mt-6 text-2xl text-[var(--text)]">{displayedStatus}</h2>
					<div className="mt-8" data-testid="allocation-progress">
						<div className="mb-3 flex items-end justify-between gap-4">
							<div className="text-left">
								<div className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--muted)]">Possibilidades totais</div>
								<div className="mt-1 text-lg font-semibold" data-testid="allocation-total-possibilities">{formatPossibilityCount(progress.totalBranches)}</div>
							</div>
							<div className="text-3xl font-semibold tabular-nums text-[var(--accent)]" data-testid="allocation-progress-percent">{percentage.toFixed(1)}%</div>
						</div>
						<div
							className="h-4 overflow-hidden rounded-full border border-[rgba(94,106,210,0.2)] bg-white shadow-inner"
							role="progressbar"
							aria-label="Progresso da alocação"
							aria-valuemin={0}
							aria-valuemax={100}
							aria-valuenow={percentage}
						>
							<div className="h-full rounded-full bg-[linear-gradient(90deg,var(--accent),#8b94ed)] transition-[width] duration-300" style={{ width: `${percentage}%` }} />
						</div>
					</div>
					<p className="mt-5 text-sm leading-6 text-[var(--muted)]">Você não precisa esperar a prova de optimalidade para visualizar uma alocação válida.</p>
				</div>
			</SectionCard>
		</div>
	);
}
