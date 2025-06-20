import React, { useState } from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import App from "./App";
import MappingPage from "./MappingPage";
import "./index.css";

function Root() {
	const [mappingData, setMappingData] = useState<any>(null);

	return (
		<React.StrictMode>
			<BrowserRouter>
				<Routes>
					<Route
						path="/"
						element={<App setMapping={setMappingData} />}
					/>
					<Route
						path="/mapping"
						element={<MappingPage mapping={mappingData} />}
					/>
				</Routes>
			</BrowserRouter>
		</React.StrictMode>
	);
}

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
	<Root />
);
