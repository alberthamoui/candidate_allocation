import React, { useEffect, useState, useId } from "react";
import {
	BeginGoogleConnection,
	DisconnectGoogle,
	GetChannelConfig,
	GetGoogleConnectionStatus,
	GetNoticeDetails,
	GetNoticeHistory,
	RetryNoticeFailures,
	SaveChannelConfig,
	TestChannels,
} from "../wailsjs/go/main/App";
import { BrowserOpenURL, EventsOn } from "../wailsjs/runtime/runtime";

const button =
	"rounded-lg border border-slate-300 px-3 py-2 text-sm font-medium disabled:opacity-40";
const defaults = {
	ativo: false,
	enviar_ao_alocar: false,
	mensagem:
		"Olá {{nome}}, sua entrevista é em {{data}}, no horário {{horario}} ({{mesa}}).",
	email: {
		ativo: false,
		provedor: "gmail",
		host: "smtp.gmail.com",
		porta: 587,
		usuario: "",
		remetente: "",
		senha: "",
		senha_definida: false,
		google_connected: false,
	},
	whatsapp: {
		ativo: false,
		provedor: "meta",
		phone_id: "",
		token: "",
		token_definido: false,
		template: "",
		idioma: "pt_BR",
		variaveis_template: ["nome", "data", "horario", "mesa"],
		url: "",
	},
};
const presets = [
	{ name: "Convocação", message: defaults.mensagem },
	{
		name: "Mudança de horário",
		message:
			"Olá {{nome}}, sua entrevista foi remarcada para {{data}}, {{horario}} ({{mesa}}).",
	},
	{
		name: "Lembrete",
		message:
			"Olá {{nome}}, lembramos da sua entrevista em {{data}}, {{horario}} ({{mesa}}).",
	},
];
function Field({
	name,
	children,
}: {
	name: string;
	children: React.ReactNode;
}) {
	const id = useId();
	return (
		<div className="text-sm font-medium">
			<label htmlFor={id}>{name}</label>
			{React.isValidElement(children)
				? React.cloneElement(children as React.ReactElement, { id })
				: children}
		</div>
	);
}

