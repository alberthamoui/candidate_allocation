import { useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ArrowDownTrayIcon, ArrowUpTrayIcon, DocumentArrowUpIcon } from "@heroicons/react/24/outline";
import { upload, configAvisosVazia } from "../api";
import AvisosForm from "../components/AvisosForm";
import { Aviso, BotaoPrimario, cn, IconeAjuda } from "../components/ui";

interface AppProps {
	setMapping: (data: any) => void;
}

function Home({ setMapping }: AppProps) {
	const [file, setFile] = useState<File | null>(null);
	const [nOpcoes, setNOpcoes] = useState(5);
	const [emailDomain, setEmailDomain] = useState("@al.insper.edu.br");
	const [usarPreferencia, setUsarPreferencia] = useState(true);
	const [configurar, setConfigurar] = useState(false);
	const [cfg, setCfg] = useState(configAvisosVazia);
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
		setLoading(true);
		setError("");
		try {
			const mapping = await upload(file, nOpcoes, emailDomain, {
				usarPreferencia,
				avisos: configurar ? { ...cfg, ativo: true } : undefined
			});
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
					<section className="space-y-3" aria-label="Configuração inicial">
						<p className="text-sm font-medium">Quer enviar avisos de entrevista por e-mail ou WhatsApp?</p>
						<div className="flex gap-3">
							<button className="text-sm underline" onClick={() => setConfigurar(true)} aria-pressed={configurar}>
								Configurar avisos
							</button>
							<button className="text-sm underline" onClick={() => setConfigurar(false)} aria-pressed={!configurar}>
								Agora não
							</button>
						</div>
						<p className="text-xs text-[var(--text-muted)]">Opcional. Você também pode configurar os canais depois.</p>
						{configurar && <AvisosForm cfg={{ ...cfg, ativo: true }} onChange={setCfg} mostrarAtivo={false} />}
					</section>
					<label className="flex gap-2 text-sm">
						<input type="checkbox" checked={usarPreferencia} onChange={(e) => setUsarPreferencia(e.target.checked)} />
						Usar preferências de horário dos candidatos
					</label>
					{!usarPreferencia && (
						<Aviso>Você definirá os horários disponíveis nos parâmetros. CPF é opcional em todos os casos.</Aviso>
					)}
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
						<DocumentArrowUpIcon
							className={cn("h-8 w-8", file ? "text-[var(--success)]" : "text-[var(--text-light)]")}
						/>
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

					<div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
						<label className="block">
							<span className="rotulo flex items-center gap-1.5">
								Opções de horário
								<IconeAjuda texto="Quantas colunas de horário (1ª opção, 2ª opção...) cada candidato preencheu." />
							</span>
							<input
								type="number"
								disabled={!usarPreferencia}
								min={1}
								max={10}
								value={nOpcoes}
								onChange={(e) => setNOpcoes(Math.max(1, Math.min(10, Number(e.target.value))))}
								className="campo"
							/>
						</label>
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
