import { PlusIcon, XMarkIcon } from "@heroicons/react/24/outline";
import { CriterioAlocacao, TipoCriterio, ValorColuna } from "../api";
import { BotaoSecundario, cn, Secao, Selo } from "./ui";

const MAX_CRITERIOS = 5;

const TIPOS: { tipo: TipoCriterio; nome: string; usaLimite?: boolean; usaValores?: "opcional" | "obrigatorio" }[] = [
	{ tipo: "misturar", nome: "Misturar" },
	{ tipo: "agrupar", nome: "Agrupar" },
	{ tipo: "maximo", nome: "No máximo N por mesa", usaLimite: true, usaValores: "opcional" },
	{ tipo: "minimo", nome: "Se aparecer, pelo menos N por mesa", usaLimite: true, usaValores: "opcional" },
	{ tipo: "um_de_cada", nome: "Pelo menos um de cada", usaValores: "obrigatorio" },
];

const PESOS = [
	{ peso: 1, nome: "Baixa" },
	{ peso: 3, nome: "Média" },
	{ peso: 10, nome: "Alta" },
];

const COLUNAS = [
	{ coluna: "curso", nome: "Curso", artigo: "o mesmo curso" },
	{ coluna: "semestre", nome: "Semestre", artigo: "o mesmo semestre" },
] as const;

// Frase que explica o critério como ele está configurado.
function explicar(c: CriterioAlocacao): string {
	const col = c.coluna === "curso" ? "curso" : "semestre";
	const quais = c.valores.length ? c.valores.join(", ") : `qualquer ${col}`;
	switch (c.tipo) {
		case "misturar":
			return `Evita pôr na mesma mesa candidatos com ${COLUNAS.find((x) => x.coluna === c.coluna)!.artigo}.`;
		case "agrupar":
			return `Tenta juntar na mesma mesa candidatos com ${COLUNAS.find((x) => x.coluna === c.coluna)!.artigo}.`;
		case "maximo":
			return `Cada mesa tem no máximo ${c.limite || "N"} candidato(s) de ${quais}.`;
		case "minimo":
			return `Se ${quais} aparecer numa mesa, que sejam pelo menos ${c.limite || "N"} (ninguém fica sozinho).`;
		case "um_de_cada":
			return c.valores.length ? `Cada mesa tem pelo menos um candidato de cada: ${quais}.` : `Escolha os valores de ${col} abaixo.`;
	}
}

export default function EditorCriterios({
	criterios,
	onChange,
	valoresColunas,
}: {
	criterios: CriterioAlocacao[];
	onChange: (c: CriterioAlocacao[]) => void;
	valoresColunas: Record<string, ValorColuna[]>;
}) {
	const alterar = (i: number, mudanca: Partial<CriterioAlocacao>) =>
		onChange(criterios.map((c, j) => (j === i ? { ...c, ...mudanca } : c)));

	return (
		<Secao
			rotulo="Critérios adicionais"
			titulo={
				<>
					Critérios adicionais <Selo className="ml-1 align-middle">opcional</Selo>
				</>
			}
			subtitulo="Regras sobre curso ou semestre. Elas pesam junto com as preferências de horário: com importância alta, a alocação pode colocar alguém numa opção de horário pior para cumprir a regra."
		>
			<div className="space-y-3">
				{criterios.map((c, i) => {
					const def = TIPOS.find((t) => t.tipo === c.tipo)!;
					const valores = valoresColunas[c.coluna] ?? [];
					return (
						<div
							key={i}
							data-testid="criterio"
							className="space-y-3 rounded-[var(--radius-lg)] border border-[var(--line)] bg-[var(--bg-subtle)] p-4"
						>
							<div className="flex flex-wrap items-end gap-3">
								<label className="block">
									<span className="rotulo-caps block">Regra</span>
									<select
										value={c.tipo}
										onChange={(e) => alterar(i, { tipo: e.target.value as TipoCriterio })}
										className="campo mt-1 w-auto py-1.5"
									>
										{TIPOS.map((t) => (
											<option key={t.tipo} value={t.tipo}>
												{t.nome}
											</option>
										))}
									</select>
								</label>
								<label className="block">
									<span className="rotulo-caps block">Coluna</span>
									<select
										value={c.coluna}
										onChange={(e) => alterar(i, { coluna: e.target.value as CriterioAlocacao["coluna"], valores: [] })}
										className="campo mt-1 w-auto py-1.5"
									>
										{COLUNAS.map((x) => (
											<option key={x.coluna} value={x.coluna}>
												{x.nome}
											</option>
										))}
									</select>
								</label>
								{def.usaLimite && (
									<label className="block">
										<span className="rotulo-caps block">N por mesa</span>
										<input
											type="number"
											min={1}
											value={c.limite || ""}
											onChange={(e) => alterar(i, { limite: Number(e.target.value) })}
											className="campo mt-1 w-20 py-1.5 tabular-nums"
										/>
									</label>
								)}
								<label className="block">
									<span className="rotulo-caps block">Importância</span>
									<select
										value={c.peso}
										onChange={(e) => alterar(i, { peso: Number(e.target.value) })}
										className="campo mt-1 w-auto py-1.5"
									>
										{PESOS.map((p) => (
											<option key={p.peso} value={p.peso}>
												{p.nome}
											</option>
										))}
									</select>
								</label>
								<button
									type="button"
									onClick={() => onChange(criterios.filter((_, j) => j !== i))}
									aria-label="Remover critério"
									className="ml-auto rounded-md p-1.5 text-[var(--text-light)] transition-colors hover:bg-[var(--danger-soft)] hover:text-[var(--danger)]"
								>
									<XMarkIcon className="h-4 w-4" />
								</button>
							</div>

							{def.usaValores && (
								<div>
									<div className="rotulo-caps mb-1.5">
										{def.usaValores === "opcional" ? "Valores (nenhum marcado = todos)" : "Valores"}
									</div>
									<div className="flex flex-wrap gap-1.5">
										{valores.map((v) => {
											const marcado = c.valores.includes(v.valor);
											return (
												<button
													key={v.valor}
													type="button"
													aria-pressed={marcado}
													onClick={() =>
														alterar(i, {
															valores: marcado ? c.valores.filter((x) => x !== v.valor) : [...c.valores, v.valor],
														})
													}
													className={cn(
														"rounded-full border px-2.5 py-1 text-xs transition-colors",
														marcado
															? "border-[var(--text)] bg-[var(--text)] text-white"
															: "border-[var(--line-strong)] bg-white hover:border-[var(--text-muted)]"
													)}
												>
													{c.coluna === "semestre" ? `${v.valor}º` : v.valor}{" "}
													<span className={marcado ? "text-white/60" : "text-[var(--text-light)]"}>{v.quantidade}</span>
												</button>
											);
										})}
									</div>
								</div>
							)}

							<p className="text-sm text-[var(--text-muted)]">{explicar(c)}</p>
						</div>
					);
				})}

				<BotaoSecundario
					disabled={criterios.length >= MAX_CRITERIOS}
					onClick={() => onChange([...criterios, { tipo: "misturar", coluna: "curso", valores: [], limite: 2, peso: 3 }])}
				>
					<PlusIcon className="h-4 w-4" />
					Adicionar critério {criterios.length >= MAX_CRITERIOS && `(máximo ${MAX_CRITERIOS})`}
				</BotaoSecundario>
			</div>
		</Secao>
	);
}