export default function ChannelsPanel({ onClose }: { onClose: () => void }) {
	const [cfg, setCfg] = useState<any>(defaults),
		[tab, setTab] = useState("channels"),
		[busy, setBusy] = useState(false),
		[error, setError] = useState(""),
		[status, setStatus] = useState(""),
		[oauth, setOauth] = useState<any>({ available: false, status: "idle" });
	const [history, setHistory] = useState<any[]>([]),
		[next, setNext] = useState(0),
		[details, setDetails] = useState<
			Record<string, { items: any[]; next: number }>
		>({});
	function loadConfig() {
		return GetChannelConfig().then((c: any) =>
			setCfg({
				...defaults,
				...c,
				email: { ...defaults.email, ...c.email },
				whatsapp: { ...defaults.whatsapp, ...c.whatsapp },
			}),
		);
	}
	useEffect(() => {
		void loadConfig().catch((e) => setError(String(e)));
		void GetGoogleConnectionStatus().then(setOauth);
	}, []);
	useEffect(() => {
		if (oauth.status !== "waiting") return;
		let active = true;
		const timer = setInterval(
			() =>
				GetGoogleConnectionStatus()
					.then((s) => {
						if (!active) return;
						setOauth(s);
						if (s.status === "connected") {
							void loadConfig();
							setStatus("Conta Google conectada");
						}
						if (s.status === "error") setError(s.message);
					})
					.catch((e) => setError(String(e))),
			1500,
		);
		return () => {
			active = false;
			clearInterval(timer);
		};
	}, [oauth.status]);
	async function act(fn: () => Promise<any>, message = "") {
		setBusy(true);
		setError("");
		setStatus("");
		try {
			await fn();
			if (message) setStatus(message);
		} catch (e) {
			setError(String(e));
		} finally {
			setBusy(false);
		}
	}
	async function loadHistory(before = 0) {
		const page = await GetNoticeHistory(before);
		setHistory((h) => (before ? [...h, ...page.items] : page.items));
		setNext(page.next);
	}
	useEffect(() => {
		if (tab !== "history") return;
		let active = true;
		const refresh = () => {
			if (active && document.visibilityState === "visible")
				void loadHistory().catch((e) => setError(String(e)));
		};
		refresh();
		const stop = EventsOn("notices:changed", refresh);
		document.addEventListener("visibilitychange", refresh);
		return () => {
			active = false;
			stop();
			document.removeEventListener("visibilitychange", refresh);
		};
	}, [tab]);
	const email = (value: any) =>
			setCfg({ ...cfg, email: { ...cfg.email, ...value } }),
		whats = (value: any) =>
			setCfg({ ...cfg, whatsapp: { ...cfg.whatsapp, ...value } });
	return (
		<div className="space-y-4">
			<div className="flex gap-2">
				{[
					{ id: "channels", name: "1. Conectar canais" },
					{ id: "message", name: "2. Mensagem" },
					{ id: "history", name: "Histórico" },
				].map((t) => (
					<button
						type="button"
						key={t.id}
						aria-pressed={tab === t.id}
						className={`${button} ${tab === t.id ? "bg-slate-100" : ""}`}
						onClick={() => setTab(t.id)}
					>
						{t.name}
					</button>
				))}
			</div>
			{tab === "channels" && (
				<div className="space-y-4">
					<p className="text-sm text-slate-600">
						Configure os canais uma vez. Você autoriza cada envio depois de
						conferir os destinatários.
					</p>
					<section className="space-y-3 rounded-xl border p-4">
						<label className="flex items-center gap-2 font-medium">
							<input
								type="checkbox"
								checked={cfg.email.ativo}
								onChange={(e) => {
									setCfg({
										...cfg,
										ativo: true,
										email: { ...cfg.email, ativo: e.target.checked },
									});
								}}
							/>
							E-mail
						</label>
						{cfg.email.ativo && (
							<>
								<Field name="Como conectar">
									<select
										className="minimal-input mt-1"
										value={cfg.email.provedor}
										onChange={(e) => email({ provedor: e.target.value })}
									>
										<option value="google">Conectar conta Google</option>
										<option value="gmail">Gmail com senha de aplicativo</option>
										<option value="smtp">Outro serviço de e-mail</option>
									</select>
								</Field>
								{cfg.email.provedor === "google" ? (
									<div className="space-y-2">
										<p className="text-sm">
											{cfg.email.google_connected
												? `Conectado: ${cfg.email.remetente}`
												: "Autorize o envio pela sua conta no navegador. Sua senha do Google não é compartilhada com o aplicativo."}
										</p>
										<button
											type="button"
											className={button}
											disabled={
												busy || !oauth.available || oauth.status === "waiting"
											}
											onClick={() =>
												void act(async () => {
													const url = await BeginGoogleConnection();
													BrowserOpenURL(url);
													setOauth({ ...oauth, status: "waiting" });
												})
											}
										>
											{oauth.status === "waiting"
												? "Aguardando autorização…"
												: "Conectar Google"}
										</button>
										{cfg.email.google_connected && (
											<button
												type="button"
												className={button}
												onClick={() =>
													void act(async () => {
														await DisconnectGoogle();
														await loadConfig();
													}, "Google desconectado")
												}
											>
												Desconectar
											</button>
										)}
										{!oauth.available && (
											<p className="text-sm text-amber-800">
												A conexão Google ainda não está habilitada nesta
												instalação. Você pode usar Gmail com senha de
												aplicativo.
											</p>
										)}
									</div>
								) : (
									<>
										<Field name="Seu e-mail de envio">
											<input
												className="minimal-input mt-1"
												type="email"
												value={cfg.email.remetente}
												onChange={(e) => email({ remetente: e.target.value })}
											/>
										</Field>
										<Field
											name={
												cfg.email.provedor === "gmail"
													? "Senha de aplicativo"
													: "Senha do serviço"
											}
										>
											<input
												className="minimal-input mt-1"
												type="password"
												autoComplete="new-password"
												value={cfg.email.senha}
												placeholder={
													cfg.email.senha_definida
														? "Credencial salva; deixe vazio para manter"
														: ""
												}
												onChange={(e) => email({ senha: e.target.value })}
											/>
										</Field>
										{cfg.email.provedor === "gmail" ? (
											<button
												type="button"
												className="text-sm underline"
												onClick={() =>
													BrowserOpenURL(
														"https://support.google.com/accounts/answer/185833",
													)
												}
											>
												Como criar uma senha de aplicativo
											</button>
										) : (
											<details>
												<summary className="cursor-pointer text-sm">
													Configuração do serviço (SMTP)
												</summary>
												<div className="mt-2 space-y-2">
													<Field name="Servidor">
														<input
															className="minimal-input mt-1"
															value={cfg.email.host}
															onChange={(e) => email({ host: e.target.value })}
														/>
													</Field>
													<Field name="Porta">
														<input
															className="minimal-input mt-1"
															type="number"
															value={cfg.email.porta}
															onChange={(e) =>
																email({ porta: Number(e.target.value) })
															}
														/>
													</Field>
													<Field name="Usuário">
														<input
															className="minimal-input mt-1"
															value={cfg.email.usuario}
															onChange={(e) =>
																email({ usuario: e.target.value })
															}
														/>
													</Field>
												</div>
											</details>
										)}
									</>
								)}
							</>
						)}
					</section>
					<section className="space-y-3 rounded-xl border p-4">
						<label className="flex gap-2 font-medium">
							<input
								type="checkbox"
								checked={cfg.whatsapp.ativo}
								onChange={(e) =>
									setCfg({
										...cfg,
										ativo: true,
										whatsapp: { ...cfg.whatsapp, ativo: e.target.checked },
									})
								}
							/>
							WhatsApp
						</label>
						{cfg.whatsapp.ativo && (
							<>
								<p className="text-sm text-slate-600">
									Use a conta WhatsApp Business da organização e um modelo
									aprovado. O aplicativo não conecta uma conta pessoal por QR
									Code.
								</p>
								<Field name="Serviço">
									<select
										className="minimal-input mt-1"
										value={cfg.whatsapp.provedor}
										onChange={(e) => whats({ provedor: e.target.value })}
									>
										<option value="meta">WhatsApp Business (Meta)</option>
										<option value="webhook">Outro provedor (avançado)</option>
									</select>
								</Field>
								{cfg.whatsapp.provedor === "meta" ? (
									<>
										<button
											type="button"
											className="text-sm underline"
											onClick={() =>
												BrowserOpenURL(
													"https://business.facebook.com/wa/manage/home/",
												)
											}
										>
											Abrir configuração do WhatsApp Business
										</button>
										<details open={!cfg.whatsapp.token_definido}>
											<summary className="cursor-pointer text-sm">
												Conectar número da organização
											</summary>
											<div className="mt-2 space-y-2">
												<Field name="ID do número na Meta">
													<input
														className="minimal-input mt-1"
														value={cfg.whatsapp.phone_id || ""}
														onChange={(e) =>
															whats({ phone_id: e.target.value })
														}
													/>
												</Field>
												<Field name="Token de acesso">
													<input
														className="minimal-input mt-1"
														type="password"
														value={cfg.whatsapp.token}
														placeholder={
															cfg.whatsapp.token_definido
																? "Conexão salva; deixe vazio para manter"
																: ""
														}
														onChange={(e) => whats({ token: e.target.value })}
													/>
												</Field>
											</div>
										</details>
										<Field name="Nome do modelo aprovado">
											<input
												className="minimal-input mt-1"
												value={cfg.whatsapp.template || ""}
												onChange={(e) => whats({ template: e.target.value })}
											/>
										</Field>
										<details>
											<summary className="cursor-pointer text-sm">
												Idioma e variáveis do modelo
											</summary>
											<Field name="Idioma">
												<input
													className="minimal-input"
													value={cfg.whatsapp.idioma}
													onChange={(e) => whats({ idioma: e.target.value })}
												/>
											</Field>
											<Field name="Variáveis na ordem do modelo">
												<input
													className="minimal-input"
													value={cfg.whatsapp.variaveis_template.join(", ")}
													onChange={(e) =>
														whats({
															variaveis_template: e.target.value
																.split(",")
																.map((v) => v.trim())
																.filter(Boolean),
														})
													}
												/>
											</Field>
										</details>
									</>
								) : (
									<>
										<Field name="Endereço HTTPS do provedor">
											<input
												className="minimal-input"
												value={cfg.whatsapp.url}
												onChange={(e) => whats({ url: e.target.value })}
											/>
										</Field>
										<Field name="Token">
											<input
												className="minimal-input"
												type="password"
												value={cfg.whatsapp.token}
												onChange={(e) => whats({ token: e.target.value })}
											/>
										</Field>
									</>
								)}
							</>
						)}
					</section>
					<button
						type="button"
						className={button}
						disabled={busy}
						onClick={() =>
							void act(async () => {
								await SaveChannelConfig(cfg);
								const result = await TestChannels();
								setStatus(Object.values(result).join(" · "));
								await loadConfig();
							})
						}
					>
						Salvar e testar conexão
					</button>
				</div>
			)}
			{tab === "message" && (
				<div className="space-y-3">
					<p className="text-sm text-slate-600">
						Escolha uma mensagem inicial e ajuste o texto. No WhatsApp Business,
						o envio usa o modelo aprovado; este editor se aplica ao e-mail e a
						provedores de texto livre.
					</p>
					<div className="flex flex-wrap gap-2">
						{presets.map((p) => (
							<button
								type="button"
								className={button}
								key={p.name}
								onClick={() => setCfg({ ...cfg, mensagem: p.message })}
							>
								{p.name}
							</button>
						))}
					</div>
					<Field name="Mensagem">
						<textarea
							className="minimal-input mt-1"
							rows={4}
							value={cfg.mensagem}
							onChange={(e) => setCfg({ ...cfg, mensagem: e.target.value })}
						/>
					</Field>
					<div className="flex flex-wrap gap-1">
						{[
							"nome",
							"data",
							"horario",
							"mesa",
							"email",
							"curso",
							"numero",
						].map((v) => (
							<button
								type="button"
								className={button}
								key={v}
								onClick={() =>
									setCfg({ ...cfg, mensagem: cfg.mensagem + ` {{${v}}}` })
								}
							>{`{{${v}}}`}</button>
						))}
					</div>
					<div className="rounded-lg bg-slate-50 p-3 text-sm">
						<b>Exemplo de e-mail</b>
						<p className="mt-2 whitespace-pre-wrap">
							{cfg.mensagem
								.replaceAll("{{nome}}", "Maria")
								.replaceAll("{{data}}", "sexta-feira")
								.replaceAll("{{horario}}", "14-16")
								.replaceAll("{{mesa}}", "Mesa 1")
								.replaceAll("{{email}}", "maria@exemplo.test")
								.replaceAll("{{curso}}", "Economia")
								.replaceAll("{{numero}}", "123456789")}
						</p>
					</div>
					<label className="flex gap-2 text-sm">
						<input
							type="checkbox"
							checked={cfg.enviar_ao_alocar}
							onChange={(e) =>
								setCfg({ ...cfg, enviar_ao_alocar: e.target.checked })
							}
						/>
						Autorizar envio automático ao concluir uma nova alocação
					</label>
				</div>
			)}
			{tab === "history" && (
				<div className="space-y-3">
					<p className="text-xs text-slate-500">
						O histórico atualiza ao abrir e quando um envio termina. Os
						destinatários são carregados apenas quando você pedir.
					</p>
					<button
						type="button"
						className={button}
						disabled={busy}
						onClick={() => void act(() => loadHistory())}
					>
						Atualizar histórico
					</button>
					{history.length === 0 && <p>Nenhum envio registrado.</p>}
					{history.map((b) => (
						<article
							key={b.id}
							className="space-y-2 rounded-lg border p-3 text-sm"
						>
							<b>
								Envio #{b.id} · {new Date(b.created).toLocaleString("pt-BR")}
							</b>
							<p>
								{b.accepted} aceito(s) · {b.failed} falha(s) · {b.pending}{" "}
								pendente(s)
								{b.unknown ? ` · ${b.unknown} a conferir no provedor` : ""}
							</p>
							<button
								type="button"
								className={button}
								onClick={() =>
									void act(async () => {
										const p = await GetNoticeDetails(b.id, 0);
										setDetails((d) => ({ ...d, [b.id]: p }));
									})
								}
							>
								Ver destinatários
							</button>
							{b.failed > 0 && !b.pending && !b.repeatedBy && (
								<button
									type="button"
									className={button}
									disabled={busy}
									onClick={() =>
										void act(async () => {
											await RetryNoticeFailures(b.id);
											await loadHistory();
										}, "Repetição das falhas iniciada")
									}
								>
									Repetir somente falhas
								</button>
							)}
							{details[b.id] && (
								<>
									<ul className="max-h-60 overflow-y-auto">
										{details[b.id].items.map((d: any) => (
											<li key={d.id} className="border-t py-2">
												{d.name} · {d.channel} ·{" "}
												{d.status === "accepted"
													? "aceito pelo provedor"
													: d.error || d.status}
											</li>
										))}
									</ul>
									{details[b.id].next > 0 && (
										<button
											type="button"
											className={button}
											onClick={() =>
												void act(async () => {
													const p = await GetNoticeDetails(
														b.id,
														details[b.id].next,
													);
													setDetails((d) => ({
														...d,
														[b.id]: {
															items: [...d[b.id].items, ...p.items],
															next: p.next,
														},
													}));
												})
											}
										>
											Mais destinatários
										</button>
									)}
								</>
							)}
						</article>
					))}
					{next > 0 && (
						<button
							type="button"
							className={button}
							disabled={busy}
							onClick={() => void act(() => loadHistory(next))}
						>
							Carregar envios anteriores
						</button>
					)}
				</div>
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
				<p role="status" className="rounded-lg bg-emerald-50 p-3 text-sm">
					{status}
				</p>
			)}
			<div className="flex justify-end gap-2 border-t pt-3">
				<button type="button" className={button} onClick={onClose}>
					Fechar
				</button>
				{tab !== "history" && (
					<button
						type="button"
						className={button}
						disabled={busy}
						onClick={() =>
							void act(async () => {
								await SaveChannelConfig(cfg);
								await loadConfig();
							}, "Configuração salva")
						}
					>
						Salvar configuração
					</button>
				)}
			</div>
		</div>
	);
}
