import { useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ArrowDownTrayIcon, ArrowUpTrayIcon, DocumentArrowUpIcon } from "@heroicons/react/24/outline";
import { upload } from "../api";
import { Aviso, BotaoPrimario, cn, IconeAjuda } from "../components/ui";

interface AppProps {
	setMapping: (data: any) => void;
}

// Separa os horários digitados como o servidor faz (preferencia.go): um por
// linha, ou separados por vírgula ou ponto e vírgula, sem repetidos.
function lerHorarios(texto: string): string[] {
	const vistos = new Set<string>();
	for (const parte of texto.split(/[\n\r,;]+/)) {
		const h = parte.trim().toLowerCase();
		if (h) vistos.add(h);
	}
	return [...vistos];
}

function Home({ setMapping }: AppProps) {
	const [file, setFile] = useState<File | null>(null);
	const [nOpcoes, setNOpcoes] = useState(5);
	// "candidatos": cada um escolhe horários em ordem de preferência (colunas
	// de opção); "definidos": quem organiza digita os horários e todos podem
	// ir em qualquer um
	const [modoHorario, setModoHorario] = useState<"candidatos" | "definidos">("candidatos");
	const [textoHorarios, setTextoHorarios] = useState("");
	const [emailDomain, setEmailDomain] = useState("@al.insper.edu.br");
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState("");
	const [arrastando, setArrastando] = useState(false);
	const inputRef = useRef<HTMLInputElement>(null);
	const navigate = useNavigate();

	async function handleUpload() {
		if (!file) {
			setError("Por favor, selecione um arquivo .xlsx.");
			return;
		}
		const horarios = modoHorario === "definidos" ? lerHorarios(textoHorarios) : undefined;
		if (horarios && horarios.length === 0) {
			setError("Informe pelo menos um horário para as mesas.");
			return;
		}
		setLoading(true);
		setError("");
		try {
			const mapping = await upload(file, nOpcoes, emailDomain, horarios);
			setMapping(mapping);
			navigate("/mapping");
		} catch (err: any) {
			setError("Erro ao processar o arquivo: " + err.message);
		} finally {
			setLoading(false);
		}
	}

	function escolher(f: File | undefined) {
		if (!f) return;
		setFile(f);
		setError("");
	}

	return (
		<div className="mx-auto flex min-h-[65vh] w-full max-w-lg flex-col justify-center">
			<div className="painel p-8">
				<div className="mb-8 text-center">
					<h1 className="text-2xl font-semibold">Candidate Allocator</h1>
					<p className="mt-2 text-sm text-[var(--text-muted)]">
						Envie a planilha com candidatos, avaliadores e restrições para montar as mesas de avaliação.
					</p>
				</div>

				<div className="space-y-5">
					{/* área de arquivo: clicar ou soltar */}
					<div
						role="button"
						tabIndex={0}
						onClick={() => inputRef.current?.click()}
						onKeyDown={(e) => (e.key === "Enter" || e.key === " ") && inputRef.current?.click()}
						onDragOver={(e) => {
							e.preventDefault();
							setArrastando(true);
						}}
						onDragLeave={() => setArrastando(false)}
						onDrop={(e) => {
							e.preventDefault();
							setArrastando(false);
							escolher(e.dataTransfer.files?.[0]);
						}}
						className={cn(
							"flex cursor-pointer flex-col items-center justify-center gap-2 rounded-[var(--radius-lg)] border border-dashed px-4 py-8 text-center transition-colors",
							arrastando
								? "border-[var(--accent)] bg-[var(--accent-soft)]"
								: file
									? "border-[rgba(31,138,59,0.35)] bg-[var(--success-soft)]"
									: "border-[var(--line-strong)] bg-[var(--bg-subtle)] hover:border-[var(--accent)]"
						)}
					>
						<DocumentArrowUpIcon className={cn("h-8 w-8", file ? "text-[var(--success)]" : "text-[var(--text-light)]")} />
						{file ? (
							<>
								<span className="text-sm font-medium">{file.name}</span>
								<span className="text-xs text-[var(--text-muted)]">Clique para trocar o arquivo</span>
							</>
						) : (
							<>
								<span className="text-sm font-medium">Clique ou arraste a planilha aqui</span>
								<span className="text-xs text-[var(--text-muted)]">Arquivo .xlsx com 3 abas</span>
							</>
						)}
						<input
							ref={inputRef}
							type="file"
							accept=".xlsx"
							aria-label="Arquivo Excel (.xlsx)"
							onChange={(e) => escolher(e.target.files?.[0])}
							className="hidden"
						/>
					</div>

					<div>
						<span className="rotulo flex items-center gap-1.5">
							Horários
							<IconeAjuda texto="Candidatos escolhem: a planilha tem colunas de opção (1ª, 2ª...) e a 1ª opção tem prioridade. Eu defino: você digita os horários das mesas, todo candidato pode ir em qualquer um e só as restrições dos avaliadores pesam." />
						</span>
						<div role="radiogroup" aria-label="Horários" className="grid grid-cols-2 gap-1 rounded-[var(--radius)] border border-[color:var(--line-strong)] bg-[color:var(--bg-subtle)] p-1">
							{(
								[
									["candidatos", "Candidatos escolhem"],
									["definidos", "Eu defino"],
								] as const
							).map(([modo, rotulo]) => (
								<button
									key={modo}
									type="button"
									role="radio"
									aria-checked={modoHorario === modo}
									onClick={() => setModoHorario(modo)}
									className={cn(
										"rounded-[calc(var(--radius)-2px)] px-3 py-1.5 text-sm transition-colors",
										modoHorario === modo
											? "bg-[color:var(--text)] font-medium text-white"
											: "text-[color:var(--text-muted)] hover:bg-white hover:text-[color:var(--text)]"
									)}
								>
									{rotulo}
								</button>
							))}
						</div>
					</div>

					{modoHorario === "definidos" && (
						<label className="block">
							<span className="rotulo flex items-center gap-1.5">
								Horários das mesas
								<IconeAjuda texto="Um por linha (ou separados por vírgula). Colunas de opção na planilha, se houver, são ignoradas." />
							</span>
							<textarea
								rows={4}
								value={textoHorarios}
								onChange={(e) => setTextoHorarios(e.target.value)}
								placeholder={"segunda 14-16\nterça 10-12\nquarta 16-18"}
								className="campo resize-y"
							/>
							<span className="mt-1 block text-xs text-[color:var(--text-muted)]">
								{lerHorarios(textoHorarios).length} horário(s)
							</span>
						</label>
					)}

					<div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
						{modoHorario === "candidatos" && (
							<label className="block">
								<span className="rotulo flex items-center gap-1.5">
									Opções de horário
									<IconeAjuda texto="Quantas colunas de horário (1ª opção, 2ª opção...) cada candidato preencheu." />
								</span>
								<input
									type="number"
									min={1}
									max={10}
									value={nOpcoes}
									onChange={(e) => setNOpcoes(Math.max(1, Math.min(10, Number(e.target.value))))}
									className="campo"
								/>
							</label>
						)}
						<label className="block">
							<span className="rotulo flex items-center gap-1.5">
								Domínio do email
								<IconeAjuda texto="Usado para validar o email institucional dos candidatos." />
							</span>
							<input
								type="text"
								value={emailDomain}
								onChange={(e) => setEmailDomain(e.target.value)}
								placeholder="@al.insper.edu.br"
								className="campo font-mono"
							/>
						</label>
					</div>

					{error && <Aviso tom="perigo">{error}</Aviso>}

					<BotaoPrimario onClick={handleUpload} disabled={loading} className="w-full py-2.5">
						<ArrowUpTrayIcon className="h-4 w-4" />
						{loading ? "Processando..." : "Iniciar →"}
					</BotaoPrimario>
				</div>
			</div>

			<a
				href="/api/exemplo"
				download="base_exemplo.xlsx"
				className="mx-auto mt-4 inline-flex items-center gap-1.5 text-sm text-[var(--text-muted)] hover:text-[var(--accent)]"
			>
				<ArrowDownTrayIcon className="h-4 w-4" />
				Baixar planilha de exemplo
			</a>
		</div>
	);
}

export default Home;
