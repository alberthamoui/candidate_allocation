import React, { createContext, useContext, useEffect, useState } from "react";
import {
	EvaluateCandidates,
	ExportAgendaExcel,
	GetMoveSuggestions,
	GetNoticeStates,
	MoveAgenda,
	PreviewAgendaNotices,
	SendAgendaNotices,
} from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";
import ChannelsPanel from "./ChannelsPanel";

type Actions = {
	busy: boolean;
	states: Record<number, string>;
	move: (kind: string, item: any) => void;
	evaluate: (ids: number[], value: string) => void;
	send: (ids: number[], group: number) => void;
	filter: string;
	setFilter: (s: string) => void;
};
const Context = createContext<Actions | null>(null);
const button =
	"rounded-md border border-slate-200 bg-white px-2 py-1.5 text-xs font-medium hover:bg-slate-100 disabled:opacity-40";
export function useAgendaActions() {
	return useContext(Context);
}

function Dialog({
	title,
	children,
	close,
}: {
	title: string;
	children: React.ReactNode;
	close: () => void;
}) {
	useEffect(() => {
		const previous = document.activeElement as HTMLElement;
		const root = document.getElementById("agenda-dialog");
		root?.focus();
		const handler = (e: KeyboardEvent) => {
			if (e.key === "Escape") close();
			if (e.key === "Tab") {
				const fields = root?.querySelectorAll<HTMLElement>(
					"button:not(:disabled), input, select, textarea, a[href]",
				);
				if (!fields?.length) return;
				const first = fields[0],
					last = fields[fields.length - 1];
				if (e.shiftKey && document.activeElement === first) {
					e.preventDefault();
					last.focus();
				} else if (!e.shiftKey && document.activeElement === last) {
					e.preventDefault();
					first.focus();
				}
			}
		};
		document.addEventListener("keydown", handler);
		return () => {
			document.removeEventListener("keydown", handler);
			previous?.focus();
		};
	}, []);
	return (
		<div className="fixed inset-0 z-[80] flex items-center justify-center bg-black/30 p-4">
			<section
				id="agenda-dialog"
				tabIndex={-1}
				role="dialog"
				aria-modal="true"
				aria-label={title}
				className="max-h-[85vh] w-full max-w-2xl overflow-y-auto rounded-2xl bg-white p-5 shadow-xl"
			>
				<div className="mb-4 flex items-start justify-between gap-4">
					<h2 className="text-lg font-semibold">{title}</h2>
					<button type="button" onClick={close} aria-label="Fechar ação">
						✕
					</button>
				</div>
				{children}
			</section>
		</div>
	);
}

