import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { ArrowUpTrayIcon } from "@heroicons/react/24/outline";
import {
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
	const [fileResult, setFileResult] = useState("");
	const [file, setFile] = useState<File | null>(null);
	const [isLoading, setIsLoading] = useState(false);
	const navigate = useNavigate();

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
			setIsLoading(true);
			const reader = new FileReader();
			reader.onload = async (event) => {
				try {
					const fileData = event.target?.result;
					if (!fileData) {
						setFileResult("Erro ao ler o arquivo.");
						setIsLoading(false);
						return;
					}
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
					setIsLoading(false);
				}
			};
			reader.readAsArrayBuffer(file);
		} else {
			setFileResult("Por favor, selecione um arquivo.");
		}
	}

	return (
		<div className="flex flex-col items-center justify-center min-h-[70vh] max-w-lg mx-auto w-full">
			<div className="w-full minimal-panel p-8">
				<div className="mb-8 text-center">
					<h1 className="text-2xl font-semibold mb-2">Importação de Dados</h1>
					<p className="text-sm text-gray-500">
						Selecione a planilha (.xlsx) para iniciar um novo processo de alocação.
					</p>
				</div>

				<div className="space-y-6">
					<div className="bg-gray-50 border border-gray-100 rounded-lg p-4">
						<FieldLabel
							label="Arquivo Excel"
						/>
						<input
							type="file"
							id="fileInput"
							data-testid="excel-file-input"
							onChange={handleFileChange}
							className="minimal-input mt-2 cursor-pointer bg-white"
							accept=".xlsx,.xls,.csv"
						/>
					</div>
					
					<PrimaryButton
						onClick={handleFile}
						className="w-full py-2.5 text-base"
						data-testid="start-import-button"
						disabled={isLoading}
					>
						<ArrowUpTrayIcon className="h-5 w-5" />
						{isLoading ? "Processando..." : "Continuar"}
					</PrimaryButton>
					
					{fileResult && (
						<div data-testid="import-error-message" className="bg-red-50 text-red-600 border border-red-100 rounded-lg p-3 text-sm font-medium text-center">
							{fileResult}
						</div>
					)}
				</div>
			</div>
		</div>
	);
}

export default App;
