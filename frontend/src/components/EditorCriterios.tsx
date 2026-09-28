import { PlusIcon, XMarkIcon } from "@heroicons/react/24/outline";
import { CriterioAlocacao, TipoCriterio, ValorColuna } from "../api";

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
		<section aria-label="Critérios adicionais" className="space-y-3">
			<div>
				<h2 className="text-lg font-bold text-gray-800">Critérios adicionais <span className="text-sm font-normal text-gray-400">(opcional)</span></h2>
				<p className="text-sm text-gray-500">
					Regras sobre curso ou semestre. Elas pesam junto com as preferências de horário: com importância alta, a
					alocação pode colocar alguém numa opção de horário pior para cumprir a regra.
				</p>
			</div>

			{criterios.map((c, i) => {
				const def = TIPOS.find((t) => t.tipo === c.tipo)!;
				const valores = valoresColunas[c.coluna] ?? [];
				return (
					<div key={i} data-testid="criterio" className="border border-purple-200 bg-purple-50/40 rounded-xl p-4 space-y-3">
						<div className="flex flex-wrap items-end gap-3">
							<label className="block">
								<span className="text-xs font-semibold text-gray-600">Regra</span>
								<select
									value={c.tipo}
									onChange={(e) => alterar(i, { tipo: e.target.value as TipoCriterio })}
									className="mt-1 block border border-gray-300 rounded-lg px-2 py-1.5 bg-white"
								>
									{TIPOS.map((t) => (
										<option key={t.tipo} value={t.tipo}>
											{t.nome}
										</option>
									))}
								</select>
							</label>
							<label className="block">
								<span className="text-xs font-semibold text-gray-600">Coluna</span>
								<select
									value={c.coluna}
									onChange={(e) => alterar(i, { coluna: e.target.value as CriterioAlocacao["coluna"], valores: [] })}
									className="mt-1 block border border-gray-300 rounded-lg px-2 py-1.5 bg-white"
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
									<span className="text-xs font-semibold text-gray-600">N por mesa</span>
									<input
										type="number"
										min={1}
										value={c.limite || ""}
										onChange={(e) => alterar(i, { limite: Number(e.target.value) })}
										className="mt-1 block w-24 border border-gray-300 rounded-lg px-2 py-1.5"
									/>
								</label>
							)}
							<label className="block">
								<span className="text-xs font-semibold text-gray-600">Importância</span>
								<select
									value={c.peso}
									onChange={(e) => alterar(i, { peso: Number(e.target.value) })}
									className="mt-1 block border border-gray-300 rounded-lg px-2 py-1.5 bg-white"
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
								className="ml-auto p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-full"
							>
								<XMarkIcon className="w-5 h-5" />
							</button>
						</div>

						{def.usaValores && (
							<div>
								<div className="text-xs font-semibold text-gray-600 mb-1">
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
												className={`text-xs px-2.5 py-1 rounded-full border transition-colors ${
													marcado
														? "bg-purple-600 border-purple-600 text-white"
														: "bg-white border-gray-300 text-gray-700 hover:border-purple-400"
												}`}
											>
												{c.coluna === "semestre" ? `${v.valor}º` : v.valor}{" "}
												<span className={marcado ? "text-purple-200" : "text-gray-400"}>{v.quantidade}</span>
											</button>
										);
									})}
								</div>
							</div>
						)}

						<p className="text-sm text-purple-900">{explicar(c)}</p>
					</div>
				);
			})}

			<button
				type="button"
				disabled={criterios.length >= MAX_CRITERIOS}
				onClick={() => onChange([...criterios, { tipo: "misturar", coluna: "curso", valores: [], limite: 2, peso: 3 }])}
				className="flex items-center gap-1.5 text-sm font-semibold text-purple-700 hover:text-purple-900 disabled:text-gray-400 disabled:cursor-not-allowed"
			>
				<PlusIcon className="w-4 h-4" />
				Adicionar critério {criterios.length >= MAX_CRITERIOS && `(máximo ${MAX_CRITERIOS})`}
			</button>
		</section>
	);
}