export function AgendaActionsProvider({
	result,
	onResult,
	running,
	children,
}: {
	result: any;
	onResult: (r: any) => void;
	running: boolean;
	children: React.ReactNode;
}) {
	const [busy, setBusy] = useState(false),
		[error, setError] = useState(""),
		[status, setStatus] = useState(""),
		[channels, setChannels] = useState(false),
		[filter, setFilter] = useState("");
	const [states, setStates] = useState<Record<number, string>>({});
	const [move, setMove] = useState<{ kind: string; item: any } | null>(null),
		[dayScope, setDayScope] = useState("interval"),
		[suggestions, setSuggestions] = useState<any[]>([]),
		[destination, setDestination] = useState(0),
		[schedule, setSchedule] = useState(""),
		[notify, setNotify] = useState(false);
	const [send, setSend] = useState<{
		ids: number[];
		group: number;
		items: any[];
	} | null>(null);
	const candidates = [
		...(result.mesas || []).flatMap((m: any) => m.candidatos || []),
		...(result.naoAlocados || []),
	];
	const refreshStates = () =>
		GetNoticeStates()
			.then(setStates)
			.catch(() => {});
	useEffect(() => {
		void refreshStates();
		return EventsOn("notices:changed", () => {
			void refreshStates();
		});
	}, []);
	async function act(fn: () => Promise<any>, message: string) {
		setBusy(true);
		setError("");
		setStatus("");
		try {
			await fn();
			setStatus(message);
		} catch (e) {
			setError(String(e));
		} finally {
			setBusy(false);
		}
	}
	function beginMove(kind: string, item: any) {
		setError("");
		setMove({ kind, item });
		setSuggestions([]);
		setDestination(0);
		setSchedule("");
		setNotify(false);
		setDayScope("interval");
		if (kind === "person")
			GetMoveSuggestions(item.id)
				.then(setSuggestions)
				.catch((e) => setError(String(e)));
	}
	async function beginSend(ids: number[], group: number) {
		await act(async () => {
			const items = await PreviewAgendaNotices(ids, group);
			setSend({ ids, group, items });
		}, "");
	}
	const value: Actions = {
		busy: busy || running,
		states,
		move: beginMove,
		evaluate: (ids, v) =>
			void act(
				async () => onResult(await EvaluateCandidates(ids, v)),
				"Avaliação salva",
			),
		send: (ids, g) => void beginSend(ids, g),
		filter,
		setFilter,
	};
	const conflicting = new Set<number>(
		(result.hardViolations || []).flatMap((v: any) =>
			v.candidateId
				? [v.candidateId]
				: (result.mesas || [])
						.filter((m: any) => m.id === v.groupId)
						.flatMap((m: any) => m.candidatos.map((c: any) => c.id)),
		),
	);
	const counts: { key: string; label: string; count: number }[] = [
		{
			key: "evaluation",
			label: "Sem avaliação",
			count: candidates.filter((c: any) => !c.avaliacao).length,
		},
		{
			key: "notice",
			label: "Avisos pendentes ou com falha",
			count: candidates.filter((c: any) => states[c.id] !== "accepted").length,
		},
		{ key: "conflict", label: "Com alerta", count: conflicting.size },
		{
			key: "unallocated",
			label: "Sem mesa",
			count: (result.naoAlocados || []).length,
		},
	];
	return (
		<Context.Provider value={value}>
			<section
				className="minimal-panel space-y-4 p-5"
				aria-label="Ações e pendências da agenda"
			>
				<div className="flex flex-wrap justify-between gap-3">
					<h2 className="font-semibold">Agenda e pendências</h2>
					<div className="flex flex-wrap gap-2">
						<button
							type="button"
							className={button}
							onClick={() => setChannels(true)}
						>
							Canais e mensagens
						</button>
						<button
							type="button"
							className={button}
							disabled={busy}
							onClick={() =>
								void act(async () => {
									const data = await ExportAgendaExcel();
									const bytes = Uint8Array.from(atob(data), (c) =>
										c.charCodeAt(0),
									);
									const url = URL.createObjectURL(
										new Blob([bytes], {
											type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
										}),
									);
									const link = document.createElement("a");
									link.href = url;
									link.download = "agenda.xlsx";
									link.click();
									setTimeout(() => URL.revokeObjectURL(url), 5000);
								}, "Excel exportado")
							}
						>
							Exportar Excel
						</button>
						<button
							type="button"
							className={button}
							disabled={busy || running}
							onClick={() => void beginSend([], 0)}
						>
							Avisar todos
						</button>
					</div>
				</div>
				<div className="flex flex-wrap gap-2">
					{counts.map((c) => (
						<button
							type="button"
							key={c.key}
							aria-pressed={filter === c.key}
							className={`${button} ${filter === c.key ? "ring-2 ring-indigo-500" : ""}`}
							onClick={() => setFilter(filter === c.key ? "" : c.key)}
						>
							{c.label}: {c.count}
						</button>
					))}
					{filter && (
						<button
							type="button"
							className={button}
							onClick={() => setFilter("")}
						>
							Mostrar todos
						</button>
					)}
				</div>
				{running && (
					<p className="text-sm text-amber-800">
						Pare a verificação para realocar, avaliar ou avisar usando esta
						solução.
					</p>
				)}
				{error && (
					<p
						role="alert"
						className="rounded-lg bg-red-50 p-3 text-sm text-red-800"
					>
						{error}
					</p>
				)}
				{status && (
					<p role="status" className="text-sm text-emerald-800">
						{status}
					</p>
				)}
			</section>
			{children}
			{channels && (
				<Dialog title="Canais e mensagens" close={() => setChannels(false)}>
					<ChannelsPanel onClose={() => setChannels(false)} />
				</Dialog>
			)}
			{move && (
				<Dialog
					title={`Realocar ${move.item.nome || move.item.descricao || move.item.horario}`}
					close={() => setMove(null)}
				>
					<p className="mb-3 text-sm text-slate-600">
						A escolha manual é livre. Confira os alertas antes de confirmar.
					</p>
					{move.kind === "person" && (
						<>
							<h3 className="font-medium">Melhores destinos</h3>
							<div className="my-3 space-y-2">
								{suggestions.slice(0, 3).map((s, i) => (
									<button
										type="button"
										key={s.groupId}
										className={`${button} block w-full text-left ${destination === s.groupId ? "ring-2 ring-indigo-500" : ""}`}
										onClick={() => {
											setDestination(s.groupId);
											setSchedule(s.schedule);
										}}
									>
										<b>
											{i + 1}º · {s.description}
										</b>
										<span className="block">
											{s.vacancies} vaga(s) ·{" "}
											{(s.alerts || []).join(" · ") || "Sem alertas"}
										</span>
									</button>
								))}
							</div>
							<label className="block text-sm">
								Mesa de destino
								<select
									className="minimal-input mt-1"
									value={destination}
									onChange={(e) => {
										const id = Number(e.target.value);
										setDestination(id);
										setSchedule(
											suggestions.find((s) => s.groupId === id)?.schedule || "",
										);
									}}
								>
									<option value={0}>Horário manual</option>
									{suggestions.map((s) => (
										<option key={s.groupId} value={s.groupId}>
											{s.description} · {s.alerts.length} alerta(s)
										</option>
									))}
								</select>
							</label>
						</>
					)}
					{move.kind === "day" && (
						<label className="block text-sm">
							O que mover?
							<select
								className="minimal-input mt-1"
								value={dayScope}
								onChange={(e) => setDayScope(e.target.value)}
							>
								<option value="interval">Somente este intervalo</option>
								<option value="day">
									Dia inteiro, preservando os intervalos
								</option>
							</select>
						</label>
					)}
					<label className="mt-3 block text-sm">
						{move.kind === "day"
							? dayScope === "day"
								? "Novo dia (intervalos serão preservados)"
								: "Novo intervalo"
							: "Novo horário"}
						<input
							className="minimal-input mt-1"
							value={schedule}
							placeholder="sexta 14-16"
							onChange={(e) => {
								setSchedule(e.target.value);
								setDestination(0);
							}}
						/>
					</label>
					{move.kind === "person" && destination === 0 && schedule && (
						<p className="mt-2 text-sm text-amber-800">
							Um horário novo pode criar uma mesa sem avaliadores. A pendência
							ficará visível.
						</p>
					)}
					{!!suggestions.find((s) => s.groupId === destination)?.alerts
						.length && (
						<p className="mt-2 text-sm text-amber-800">
							{suggestions
								.find((s) => s.groupId === destination)
								.alerts.join(" · ")}
						</p>
					)}
					<label className="my-4 flex gap-2 text-sm">
						<input
							type="checkbox"
							checked={notify}
							onChange={(e) => setNotify(e.target.checked)}
						/>
						Avisar candidatos e avaliadores afetados após a mudança
					</label>
					{error && (
						<p role="alert" className="text-red-700">
							{error}
						</p>
					)}
					<button
						type="button"
						disabled={busy || (!schedule.trim() && !destination)}
						className={button}
						onClick={() =>
							void act(async () => {
								const op = await MoveAgenda({
									kind: move.kind,
									scope: dayScope,
									candidateId: move.kind === "person" ? move.item.id : 0,
									groupId: move.kind === "group" ? move.item.id : 0,
									destinationId: destination,
									schedule,
									from: move.item.horario || "",
									to: schedule,
									notify,
								});
								onResult(op.result);
								setMove(null);
								if (op.warning) setError(op.warning);
								void refreshStates();
							}, "Agenda atualizada")
						}
					>
						Confirmar realocação
					</button>
				</Dialog>
			)}
			{send && (
				<Dialog title="Confirmar envio" close={() => setSend(null)}>
					<p className="text-sm">
						{send.items.length} aviso(s) para os candidatos selecionados e seus
						avaliadores, pelos canais ativos.
					</p>
					<div className="my-3 max-h-64 space-y-3 overflow-y-auto">
						{send.items.slice(0, 20).map((d, i) => (
							<div key={i} className="rounded-lg border p-3 text-sm">
								<b>
									{d.name} · {d.channel}
								</b>
								<p>{d.destination || "Contato ausente"}</p>
								<p className="whitespace-pre-wrap">
									{d.channel === "whatsapp"
										? "O WhatsApp Business usará o modelo aprovado configurado, com as variáveis desta entrevista."
										: d.message}
								</p>
								{d.error && <p className="text-red-700">{d.error}</p>}
							</div>
						))}
					</div>
					<p className="mb-3 text-xs text-slate-500">
						Aceitação pelo provedor não confirma entrega. Nenhuma mensagem foi
						enviada ainda.
					</p>
					<button
						type="button"
						className={button}
						disabled={busy}
						onClick={() =>
							void act(async () => {
								const id = await SendAgendaNotices(send.ids, send.group);
								setSend(null);
								setStatus(
									`Envio ${id} iniciado; consulte o histórico em Canais e mensagens.`,
								);
							}, "Envio iniciado")
						}
					>
						Confirmar envio
					</button>
					{error && <p role="alert">{error}</p>}
				</Dialog>
			)}
		</Context.Provider>
	);
}

