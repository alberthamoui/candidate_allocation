import React, { useState } from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import Home from "./pages/Home";
import MappingPage from "./pages/MappingPage";
import UploadAvaliador from "./pages/UploadAvaliador";
import UploadRestricao from "./pages/UploadRestricao";
import Resultado from "./pages/Resultado";
import Parametros from "./pages/Parametros";
import "./index.css";
import VerifyUserPage from "./pages/VerifyUsers";
import Layout from "./components/Layout";
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

  return (
    <React.StrictMode>
      <BrowserRouter>
        <Routes>
          <Route element={<Layout />}>
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
                titulo="Mapeamento de candidatos"
                descricao="Confira qual coluna da planilha alimenta cada campo. A sugestão é feita pelo nome das colunas; arraste para trocar."
                rotuloConfirmar="Continuar para a revisão"
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
                titulo="Mapeamento de avaliadores"
                descricao="Confira as colunas de nome, email e sigla da aba de avaliadores. Ao confirmar, os avaliadores são salvos."
                rotuloConfirmar="Salvar avaliadores"
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
                nextRoute="/parametros"
                titulo="Mapeamento de restrições"
                descricao="Confira as colunas da aba de restrições: o candidato e as siglas dos avaliadores que não podem ou preferem não avaliá-lo."
                rotuloConfirmar="Salvar restrições"
              />
            }
          />

          {/* Step 4 — parâmetros da alocação */}
          <Route path="/parametros" element={<Parametros />} />

          <Route
            path="/resultado"
            element={<Resultado />}
          />
          </Route>
        </Routes>
      </BrowserRouter>
    </React.StrictMode>
  );
}

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <Root />
);
