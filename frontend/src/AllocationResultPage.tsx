import React, { useMemo, useState } from "react";
import { HelpIcon } from "./components/Tooltip";
import { useLocation } from "react-router-dom";
import {
	EmptyState,
	FieldLabel,
	
	MetricPill,
	SectionCard,
	StatusBadge,
} from "./workflowShell";

export default function AllocationResultPage() {
	const location = useLocation();
	const result = location.state?.result;

	const [selectedSemestre, setSelectedSemestre] = useState("");
	const [selectedCurso, setSelectedCurso] = useState("");
	const [searchTerm, setSearchTerm] = useState("");

	if (!result) {
		return (
			<SectionCard
				title="Resultados indisponíveis"
				description="Nenhum resultado de alocação foi encontrado nesta navegação."
			>
				<EmptyState
					title="Sem resultados de alocação disponíveis."
					description="Volte para a configuração e execute uma nova simulação para gerar o painel final."
				/>
			</SectionCard>
		);
	}

	const allCandidates = result.mesas ? result.mesas.flatMap((mesa: any) => mesa.candidatos) : [];
	if (result.naoAlocados) {
		allCandidates.push(...result.naoAlocados);
	}

	const semestres = useMemo(() => {
		const values = new Set<string>();
		allCandidates.forEach((candidate: any) => {
			if (candidate.semestre) {
				values.add(candidate.semestre.toString());
			}
		});
		return Array.from(values);
	}, [result]);

	const cursos = useMemo(() => {
		const values = new Set<string>();
		allCandidates.forEach((candidate: any) => {
			if (candidate.curso) {
				values.add(candidate.curso);
			}
		});
		return Array.from(values);
	}, [result]);

	const groupedByHorario: Record<string, any[]> = {};
	if (result.mesas) {
		result.mesas.forEach((mesa: any) => {
			if (!groupedByHorario[mesa.horario]) {
				groupedByHorario[mesa.horario] = [];
			}
			groupedByHorario[mesa.horario].push(mesa);
		});
	}

	const getCandidateClassName = (candidate: any) => {
		if (searchTerm) {
			const searchLower = searchTerm.toLowerCase();
			const textToSearch = `${candidate.nome} ${candidate.curso} ${candidate.semestre}`.toLowerCase();
			if (textToSearch.includes(searchLower)) {
				return "border-[rgba(178,122,68,0.28)] bg-[rgba(178,122,68,0.12)] text-[var(--accent-strong)]";
			}
			return "border-[var(--line)] bg-white/60 text-[rgba(24,35,45,0.45)] opacity-55";
		}

		if (selectedSemestre && selectedCurso) {
			if (candidate.semestre?.toString() === selectedSemestre && candidate.curso === selectedCurso) {
				return "border-[rgba(156,66,63,0.22)] bg-[rgba(156,66,63,0.12)] text-[var(--danger)]";
			}
			return "border-[var(--line)] bg-white/72 text-[var(--text)]";
		}
		if (selectedSemestre && candidate.semestre?.toString() === selectedSemestre) {
			return "border-[rgba(18,48,71,0.2)] bg-[rgba(18,48,71,0.08)] text-[var(--text)]";
		}
		if (selectedCurso && candidate.curso === selectedCurso) {
			return "border-[rgba(37,100,84,0.2)] bg-[var(--success-soft)] text-[var(--success)]";
		}

		return "border-[var(--line)] bg-white/72 text-[var(--text)]";
	};

	return (
		<div className="space-y-6">
			<SectionCard
				title="Painel executivo da rodada"
				description="Use os filtros para inspecionar a distribuição por perfil e identifique rapidamente exceções operacionais."
				aside={
					<StatusBadge tone={result.status === "Sucesso!" ? "success" : "accent"}>
						Status: {result.status || "Finalizado"}
					</StatusBadge>
				}
			>
				<div className="grid gap-4 md:grid-cols-4">
					<MetricPill label="Mesas criadas" value={result.mesas?.length || 0} tone="accent" />
					<MetricPill label="Horários" value={Object.keys(groupedByHorario).length} />
					<MetricPill label="Não alocados" value={result.naoAlocados?.length || 0} tone={(result.naoAlocados?.length || 0) > 0 ? "danger" : "success"} />
					<MetricPill label="Score" value={result.score?.totalPenalty ?? 0} tone="success" />
				</div>
			</SectionCard>

			<SectionCard
				title="Diagnóstico do solver"
				description="Leitura técnica da execução real usada para chegar na distribuição exibida."
				aside={<StatusBadge tone={result.solverStatus === "optimal" ? "success" : "danger"}>{result.solverStatus || "finalizado"}</StatusBadge>}
			>
				<div className="grid gap-4 md:grid-cols-4">
					<MetricPill label="Estados completos" value={result.metrics?.completeStates ?? 0} />
					<MetricPill label="Nós visitados" value={result.metrics?.nodesVisited ?? 0} />
					<MetricPill label="Podas hard" value={result.metrics?.nodesPrunedByHard ?? 0} />
					<MetricPill label="Podas por score" value={result.metrics?.nodesPrunedByBound ?? 0} />
				</div>

				{result.hardViolations?.length ? (
					<div className="mt-6 rounded-[24px] border border-[rgba(156,66,63,0.18)] bg-[rgba(156,66,63,0.1)] p-5">
						<div className="text-xs uppercase tracking-[0.18em] text-[var(--danger)]">
							Violações obrigatórias
						</div>
						<ul className="mt-3 space-y-2 text-sm leading-6 text-[var(--danger)]">
							{result.hardViolations.map((violation: any, index: number) => (
								<li key={`${violation.code}-${index}`}>{violation.message || violation.code}</li>
							))}
						</ul>
					</div>
				) : null}

				{result.score?.components?.length ? (
					<div className="mt-6 grid gap-3 lg:grid-cols-2">
						{result.score.components.slice(0, 6).map((component: any, index: number) => (
							<div key={`${component.code}-${index}`} className="rounded-[20px] border border-[var(--line)] bg-white/70 px-4 py-3">
								<div className="flex items-center justify-between gap-3">
									<span className="text-sm font-semibold text-[var(--text)]">{component.code}</span>
									<span className="text-sm font-bold text-[var(--accent-strong)]">{component.penalty}</span>
								</div>
								<p className="mt-2 text-xs leading-5 text-[var(--muted)]">{component.message}</p>
							</div>
						))}
					</div>
				) : null}
			</SectionCard>

			<SectionCard
				title="Destacar Candidatos"
				description="Aplique filtros para revisar diversidade, distribuição e exceções sem alterar os dados."
			>
				<div className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
					<div>
						<FieldLabel
							label="Pesquisa livre"
							description="Busca por nome, curso ou semestre."
							help="Ao usar a busca livre, os filtros suaves de semestre e curso são limpos para destacar apenas o texto pesquisado."
						/>
						<input
							type="text"
							placeholder="Pesquisa livre..."
							className="executive-input"
							value={searchTerm}
							onChange={(e) => {
								setSearchTerm(e.target.value);
								setSelectedSemestre("");
								setSelectedCurso("");
							}}
						/>
					</div>

					<div className="rounded-[24px] border border-[var(--line)] bg-[rgba(178,122,68,0.08)] px-5 py-4">
						<div className="flex items-center gap-2 text-xs uppercase tracking-[0.18em] text-[var(--accent-strong)]">
							Legenda de destaque
							<HelpIcon text="Semestre e curso podem ser usados separadamente ou juntos. Quando ambos estiverem ativos, o sistema destaca a combinação com maior ênfase." />
						</div>
						<p className="mt-3 text-sm leading-6 text-[var(--muted)]">
							Busca livre destaca em dourado. Semestre destaca em azul-ardósia. Curso destaca em verde. A combinação semestre + curso destaca em vermelho de exceção.
						</p>
					</div>
				</div>

				<div className="mt-6 grid gap-6 md:grid-cols-2">
					<div>
						<div className="mb-3 text-sm font-semibold text-[var(--text)]">Por Semestre:</div>
						<div className="flex flex-wrap gap-2">
							<button
								onClick={() => setSelectedSemestre("")}
								className={`rounded-full border px-4 py-2 text-sm font-semibold ${
									!selectedSemestre
										? "border-[rgba(18,48,71,0.18)] bg-[linear-gradient(135deg,#173955_0%,#123047_100%)] text-[#fff8ee]"
										: "border-[var(--line)] bg-white/70 text-[var(--text)]"
								}`}
							>
								Todos
							</button>
							{semestres.map((semestre) => (
								<button
									key={semestre}
									onClick={() => setSelectedSemestre(semestre)}
									className={`rounded-full border px-4 py-2 text-sm font-semibold ${
										selectedSemestre === semestre
											? "border-[rgba(18,48,71,0.18)] bg-[rgba(18,48,71,0.08)] text-[var(--text)]"
											: "border-[var(--line)] bg-white/70 text-[var(--text)] hover:bg-[rgba(18,48,71,0.05)]"
									}`}
								>
									{semestre}º Semestre
								</button>
							))}
						</div>
					</div>
					<div>
						<div className="mb-3 text-sm font-semibold text-[var(--text)]">Por Curso:</div>
						<div className="flex flex-wrap gap-2">
							<button
								onClick={() => setSelectedCurso("")}
								className={`rounded-full border px-4 py-2 text-sm font-semibold ${
									!selectedCurso
										? "border-[rgba(18,48,71,0.18)] bg-[linear-gradient(135deg,#173955_0%,#123047_100%)] text-[#fff8ee]"
										: "border-[var(--line)] bg-white/70 text-[var(--text)]"
								}`}
							>
								Todos
							</button>
							{cursos.map((curso) => (
								<button
									key={curso}
									onClick={() => setSelectedCurso(curso)}
									className={`rounded-full border px-4 py-2 text-sm font-semibold ${
										selectedCurso === curso
											? "border-[rgba(37,100,84,0.2)] bg-[var(--success-soft)] text-[var(--success)]"
											: "border-[var(--line)] bg-white/70 text-[var(--text)] hover:bg-[rgba(37,100,84,0.06)]"
									}`}
								>
									{curso}
								</button>
							))}
						</div>
					</div>
				</div>

				{selectedSemestre || selectedCurso ? (
					<div className="mt-6 rounded-[22px] border border-[var(--line)] bg-white/70 px-5 py-4 text-sm text-[var(--muted)]">
						<span className="font-semibold text-[var(--text)]">Regra Ativa:</span>{" "}
						{selectedSemestre && selectedCurso ? (
							<span>
								Destacando em <b className="text-[var(--danger)]">vermelho</b> quem é do {selectedSemestre}º semestre e de {selectedCurso}.
							</span>
						) : selectedSemestre ? (
							<span>
								Destacando em <b className="text-[var(--text)]">azul-ardósia</b> quem é do {selectedSemestre}º semestre.
							</span>
						) : (
							<span>
								Destacando em <b className="text-[var(--success)]">verde</b> quem é de {selectedCurso}.
							</span>
						)}
					</div>
				) : null}
			</SectionCard>

			<div className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
				<div className="space-y-6">
					{Object.keys(groupedByHorario).length === 0 ? (
						<SectionCard
							title="Mesas por horário"
							description="Nenhum grupo foi formado nesta execução."
						>
							<EmptyState
								title="Nenhum grupo foi formado."
								description="Revise a configuração de capacidade e execute a alocação novamente."
							/>
						</SectionCard>
					) : (
						Object.keys(groupedByHorario).map((horario) => (
							<SectionCard
								key={horario}
								title={`Horário: ${horario}`}
								description="Cada card representa uma mesa com candidatos e avaliadores vinculados."
							>
								<div className="grid gap-5 lg:grid-cols-2">
									{groupedByHorario[horario].map((mesa: any) => (
										<div
											key={mesa.id}
											className="overflow-hidden rounded-[28px] border border-[var(--line)] bg-white/76"
										>
											<div className="border-b border-[var(--line)] bg-[rgba(18,48,71,0.04)] px-5 py-4">
												<h3 className="text-2xl text-[var(--text)]">{mesa.descricao}</h3>
											</div>
											<div className="p-5">
												<div className="text-xs uppercase tracking-[0.18em] text-[var(--muted)]">
													Candidatos
												</div>
												<ul className="mt-4 space-y-3">
													{mesa.candidatos?.map((candidate: any) => (
														<li
															key={candidate.id}
															className={`flex items-center justify-between gap-4 rounded-[20px] border px-4 py-3 text-sm ${getCandidateClassName(candidate)}`}
														>
															<span className="font-semibold">{candidate.nome}</span>
															<span className="text-xs opacity-80">
																{candidate.curso} ({candidate.semestre}º)
															</span>
														</li>
													))}
												</ul>
											</div>
											<div className="border-t border-[rgba(18,48,71,0.1)] bg-[rgba(178,122,68,0.08)] px-5 py-4">
												<div className="text-xs uppercase tracking-[0.18em] text-[var(--accent-strong)]">
													Avaliadores
												</div>
												<ul className="mt-3 space-y-2">
													{mesa.avaliadores?.map((avaliador: any) => (
														<li key={avaliador.id} className="rounded-[18px] border border-[rgba(18,48,71,0.12)] bg-white/70 px-4 py-3 text-sm font-semibold text-[var(--text)]">
															{avaliador.nome}
														</li>
													))}
												</ul>
											</div>
										</div>
									))}
								</div>
							</SectionCard>
						))
					)}
				</div>

				<div className="space-y-6">
					<SectionCard
						title="Não Alocados"
						description="Leitura direta das exceções da rodada."
						aside={
							<StatusBadge tone={(result.naoAlocados?.length || 0) > 0 ? "danger" : "success"}>
								{result.naoAlocados?.length || 0} registro(s)
							</StatusBadge>
						}
					>
						{!result.naoAlocados || result.naoAlocados.length === 0 ? (
							<EmptyState
								title="Todos foram alocados!"
								description="A configuração atual conseguiu distribuir todos os candidatos disponíveis."
							/>
						) : (
							<ul className="space-y-3">
								{result.naoAlocados.map((candidate: any) => (
									<li
										key={candidate.id}
										className="rounded-[22px] border border-[rgba(156,66,63,0.18)] bg-[rgba(156,66,63,0.1)] px-4 py-4"
									>
										<div className="font-bold text-[var(--danger)]">{candidate.nome}</div>
										<div className="mt-1 text-sm text-[var(--danger)]">
											{candidate.curso} - {candidate.semestre}º Sem.
										</div>
									</li>
								))}
							</ul>
						)}
					</SectionCard>
				</div>
			</div>
		</div>
	);
}
