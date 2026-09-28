import { useEffect, useMemo, useState } from "react";
import {
	CheckCircleIcon,
	ExclamationTriangleIcon,
	MagnifyingGlassIcon,
	TableCellsIcon,
	UserGroupIcon,
	XMarkIcon,
} from "@heroicons/react/24/outline";
import { AlocacaoResponse, AvaliadorResultado, CandidatoResultado, ItemQualidade, MesaResult } from "../api";

type Selecao =
	| { tipo: "candidato"; pessoa: CandidatoResultado; mesa?: MesaResult }
	| { tipo: "avaliador"; pessoa: AvaliadorResultado }
	| null;

// sem caixa e sem acento, para busca e filtros
const normalizar = (s: unknown) =>
	String(s ?? "")
		.normalize("NFD")
		.replace(/[̀-ͯ]/g, "")
		.toLowerCase()
		.trim();

const distintos = (valores: (string | number)[]) =>
	Array.from(new Set(valores.map(String).filter((v) => v.trim() && v !== "0"))).sort((a, b) =>
		a.localeCompare(b, "pt-BR", { numeric: true })
	);

const TOM: Record<ItemQualidade["tom"], string> = {
	bom: "border-green-200 bg-green-50 text-green-900",
	neutro: "border-gray-200 bg-gray-50 text-gray-900",
	atencao: "border-amber-200 bg-amber-50 text-amber-900",
	ruim: "border-red-200 bg-red-50 text-red-900",
};

