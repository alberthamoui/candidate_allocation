import { useState } from "react";
import { useNavigate } from "react-router-dom";
import {
	ArrowUpTrayIcon,
	CheckBadgeIcon,
	CommandLineIcon,
	DocumentChartBarIcon,
	ShieldCheckIcon,
} from "@heroicons/react/24/outline";
import {
	Greet,
	GetAvaliadorMappingFieldInfos,
	GetCandidateMappingFieldInfos,
	GetRestricaoMappingFieldInfos,
	SuggestMapping,
	SuggestMappingAvaliador,
	SuggestMappingRestricao,
} from "../wailsjs/go/main/App";
import type { MappingDraft, MappingFieldInfo, MappingItem } from "./importTypes";
import {
	FieldLabel,
	PrimaryButton,
	SecondaryButton,
	SectionCard,
	StatusBadge,
} from "./workflowShell";

interface AppProps {
	setMapping: (data: MappingDraft[]) => void;
	setMappingAvaliadores: (data: MappingDraft[]) => void;
	setMappingRestricoes: (data: MappingDraft[]) => void;
	setCandidateFieldInfos: (data: MappingFieldInfo[]) => void;
	setAvaliadorFieldInfos: (data: MappingFieldInfo[]) => void;
	setRestricaoFieldInfos: (data: MappingFieldInfo[]) => void;
}

