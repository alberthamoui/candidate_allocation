import React, { useState } from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import App from "./App";
import MappingPage from "./MappingPage";
import "./index.css";
import VerifyUserPage from "./VerifyUsers";
import VerifyAvaliadoresPage from "./VerifyAvaliadores";
import VerifyRestricoesPage from "./VerifyRestricoes";
import MappingAvaliadoresPage from "./MappingAvaliadoresPage";
import MappingResticoesPage from "./MappingRerstricoesPage";
import SuccessPage from "./SuccessPage";
import AllocationLoadingPage from "./AllocationLoadingPage";
import AllocationResultPage from "./AllocationResultPage";
import AllocationConfigPage from "./AllocationConfigPage";
import type { MappingDraft, MappingFieldInfo } from "./importTypes";
import { WorkflowLayout } from "./workflowShell";
import ScrollToTop from "./ScrollToTop";
import { AllocationRunProvider } from "./AllocationRunContext";

function Root() {
	const [mappingData, setMappingData] = useState<MappingDraft[] | null>(null);
	const [mappingAvaliadores, setMappingAvaliadores] = useState<MappingDraft[] | null>(null);
	const [mappingRestricoes, setMappingRestricoes] = useState<MappingDraft[] | null>(null);
	const [candidateFieldInfos, setCandidateFieldInfos] = useState<MappingFieldInfo[]>([]);
	const [avaliadorFieldInfos, setAvaliadorFieldInfos] = useState<MappingFieldInfo[]>([]);
	const [restricaoFieldInfos, setRestricaoFieldInfos] = useState<MappingFieldInfo[]>([]);

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
				<AllocationRunProvider>
					<ScrollToTop />
					<Routes>
					<Route element={<WorkflowLayout />}>
						<Route
							path="/"
							element={
								<App
									setMapping={setMappingData}
									setMappingAvaliadores={setMappingAvaliadores}
									setMappingRestricoes={setMappingRestricoes}
									setCandidateFieldInfos={setCandidateFieldInfos}
									setAvaliadorFieldInfos={setAvaliadorFieldInfos}
									setRestricaoFieldInfos={setRestricaoFieldInfos}
								/>
							}
						/>
						<Route
							path="/mapping"
							element={
								<MappingPage
									mapping={mappingData}
									setMapping={setMappingData}
									fieldInfos={candidateFieldInfos}
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
									setMapping={setMappingAvaliadores}
									fieldInfos={avaliadorFieldInfos}
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
									setMapping={setMappingRestricoes}
									fieldInfos={restricaoFieldInfos}
									setRestricoes={setRestricoes}
								/>
							}
						/>
						<Route
							path="/verify"
							element={
								<VerifyUserPage
									usuarios={users}
									duplicates={duplicatas}
									duplicateFields={duplicateFields}
								/>
							}
						/>
						<Route
							path="/verifyRestricoes"
							element={<VerifyRestricoesPage restricoes={restricoes} />}
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
						<Route path="/allocation-config" element={<AllocationConfigPage />} />
						<Route path="/allocation-loading" element={<AllocationLoadingPage />} />
						<Route path="/allocation-result" element={<AllocationResultPage />} />
					</Route>
					</Routes>
				</AllocationRunProvider>
			</BrowserRouter>
		</React.StrictMode>
	);
}

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
	<Root />
);
