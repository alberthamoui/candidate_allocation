import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { ConfigAvisos, configAvisosVazia, upload } from "../api";

interface AppProps {
  setMapping: (data: any) => void;
}

const VARIAVEIS = ["nome", "data", "horario", "mesa", "email", "curso"];

function Home({ setMapping }: AppProps) {
  const [file, setFile] = useState<File | null>(null);
  const [nOpcoes, setNOpcoes] = useState(5);
  const [emailDomain, setEmailDomain] = useState("@al.insper.edu.br");
  const [usarPreferencia, setUsarPreferencia] = useState(true);
  const [querAvisos, setQuerAvisos] = useState(false);
  const [avisos, setAvisos] = useState<ConfigAvisos>(configAvisosVazia());
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const navigate = useNavigate();

  function inserirVariavel(nome: string) {
    setAvisos((atual) => ({ ...atual, mensagem: `${atual.mensagem}{{${nome}}}` }));
  }

  async function handleUpload() {
    if (!file) {
      setError("Por favor, selecione um arquivo .xlsx.");
      return;
    }
    setLoading(true);
    setError("");
    try {
      const cfg: ConfigAvisos = { ...avisos, ativo: querAvisos };
      const mapping = await upload(file, nOpcoes, emailDomain, {
        usarPreferencia,
        avisos: cfg,
      });
      setMapping(mapping);
      navigate("/mapping");
    } catch (err: any) {
      setError("Erro ao processar o arquivo: " + err.message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-green-100 flex flex-col items-center justify-center py-8 px-4">
      <div className="bg-white shadow-xl rounded-2xl p-10 w-full max-w-xl flex flex-col items-center space-y-8">
        <h1 className="text-3xl font-bold text-blue-700 text-center">
          Candidate Allocator
        </h1>
        <p className="text-gray-500 text-sm text-center">
          Faça upload do arquivo Excel com os dados de candidatos, avaliadores e restrições para iniciar a alocação.
        </p>

        <div className="flex flex-col space-y-4 w-full">
          <label className="flex items-start gap-3 text-sm text-gray-700">
            <input
              type="checkbox"
              checked={usarPreferencia}
              onChange={(e) => setUsarPreferencia(e.target.checked)}
              className="mt-1"
            />
            <span>
              <span className="font-medium">Candidatos informam preferência de horário</span>
              <span className="block text-gray-500">
                Desligue se a planilha não tiver colunas de opção. Você define os horários na etapa de parâmetros e todo mundo pode ir em qualquer um.
              </span>
            </span>
          </label>

          {usarPreferencia && (
            <label className="block text-sm font-medium text-gray-700">
              Número de opções de horário por candidato
              <input
                type="number"
                min={1}
                max={10}
                value={nOpcoes}
                onChange={(e) =>
                  setNOpcoes(Math.max(1, Math.min(10, Number(e.target.value))))
                }
                className="mt-1 border border-gray-300 p-2 rounded-lg w-full focus:outline-none focus:ring-2 focus:ring-blue-300 transition"
              />
            </label>
          )}

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
            <p className="text-xs text-gray-500 mt-1">
              CPF é opcional. Se vier preenchido, precisa ter 11 dígitos. A coluna Número aceita o RA (9 dígitos) ou um telefone com DDD.
            </p>
          </div>

          <div className="border border-gray-200 rounded-xl p-4 space-y-3 bg-gray-50">
            <label className="flex items-start gap-3 text-sm text-gray-700">
              <input
                type="checkbox"
                checked={querAvisos}
                onChange={(e) => setQuerAvisos(e.target.checked)}
                className="mt-1"
              />
              <span>
                <span className="font-medium">Configurar avisos por e-mail e WhatsApp</span>
                <span className="block text-gray-500">
                  Opcional. Escreva a mensagem uma vez; o sistema troca as variáveis por cada pessoa. O envio só acontece quando você aceitar, depois da alocação ou de uma mudança de horário.
                </span>
              </span>
            </label>

            {querAvisos && (
              <div className="space-y-3">
                <label className="block text-sm font-medium text-gray-700">
                  Mensagem
                  <textarea
                    value={avisos.mensagem}
                    onChange={(e) => setAvisos({ ...avisos, mensagem: e.target.value })}
                    rows={3}
                    className="mt-1 border border-gray-300 p-2 rounded-lg w-full text-sm font-normal focus:outline-none focus:ring-2 focus:ring-blue-300"
                  />
                </label>
                <div className="flex flex-wrap gap-1">
                  {VARIAVEIS.map((v) => (
                    <button
                      key={v}
                      type="button"
                      onClick={() => inserirVariavel(v)}
                      className="text-xs font-mono px-2 py-1 rounded-full bg-white border border-blue-200 text-blue-700 hover:bg-blue-50"
                    >
                      {`{{${v}}}`}
                    </button>
                  ))}
                </div>

                <label className="flex items-center gap-2 text-sm text-gray-700">
                  <input
                    type="checkbox"
                    checked={avisos.email.ativo}
                    onChange={(e) => setAvisos({ ...avisos, email: { ...avisos.email, ativo: e.target.checked } })}
                  />
                  Enviar por e-mail (SMTP)
                </label>
                {avisos.email.ativo && (
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                    <input className="border rounded-lg p-2 text-sm" placeholder="servidor SMTP" value={avisos.email.host} onChange={(e) => setAvisos({ ...avisos, email: { ...avisos.email, host: e.target.value } })} />
                    <input className="border rounded-lg p-2 text-sm" type="number" placeholder="porta" value={avisos.email.porta} onChange={(e) => setAvisos({ ...avisos, email: { ...avisos.email, porta: Number(e.target.value) } })} />
                    <input className="border rounded-lg p-2 text-sm" placeholder="usuário" value={avisos.email.usuario} onChange={(e) => setAvisos({ ...avisos, email: { ...avisos.email, usuario: e.target.value } })} />
                    <input className="border rounded-lg p-2 text-sm" type="password" placeholder="senha" value={avisos.email.senha} onChange={(e) => setAvisos({ ...avisos, email: { ...avisos.email, senha: e.target.value } })} />
                    <input className="border rounded-lg p-2 text-sm sm:col-span-2" placeholder="remetente" value={avisos.email.remetente} onChange={(e) => setAvisos({ ...avisos, email: { ...avisos.email, remetente: e.target.value } })} />
                  </div>
                )}

                <label className="flex items-center gap-2 text-sm text-gray-700">
                  <input
                    type="checkbox"
                    checked={avisos.whatsapp.ativo}
                    onChange={(e) => setAvisos({ ...avisos, whatsapp: { ...avisos.whatsapp, ativo: e.target.checked } })}
                  />
                  Enviar por WhatsApp
                </label>
                {avisos.whatsapp.ativo && (
                  <div className="space-y-2">
                    <input className="border rounded-lg p-2 text-sm w-full" placeholder="URL do webhook (POST JSON com to e message)" value={avisos.whatsapp.url} onChange={(e) => setAvisos({ ...avisos, whatsapp: { ...avisos.whatsapp, url: e.target.value } })} />
                    <input className="border rounded-lg p-2 text-sm w-full" placeholder="token (opcional)" value={avisos.whatsapp.token} onChange={(e) => setAvisos({ ...avisos, whatsapp: { ...avisos.whatsapp, token: e.target.value } })} />
                    <p className="text-xs text-gray-500">O WhatsApp usa a coluna Número. Um telefone com DDD sai direto; um RA de 9 dígitos é enviado assim para o webhook. Sem URL, a mensagem é montada e mostrada, mas não sai.</p>
                  </div>
                )}
              </div>
            )}
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
