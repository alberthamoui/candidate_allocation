import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { upload } from "../api";

interface AppProps {
  setMapping: (data: any) => void;
}

function Home({ setMapping }: AppProps) {
  const [file, setFile] = useState<File | null>(null);
  const [nOpcoes, setNOpcoes] = useState(5);
  const [emailDomain, setEmailDomain] = useState("@al.insper.edu.br");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const navigate = useNavigate();

  async function handleUpload() {
    if (!file) {
      setError("Por favor, selecione um arquivo .xlsx.");
      return;
    }
    setLoading(true);
    setError("");
    try {
      const mapping = await upload(file, nOpcoes, emailDomain);
      setMapping(mapping);
      navigate("/mapping");
    } catch (err: any) {
      setError("Erro ao processar o arquivo: " + err.message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-green-100 flex flex-col items-center justify-center py-8">
      <div className="bg-white shadow-xl rounded-2xl p-10 w-full max-w-md flex flex-col items-center space-y-8">
        <h1 className="text-3xl font-bold text-blue-700 text-center">
          Candidate Allocator
        </h1>
        <p className="text-gray-500 text-sm text-center">
          Faça upload do arquivo Excel com os dados de candidatos, avaliadores e restrições para iniciar a alocação.
        </p>

        <div className="flex flex-col space-y-4 w-full">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Número de opções de horário por candidato
            </label>
            <input
              type="number"
              min={1}
              max={10}
              value={nOpcoes}
              onChange={(e) =>
                setNOpcoes(Math.max(1, Math.min(10, Number(e.target.value))))
              }
              className="border border-gray-300 p-2 rounded-lg w-full focus:outline-none focus:ring-2 focus:ring-blue-300 transition"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Domínio do email institucional
            </label>
            <input
              type="text"
              value={emailDomain}
              onChange={(e) => setEmailDomain(e.target.value)}
              placeholder="@al.insper.edu.br"
              className="border border-gray-300 p-2 rounded-lg w-full focus:outline-none focus:ring-2 focus:ring-blue-300 transition font-mono text-sm"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Arquivo Excel (.xlsx)
            </label>
            <input
              type="file"
              accept=".xlsx"
              onChange={(e) => setFile(e.target.files?.[0] ?? null)}
              className="border border-gray-300 p-2 rounded-lg w-full bg-gray-50"
            />
          </div>
        </div>

        {error && (
          <div className="w-full bg-red-50 border border-red-200 rounded-lg p-3 text-sm text-red-700">
            {error}
          </div>
        )}

        <button
          onClick={handleUpload}
          disabled={loading}
          className={`w-full py-3 rounded-xl font-semibold text-white shadow-lg transition-all ${
            loading
              ? "bg-blue-300 cursor-not-allowed"
              : "bg-blue-600 hover:bg-blue-700"
          }`}
        >
          {loading ? "Processando..." : "Iniciar →"}
        </button>

        <a
          href="/api/exemplo"
          download="base_exemplo.xlsx"
          className="w-full py-2 rounded-xl font-medium text-blue-600 border border-blue-300 bg-blue-50 hover:bg-blue-100 transition-all text-center text-sm"
        >
          Baixar planilha de exemplo
        </a>
      </div>
    </div>
  );
}

export default Home;
