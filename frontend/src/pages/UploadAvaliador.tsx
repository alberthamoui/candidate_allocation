import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { motion } from "framer-motion";
import {
  UserGroupIcon,
  ExclamationTriangleIcon,
  ArrowLeftIcon,
} from "@heroicons/react/24/outline";
import { suggestMappingAvaliador } from "../api";

interface UploadAvaliadorProps {
  setMappingAvaliador: (data: any) => void;
}

export default function UploadAvaliador({
  setMappingAvaliador,
}: UploadAvaliadorProps) {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleProcess() {
    setLoading(true);
    setError(null);
    try {
      const result = await suggestMappingAvaliador();
      setMappingAvaliador(result);
      navigate("/mapping-avaliador");
    } catch (err: any) {
      setError(
        "Erro ao processar avaliadores. Certifique-se de que o arquivo possui uma aba de avaliadores. Detalhe: " +
          err.message
      );
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-green-100 flex flex-col items-center justify-center py-8">
      <div className="bg-white shadow-xl rounded-2xl p-10 w-full max-w-md flex flex-col items-center space-y-8">
        {/* Step indicator */}
        <div className="flex items-center space-x-2 text-sm text-gray-500">
          <span className="bg-green-100 text-green-700 font-semibold px-3 py-1 rounded-full">
            Passo 1 ✓
          </span>
          <span className="text-gray-300">→</span>
          <span className="bg-blue-600 text-white font-semibold px-3 py-1 rounded-full">
            Passo 2
          </span>
          <span className="text-gray-300">→</span>
          <span className="bg-gray-100 text-gray-400 font-semibold px-3 py-1 rounded-full">
            Passo 3
          </span>
        </div>

        <UserGroupIcon className="w-20 h-20 text-blue-500" />

        <div className="text-center space-y-2">
          <h1 className="text-3xl font-bold text-blue-700">Avaliadores</h1>
          <p className="text-gray-500 text-sm">
            Os avaliadores serão lidos da aba{" "}
            <span className="font-semibold text-gray-700">Avaliadores</span>{" "}
            do mesmo arquivo já carregado.
          </p>
        </div>

        {error && (
          <div className="flex items-start space-x-2 bg-red-50 border border-red-200 rounded-lg p-3 w-full">
            <ExclamationTriangleIcon className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" />
            <p className="text-sm text-red-700">{error}</p>
          </div>
        )}

        <motion.button
          onClick={handleProcess}
          disabled={loading}
          whileHover={{ scale: loading ? 1 : 1.03 }}
          whileTap={{ scale: loading ? 1 : 0.97 }}
          className={`w-full py-3 rounded-xl font-semibold text-white shadow-lg transition-all ${
            loading
              ? "bg-blue-300 cursor-not-allowed"
              : "bg-blue-600 hover:bg-blue-700"
          }`}
        >
          {loading ? "Processando..." : "Processar Avaliadores"}
        </motion.button>

        <button
          onClick={() => navigate(-1)}
          className="flex items-center justify-center space-x-2 w-full py-2.5 rounded-xl border border-gray-300 text-gray-600 hover:bg-gray-50 font-medium transition-all text-sm"
        >
          <ArrowLeftIcon className="w-4 h-4" />
          <span>Voltar</span>
        </button>
      </div>
    </div>
  );
}
