import React, { useState } from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import App from "./App";
import MappingPage from "./MappingPage";
import "./index.css";
import VerifyUserPage from "./VerifyUsers";
import VerifyAvaliadoresPage from "./VerifyAvaliadores";
import MappingAvaliadoresPage from "./MappingAvaliadoresPage";
import MappingResticoesPage from "./MappingRerstricoesPage";
import SuccessPage from "./SuccessPage";

function Root() {
	const [mappingData, setMappingData] = useState<any>(null);
	const [mappingAvaliadores, setMappingAvaliadores] = useState<any>(null);
	const [mappingRestricoes, setMappingRestricoes] = useState<any>(null);

	const [users, setUsers] = useState<any>(null);
	const [avaliadores, setAvaliadores] = useState<any>(null);
	const [restricoes, setRestricoes] = useState<any>(null);

	const [duplicatas, setDuplicatas] = useState<any>(null);
	const [duplicateFields, setDuplicateFields] = useState<string[]>([]);

	const [avaliadoresDuplicatas, setAvaliadoresDuplicatas] = useState<any>(null);
	const [avaliadoresDuplicateFields, setAvaliadoresDuplicateFields] = useState<string[]>([]);

	return (
		<React.StrictMode>
			<BrowserRouter>
				<Routes>
					<Route
						path="/"
						element={
							<App
								setMapping={setMappingData}
								setMappingAvaliadores={setMappingAvaliadores}
								setMappingRestricoes={setMappingRestricoes}
							/>
						}
					/>
					<Route
						path="/mapping"
						element={
							<MappingPage
								mapping={mappingData}
								setUsers={setUsers}
								setDuplicatas={setDuplicatas}
								setDuplicateFields={setDuplicateFields}
							/>
						}
					/>
					<Route
						path="/mappingAvaliadores"
						element={
							<MappingAvaliadoresPage
								mapping={mappingAvaliadores}
								setAvaliadores={setAvaliadores}
								setDuplicatas={setAvaliadoresDuplicatas}
								setDuplicateFields={setAvaliadoresDuplicateFields}
							/>
						}
					/>
					<Route
						path="/mappingRestricoes"
						element={
							<MappingResticoesPage
								mapping={mappingRestricoes}
								setRestricoes={setRestricoes}
							/>
						}
					/>
					<Route
						path="/verify"
						element={
							<VerifyUserPage
								usuarios={users}
								restricoes={restricoes}
								duplicates={duplicatas}
								duplicateFields={duplicateFields}
							/>
						}
					/>
					<Route
						path="/verifyAvaliadores"
						element={
							<VerifyAvaliadoresPage
								avaliadores={avaliadores}
								duplicates={avaliadoresDuplicatas}
								duplicateFields={avaliadoresDuplicateFields}
							/>
						}
					/>
					<Route path="/success" element={<SuccessPage />} />
				</Routes>
			</BrowserRouter>
		</React.StrictMode>
	);
}

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
	<Root />
);
