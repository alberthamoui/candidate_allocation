import React, { useMemo, useState } from "react";
import { useLocation } from "react-router-dom";
import { Tooltip } from "./components/Tooltip";
import {
	EmptyState,
	MetricPill,
	PrimaryButton,
	SecondaryButton,
	SectionCard,
	StatusBadge,
} from "./workflowShell";
import { useAllocationRun } from "./AllocationRunContext";

type PersonSelection =
	| { kind: "candidate"; person: any }
	| { kind: "evaluator"; person: any }
	| null;

const normalizeFilterValue = (value: unknown) =>
	String(value ?? "").trim().toLocaleLowerCase("pt-BR");

const formatLargeCount = (value: string) => {
	try {
		return BigInt(value).toLocaleString("pt-BR");
	} catch {
		return value;
	}
};

const uniqueDisplayValues = (values: unknown[]) => {
	const byNormalizedValue = new Map<string, string>();
	values.forEach((value) => {
		const displayValue = String(value ?? "").trim();
		const normalizedValue = normalizeFilterValue(displayValue);
		if (normalizedValue && !byNormalizedValue.has(normalizedValue)) {
			byNormalizedValue.set(normalizedValue, displayValue);
		}
	});
	return Array.from(byNormalizedValue.values()).sort((left, right) =>
		left.localeCompare(right, "pt-BR", { numeric: true }),
	);
};

const qualityToneClass = (tone: string) => {
	if (tone === "success") return "border-emerald-200 bg-emerald-50 text-emerald-900";
	if (tone === "warning") return "border-amber-200 bg-amber-50 text-amber-950";
	return "border-slate-200 bg-slate-50 text-slate-900";
};

function SearchIcon() {
	return (
		<svg viewBox="0 0 24 24" aria-hidden="true" className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="2">
			<circle cx="11" cy="11" r="7" />
			<path d="m20 20-3.5-3.5" />
		</svg>
	);
}

function PersonDetailsPopover({ selection, onClose }: { selection: PersonSelection; onClose: () => void }) {
	if (!selection) return null;
	const { kind, person } = selection;
	const extras = Object.entries(person.extras || {}).filter(([, value]) => String(value || "").trim());

	return (
		<div
			role="dialog"
			aria-label={`Informações de ${person.nome}`}
			data-testid="person-details-tooltip"
			className="fixed bottom-16 right-5 z-[70] w-[min(390px,calc(100vw-2rem))] rounded-2xl border border-slate-200 bg-white p-5 shadow-2xl"
		>
			<div className="flex items-start justify-between gap-4">
				<div>
					<div className="text-[11px] font-semibold uppercase tracking-[0.16em] text-[var(--accent)]">
						{kind === "candidate" ? "Candidato" : "Avaliador"}
					</div>
					<h3 className="mt-1 text-lg font-semibold text-slate-950">{person.nome}</h3>
				</div>
				<button type="button" onClick={onClose} aria-label="Fechar informações" className="rounded-lg p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-700">✕</button>
			</div>

			<div className="mt-4 space-y-4 text-sm text-slate-700">
				{kind === "candidate" ? (
					<>
						<div className="grid grid-cols-2 gap-3 rounded-xl bg-slate-50 p-3">
							<div><span className="block text-xs text-slate-500">Curso</span><b>{person.curso || "—"}</b></div>
							<div><span className="block text-xs text-slate-500">Semestre</span><b>{person.semestre ? `${person.semestre}º` : "—"}</b></div>
						</div>
						<div>
							<div className="text-xs font-semibold uppercase tracking-wide text-slate-500">Preferências de horário</div>
							{person.opcoes?.length ? (
								<ol className="mt-2 space-y-1">
									{person.opcoes.map((option: string, index: number) => <li key={`${option}-${index}`}><b>{index + 1}.</b> {option}</li>)}
								</ol>
							) : <p className="mt-1 text-slate-500">Nenhuma preferência informada.</p>}
						</div>
						{person.emailSecundario || person.emailPessoal ? <div className="break-all text-xs text-slate-500">{person.emailSecundario || person.emailPessoal}</div> : null}
					</>
				) : (
					<>
						<div className="grid grid-cols-2 gap-3 rounded-xl bg-slate-50 p-3">
							<div><span className="block text-xs text-slate-500">Sigla</span><b>{person.sigla || "—"}</b></div>
							<div className="break-all"><span className="block text-xs text-slate-500">E-mail</span><b>{person.email || "—"}</b></div>
						</div>
						<div><span className="font-semibold">Não pode avaliar:</span> {person.naoPosso?.join(", ") || "nenhum candidato"}</div>
						<div><span className="font-semibold">Prefere não avaliar:</span> {person.prefiroNao?.join(", ") || "nenhum candidato"}</div>
					</>
				)}
				{kind === "candidate" && (person.naoPosso?.length || person.prefiroNao?.length) ? (
					<div className="rounded-xl border border-amber-200 bg-amber-50 p-3 text-amber-950">
						{person.naoPosso?.length ? <div><b>Não pode:</b> {person.naoPosso.join(", ")}</div> : null}
						{person.prefiroNao?.length ? <div><b>Prefiro não:</b> {person.prefiroNao.join(", ")}</div> : null}
					</div>
				) : null}
				{extras.length ? (
					<div className="border-t border-slate-100 pt-3">
						{extras.map(([key, value]) => <div key={key}><span className="font-semibold">{key}:</span> {String(value)}</div>)}
					</div>
				) : null}
			</div>
		</div>
	);
}

