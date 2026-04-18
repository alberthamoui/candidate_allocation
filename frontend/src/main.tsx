import React, { useState } from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import Home from "./pages/Home";
import MappingPage from "./pages/MappingPage";
import UploadAvaliador from "./pages/UploadAvaliador";
import UploadRestricao from "./pages/UploadRestricao";
import Resultado from "./pages/Resultado";
import "./index.css";
import VerifyUserPage from "./pages/VerifyUsers";
import {
  buildUsuarios,
  buildAvaliadores,
  buildRestricoes,
  saveAvaliadores,
  saveRestricoes,
} from "./api";

function Root() {
  const [mappingData, setMappingData] = useState<any>(null);

  const [users, setUsers] = useState<any>(null);
  const [duplicatas, setDuplicatas] = useState<any>(null);

  const [mappingAvaliador, setMappingAvaliador] = useState<any>(null);
  const [mappingRestricao, setMappingRestricao] = useState<any>(null);

  const [alocacaoResult, setAlocacaoResult] = useState<any>(null);

  return (
    <React.StrictMode>
      <BrowserRouter>
        <Routes>
          <Route
            path="/"
            element={<Home setMapping={setMappingData} />}
          />

          {/* Step 1 — candidatos */}
          <Route
            path="/mapping"
            element={
              <MappingPage
                mapping={mappingData}
                buildFn={buildUsuarios}
                onSuccess={(result) => {
                  setUsers(result.usuarios);
                  setDuplicatas(result.duplicates);
                }}
                nextRoute="/verify"
              />
            }
          />
          <Route
            path="/verify"
            element={
              <VerifyUserPage
                usuarios={users}
                duplicates={duplicatas}
              />
            }
          />

          {/* Step 2 — avaliadores */}
          <Route
            path="/upload-avaliador"
            element={<UploadAvaliador setMappingAvaliador={setMappingAvaliador} />}
          />
          <Route
            path="/mapping-avaliador"
            element={
              <MappingPage
                mapping={mappingAvaliador}
                buildFn={buildAvaliadores}
                onSuccess={async (result) => {
                  await saveAvaliadores(result);
                }}
                nextRoute="/upload-restricao"
              />
            }
          />

          {/* Step 3 — restrições */}
          <Route
            path="/upload-restricao"
            element={<UploadRestricao setMappingRestricao={setMappingRestricao} />}
          />
          <Route
            path="/mapping-restricao"
            element={
              <MappingPage
                mapping={mappingRestricao}
                buildFn={buildRestricoes}
                onSuccess={async (result) => {
                  await saveRestricoes(result);
                }}
                nextRoute="/resultado"
              />
            }
          />

          <Route
            path="/resultado"
            element={<Resultado setAlocacaoResult={setAlocacaoResult} />}
          />
        </Routes>
      </BrowserRouter>
    </React.StrictMode>
  );
}

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <Root />
);
