import { useState } from "react";
import wailsLogo from "./assets/wails.png";
import "./App.css";
import { Greet, SuggestMapping } from "../wailsjs/go/main/App";

function App() {
	const [resultText, setResultText] = useState(
		"Por favor, digite seu nome abaixo 👇"
	);
	const [name, setName] = useState("");
	const [fileResult, setFileResult] = useState("");
	const [file, setFile] = useState<File | null>(null);
	const updateName = (e: React.ChangeEvent<HTMLInputElement>) =>
		setName(e.target.value);
	const updateResultText = (result: string) => setResultText(result);

	function greet() {
		Greet(name).then(updateResultText);
	}
	function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
		const selected = e.target.files?.[0] ?? null;
		if (!selected) return;
		setFile(selected);
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
					// Chama a nova função que aceita os dados do arquivo.
					const result = await SuggestMapping(Array.from(data), 5);
					setFileResult(JSON.stringify(result, null, 2));
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
		<div className="min-h-screen bg-gradient-to-br from-blue-50 to-green-100 flex flex-col items-center justify-center py-8">
			<div className="bg-white shadow-xl rounded-2xl p-10 w-full max-w-md flex flex-col items-center space-y-8">
				<img
					src={wailsLogo}
					alt="Wails Logo"
					className="w-24 h-24 mb-2 drop-shadow-lg"
				/>
				<h1 className="text-3xl font-bold text-blue-700 mb-2 text-center">
					Candidate Allocator
				</h1>
				<div
					id="result"
					className="text-lg font-medium text-gray-700 text-center"
				>
					{resultText}
				</div>
				<div
					id="input"
					className="flex flex-col items-center space-y-4 w-full"
				>
					<input
						id="name"
						onChange={updateName}
						autoComplete="off"
						name="input"
						type="text"
						className="border border-gray-300 p-3 rounded-lg w-full focus:outline-none focus:ring-2 focus:ring-blue-300 transition"
						placeholder="Digite seu nome"
					/>
					<button
						onClick={greet}
						className="bg-blue-500 hover:bg-blue-600 transition text-white px-6 py-2 rounded-lg shadow font-semibold w-full"
					>
						Greet
					</button>
				</div>
				<div
					id="file-section"
					className="flex flex-col items-center space-y-4 w-full pt-4 border-t border-gray-200"
				>
					<label htmlFor="fileInput" className="w-full">
						<input
							type="file"
							id="fileInput"
							onChange={handleFileChange}
							className="border border-gray-300 p-2 rounded-lg w-full bg-gray-50"
						/>
					</label>
					<button
						onClick={handleFile}
						className="bg-green-500 hover:bg-green-600 transition text-white px-6 py-2 rounded-lg shadow font-semibold w-full"
					>
						Executar função de arquivo
					</button>
					{fileResult && (
						<div
							id="fileResult"
							className="text-base font-medium text-green-700 bg-green-100 rounded p-2 w-full text-center"
						>
							{fileResult}
						</div>
					)}
				</div>
			</div>
		</div>
	);
}

export default App;