export default function AllocationResultPage() {
	const location = useLocation();
	const allocationRun = useAllocationRun();
	const result = allocationRun.result || location.state?.result;
	const [continueConfirmed, setContinueConfirmed] = useState(false);
	const [selectedSemestre, setSelectedSemestre] = useState("");
	const [selectedCurso, setSelectedCurso] = useState("");
	const [searchTerm, setSearchTerm] = useState("");
	const [activeQualityCode, setActiveQualityCode] = useState("");
	const [personSelection, setPersonSelection] = useState<PersonSelection>(null);

	const allCandidates = useMemo(() => {
		if (!result) return [];
		return [
			...(result.mesas?.flatMap((mesa: any) => mesa.candidatos || []) || []),
			...(result.naoAlocados || []),
		];
	}, [result]);

	const semestres = useMemo(() => uniqueDisplayValues(allCandidates.map((candidate: any) => candidate.semestre)), [allCandidates]);
	const cursos = useMemo(() => uniqueDisplayValues(allCandidates.map((candidate: any) => candidate.curso)), [allCandidates]);
	const groupedByHorario = useMemo(() => {
		const groups: Record<string, any[]> = {};
		result?.mesas?.forEach((mesa: any) => {
			groups[mesa.horario] = [...(groups[mesa.horario] || []), mesa];
		});
		return groups;
	}, [result]);
	const qualityCharacteristics = result?.quality?.characteristics || [];
	const activeQuality = qualityCharacteristics.find((characteristic: any) => characteristic.code === activeQualityCode);
	const activeQualityCandidateIDs = new Set<number>(activeQuality?.candidateIds || []);

	if (!result) {
		return (
			<SectionCard title="Resultados indisponíveis" description="Nenhum resultado de alocação foi encontrado nesta navegação.">
				<EmptyState title="Sem resultados de alocação disponíveis." description="Volte para a configuração e execute uma nova simulação para gerar o painel final." />
			</SectionCard>
		);
	}

	const resetQualityHighlight = () => setActiveQualityCode("");
	const activateQuality = (characteristic: any) => {
		const nextCode = activeQualityCode === characteristic.code ? "" : characteristic.code;
		setActiveQualityCode(nextCode);
		setSearchTerm("");
		setSelectedSemestre("");
		setSelectedCurso("");
		if (nextCode && characteristic.candidateIds?.length) {
			window.setTimeout(() => {
				document.querySelector(`[data-candidate-id="${characteristic.candidateIds[0]}"]`)?.scrollIntoView({ behavior: "smooth", block: "center" });
			}, 0);
		}
	};
	const getCandidateClassName = (candidate: any) => {
		if (activeQualityCode) {
			return activeQualityCandidateIDs.has(Number(candidate.id))
				? "border-violet-400 bg-violet-100 text-violet-950 ring-2 ring-violet-300 shadow-sm"
				: "border-slate-100 bg-white text-slate-400 opacity-45";
		}
		if (searchTerm.trim()) {
			const searchable = normalizeFilterValue(`${candidate.nome} ${candidate.curso} ${candidate.semestre}`);
			return searchable.includes(normalizeFilterValue(searchTerm))
				? "border-indigo-300 bg-indigo-50 text-indigo-950 ring-1 ring-indigo-200"
				: "border-slate-100 bg-white text-slate-400 opacity-45";
		}
		const semesterMatches = !selectedSemestre || normalizeFilterValue(candidate.semestre) === normalizeFilterValue(selectedSemestre);
		const courseMatches = !selectedCurso || normalizeFilterValue(candidate.curso) === normalizeFilterValue(selectedCurso);
		if (selectedSemestre || selectedCurso) {
			return semesterMatches && courseMatches
				? "border-emerald-300 bg-emerald-50 text-emerald-950 ring-1 ring-emerald-200"
				: "border-slate-100 bg-white text-slate-400 opacity-45";
		}
		return "border-slate-200 bg-white text-slate-900 hover:border-indigo-200 hover:bg-indigo-50/40";
	};
	const scoreExplanation = (
		<div className="space-y-2 text-left">
			<div className="font-semibold">Como o score foi calculado</div>
			<p className="text-slate-600">Quanto menor, melhor. O valor soma as penalidades das preferências e dos critérios adicionais.</p>
			{result.score?.components?.filter((component: any) => component.penalty > 0).slice(0, 6).map((component: any, index: number) => (
				<div key={`${component.code}-${index}`} className="border-t border-slate-100 pt-2"><b>+{component.penalty}</b> {component.message}</div>
			))}
			{!result.score?.components?.some((component: any) => component.penalty > 0) ? <div className="text-emerald-700">Nenhuma penalidade aplicada.</div> : null}
		</div>
	);

	return (
		<div className="space-y-6" data-testid="allocation-result-page">
		{allocationRun.running ? (
			<SectionCard
				title="Solução válida encontrada — verificação continua"
				description="Esta distribuição já pode ser usada. O solver continua analisando as possibilidades restantes para tentar reduzir o score e provar a melhor solução."
				aside={<StatusBadge tone="accent">Verificando alternativas</StatusBadge>}
				className="border-indigo-200 ring-1 ring-indigo-100"
			>
				<div className="grid gap-5 lg:grid-cols-[1fr_auto] lg:items-end">
					<div>
						<div className="mb-3 flex items-end justify-between gap-4">
							<div>
								<div className="text-xs font-semibold uppercase tracking-[0.16em] text-[var(--muted)]">Possibilidades totais</div>
								<div className="mt-1 font-semibold" data-testid="result-total-possibilities">{formatLargeCount(allocationRun.progress.totalBranches)}</div>
							</div>
							<div className="text-2xl font-semibold text-[var(--accent)]" data-testid="result-progress-percent">{Math.min(100, Math.max(0, allocationRun.progress.percent)).toFixed(1)}%</div>
						</div>
						<div className="h-3 overflow-hidden rounded-full bg-slate-100" role="progressbar" aria-label="Verificação de outras soluções" aria-valuemin={0} aria-valuemax={100} aria-valuenow={allocationRun.progress.percent}>
							<div className="h-full rounded-full bg-[linear-gradient(90deg,var(--accent),#8b94ed)] transition-[width] duration-300" style={{ width: `${Math.min(100, Math.max(0, allocationRun.progress.percent))}%` }} />
						</div>
						{continueConfirmed ? <p className="mt-3 text-sm font-medium text-indigo-800">A verificação continuará; esta tela será atualizada quando surgir uma solução melhor.</p> : null}
					</div>
					<div className="flex flex-col gap-3 sm:flex-row lg:flex-col">
						<PrimaryButton type="button" data-testid="continue-search-button" onClick={() => setContinueConfirmed(true)}>Continuar verificando</PrimaryButton>
						<SecondaryButton type="button" data-testid="stop-search-button" onClick={() => allocationRun.stop()}>Parar e ficar com esta solução</SecondaryButton>
					</div>
				</div>
			</SectionCard>
		) : result.solverStatus === "feasible" ? (
			<SectionCard title="Verificação interrompida" description="A melhor solução encontrada foi preservada, mas a optimalidade não foi comprovada." aside={<StatusBadge tone="success">Solução mantida</StatusBadge>}>
				<p className="text-sm text-[var(--muted)]">Você pode usar esta alocação normalmente ou iniciar uma nova execução para continuar refinando.</p>
			</SectionCard>
		) : null}
			<SectionCard
				title="Resumo da alocação"
				description={allocationRun.running ? "Melhor solução válida encontrada até agora." : "Visão geral da rodada concluída."}
				aside={<StatusBadge tone={result.status === "Sucesso!" ? "success" : "accent"}>{result.status || "Finalizado"}</StatusBadge>}
			>
				<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
					<MetricPill label="Mesas criadas" value={result.mesas?.length || 0} tone="accent" />
					<MetricPill label="Horários" value={Object.keys(groupedByHorario).length} />
					<MetricPill label="Não alocados" value={result.naoAlocados?.length || 0} tone={(result.naoAlocados?.length || 0) > 0 ? "danger" : "success"} />
					<Tooltip content={scoreExplanation}>
						<div data-testid="score-metric" className="minimal-panel flex w-full cursor-help flex-col items-start gap-1 border-indigo-200 p-4 ring-1 ring-indigo-100">
							<div className="flex items-center gap-2 text-xs font-medium text-gray-500">Score <span className="rounded-full bg-indigo-100 px-1.5 text-[10px] text-indigo-700">?</span></div>
							<div className="text-xl font-semibold text-gray-900">{result.score?.totalPenalty ?? 0}</div>
						</div>
					</Tooltip>
				</div>
			</SectionCard>

			<SectionCard title="Qualidade da alocação" description="Clique em uma característica para destacar na alocação os candidatos relacionados.">
				<div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
					{qualityCharacteristics.map((characteristic: any) => {
						const active = activeQualityCode === characteristic.code;
						return (
							<button
								type="button"
								key={characteristic.code}
								onClick={() => activateQuality(characteristic)}
								data-testid={`quality-${characteristic.code}`}
								aria-pressed={active}
								className={`rounded-2xl border p-4 text-left transition-all hover:-translate-y-0.5 hover:shadow-md ${qualityToneClass(characteristic.tone)} ${active ? "ring-2 ring-violet-500 ring-offset-2" : ""}`}
							>
								<div className="flex items-start justify-between gap-3">
									<div className="text-sm font-semibold">{characteristic.label}</div>
									<div className="whitespace-nowrap text-lg font-bold">{characteristic.value} <span className="text-xs font-medium">{characteristic.valueLabel}</span></div>
								</div>
								<p className="mt-2 text-xs leading-5 opacity-75">{characteristic.description}</p>
								{characteristic.penalty > 0 ? <div className="mt-2 text-xs font-semibold">+{characteristic.penalty} no score</div> : null}
							</button>
						);
					})}
				</div>
				{activeQuality ? (
					<div className="mt-4 flex items-center justify-between rounded-xl border border-violet-200 bg-violet-50 px-4 py-3 text-sm text-violet-950">
						<span><b>Destaque ativo:</b> {activeQuality.label} ({activeQuality.candidateIds?.length || 0} candidato(s))</span>
						<button type="button" onClick={resetQualityHighlight} className="font-semibold underline underline-offset-2">Limpar</button>
					</div>
				) : null}
			</SectionCard>

			<SectionCard title="Encontrar e destacar candidatos" description="Pesquise por texto ou combine semestre e curso.">
				<label htmlFor="candidate-search" className="mb-2 block text-sm font-semibold text-slate-800">Pesquisar candidato</label>
				<div className="relative">
					<div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4 text-slate-400"><SearchIcon /></div>
					<input
						id="candidate-search"
						type="search"
						placeholder="Digite nome, curso ou semestre..."
						className="minimal-input h-12 bg-white pl-12 pr-4 text-base shadow-sm"
						data-testid="candidate-search-input"
						value={searchTerm}
						onChange={(event) => {
							setSearchTerm(event.target.value);
							setSelectedSemestre("");
							setSelectedCurso("");
							resetQualityHighlight();
						}}
					/>
				</div>
				<div className="mt-5 grid gap-5 lg:grid-cols-2">
					<div>
						<div className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-500">Semestre</div>
						<div className="flex flex-wrap gap-2">
							<button type="button" onClick={() => { setSelectedSemestre(""); setSearchTerm(""); resetQualityHighlight(); }} className={`rounded-full border px-3 py-1.5 text-sm font-medium ${!selectedSemestre ? "border-slate-900 bg-slate-900 text-white" : "border-slate-200 bg-white"}`}>Todos</button>
							{semestres.map((semestre) => <button type="button" key={semestre} data-testid={`semester-filter-${semestre}`} onClick={() => { setSelectedSemestre(semestre); setSearchTerm(""); resetQualityHighlight(); }} className={`rounded-full border px-3 py-1.5 text-sm font-medium ${normalizeFilterValue(selectedSemestre) === normalizeFilterValue(semestre) ? "border-indigo-600 bg-indigo-600 text-white" : "border-slate-200 bg-white hover:border-indigo-300"}`}>{semestre}º</button>)}
						</div>
					</div>
					<div>
						<div className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-500">Curso</div>
						<div className="flex flex-wrap gap-2">
							<button type="button" onClick={() => { setSelectedCurso(""); setSearchTerm(""); resetQualityHighlight(); }} className={`rounded-full border px-3 py-1.5 text-sm font-medium ${!selectedCurso ? "border-slate-900 bg-slate-900 text-white" : "border-slate-200 bg-white"}`}>Todos</button>
							{cursos.map((curso) => <button type="button" key={curso} data-testid={`course-filter-${normalizeFilterValue(curso)}`} onClick={() => { setSelectedCurso(curso); setSearchTerm(""); resetQualityHighlight(); }} className={`rounded-full border px-3 py-1.5 text-sm font-medium ${normalizeFilterValue(selectedCurso) === normalizeFilterValue(curso) ? "border-emerald-700 bg-emerald-700 text-white" : "border-slate-200 bg-white hover:border-emerald-300"}`}>{curso}</button>)}
						</div>
					</div>
				</div>
			</SectionCard>

			{Object.keys(groupedByHorario).length === 0 ? (
				<SectionCard title="Mesas por horário" description="Nenhum grupo foi formado nesta execução.">
					<EmptyState title="Nenhum grupo foi formado." description="Revise a configuração de capacidade e execute a alocação novamente." />
				</SectionCard>
			) : Object.keys(groupedByHorario).map((horario) => (
				<SectionCard key={horario} title={`Horário: ${horario}`} description="Clique em qualquer pessoa para consultar informações e preferências.">
					<div className="grid gap-5 lg:grid-cols-2 xl:grid-cols-3">
						{groupedByHorario[horario].map((mesa: any) => (
							<div key={mesa.id} data-group-id={mesa.id} className="rounded-2xl border border-slate-200 bg-white shadow-sm">
								<div className="rounded-t-2xl border-b border-slate-100 bg-slate-50 px-5 py-4"><h3 className="text-base font-semibold text-slate-950">{mesa.descricao}</h3></div>
								<div className="p-4">
									<div className="text-[11px] font-semibold uppercase tracking-[0.15em] text-slate-500">Candidatos</div>
									<ul className="mt-3 space-y-2">
										{mesa.candidatos?.map((candidate: any) => (
											<li key={candidate.id} data-candidate-id={candidate.id} data-course={normalizeFilterValue(candidate.curso)} className={`rounded-xl border text-sm transition-all ${getCandidateClassName(candidate)}`}>
												<button type="button" onClick={() => setPersonSelection({ kind: "candidate", person: candidate })} className="flex w-full items-center justify-between gap-3 px-3 py-2.5 text-left">
													<span className="font-semibold">{candidate.nome}</span><span className="text-xs opacity-75">{candidate.curso} · {candidate.semestre}º</span>
												</button>
											</li>
										))}
									</ul>
								</div>
								<div className="rounded-b-2xl border-t border-indigo-100 bg-indigo-50/60 px-4 py-3">
									<div className="text-[11px] font-semibold uppercase tracking-[0.15em] text-indigo-600">Avaliadores</div>
									<ul className="mt-2 space-y-1.5">
										{mesa.avaliadores?.map((evaluator: any) => (
											<li key={evaluator.id}><button type="button" onClick={() => setPersonSelection({ kind: "evaluator", person: evaluator })} className="flex w-full items-center justify-between rounded-lg border border-indigo-100 bg-white px-3 py-2 text-left text-sm font-semibold text-slate-800 hover:border-indigo-300"><span>{evaluator.nome}</span><span className="text-xs font-medium text-indigo-500">{evaluator.sigla}</span></button></li>
										))}
									</ul>
								</div>
							</div>
						))}
					</div>
				</SectionCard>
			))}

			<SectionCard
				title="Candidatos não alocados"
				description="Exceções ficam abaixo da grade para que as mesas usem toda a largura disponível."
				aside={<StatusBadge tone={(result.naoAlocados?.length || 0) > 0 ? "danger" : "success"}>{result.naoAlocados?.length || 0} registro(s)</StatusBadge>}
			>
				{!result.naoAlocados?.length ? (
					<div className="rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm font-medium text-emerald-900">Todos os candidatos foram alocados.</div>
				) : (
					<div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
						{result.naoAlocados.map((candidate: any) => (
							<button type="button" key={candidate.id} data-candidate-id={candidate.id} onClick={() => setPersonSelection({ kind: "candidate", person: candidate })} className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-left hover:border-red-300">
								<div className="font-semibold text-red-900">{candidate.nome}</div><div className="mt-1 text-xs text-red-700">{candidate.curso} · {candidate.semestre}º semestre</div>
							</button>
						))}
					</div>
				)}
			</SectionCard>

			<details className="minimal-panel p-5 text-sm text-slate-600">
				<summary className="cursor-pointer font-semibold text-slate-800">Diagnóstico técnico do solver</summary>
				<div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
					<MetricPill label="Nós visitados" value={result.metrics?.nodesVisited ?? 0} />
					<MetricPill label="Podas hard" value={result.metrics?.nodesPrunedByHard ?? 0} />
					<MetricPill label="Podas por fluxo" value={result.metrics?.nodesPrunedByFlow ?? 0} />
					<MetricPill label="Simetrias evitadas" value={result.metrics?.branchesSkippedBySymmetry ?? 0} />
				</div>
			</details>

			<PersonDetailsPopover selection={personSelection} onClose={() => setPersonSelection(null)} />
		</div>
	);
}