export default function PainelResultado({ result }: { result: AlocacaoResponse }) {
	const [busca, setBusca] = useState("");
	const [horario, setHorario] = useState("");
	const [curso, setCurso] = useState("");
	const [semestre, setSemestre] = useState("");
	const [itemAtivo, setItemAtivo] = useState<string>("");
	const [selecao, setSelecao] = useState<Selecao>(null);

	const mesas = result.mesas ?? [];
	const naoAlocados = result.nao_alocados_info ?? [];
	const qualidade = result.qualidade ?? [];
	const todos = useMemo(() => [...mesas.flatMap((m) => m.candidatos), ...naoAlocados], [result]);
	const horarios = useMemo(() => Array.from(new Set(mesas.map((m) => m.dia_nome))), [result]);
	const cursos = useMemo(() => distintos(todos.map((c) => c.curso)), [todos]);
	const semestres = useMemo(() => distintos(todos.map((c) => c.semestre)), [todos]);

	const item = qualidade.find((q) => q.codigo === itemAtivo);
	const idsDoItem = useMemo(() => new Set(item?.candidatos ?? []), [item]);
	const filtrando = !!(busca.trim() || curso || semestre || item);

	// um candidato "bate" se atende a todos os filtros ativos
	const bate = (c: CandidatoResultado) =>
		(!busca.trim() || normalizar(`${c.nome} ${c.email_insper} ${c.curso}`).includes(normalizar(busca))) &&
		(!curso || normalizar(c.curso) === normalizar(curso)) &&
		(!semestre || String(c.semestre) === semestre) &&
		(!item || idsDoItem.has(c.id));

	// só com a busca de texto ativa, ela também procura avaliadores (nome, sigla, email)
	const buscaAvaliador = !!busca.trim() && !curso && !semestre && !item;
	const bateAvaliador = (a: AvaliadorResultado) =>
		buscaAvaliador && normalizar(`${a.nome} ${a.sigla} ${a.email}`).includes(normalizar(busca));

	const mesasVisiveis = mesas.filter(
		(m) =>
			(!horario || m.dia_nome === horario) &&
			(!filtrando || m.candidatos.some(bate) || m.avaliadores.some(bateAvaliador))
	);
	const porHorario = mesasVisiveis.reduce<Record<string, MesaResult[]>>((acc, m) => {
		(acc[m.dia_nome] ||= []).push(m);
		return acc;
	}, {});
	const encontrados = filtrando
		? mesasVisiveis.flatMap((m) => m.candidatos).filter(bate).length + (horario ? 0 : naoAlocados.filter(bate).length)
		: 0;
	const avaliadoresEncontrados = new Set(
		mesasVisiveis.flatMap((m) => m.avaliadores).filter(bateAvaliador).map((a) => a.id)
	).size;
	const qtd = (n: number, um: string, varios: string) => `${n} ${n === 1 ? um : varios}`;

	function limpar() {
		setBusca("");
		setHorario("");
		setCurso("");
		setSemestre("");
		setItemAtivo("");
	}

	useEffect(() => {
		const fechar = (e: KeyboardEvent) => e.key === "Escape" && setSelecao(null);
		window.addEventListener("keydown", fechar);
		return () => window.removeEventListener("keydown", fechar);
	}, []);

	const classeCandidato = (c: CandidatoResultado) =>
		!filtrando
			? "hover:bg-blue-50"
			: bate(c)
				? "bg-yellow-100 ring-1 ring-yellow-300 font-semibold"
				: "opacity-40";

	return (
		<div className="space-y-8">
			{/* ── Resumo ─────────────────────────────────────────────────── */}
			<div className="grid grid-cols-2 sm:grid-cols-4 gap-3" data-testid="resumo">
				<Numero valor={mesas.length} rotulo="mesas" />
				<Numero valor={horarios.length} rotulo="horários com mesa" />
				<Numero valor={result.total_alocados} rotulo="candidatos alocados" />
				<Numero
					valor={result.pontuacao}
					rotulo="pontuação"
					dica={'Começa em 100 e perde pontos: 2ª opção −1, 3ª −3, 4ª −5, 5ª −7; cada avaliador "prefiro não" na mesa −5; cada candidato sem mesa −1000; critérios adicionais descontam pela importância.'}
				/>
			</div>

			{/* ── Qualidade ──────────────────────────────────────────────── */}
			<section>
				<h2 className="text-lg font-bold text-gray-900">Qualidade da alocação</h2>
				<p className="text-sm text-gray-500 mb-3">Clique em um item para destacar os candidatos dele.</p>
				<div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
					{qualidade.map((q) => {
						const ativo = itemAtivo === q.codigo;
						return (
							<button
								key={q.codigo}
								type="button"
								aria-pressed={ativo}
								data-testid={`qualidade-${q.codigo}`}
								disabled={q.valor === 0}
								onClick={() => setItemAtivo(ativo ? "" : q.codigo)}
								title={q.descricao}
								className={`rounded-xl border p-4 text-left transition-all ${TOM[q.tom]} ${
									q.valor === 0 ? "cursor-default" : "hover:shadow-md"
								} ${ativo ? "ring-2 ring-blue-500 ring-offset-2" : ""}`}
							>
								<div className="text-2xl font-bold">{q.valor}</div>
								<div className="text-sm font-semibold">{q.titulo}</div>
							</button>
						);
					})}
				</div>
			</section>

			{/* ── Busca e filtros ────────────────────────────────────────── */}
			<section className="bg-white rounded-2xl shadow p-5 space-y-4">
				<div className="relative">
					<MagnifyingGlassIcon className="w-5 h-5 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
					<input
						type="search"
						aria-label="Buscar"
						placeholder="Buscar candidato (nome, email, curso) ou avaliador (nome, sigla)..."
						value={busca}
						onChange={(e) => setBusca(e.target.value)}
						className="w-full border border-gray-300 rounded-xl pl-10 pr-4 py-2.5 focus:outline-none focus:ring-2 focus:ring-blue-300"
					/>
				</div>
				<div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
					<Filtro rotulo="Horário" valor={horario} opcoes={horarios} onChange={setHorario} />
					<Filtro rotulo="Curso" valor={curso} opcoes={cursos} onChange={setCurso} />
					<Filtro rotulo="Semestre" valor={semestre} opcoes={semestres} onChange={setSemestre} formatar={(s) => `${s}º`} />
				</div>
				{(filtrando || horario) && (
					<div className="flex items-center justify-between text-sm">
						<span className="text-gray-600" data-testid="contagem-filtro">
							{filtrando
								? qtd(encontrados, "candidato", "candidatos") +
								  (avaliadoresEncontrados ? ` e ${qtd(avaliadoresEncontrados, "avaliador", "avaliadores")}` : "") +
								  (encontrados + avaliadoresEncontrados === 1 ? " encontrado" : " encontrados")
								: `${qtd(mesasVisiveis.length, "mesa", "mesas")} no horário`}
							{item && <> · destaque: <b>{item.titulo}</b></>}
						</span>
						<button type="button" onClick={limpar} className="text-blue-600 font-semibold hover:underline">
							Limpar filtros
						</button>
					</div>
				)}
			</section>

			{/* ── Mesas por horário ──────────────────────────────────────── */}
			{mesasVisiveis.length === 0 && (filtrando || horario) ? (
				// com "sem mesa" em destaque, é esperado não haver mesas: a tabela abaixo basta
				item?.codigo !== "nao_alocados" && (
					<div className="bg-white rounded-2xl shadow p-8 text-center text-gray-500">
						Nenhuma mesa com candidatos que atendam aos filtros.
					</div>
				)
			) : (
				Object.entries(porHorario).map(([hor, ms]) => (
					<section key={hor} aria-label={`Mesas de ${hor}`}>
						<h2 className="text-lg font-bold text-gray-900 capitalize mb-3">{hor}</h2>
						<div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-5">
							{ms.map((mesa) => (
								<div key={mesa.id} data-testid="mesa" className="bg-white rounded-2xl shadow-lg border border-gray-100 overflow-hidden">
									<div className="bg-gradient-to-r from-blue-600 to-blue-700 text-white px-5 py-3 flex items-center justify-between">
										<div className="flex items-center space-x-2">
											<TableCellsIcon className="w-5 h-5" />
											<span className="font-bold capitalize">{mesa.descricao}</span>
										</div>
										<span className="bg-white/20 px-3 py-0.5 rounded-full text-sm font-semibold">
											{mesa.candidatos.length} candidato{mesa.candidatos.length !== 1 ? "s" : ""}
										</span>
									</div>
									<div className="p-5 grid grid-cols-1 sm:grid-cols-2 gap-5">
										<div>
											<Titulo icone={<UserGroupIcon className="w-4 h-4 text-blue-500" />} texto="Candidatos" />
											<ul className="space-y-1" data-testid="candidatos">
												{mesa.candidatos.map((c) => (
													<li key={c.id}>
														<button
															type="button"
															onClick={() => setSelecao({ tipo: "candidato", pessoa: c, mesa })}
															className={`w-full text-left text-sm text-gray-700 rounded-md px-1.5 py-0.5 flex items-center gap-1.5 ${classeCandidato(c)}`}
														>
															<span className="truncate">{c.nome}</span>
															{c.opcao > 1 && (
																<span className="text-[10px] font-semibold text-amber-800 bg-amber-100 px-1.5 rounded-full whitespace-nowrap">
																	{c.opcao}ª opção
																</span>
															)}
															{c.conflitos.length > 0 && (
																<ExclamationTriangleIcon className="w-4 h-4 text-amber-500 flex-shrink-0" aria-label="avaliador prefiro não na mesa" />
															)}
														</button>
													</li>
												))}
											</ul>
										</div>
										<div>
											<Titulo icone={<CheckCircleIcon className="w-4 h-4 text-green-500" />} texto="Avaliadores" />
											<ul className="space-y-1" data-testid="avaliadores">
												{mesa.avaliadores.map((a) => (
													<li key={a.id}>
														<button
															type="button"
															onClick={() => setSelecao({ tipo: "avaliador", pessoa: a })}
															className={`w-full text-left text-sm text-gray-700 rounded-md px-1.5 py-0.5 ${
																bateAvaliador(a) ? "bg-yellow-100 ring-1 ring-yellow-300 font-semibold" : "hover:bg-green-50"
															}`}
														>
															{a.nome} <span className="text-gray-400 text-xs">{a.sigla}</span>
														</button>
													</li>
												))}
											</ul>
										</div>
									</div>
								</div>
							))}
						</div>
					</section>
				))
			)}

			{/* ── Não alocados ───────────────────────────────────────────── */}
			{naoAlocados.length > 0 && !horario && (
				<section className="bg-white rounded-2xl shadow-lg border border-red-200 overflow-hidden" aria-label="Candidatos não alocados">
					<div className="bg-gradient-to-r from-red-500 to-red-600 text-white px-6 py-4 flex items-center justify-between">
						<div className="flex items-center space-x-3">
							<ExclamationTriangleIcon className="w-5 h-5" />
							<span className="font-bold text-lg">Candidatos Não Alocados</span>
						</div>
						<span className="bg-white/20 px-3 py-1 rounded-full text-sm font-semibold">
							{naoAlocados.length} candidato{naoAlocados.length !== 1 ? "s" : ""}
						</span>
					</div>
					<div className="overflow-x-auto">
						<table className="w-full text-sm">
							<thead className="bg-red-50 text-red-800">
								<tr>
									<th className="text-left px-6 py-3 font-semibold">Nome</th>
									<th className="text-left px-6 py-3 font-semibold">Curso</th>
									<th className="text-left px-6 py-3 font-semibold">Semestre</th>
									<th className="text-left px-6 py-3 font-semibold">Horários que escolheu</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-red-100">
								{naoAlocados.map((c) => (
									<tr key={c.id} className={filtrando && !bate(c) ? "opacity-40" : filtrando ? "bg-yellow-50" : ""}>
										<td className="px-6 py-3">
											<button
												type="button"
												onClick={() => setSelecao({ tipo: "candidato", pessoa: c })}
												className="font-medium text-gray-900 hover:text-blue-700 hover:underline text-left"
											>
												{c.nome}
											</button>
										</td>
										<td className="px-6 py-3 text-gray-600">{c.curso}</td>
										<td className="px-6 py-3 text-gray-600">{c.semestre ? `${c.semestre}º` : ""}</td>
										<td className="px-6 py-3 text-gray-600">{c.opcoes.join(" · ")}</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
				</section>
			)}

			<Detalhes selecao={selecao} mesas={mesas} onFechar={() => setSelecao(null)} />
		</div>
	);
}

function Numero({ valor, rotulo, dica }: { valor: number; rotulo: string; dica?: string }) {
	return (
		<div className="bg-white rounded-xl shadow p-4" title={dica}>
			<div className="text-2xl font-bold text-gray-900">{valor}</div>
			<div className="text-xs font-semibold text-gray-500">
				{rotulo}
				{dica && <span className="ml-1 text-blue-500 cursor-help">(?)</span>}
			</div>
		</div>
	);
}

function Filtro({
	rotulo,
	valor,
	opcoes,
	onChange,
	formatar = (s) => s,
}: {
	rotulo: string;
	valor: string;
	opcoes: string[];
	onChange: (v: string) => void;
	formatar?: (s: string) => string;
}) {
	return (
		<label className="block">
			<span className="text-xs font-semibold text-gray-500 uppercase tracking-wide">{rotulo}</span>
			<select
				value={valor}
				onChange={(e) => onChange(e.target.value)}
				className="mt-1 w-full border border-gray-300 rounded-lg px-3 py-2 bg-white capitalize focus:outline-none focus:ring-2 focus:ring-blue-300"
			>
				<option value="">Todos</option>
				{opcoes.map((o) => (
					<option key={o} value={o}>
						{formatar(o)}
					</option>
				))}
			</select>
		</label>
	);
}

function Titulo({ icone, texto }: { icone: React.ReactNode; texto: string }) {
	return (
		<div className="flex items-center space-x-2 mb-2">
			{icone}
			<span className="text-xs font-semibold text-gray-500 uppercase tracking-wide">{texto}</span>
		</div>
	);
}

// Painel lateral com os dados de quem foi clicado.
function Detalhes({ selecao, mesas, onFechar }: { selecao: Selecao; mesas: MesaResult[]; onFechar: () => void }) {
	if (!selecao) return null;
	const { tipo, pessoa } = selecao;
	return (
		<aside
			role="dialog"
			aria-label={`Detalhes de ${pessoa.nome}`}
			className="fixed top-4 right-4 bottom-4 z-40 w-[min(380px,calc(100vw-2rem))] bg-white rounded-2xl shadow-2xl border border-gray-200 p-6 overflow-y-auto"
		>
			<div className="flex items-start justify-between gap-3">
				<div>
					<div className="text-xs font-semibold uppercase tracking-wide text-blue-600">
						{tipo === "candidato" ? "Candidato" : "Avaliador"}
					</div>
					<h3 className="text-xl font-bold text-gray-900">{pessoa.nome}</h3>
				</div>
				<button type="button" onClick={onFechar} aria-label="Fechar detalhes" className="p-1 text-gray-400 hover:text-gray-700 hover:bg-gray-100 rounded-lg">
					<XMarkIcon className="w-5 h-5" />
				</button>
			</div>

			<div className="mt-5 space-y-5 text-sm text-gray-700">
				{tipo === "candidato" ? (
					<DetalhesCandidato c={selecao.pessoa} mesa={selecao.mesa} />
				) : (
					<DetalhesAvaliador a={selecao.pessoa} mesas={mesas.filter((m) => m.avaliadores.some((x) => x.id === pessoa.id))} />
				)}
			</div>
		</aside>
	);
}

function DetalhesCandidato({ c, mesa }: { c: CandidatoResultado; mesa?: MesaResult }) {
	return (
		<>
			<div className="grid grid-cols-2 gap-3 bg-gray-50 rounded-xl p-3">
				<Campo rotulo="Curso" valor={c.curso} />
				<Campo rotulo="Semestre" valor={c.semestre ? `${c.semestre}º` : ""} />
				<div className="col-span-2">
					<Campo rotulo="Email" valor={c.email_insper} />
				</div>
			</div>
			<div className={`rounded-xl p-3 ${mesa ? "bg-blue-50 text-blue-900" : "bg-red-50 text-red-900"}`}>
				{mesa ? (
					<>
						Alocado em <b className="capitalize">{mesa.descricao}</b>, a sua <b>{c.opcao}ª opção</b>.
					</>
				) : (
					<>Sem mesa: não coube em nenhum dos horários que escolheu.</>
				)}
			</div>
			<div>
				<div className="text-xs font-semibold uppercase tracking-wide text-gray-500 mb-1">Horários que escolheu</div>
				{c.opcoes.length ? (
					<ol className="space-y-1">
						{c.opcoes.map((o, i) => (
							<li key={i} className={`capitalize ${mesa && i + 1 === c.opcao ? "font-bold text-blue-700" : ""}`}>
								{i + 1}. {o} {mesa && i + 1 === c.opcao && <span className="normal-case">← alocado</span>}
							</li>
						))}
					</ol>
				) : (
					<p className="text-gray-500">Nenhum horário informado.</p>
				)}
			</div>
			{c.conflitos.length > 0 && (
				<div className="rounded-xl border border-amber-200 bg-amber-50 p-3 text-amber-900">
					Ficou na mesa de <b>{c.conflitos.join(", ")}</b>, que marcou "prefiro não" para ele.
				</div>
			)}
			<Lista rotulo="Não podem avaliá-lo" nomes={c.nao_posso} />
			<Lista rotulo="Preferem não avaliá-lo" nomes={c.prefiro_nao} />
		</>
	);
}

function DetalhesAvaliador({ a, mesas }: { a: AvaliadorResultado; mesas: MesaResult[] }) {
	return (
		<>
			<div className="grid grid-cols-2 gap-3 bg-gray-50 rounded-xl p-3">
				<Campo rotulo="Sigla" valor={a.sigla} />
				<div className="col-span-2">
					<Campo rotulo="Email" valor={a.email} />
				</div>
			</div>
			<div>
				<div className="text-xs font-semibold uppercase tracking-wide text-gray-500 mb-1">Mesas ({mesas.length})</div>
				<ul className="space-y-1">
					{mesas.map((m) => (
						<li key={m.id} className="capitalize">
							{m.descricao}
						</li>
					))}
				</ul>
			</div>
			<Lista rotulo="Não pode avaliar" nomes={a.nao_posso} />
			<Lista rotulo="Prefere não avaliar" nomes={a.prefiro_nao} />
		</>
	);
}

function Campo({ rotulo, valor }: { rotulo: string; valor: string }) {
	return (
		<div className="break-words">
			<span className="block text-xs text-gray-500">{rotulo}</span>
			<b>{valor || "—"}</b>
		</div>
	);
}

function Lista({ rotulo, nomes }: { rotulo: string; nomes: string[] }) {
	return (
		<div>
			<div className="text-xs font-semibold uppercase tracking-wide text-gray-500 mb-1">
				{rotulo} ({nomes.length})
			</div>
			{nomes.length ? <p>{nomes.join(", ")}</p> : <p className="text-gray-400">ninguém</p>}
		</div>
	);
}
