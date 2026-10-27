import { AcoesPessoa, AcoesMesa, AcoesHorario } from "./OperacoesAgenda";
import { ReactNode, useEffect, useMemo, useState } from "react";
import { ExclamationTriangleIcon, MagnifyingGlassIcon, XMarkIcon } from "@heroicons/react/24/outline";
import { AlocacaoResponse, AvaliadorResultado, CandidatoResultado, ItemQualidade, MesaResult } from "../api";
import { cn, EstadoVazio, Metrica, Secao, Selo } from "./ui";

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

const TOM: Record<ItemQualidade["tom"], { cartao: string; valor: string }> = {
	bom: { cartao: "border-[rgba(31,138,59,0.2)]", valor: "text-[var(--success)]" },
	neutro: { cartao: "border-[var(--line)]", valor: "text-[var(--text)]" },
	atencao: { cartao: "border-[rgba(245,166,35,0.35)]", valor: "text-[var(--warning)]" },
	ruim: { cartao: "border-[rgba(212,32,32,0.25)]", valor: "text-[var(--danger)]" },
};

// destaque de quem atende aos filtros ou ao item de qualidade ativo
const DESTACADO = "bg-[var(--warning-soft)] font-semibold text-[var(--text)] ring-1 ring-[rgba(245,166,35,0.45)]";

