import wailsLogo from "./assets/wails.png";
import "./App.css";
import { Greet } from "../wailsjs/go/main/App";
import { useState } from "react";

function App() {
	const [resultText, setResultText] = useState(
		"Please enter your name below 👇"
	);
	const [name, setName] = useState("");
	const [fileResult, setFileResult] = useState("");
	const updateName = (e: React.ChangeEvent<HTMLInputElement>) =>
		setName(e.target.value);
	const updateResultText = (result: string) => setResultText(result);

	function greet() {
		Greet(name).then(updateResultText);
	}

	function handleFile() {
		// Função executada ao clicar no botão associado ao input de arquivo.
		setFileResult("rodou");
	}
	return (
		<div className="min-h-screen bg-white grid grid-cols-1 place-items-center justify-items-center mx-auto py-8">
			<div id="App" className="space-y-8">
				<div id="result" className="text-xl font-medium">
					{resultText}
				</div>
				<div
					id="input"
					className="flex flex-col items-center space-y-4"
				>
					<input
						id="name"
						onChange={updateName}
						autoComplete="off"
						name="input"
						type="text"
						className="border border-gray-300 p-2 rounded-md"
						placeholder="Digite seu nome"
					/>
					<button
						onClick={greet}
						className="bg-blue-500 text-white px-4 py-2 rounded-md"
					>
						Greet
					</button>
				</div>
				<div
					id="file-section"
					className="flex flex-col items-center space-y-4"
				>
					<input
						type="file"
						id="fileInput"
						className="border border-gray-300 p-2 rounded-md"
					/>
					<button
						onClick={handleFile}
						className="bg-green-500 text-white px-4 py-2 rounded-md"
					>
						Executar função de arquivo
					</button>
					{fileResult && (
						<div id="fileResult" className="text-lg font-medium">
							{fileResult}
						</div>
					)}
				</div>
			</div>
		</div>
	);
}

export default App;
