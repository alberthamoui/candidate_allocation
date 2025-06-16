import { useState } from "react";
import logo from "./assets/images/logo-universal.png";
import "./App.css";
import { Greet, ImportExcelInteractive } from "../wailsjs/go/main/App";

function App() {
	const [resultText, setResultText] = useState(
		"Please enter your name below 👇"
	);
	const [name, setName] = useState("");
	const updateName = (e: any) => setName(e.target.value);
	const updateResultText = (result: string) => setResultText(result);

	function greet() {
		Greet(name).then(updateResultText);
	}

	const [filePath, setFilePath] = useState<string>("");
	const [result, setResult] = useState<any[]>([]);
	// captura o path do arquivo selecionado
	const onFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
		const file = e.target.files?.[0];
		if (!file) {
			setFilePath("");
			return;
		}
		// no desktop (wails) o arquivo traz a propriedade .path
		// caso não venha, cai no file.name
		// @ts-ignore
		setFilePath((file as any).path || file.name);
	};
	const processExcel = async () => {
		if (!filePath) {
			alert("Selecione primeiro um arquivo .xlsx");
			return;
		}

		// 1) pergunta nOpções
		const nOpcoes = parseInt(
			prompt("Quantas opções de alocação?") || "0",
			10
		);
		if (isNaN(nOpcoes) || nOpcoes < 1) {
			alert("Número de opções inválido");
			return;
		}
		// 2) dispara o parsing interativo no Go
		//    ele ainda vai tentar ler do stdin, mas mostramos
		//    no front os prompts para você copiar as respostas
		//    ou você pode manter o terminal aberto para responder lá.
		try {
			const users = await ImportExcelInteractive(filePath);
			console.log("Resultado do parsing:", users);
			setResult(users);
		} catch (err) {
			console.error(err);
			alert("Erro ao processar Excel: " + err);
		}
	};

	return (
		<div id="App">
			<div id="result" className="result">
				{resultText}
			</div>
			<div id="input" className="input-box">
				<input
					id="name"
					className="input"
					onChange={updateName}
					autoComplete="off"
					name="input"
					type="text"
				/>
				<button className="btn" onClick={greet}>
					Greet
				</button>
			</div>
			<h1>Processar Excel</h1>
			<div className="input-box">
				<input type="file" accept=".xlsx" onChange={onFileChange} />
				<button className="btn" onClick={processExcel}>
					Processar
				</button>
			</div>
			<h2>Resultado</h2>
			<pre>{JSON.stringify(result, null, 2)}</pre>
		</div>
	);
}

export default App;
