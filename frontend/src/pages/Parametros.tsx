import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ArrowLeftIcon, ArrowUturnLeftIcon, CheckCircleIcon, ExclamationTriangleIcon } from "@heroicons/react/24/outline";
import {
	getCapacidade,
	getParametrosSalvos,
	salvarParametros,
	CapacidadeResponse,
	CriterioAlocacao,
	ParametrosAlocacao,
} from "../api";
import EditorCriterios from "../components/EditorCriterios";
import { Aviso, BarraAcoes, BotaoPrimario, BotaoSecundario, CabecalhoPagina, cn, Metrica, Secao, Selo } from "../components/ui";

type Campo = Exclude<keyof ParametrosAlocacao, "criterios">; // campos numéricos

const CAMPOS: { campo: Campo; label: string; dica: string }[] = [
	{ campo: "mesas_por_horario", label: "Mesas por horário", dica: "Máximo de mesas em cada horário" },
	{ campo: "avaliadores_por_mesa", label: "Avaliadores por mesa", dica: "Cada avaliador fica em uma mesa por horário" },
	{ campo: "min_pessoas_por_mesa", label: "Mínimo de candidatos por mesa", dica: "Mesa com menos que isso não é formada" },
	{ campo: "max_pessoas_por_mesa", label: "Máximo de candidatos por mesa", dica: "" },
];

// Texto digitado → parâmetros; null se algum campo estiver vazio ou não for inteiro.
function paraParametros(valores: Record<Campo, string>, criterios: CriterioAlocacao[]): ParametrosAlocacao | null {
	const p = { criterios } as ParametrosAlocacao;
	for (const { campo } of CAMPOS) {
		const n = Number(valores[campo]);
		if (valores[campo].trim() === "" || !Number.isInteger(n)) return null;
		p[campo] = n;
	}
	return p;
}

function paraTexto(p: ParametrosAlocacao): Record<Campo, string> {
	return Object.fromEntries(CAMPOS.map(({ campo }) => [campo, String(p[campo])])) as Record<Campo, string>;
}