export function CandidateActions({
	candidate,
	unallocated = false,
	conflict = false,
}: {
	candidate: any;
	unallocated?: boolean;
	conflict?: boolean;
}) {
	const actions = useAgendaActions();
	if (!actions) return null;
	const state = actions.states[candidate.id];
	const hidden =
		(actions.filter === "evaluation" && !!candidate.avaliacao) ||
		(actions.filter === "notice" && state === "accepted") ||
		(actions.filter === "unallocated" && !unallocated) ||
		(actions.filter === "conflict" && !conflict);
	return (
		<div
			data-agenda-hidden={hidden ? "true" : "false"}
			className="space-y-2 border-t border-slate-100 px-3 py-2"
		>
			<p className="flex flex-wrap gap-2 text-xs">
				<span>
					{candidate.avaliacao === "aprovado"
						? "Passou"
						: candidate.avaliacao === "reprovado"
							? "Não passou"
							: "Sem avaliação"}
				</span>
				<span>
					{state === "accepted"
						? "Aviso aceito"
						: state === "failed"
							? "Falha no aviso"
							: state === "pending" || state === "sending"
								? "Aviso em andamento"
								: state === "unknown"
									? "Envio a conferir"
									: "Sem aviso"}
				</span>
			</p>
			<div className="flex flex-wrap gap-1">
				<button
					type="button"
					className={button}
					disabled={actions.busy}
					aria-label={`Realocar ${candidate.nome}`}
					onClick={() => actions.move("person", candidate)}
				>
					Realocar
				</button>
				<button
					type="button"
					className={button}
					disabled={actions.busy}
					aria-label={`Aprovar ${candidate.nome}`}
					aria-pressed={candidate.avaliacao === "aprovado"}
					onClick={() =>
						actions.evaluate(
							[candidate.id],
							candidate.avaliacao === "aprovado" ? "" : "aprovado",
						)
					}
				>
					Passou
				</button>
				<button
					type="button"
					className={button}
					disabled={actions.busy}
					aria-label={`Reprovar ${candidate.nome}`}
					aria-pressed={candidate.avaliacao === "reprovado"}
					onClick={() =>
						actions.evaluate(
							[candidate.id],
							candidate.avaliacao === "reprovado" ? "" : "reprovado",
						)
					}
				>
					Não passou
				</button>
				<button
					type="button"
					className={button}
					disabled={actions.busy || unallocated}
					aria-label={`Avisar ${candidate.nome}`}
					onClick={() => actions.send([candidate.id], 0)}
				>
					Avisar
				</button>
			</div>
		</div>
	);
}

export function GroupActions({ group }: { group: any }) {
	const a = useAgendaActions();
	if (!a) return null;
	return (
		<div className="flex flex-wrap gap-1 px-4 pb-3">
			<button
				type="button"
				className={button}
				disabled={a.busy}
				onClick={() => a.move("group", group)}
			>
				Realocar mesa
			</button>
			<button
				type="button"
				className={button}
				disabled={a.busy}
				onClick={() => a.send([], group.id)}
			>
				Avisar mesa
			</button>
			<button
				type="button"
				className={button}
				disabled={a.busy}
				onClick={() =>
					a.evaluate(
						group.candidatos.map((c: any) => c.id),
						"aprovado",
					)
				}
			>
				Aprovar mesa
			</button>
		</div>
	);
}
export function DayActions({ schedule }: { schedule: string }) {
	const a = useAgendaActions();
	if (!a) return null;
	return (
		<button
			type="button"
			className={button}
			disabled={a.busy}
			onClick={() => a.move("day", { horario: schedule })}
		>
			Mover intervalo / dia
		</button>
	);
}