function App({
	setMapping,
	setMappingAvaliadores,
	setMappingRestricoes,
	setCandidateFieldInfos,
	setAvaliadorFieldInfos,
	setRestricaoFieldInfos,
}: AppProps) {
	const [resultText, setResultText] = useState(
		"Por favor, digite seu nome abaixo 👇"
	);
	const [name, setName] = useState("");
	const [fileResult, setFileResult] = useState("");
	const [file, setFile] = useState<File | null>(null);
	const updateName = (e: React.ChangeEvent<HTMLInputElement>) =>
		setName(e.target.value);
	const updateResultText = (result: string) => setResultText(result);
	const navigate = useNavigate();
	function greet() {
		Greet(name).then(updateResultText);
	}
	function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
		const selected = e.target.files?.[0] ?? null;
		if (!selected) return;
		setFile(selected);
	}

	function makeDrafts(
		items: MappingItem[],
		fieldInfos: MappingFieldInfo[],
		allowExtraFields: boolean
	): MappingDraft[] {
		const coreVariables = new Set(fieldInfos.map((fieldInfo) => fieldInfo.variavel));

		return items.map((item) => {
			const isCore = coreVariables.has(item.variavel);
			const isAvailableSuggestion = allowExtraFields && !isCore;

			return {
				...item,
				variavel: isAvailableSuggestion ? "" : item.variavel,
				includeWhenUnmapped: item.includeWhenUnmapped ?? false,
				clientId:
					typeof crypto !== "undefined" && "randomUUID" in crypto
						? crypto.randomUUID()
						: `${Date.now()}-${Math.random()}`,
				manualExtra: false,
			};
		});
	}

	async function handleFile() {
		if (file) {
			const reader = new FileReader();
			reader.onload = async (event) => {
				try {
					const fileData = event.target?.result;
					if (!fileData) {
						setFileResult("Erro ao ler o arquivo.");
						return;
					}
					// Converte o ArrayBuffer para Uint8Array
					const data = new Uint8Array(fileData as ArrayBuffer);
					const mappingCandidatos = await SuggestMapping(Array.from(data), 5);
					const [
						candidateFieldInfos,
						mappingAvaliadores,
						avaliadorFieldInfos,
						mappingRestricoes,
						restricaoFieldInfos,
					] = await Promise.all([
						GetCandidateMappingFieldInfos(),
						SuggestMappingAvaliador(),
						GetAvaliadorMappingFieldInfos(),
						SuggestMappingRestricao(),
						GetRestricaoMappingFieldInfos(),
					]);

					setMapping(makeDrafts(mappingCandidatos, candidateFieldInfos, true));
					setCandidateFieldInfos(candidateFieldInfos);
					setMappingAvaliadores(
						makeDrafts(mappingAvaliadores, avaliadorFieldInfos, true)
					);
					setAvaliadorFieldInfos(avaliadorFieldInfos);
					setMappingRestricoes(
						makeDrafts(mappingRestricoes, restricaoFieldInfos, false)
					);
					setRestricaoFieldInfos(restricaoFieldInfos);
					navigate("/mapping");
				} catch (error) {
					setFileResult("Erro ao processar o arquivo: " + error);
				}
			};
			reader.readAsArrayBuffer(file);
		} else {
			setFileResult("Por favor, selecione um arquivo.");
		}
	}

	return (
		<div className="space-y-6">
			<section className="executive-card executive-card-strong relative overflow-hidden p-8 md:p-10">
				<div className="absolute inset-y-0 right-0 hidden w-1/3 bg-[radial-gradient(circle_at_top,rgba(178,122,68,0.18),transparent_62%)] lg:block" />
				<div className="relative grid gap-8 lg:grid-cols-[1.2fr_0.8fr] lg:items-center">
					<div>
						<StatusBadge tone="accent">Confiança, clareza e controle</StatusBadge>
						<h2 className="mt-5 max-w-3xl text-4xl text-[var(--text)] md:text-5xl">
							Conduza a alocação de candidatos com uma experiência feita para RH corporativo.
						</h2>
						<p className="mt-4 max-w-2xl text-base leading-8 text-[var(--muted)]">
							Este fluxo importa a planilha, organiza o mapeamento das entidades,
							revisa conflitos e prepara uma distribuição configurável sem expor a
							operação a telas confusas ou decisões opacas.
						</p>
						<div className="mt-8 grid gap-4 md:grid-cols-3">
							<div className="rounded-[24px] border border-[var(--line)] bg-white/70 p-4">
								<ShieldCheckIcon className="h-7 w-7 text-[var(--accent-strong)]" />
								<h3 className="mt-3 text-xl text-[var(--text)]">Confiável</h3>
								<p className="mt-2 text-sm leading-6 text-[var(--muted)]">
									Revisão por etapas, validação de duplicidades e rastreabilidade do processo.
								</p>
							</div>
							<div className="rounded-[24px] border border-[var(--line)] bg-white/70 p-4">
								<DocumentChartBarIcon className="h-7 w-7 text-[var(--accent-strong)]" />
								<h3 className="mt-3 text-xl text-[var(--text)]">Configurable</h3>
								<p className="mt-2 text-sm leading-6 text-[var(--muted)]">
									Parâmetros e critérios adaptáveis para necessidades específicas de cada empresa.
								</p>
							</div>
							<div className="rounded-[24px] border border-[var(--line)] bg-white/70 p-4">
								<CheckBadgeIcon className="h-7 w-7 text-[var(--accent-strong)]" />
								<h3 className="mt-3 text-xl text-[var(--text)]">Intuitivo</h3>
								<p className="mt-2 text-sm leading-6 text-[var(--muted)]">
									Ajuda contextual fixa e fluxo progressivo para evitar dúvida operacional.
								</p>
							</div>
						</div>
					</div>

					<div id="file-section" className="executive-panel-dark p-6 md:p-7">
						<div className="executive-pill border-white/10 bg-white/5 text-[#f2debf]">
							Entrada do Excel
						</div>
						<h3 className="mt-4 text-3xl text-white">Importar base principal</h3>
						<p className="mt-3 text-sm leading-6 text-[#e2d3c1]">
							Selecione a planilha oficial do processo seletivo. O sistema abrirá
							o mapeamento de candidatos e preparará o restante do wizard.
						</p>
						<div className="mt-6 space-y-4">
							<div className="rounded-[24px] border border-white/10 bg-white/5 p-4">
								<FieldLabel
									label="Arquivo Excel"
									description="Use a planilha consolidada do processo seletivo."
								/>
								<label htmlFor="fileInput" className="block">
									<input
										type="file"
										id="fileInput"
										onChange={handleFileChange}
										className="executive-input w-full cursor-pointer border-white/10 bg-white/90 text-[var(--text)]"
									/>
								</label>
							</div>
							<PrimaryButton
								onClick={handleFile}
								className="w-full justify-center bg-[linear-gradient(135deg,#b27a44_0%,#85562e_100%)]"
								data-testid="start-import-button"
							>
								<ArrowUpTrayIcon className="h-5 w-5" />
								Executar função de arquivo
							</PrimaryButton>
							{fileResult && (
								<div
									id="fileResult"
									className="rounded-[20px] border border-[rgba(156,66,63,0.24)] bg-[rgba(156,66,63,0.12)] px-4 py-3 text-sm font-semibold text-[#f7d6d2]"
								>
									{fileResult}
								</div>
							)}
						</div>
					</div>
				</div>
			</section>

			<div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
				<SectionCard
					title="Como o fluxo opera"
					description="O sistema trabalha em uma sequência previsível. Isso reduz retrabalho e facilita auditoria interna."
				>
					<div className="grid gap-4 md:grid-cols-2">
						{[
							"Importação da planilha e leitura inicial das abas.",
							"Mapeamento dos candidatos e revisão dos conflitos.",
							"Conferência das restrições com foco no schema central.",
							"Preparação dos avaliadores e configuração final da alocação.",
						].map((item, index) => (
							<div
								key={item}
								className="rounded-[22px] border border-[var(--line)] bg-white/70 px-4 py-4"
							>
								<div className="text-xs font-semibold uppercase tracking-[0.2em] text-[var(--accent-strong)]">
									Etapa {index + 1}
								</div>
								<p className="mt-3 text-sm leading-6 text-[var(--text)]">{item}</p>
							</div>
						))}
					</div>
				</SectionCard>

				<SectionCard
					title="Console técnico"
					description="Bloco secundário do ambiente Wails. Mantido apenas para testes e verificação local."
					aside={<StatusBadge>Utilitário</StatusBadge>}
				>
					<div id="result" className="rounded-[20px] border border-[var(--line)] bg-white/80 px-4 py-4 text-sm text-[var(--muted)]">
						{resultText}
					</div>
					<div id="input" className="mt-4 space-y-4">
						<div>
							<FieldLabel label="Nome para saudação" description="Usado pela função demonstrativa do backend Wails." />
							<input
								id="name"
								onChange={updateName}
								autoComplete="off"
								name="input"
								type="text"
								className="executive-input"
								placeholder="Digite seu nome"
							/>
						</div>
						<SecondaryButton onClick={greet} className="w-full justify-center">
							<CommandLineIcon className="h-5 w-5" />
							Greet
						</SecondaryButton>
					</div>
				</SectionCard>
			</div>
		</div>
	);
}

export default App;