export default function Parametros() {
	const navigate = useNavigate();
	const [valores, setValores] = useState<Record<Campo, string> | null>(null);
	const [criterios, setCriterios] = useState<CriterioAlocacao[]>([]);
	const [padrao, setPadrao] = useState<ParametrosAlocacao | null>(null);
	const [previa, setPrevia] = useState<CapacidadeResponse | null>(null);
	const [erro, setErro] = useState<string | null>(null);
	const [carregando, setCarregando] = useState(true);
	const requisicao = useRef(0);

	// Primeira carga: padrões do backend e, se houver, os últimos parâmetros usados
	useEffect(() => {
		(async () => {
			try {
				const inicial = await getCapacidade(null);
				setPadrao(inicial.parametros);
				const salvos = getParametrosSalvos();
				setValores(paraTexto(salvos ?? inicial.parametros));
				setCriterios(salvos?.criterios ?? []);
				if (!salvos) setPrevia(inicial);
			} catch (err: any) {
				setErro(err.message);
				setCarregando(false);
			}
		})();
	}, []);

	// Recalcula a prévia a cada mudança (com um pequeno atraso enquanto digita)
	useEffect(() => {
		if (!valores) return;
		const params = paraParametros(valores, criterios);
		if (!params) {
			setErro("Preencha todos os campos com números inteiros.");
			return;
		}
		const id = ++requisicao.current;
		setCarregando(true);
		const t = setTimeout(async () => {
			try {
				const r = await getCapacidade(params);
				if (id !== requisicao.current) return;
				setPrevia(r);
				setErro(null);
			} catch (err: any) {
				if (id !== requisicao.current) return;
				setErro(err.message);
			} finally {
				if (id === requisicao.current) setCarregando(false);
			}
		}, 250);
		return () => clearTimeout(t);
	}, [valores, criterios]);

	const params = valores ? paraParametros(valores, criterios) : null;
	const podeRodar = !!params && !erro && !carregando && !!previa && previa.mesas_por_horario > 0;

	function rodar() {
		if (!params || !podeRodar) return;
		salvarParametros(params);
		navigate("/resultado");
	}

	const todosCabem = previa && previa.max_alocaveis >= previa.candidatos;
	const alterado = !!valores && !!padrao && CAMPOS.some(({ campo }) => valores[campo] !== String(padrao[campo]));

	return (
		<div className="space-y-6">
			<CabecalhoPagina
				titulo="Parâmetros da Alocação"
				descricao="Defina como as mesas são formadas. A prévia mostra o que cabe com esses valores, antes de rodar a alocação."
			/>

			<div className="grid grid-cols-1 items-start gap-6 lg:grid-cols-2">
				<div className="space-y-6">
					{valores && (
						<Secao
							titulo="Mesas"
							ajuda="O número de mesas também é limitado pela quantidade de avaliadores: cada um fica em no máximo uma mesa em cada horário."
							acoes={
								alterado && (
									<BotaoSecundario onClick={() => padrao && setValores(paraTexto(padrao))} className="px-3 py-1.5 text-xs">
										<ArrowUturnLeftIcon className="h-3.5 w-3.5" />
										Restaurar valores padrão
									</BotaoSecundario>
								)
							}
						>
							<div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
								{CAMPOS.map(({ campo, label, dica }) => (
									<label key={campo} className="block">
										<span className="rotulo">{label}</span>
										<input
											type="number"
											min={1}
											value={valores[campo]}
											onChange={(e) => setValores({ ...valores, [campo]: e.target.value })}
											className="campo tabular-nums"
										/>
										{dica && <span className="mt-1 block text-xs text-[var(--text-muted)]">{dica}</span>}
									</label>
								))}
							</div>
						</Secao>
					)}

					{valores && (
						<EditorCriterios
							criterios={criterios}
							onChange={setCriterios}
							valoresColunas={previa?.valores_colunas ?? {}}
							semPreferencia={previa?.sem_preferencia}
						/>
					)}
				</div>

				<Secao
					titulo="Prévia"
					ajuda="Calculada a cada mudança. Não roda a alocação: mostra vagas, quantos cabem pelos horários escolhidos e avisos."
					acoes={carregando && !erro ? <Selo>Atualizando...</Selo> : undefined}
				>
					<div className="space-y-4">
						{erro && (
							<Aviso tom="perigo" icone={<ExclamationTriangleIcon className="h-4 w-4" />}>
								{erro}
							</Aviso>
						)}

						{previa && (
							<div className={cn("space-y-4 transition-opacity", (carregando || erro) && "opacity-50")}>
								<div className="grid grid-cols-2 gap-3">
									<Metrica
										testId="previa-mesas"
										rotulo="Mesas por horário"
										valor={previa.mesas_por_horario}
										detalhe={
											previa.mesas_por_horario < previa.parametros.mesas_por_horario
												? `de ${previa.parametros.mesas_por_horario} pedidas`
												: undefined
										}
										tom={previa.mesas_por_horario < previa.parametros.mesas_por_horario ? "atencao" : "neutro"}
									/>
									<Metrica
										testId="previa-vagas"
										rotulo="Vagas no total"
										valor={previa.capacidade_total}
										detalhe={`${previa.capacidade_por_horario} por horário`}
									/>
									<Metrica
										testId="previa-candidatos"
										rotulo="Candidatos"
										valor={previa.candidatos}
										detalhe={`${previa.avaliadores} avaliadores`}
									/>
									<Metrica
										testId="previa-cabem"
										rotulo="Cabem no máximo"
										valor={previa.max_alocaveis}
										detalhe={previa.sem_preferencia ? "nos horários definidos" : "pelos horários escolhidos"}
										tom={todosCabem ? "sucesso" : "atencao"}
									/>
								</div>

								{previa.avisos.length > 0 ? (
									<div className="space-y-2">
										{previa.avisos.map((a, i) => (
											<Aviso key={i} tom="atencao" icone={<ExclamationTriangleIcon className="h-4 w-4" />}>
												{a}
											</Aviso>
										))}
									</div>
								) : (
									<Aviso tom="sucesso" icone={<CheckCircleIcon className="h-4 w-4" />}>
										{previa.sem_preferencia
											? "Há mesas e vagas para todos os candidatos nos horários definidos."
											: "Há mesas e vagas para todos os candidatos nos horários que escolheram."}
									</Aviso>
								)}

								<div className="overflow-x-auto rounded-[var(--radius-lg)] border border-[var(--line)]">
									<table className="w-full text-sm">
										<thead className="bg-[var(--bg-subtle)] text-left">
											<tr>
												<th className="rotulo-caps px-3 py-2">Horário</th>
												{/* sem preferência, todos podem ir em qualquer horário */}
												{!previa.sem_preferencia && (
													<>
														<th className="rotulo-caps px-3 py-2 text-right">1ª opção</th>
														<th className="rotulo-caps px-3 py-2 text-right" title="Escolheram em qualquer opção">
															Escolheram
														</th>
													</>
												)}
												<th className="rotulo-caps px-3 py-2 text-right">Vagas</th>
											</tr>
										</thead>
										<tbody className="divide-y divide-[var(--line)]">
											{previa.horarios.map((h) => {
												const lotado = h.primeira_opcao > previa.capacidade_por_horario;
												return (
													<tr key={h.descricao} className="hover:bg-[var(--bg-subtle)]">
														<td className="px-3 py-2 capitalize">{h.descricao}</td>
														{!previa.sem_preferencia && (
															<>
																<td
																	className={cn(
																		"px-3 py-2 text-right tabular-nums",
																		lotado ? "font-semibold text-[var(--warning)]" : "text-[var(--text-muted)]"
																	)}
																	title={lotado ? "Mais gente quer este horário como 1ª opção do que há vagas" : undefined}
																>
																	{h.primeira_opcao}
																</td>
																<td className="px-3 py-2 text-right tabular-nums text-[var(--text-muted)]">{h.interessados}</td>
															</>
														)}
														<td className="px-3 py-2 text-right tabular-nums text-[var(--text-muted)]">
															{previa.capacidade_por_horario}
														</td>
													</tr>
												);
											})}
										</tbody>
									</table>
								</div>
							</div>
						)}
					</div>
				</Secao>
			</div>

			<BarraAcoes dica="A alocação usa estes parâmetros e critérios.">
				<BotaoSecundario onClick={() => navigate(-1)}>
					<ArrowLeftIcon className="h-4 w-4" />
					Voltar
				</BotaoSecundario>
				<BotaoPrimario onClick={rodar} disabled={!podeRodar}>
					Rodar alocação
				</BotaoPrimario>
			</BarraAcoes>
		</div>
	);
}