export default function PainelResultado({ result }: { result: AlocacaoResponse }) {
	const [busca, setBusca] = useState("");
	const [horario, setHorario] = useState("");
	const [avaliacao, setAvaliacao] = useState("");
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
	const filtrando = !!(busca.trim() || curso || semestre || item || avaliacao);

	// um candidato "bate" se atende a todos os filtros ativos
	const bate = (c: CandidatoResultado) =>
		(!busca.trim() || normalizar(`${c.nome} ${c.email_insper} ${c.curso}`).includes(normalizar(busca))) &&
		(!avaliacao || (avaliacao === "pendente" ? !c.avaliacao : c.avaliacao === avaliacao)) &&
		(!curso || normalizar(c.curso) === normalizar(curso)) &&
		(!semestre || String(c.semestre) === semestre) &&
		(!item || idsDoItem.has(c.id));

	// só com a busca de texto ativa, ela também procura avaliadores (nome, sigla, email)
	const buscaAvaliador = !!busca.trim() && !curso && !semestre && !item && !avaliacao;
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
		setAvaliacao("");
	}

	useEffect(() => {
		const fechar = (e: KeyboardEvent) => e.key === "Escape" && setSelecao(null);
		window.addEventListener("keydown", fechar);
		return () => window.removeEventListener("keydown", fechar);
	}, []);

	const classeCandidato = (c: CandidatoResultado) =>
		!filtrando ? "hover:bg-[var(--bg-hover)]" : bate(c) ? DESTACADO : "opacity-40";

	return (
		<div className="space-y-6">
			{/* ── Resumo ─────────────────────────────────────────────────── */}
			<div className="grid grid-cols-2 gap-3 md:grid-cols-4" data-testid="resumo">
				<Metrica rotulo="Mesas" valor={mesas.length} />
				<Metrica rotulo="Horários com mesa" valor={horarios.length} />
				<Metrica rotulo="Candidatos alocados" valor={result.total_alocados} tom="sucesso" />
				<Metrica
					rotulo="Pontuação"
					valor={result.pontuacao}
					tom="destaque"
					ajuda={'Começa em 100 e perde pontos: 2ª opção −1, 3ª −3, 4ª −5, 5ª −7; cada avaliador "prefiro não" na mesa −5; cada candidato sem mesa −1000; critérios adicionais descontam pela importância.'}
				/>
			</div>

			<button className="painel p-4 text-left text-sm" onClick={() => setAvaliacao("pendente")} aria-pressed={avaliacao === "pendente"}>
				{todos.filter(c => !c.avaliacao).length} avaliações pendentes — avaliar participantes
			</button>
			{/* ── Qualidade ──────────────────────────────────────────────── */}
			<Secao titulo="Qualidade da alocação" subtitulo="Clique em um item para destacar os candidatos dele.">
				<div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
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
								className={cn(
									"rounded-[var(--radius-lg)] border bg-white p-4 text-left transition-all",
									TOM[q.tom].cartao,
									q.valor === 0 ? "cursor-default" : "hover:shadow-[var(--shadow-md)]",
									ativo && "border-[var(--accent)] bg-[var(--accent-soft)] ring-1 ring-[var(--accent)]"
								)}
							>
								<div className={cn("text-2xl font-semibold tabular-nums", TOM[q.tom].valor)}>{q.valor}</div>
								<div className="text-sm font-medium">{q.titulo}</div>
							</button>
						);
					})}
				</div>
			</Secao>

			{/* ── Busca e filtros ────────────────────────────────────────── */}
			<section className="painel space-y-4 p-5" aria-label="Busca e filtros">
				<div className="relative">
					<MagnifyingGlassIcon className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--text-light)]" />
					<input
						type="search"
						aria-label="Buscar"
						placeholder="Buscar candidato (nome, email, curso) ou avaliador (nome, sigla)..."
						value={busca}
						onChange={(e) => setBusca(e.target.value)}
						className="campo pl-9"
					/>
				</div>
				<div className="grid grid-cols-1 gap-3 sm:grid-cols-4">
					<Filtro rotulo="Horário" valor={horario} opcoes={horarios} onChange={setHorario} />
					<label className="block"><span className="rotulo">Avaliação</span>
						<select className="campo" value={avaliacao} onChange={e => setAvaliacao(e.target.value)}>
							<option value="">Todas</option><option value="pendente">Pendente</option>
							<option value="aprovado">Passou</option><option value="reprovado">Não passou</option>
						</select>
					</label>
					<Filtro rotulo="Curso" valor={curso} opcoes={cursos} onChange={setCurso} />
					<Filtro rotulo="Semestre" valor={semestre} opcoes={semestres} onChange={setSemestre} formatar={(s) => `${s}º`} />
				</div>
				{(filtrando || horario) && (
					<div className="flex items-center justify-between gap-3 border-t border-[var(--line)] pt-3 text-sm">
						<span className="text-[var(--text-muted)]" data-testid="contagem-filtro">
							{filtrando
								? qtd(encontrados, "candidato", "candidatos") +
								  (avaliadoresEncontrados ? ` e ${qtd(avaliadoresEncontrados, "avaliador", "avaliadores")}` : "") +
								  (encontrados + avaliadoresEncontrados === 1 ? " encontrado" : " encontrados")
								: `${qtd(mesasVisiveis.length, "mesa", "mesas")} no horário`}
							{item && (
								<>
									{" "}
									· destaque: <b className="text-[var(--text)]">{item.titulo}</b>
								</>
							)}
						</span>
						<button type="button" onClick={limpar} className="font-medium text-[var(--accent)] hover:underline">
							Limpar filtros
						</button>
					</div>
				)}
			</section>

			{/* ── Mesas por horário ──────────────────────────────────────── */}
			{mesasVisiveis.length === 0 && (filtrando || horario)
				? // com "sem mesa" em destaque, é esperado não haver mesas: a tabela abaixo basta
				  item?.codigo !== "nao_alocados" && (
						<EstadoVazio titulo="Nenhuma mesa com candidatos que atendam aos filtros." descricao="Mude a busca ou limpe os filtros." />
				  )
				: Object.entries(porHorario).map(([hor, ms]) => (
						<section key={hor} aria-label={`Mesas de ${hor}`} className="space-y-3">
							<div className="flex items-center gap-2">
								<h2 className="text-base font-semibold capitalize">{hor}</h2>
								<AcoesHorario horario={hor} />
								<Selo>{qtd(ms.length, "mesa", "mesas")}</Selo>
							</div>
							<div className="grid grid-cols-1 gap-4 lg:grid-cols-2 xl:grid-cols-3">
								{ms.map((mesa) => (
									<div key={mesa.id} data-testid="mesa" className="painel overflow-hidden">
										<div className="flex items-center justify-between gap-2 border-b border-[var(--line)] bg-[var(--bg-subtle)] px-4 py-2.5">
											<span data-testid="nome-mesa" className="truncate text-sm font-semibold capitalize">
												{mesa.descricao}
											</span>
											<Selo>
												{mesa.candidatos.length} candidato{mesa.candidatos.length !== 1 ? "s" : ""}
											</Selo>
											<AcoesMesa mesa={mesa} />
										</div>
										<div className="grid grid-cols-1 gap-4 p-4 sm:grid-cols-2">
											<div>
												<div className="rotulo-caps mb-1.5">Candidatos</div>
												<ul className="space-y-0.5" data-testid="candidatos">
													{mesa.candidatos.map((c) => (
														<li key={c.id}>
															<button
																type="button"
																data-destacado={filtrando && bate(c) ? true : undefined}
																onClick={() => setSelecao({ tipo: "candidato", pessoa: c, mesa })}
																className={cn(
																	"flex w-full items-center gap-1.5 rounded-md px-1.5 py-0.5 text-left text-sm",
																	classeCandidato(c)
																)}
															>
																<span className="truncate">{c.nome}</span>
																{c.opcao > 1 && (
																	<Selo tom="atencao" className="px-1.5 text-[10px]">
																		{c.opcao}ª opção
																	</Selo>
																)}
																{c.conflitos.length > 0 && (
																	<ExclamationTriangleIcon
																		className="h-4 w-4 flex-shrink-0 text-[var(--warning)]"
																		aria-label="avaliador prefiro não na mesa"
																	/>
																)}
															</button>
															<AcoesPessoa pessoa={c} />
														</li>
													))}
												</ul>
											</div>
											<div>
												<div className="rotulo-caps mb-1.5">Avaliadores</div>
												<ul className="space-y-0.5" data-testid="avaliadores">
													{mesa.avaliadores.map((a) => (
														<li key={a.id}>
															<button
																type="button"
																data-destacado={bateAvaliador(a) ? true : undefined}
																onClick={() => setSelecao({ tipo: "avaliador", pessoa: a })}
																className={cn(
																	"w-full rounded-md px-1.5 py-0.5 text-left text-sm",
																	bateAvaliador(a) ? DESTACADO : "hover:bg-[var(--bg-hover)]"
																)}
															>
																{a.nome} <span className="font-mono text-[11px] text-[var(--text-light)]">{a.sigla}</span>
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
				  ))}

			{/* ── Não alocados ───────────────────────────────────────────── */}
			{naoAlocados.length > 0 && !horario && (
				<Secao
					rotulo="Candidatos não alocados"
					titulo="Candidatos Não Alocados"
					subtitulo="Não couberam em nenhum dos horários que escolheram."
					acoes={
						<Selo tom="perigo">
							{naoAlocados.length} candidato{naoAlocados.length !== 1 ? "s" : ""}
						</Selo>
					}
					className="border-[rgba(212,32,32,0.25)]"
				>
					<div className="overflow-x-auto rounded-[var(--radius-lg)] border border-[var(--line)]">
						<table className="w-full text-sm">
							<thead className="bg-[var(--bg-subtle)] text-left">
								<tr>
									<th className="rotulo-caps px-4 py-2">Nome</th>
									<th className="rotulo-caps px-4 py-2">Curso</th>
									<th className="rotulo-caps px-4 py-2">Semestre</th>
									<th className="rotulo-caps px-4 py-2">Horários que escolheu</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-[var(--line)]">
								{naoAlocados.map((c) => (
									<tr
										key={c.id}
										data-destacado={filtrando && bate(c) ? true : undefined}
										className={filtrando && !bate(c) ? "opacity-40" : filtrando ? "bg-[var(--warning-soft)]" : ""}
									>
										<td className="px-4 py-2.5">
											<button
												type="button"
												onClick={() => setSelecao({ tipo: "candidato", pessoa: c })}
												className="text-left font-medium hover:text-[var(--accent)] hover:underline"
											>
												{c.nome}
											</button>
											<AcoesPessoa pessoa={c} alocado={false} />
										</td>
										<td className="px-4 py-2.5 text-[var(--text-muted)]">{c.curso}</td>
										<td className="px-4 py-2.5 text-[var(--text-muted)]">{c.semestre ? `${c.semestre}º` : ""}</td>
										<td className="px-4 py-2.5 capitalize text-[var(--text-muted)]">{c.opcoes.join(" · ")}</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
				</Secao>
			)}

			<Detalhes selecao={selecao} mesas={mesas} onFechar={() => setSelecao(null)} />
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
			<span className="rotulo-caps block">{rotulo}</span>
			<select value={valor} onChange={(e) => onChange(e.target.value)} className="campo mt-1 capitalize">
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

// Painel lateral com os dados de quem foi clicado.
function Detalhes({ selecao, mesas, onFechar }: { selecao: Selecao; mesas: MesaResult[]; onFechar: () => void }) {
	if (!selecao) return null;
	const { tipo, pessoa } = selecao;
	return (
		<aside
			role="dialog"
			aria-label={`Detalhes de ${pessoa.nome}`}
			className="painel fixed bottom-4 right-4 top-4 z-50 w-[min(380px,calc(100vw-2rem))] overflow-y-auto p-6 shadow-[var(--shadow-lg)]"
		>
			<div className="flex items-start justify-between gap-3 border-b border-[var(--line)] pb-4">
				<div>
					<div className="rotulo-caps">{tipo === "candidato" ? "Candidato" : "Avaliador"}</div>
					<h3 className="mt-1 text-xl font-semibold">{pessoa.nome}</h3>
				</div>
				<button
					type="button"
					onClick={onFechar}
					aria-label="Fechar detalhes"
					className="rounded-md p-1.5 text-[var(--text-muted)] hover:bg-[var(--bg-hover)]"
				>
					<XMarkIcon className="h-5 w-5" />
				</button>
			</div>

			<div className="mt-5 space-y-5 text-sm">
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
			<div className="grid grid-cols-2 gap-3 rounded-[var(--radius-lg)] bg-[var(--bg-subtle)] p-3">
				<Campo rotulo="Curso" valor={c.curso} />
				<Campo rotulo="Semestre" valor={c.semestre ? `${c.semestre}º` : ""} />
				<div className="col-span-2">
					<Campo rotulo="Email" valor={c.email_insper} />
				</div>
			</div>
			<div
				className={cn(
					"rounded-[var(--radius-lg)] border p-3",
					mesa
						? "border-[rgba(94,106,210,0.2)] bg-[var(--accent-soft)] text-[#3b449a]"
						: "border-[rgba(212,32,32,0.2)] bg-[var(--danger-soft)] text-[#a51818]"
				)}
			>
				{mesa ? (
					<>
						Alocado em <b className="capitalize">{mesa.descricao}</b>, a sua <b>{c.opcao}ª opção</b>.
					</>
				) : (
					<>Sem mesa: não coube em nenhum dos horários que escolheu.</>
				)}
			</div>
			<Bloco rotulo="Horários que escolheu">
				{c.opcoes.length ? (
					<ol className="space-y-1">
						{c.opcoes.map((o, i) => {
							const alocado = mesa && i + 1 === c.opcao;
							return (
								<li key={i} className={cn("capitalize", alocado ? "font-semibold text-[var(--accent)]" : "text-[var(--text-muted)]")}>
									<span className="text-[var(--text-light)]">{i + 1}.</span> {o}{" "}
									{alocado && <span className="normal-case">← alocado</span>}
								</li>
							);
						})}
					</ol>
				) : (
					<p className="text-[var(--text-muted)]">Nenhum horário informado.</p>
				)}
			</Bloco>
			{c.conflitos.length > 0 && (
				<div className="rounded-[var(--radius-lg)] border border-[rgba(245,166,35,0.3)] bg-[var(--warning-soft)] p-3 text-[#8a5a12]">
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
			<div className="grid grid-cols-2 gap-3 rounded-[var(--radius-lg)] bg-[var(--bg-subtle)] p-3">
				<Campo rotulo="Sigla" valor={a.sigla} />
				<div className="col-span-2">
					<Campo rotulo="Email" valor={a.email} />
				</div>
			</div>
			<Bloco rotulo={`Mesas (${mesas.length})`}>
				<ul className="space-y-1">
					{mesas.map((m) => (
						<li key={m.id} className="capitalize text-[var(--text-muted)]">
							{m.descricao}
						</li>
					))}
				</ul>
			</Bloco>
			<Lista rotulo="Não pode avaliar" nomes={a.nao_posso} />
			<Lista rotulo="Prefere não avaliar" nomes={a.prefiro_nao} />
		</>
	);
}

function Bloco({ rotulo, children }: { rotulo: string; children: ReactNode }) {
	return (
		<div>
			<div className="rotulo-caps mb-1.5">{rotulo}</div>
			{children}
		</div>
	);
}

function Campo({ rotulo, valor }: { rotulo: string; valor: string }) {
	return (
		<div className="break-words">
			<span className="block text-xs text-[var(--text-muted)]">{rotulo}</span>
			<span className="font-medium">{valor || "—"}</span>
		</div>
	);
}

function Lista({ rotulo, nomes }: { rotulo: string; nomes: string[] }) {
	return (
		<Bloco rotulo={`${rotulo} (${nomes.length})`}>
			{nomes.length ? <p>{nomes.join(", ")}</p> : <p className="text-[var(--text-light)]">ninguém</p>}
		</Bloco>
	);
}
