import React, { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { TrashIcon, PlusIcon } from "@heroicons/react/24/outline";
import { GetCriteriaOptions } from "../wailsjs/go/main/App";
import {
	EmptyState,
	FieldLabel,
	HelpHint,
	MetricPill,
	PrimaryButton,
	SectionCard,
	SecondaryButton,
	StatusBadge,
	StickyActionBar,
} from "./workflowShell";
import { waitForWailsBindings } from "./wailsReady";

interface SoftCriterionDraft {
	type: string;
	columnKey: string;
	selectedValues: string[];
	threshold: number;
}

const CRITERION_OPTIONS = [
	{ value: "max_value", label: "Valor Máximo Permitido" },
	{ value: "min_value", label: "Valor Mínimo Necessário" },
	{ value: "at_least_one_each", label: "Pelo Menos Um de Cada" },
	{ value: "balanced_distribution", label: "Distribuição Balanceada" },
	{ value: "group_together", label: "Agrupar Valores Iguais" },
];

const CRITERION_HELP: Record<string, string> = {
	max_value:
		"Tenta limitar quantas pessoas de um valor específico podem aparecer juntas em um grupo.",
	min_value:
		"Tenta manter um mínimo do valor selecionado quando ele aparecer dentro de um grupo.",
	at_least_one_each:
		"Prioriza a presença de pelo menos um representante de cada valor selecionado.",
	balanced_distribution:
		"Busca espalhar os valores selecionados de forma mais homogênea entre os grupos.",
	group_together:
		"Prefere aproximar pessoas com o mesmo valor selecionado na mesma composição.",
};

function criterionNeedsThreshold(type: string) {
	return type === "max_value" || type === "min_value";
}

export default function AllocationConfigPage() {
	const navigate = useNavigate();

	const [gruposPorHorario, setGrupos] = useState(2);
	const [minPessoas, setMin] = useState(4);
	const [maxPessoas, setMax] = useState(8);
	const [avaliadoresPorGrupo, setAvaliadores] = useState(3);
	const [criteria, setCriteria] = useState<SoftCriterionDraft[]>([]);
	const [criteriaOptions, setCriteriaOptions] = useState<Record<string, string[]>>({});

	useEffect(() => {
		waitForWailsBindings()
			.then(() => GetCriteriaOptions())
			.then((opts) => {
				setCriteriaOptions(opts || {});
			})
			.catch((err) => {
				console.error("Erro ao carregar opções de critério:", err);
			});
	}, []);

	const columns = useMemo(() => Object.keys(criteriaOptions).sort(), [criteriaOptions]);

	const addCriterion = () => {
		setCriteria((current) => [
			...current,
			{
				type: "max_value",
				columnKey: columns[0] || "curso",
				selectedValues: [],
				threshold: 1,
			},
		]);
	};

	const removeCriterion = (index: number) => {
		setCriteria((current) => current.filter((_, i) => i !== index));
	};

	const updateCriterion = (
		index: number,
		field: keyof SoftCriterionDraft,
		value: SoftCriterionDraft[keyof SoftCriterionDraft]
	) => {
		setCriteria((current) =>
			current.map((criterion, i) => {
				if (i !== index) {
					return criterion;
				}

				const next = { ...criterion, [field]: value };
				if (field === "columnKey") {
					next.selectedValues = [];
				}
				if (field === "type" && !criterionNeedsThreshold(String(value))) {
					next.threshold = 0;
				}
				if (field === "type" && criterionNeedsThreshold(String(value)) && next.threshold < 1) {
					next.threshold = 1;
				}
				return next;
			})
		);
	};

	const toggleValue = (index: number, val: string) => {
		setCriteria((current) =>
			current.map((criterion, i) => {
				if (i !== index) {
					return criterion;
				}
				const selected = criterion.selectedValues.includes(val)
					? criterion.selectedValues.filter((item) => item !== val)
					: [...criterion.selectedValues, val];
				return { ...criterion, selectedValues: selected };
			})
		);
	};

	const capacityPerHorario = gruposPorHorario * maxPessoas;
	const minCapacityPerHorario = gruposPorHorario * minPessoas;
	const impactSummary =
		criteria.length === 0
			? "Sem critérios soft, o algoritmo terá uma busca mais direta e orientada apenas pelos parâmetros base."
			: `${criteria.length} critério${
					criteria.length > 1 ? "s" : ""
			  } soft ativo${criteria.length > 1 ? "s" : ""} aumentam o refinamento da distribuição e podem reduzir o conjunto de soluções preferíveis.`;

	const handleStart = () => {
		const formattedCriteria = criteria.map((criterion) => ({
			Type: criterion.type,
			ColumnKey: criterion.columnKey,
			SelectedValues: criterion.selectedValues,
			Threshold: parseInt(String(criterion.threshold), 10) || 0,
		}));

		const params = {
			GruposPorHorario: gruposPorHorario,
			MinPessoasPorGrupo: minPessoas,
			MaxPessoasPorGrupo: maxPessoas,
			AvaliadoresPorGrupo: avaliadoresPorGrupo,
			SoftCriteria: formattedCriteria,
		};

		navigate("/allocation-loading", { state: { params } });
	};

	return (
		<div className="space-y-6">
			<SectionCard
				title="Parâmetros centrais"
				description="Defina a estrutura básica da rodada. Cada decisão altera a capacidade dos grupos e o espaço de soluções do algoritmo."
				aside={<StatusBadge tone="accent">Configuração ativa</StatusBadge>}
			>
				<div className="grid gap-4 md:grid-cols-3">
					<MetricPill label="Capacidade máxima por horário" value={`${capacityPerHorario} candidatos`} tone="accent" />
					<MetricPill label="Capacidade mínima por horário" value={`${minCapacityPerHorario} candidatos`} />
					<MetricPill label="Critérios soft" value={criteria.length} tone={criteria.length > 0 ? "success" : "neutral"} />
				</div>
			</SectionCard>

			<div className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
				<div className="space-y-6">
					<SectionCard
						title="Capacidade e composição"
						description="Esses números determinam quantos grupos serão montados por horário e o volume de pessoas por grupo."
					>
						<div className="grid gap-5 md:grid-cols-2">
							<div>
								<FieldLabel
									label="Grupos por Horário"
									description="Quantidade de mesas que o sistema pode formar em cada faixa de horário."
									help="Mais grupos por horário aumentam a capacidade total, mas também multiplicam as combinações possíveis."
								/>
								<input
									type="number"
									className="executive-input"
									value={gruposPorHorario}
									onChange={(e) => setGrupos(parseInt(e.target.value, 10) || 0)}
								/>
							</div>
							<div>
								<FieldLabel
									label="Avaliadores por Grupo"
									description="Número de avaliadores associados a cada mesa formada."
									help="Esse valor influencia o dimensionamento operacional da rodada e precisa refletir a disponibilidade real."
								/>
								<input
									type="number"
									className="executive-input"
									value={avaliadoresPorGrupo}
									onChange={(e) => setAvaliadores(parseInt(e.target.value, 10) || 0)}
								/>
							</div>
							<div>
								<FieldLabel
									label="Mínimo de Candidatos"
									description="Limite inferior por grupo. Usado para evitar mesas subutilizadas."
									help="Aumentar esse valor exige grupos mais cheios e pode inviabilizar parte das combinações."
								/>
								<input
									type="number"
									className="executive-input"
									value={minPessoas}
									onChange={(e) => setMin(parseInt(e.target.value, 10) || 0)}
								/>
							</div>
							<div>
								<FieldLabel
									label="Máximo de Candidatos"
									description="Limite superior por grupo. Usado para controlar densidade e capacidade."
									help="Reduzir o máximo deixa a distribuição mais controlada, mas pode aumentar não alocados se a capacidade total ficar curta."
								/>
								<input
									type="number"
									className="executive-input"
									value={maxPessoas}
									onChange={(e) => setMax(parseInt(e.target.value, 10) || 0)}
								/>
							</div>
						</div>
					</SectionCard>

					<SectionCard
						title="Critérios Soft"
						description="Use regras desejáveis para orientar a distribuição sem transformar cada preferência em bloqueio absoluto."
						aside={
							<PrimaryButton onClick={addCriterion} className="justify-center" data-testid="add-criterion-button">
								<PlusIcon className="h-4 w-4" />
								Adicionar Regra
							</PrimaryButton>
						}
					>
						{criteria.length === 0 ? (
							<EmptyState
								title="Nenhum critério soft configurado"
								description="Sem critérios soft, o algoritmo seguirá apenas as capacidades e regras obrigatórias. Adicione uma regra quando quiser refinar diversidade, agrupamento ou limites desejáveis."
								action={
									<SecondaryButton onClick={addCriterion}>
										<PlusIcon className="h-4 w-4" />
										Adicionar
									</SecondaryButton>
								}
							/>
						) : (
							<div className="space-y-5">
								{criteria.map((criterion, idx) => {
									const availableValues = criteriaOptions[criterion.columnKey] || [];
									return (
										<div
											key={`${criterion.columnKey}-${idx}`}
											className="rounded-[28px] border border-[var(--line)] bg-white/74 p-5"
										>
											<div className="flex flex-col gap-4 border-b border-[var(--line)] pb-5 lg:flex-row lg:items-start lg:justify-between">
												<div>
													<div className="flex items-center gap-2 text-xs uppercase tracking-[0.18em] text-[var(--accent-strong)]">
														Regra {idx + 1}
														<HelpHint
															label="Tipo de regra"
															content={CRITERION_HELP[criterion.type] || "Regra desejável aplicada à distribuição."}
														/>
													</div>
													<p className="mt-2 text-sm leading-6 text-[var(--muted)]">
														{CRITERION_HELP[criterion.type] || "Regra desejável aplicada à distribuição."}
													</p>
												</div>
												<button
													onClick={() => removeCriterion(idx)}
													className="inline-flex items-center gap-2 rounded-full border border-[rgba(156,66,63,0.18)] bg-[rgba(156,66,63,0.1)] px-4 py-2 text-sm font-semibold text-[var(--danger)] transition-colors hover:bg-[rgba(156,66,63,0.16)]"
													title="Remover Regra"
												>
													<TrashIcon className="h-4 w-4" />
													Remover
												</button>
											</div>

											<div className="mt-5 grid gap-5 xl:grid-cols-[1fr_1fr_1fr_auto]">
												<div>
													<FieldLabel label="Tipo de Regra" description="Define o comportamento desejado da distribuição." />
													<select
														className="executive-input"
														value={criterion.type}
														onChange={(e) => updateCriterion(idx, "type", e.target.value)}
													>
														{CRITERION_OPTIONS.map((option) => (
															<option key={option.value} value={option.value}>
																{option.label}
															</option>
														))}
													</select>
												</div>
												<div>
													<FieldLabel
														label="Coluna de referência"
														description="Campo usado como base para a regra."
														help="As opções vêm das colunas detectadas pelo backend para critérios soft."
													/>
													<select
														className="executive-input"
														value={criterion.columnKey}
														onChange={(e) => updateCriterion(idx, "columnKey", e.target.value)}
													>
														{columns.map((column) => (
															<option key={column} value={column}>
																{column}
															</option>
														))}
														{columns.length === 0 ? <option value="curso">curso</option> : null}
													</select>
												</div>
												<div>
													<FieldLabel
														label="Limite"
														description="Usado apenas em regras de mínimo e máximo."
														help="Quando a regra não usa limite numérico, esse campo é apenas informativo."
													/>
													<input
														type="number"
														className="executive-input"
														value={criterion.threshold}
														disabled={!criterionNeedsThreshold(criterion.type)}
														onChange={(e) =>
															updateCriterion(idx, "threshold", parseInt(e.target.value, 10) || 0)
														}
													/>
												</div>
												<div className="rounded-[24px] border border-[var(--line)] bg-[rgba(178,122,68,0.08)] px-4 py-4">
													<div className="text-xs uppercase tracking-[0.18em] text-[var(--accent-strong)]">
														Impacto
													</div>
													<p className="mt-2 text-sm leading-6 text-[var(--muted)]">
														{criterionNeedsThreshold(criterion.type)
															? `O algoritmo tentará respeitar o limite ${criterion.threshold} para os valores escolhidos.`
															: "O algoritmo tentará orientar a composição conforme a regra escolhida, sem usar limite numérico."}
													</p>
												</div>
											</div>

											<div className="mt-5">
												<FieldLabel
													label="Valores a aplicar"
													description="Selecione os valores desta coluna que serão observados por esta regra."
													help="Esses chips controlam exatamente quais valores entram na lógica do critério."
												/>
												{availableValues.length === 0 ? (
													<div className="rounded-[22px] border border-dashed border-[var(--line-strong)] bg-white/60 px-4 py-4 text-sm text-[var(--muted)]">
														Nenhum valor mapeado para essa coluna.
													</div>
												) : (
													<div className="flex flex-wrap gap-3">
														{availableValues.map((val) => {
															const isSelected = criterion.selectedValues.includes(val);
															return (
																<button
																	key={val}
																	onClick={() => toggleValue(idx, val)}
																	className={`rounded-full border px-4 py-2 text-sm font-semibold transition-colors ${
																		isSelected
																			? "border-[rgba(18,48,71,0.18)] bg-[linear-gradient(135deg,#173955_0%,#123047_100%)] text-[#fff8ee]"
																			: "border-[var(--line)] bg-white/78 text-[var(--text)] hover:bg-[rgba(178,122,68,0.08)]"
																	}`}
																>
																	{val}
																</button>
															);
														})}
													</div>
												)}
											</div>
										</div>
									);
								})}
							</div>
						)}
					</SectionCard>
				</div>

				<div className="space-y-6">
					<SectionCard
						title="Leitura de impacto"
						description="Resumo rápido para apoiar a decisão antes da execução."
					>
						<div className="space-y-4">
							<div className="rounded-[24px] border border-[var(--line)] bg-white/70 p-5">
								<div className="text-xs uppercase tracking-[0.18em] text-[var(--accent-strong)]">
									Capacidade operacional
								</div>
								<p className="mt-3 text-sm leading-6 text-[var(--muted)]">
									Com {gruposPorHorario} grupo{gruposPorHorario !== 1 ? "s" : ""} por horário, cada rodada comporta entre {minCapacityPerHorario} e {capacityPerHorario} candidato{capacityPerHorario !== 1 ? "s" : ""}.
								</p>
							</div>
							<div className="rounded-[24px] border border-[var(--line)] bg-white/70 p-5">
								<div className="text-xs uppercase tracking-[0.18em] text-[var(--accent-strong)]">
									Refinamento do algoritmo
								</div>
								<p className="mt-3 text-sm leading-6 text-[var(--muted)]">{impactSummary}</p>
							</div>
							<div className="rounded-[24px] border border-[var(--line)] bg-white/70 p-5">
								<div className="flex items-center gap-2 text-xs uppercase tracking-[0.18em] text-[var(--accent-strong)]">
									Boas práticas
									<HelpHint
										label="Boas práticas"
										content="Comece com parâmetros realistas e poucos critérios soft. Depois refine com base na leitura do resultado."
									/>
								</div>
								<ul className="mt-3 space-y-3 text-sm leading-6 text-[var(--muted)]">
									<li>Mantenha o mínimo e máximo coerentes com a capacidade real de entrevistas.</li>
									<li>Use critérios soft apenas quando houver um objetivo claro de composição.</li>
									<li>Revise os não alocados no resultado para recalibrar a configuração da próxima rodada.</li>
								</ul>
							</div>
						</div>
					</SectionCard>
				</div>
			</div>

			<StickyActionBar>
				<div>
					<div className="text-xs uppercase tracking-[0.18em] text-[var(--muted)]">
						Pronto para executar
					</div>
					<p className="mt-2 text-sm leading-6 text-[var(--muted)]">
						Revise a leitura de impacto e inicie a simulação quando os parâmetros refletirem a operação desejada.
					</p>
				</div>
				<PrimaryButton onClick={handleStart} className="px-8 py-4 text-base" data-testid="start-allocation-button">
					Iniciar Simulação
				</PrimaryButton>
			</StickyActionBar>
		</div>
	);
}
