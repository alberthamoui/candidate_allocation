import { ReactNode, useRef, useState } from "react";
import { AvisoGerado, ConfigAvisos, previaAvisos, testarAvisos } from "../api";

const classe =
  "mt-1 w-full border border-gray-300 rounded-lg p-2 text-sm bg-white";
const variaveis = [
  "nome",
  "data",
  "horario",
  "mesa",
  "email",
  "curso",
  "numero",
];
export default function AvisosForm({
  cfg,
  onChange,
  mostrarAtivo = true,
}: {
  cfg: ConfigAvisos;
  onChange: (c: ConfigAvisos) => void;
  mostrarAtivo?: boolean;
}) {
  const mensagem = useRef<HTMLTextAreaElement>(null);
  const cursorDefinido = useRef(false);
  const [status, setStatus] = useState<Record<string, string>>({});
  const [previa, setPrevia] = useState("");
  const [destinosPrevia, setDestinosPrevia] = useState<AvisoGerado[]>([]);
  const [ocupado, setOcupado] = useState(false);
  const email = (val: Partial<ConfigAvisos["email"]>) =>
    onChange({ ...cfg, email: { ...cfg.email, ...val } });
  const whats = (val: Partial<ConfigAvisos["whatsapp"]>) =>
    onChange({ ...cfg, whatsapp: { ...cfg.whatsapp, ...val } });
  async function testar() {
    setOcupado(true);
    try {
      setStatus(await testarAvisos(cfg));
    } catch (e: any) {
      setStatus({ erro: e.message });
    } finally {
      setOcupado(false);
    }
  }
  async function verPrevia() {
    setOcupado(true);
    try {
      const p = await previaAvisos(cfg);
      setPrevia(p.mensagem);
      setDestinosPrevia(p.destinatarios);
    } catch (e: any) {
      setStatus({ erro: e.message });
    } finally {
      setOcupado(false);
    }
  }
  function inserir(v: string) {
    const el = mensagem.current;
    const ini = cursorDefinido.current
      ? (el?.selectionStart ?? cfg.mensagem.length)
      : cfg.mensagem.length;
    const fim = cursorDefinido.current ? (el?.selectionEnd ?? ini) : ini;
    const texto = `{{${v}}}`;
    onChange({
      ...cfg,
      mensagem: cfg.mensagem.slice(0, ini) + texto + cfg.mensagem.slice(fim),
    });
    requestAnimationFrame(() => {
      el?.focus();
      el?.setSelectionRange(ini + texto.length, ini + texto.length);
    });
  }
  return (
    <div className="space-y-4">
      {mostrarAtivo && (
        <label className="flex gap-2 items-center">
          <input
            type="checkbox"
            checked={cfg.ativo}
            onChange={(e) => onChange({ ...cfg, ativo: e.target.checked })}
          />
          Avisos ligados
        </label>
      )}
      <Campo titulo="Mensagem">
        <textarea
          ref={mensagem}
          onSelect={() => {
            cursorDefinido.current = true;
          }}
          className={classe}
          value={cfg.mensagem}
          rows={3}
          onChange={(e) => onChange({ ...cfg, mensagem: e.target.value })}
        />
      </Campo>
      <div className="flex flex-wrap gap-1">
        {variaveis.map((v) => (
          <button
            type="button"
            key={v}
            className="text-xs rounded-full px-2 py-1 border border-blue-200 text-blue-700"
            onClick={() => inserir(v)}
          >{`{{${v}}}`}</button>
        ))}
      </div>
      <label className="flex gap-2 items-center text-sm">
        <input
          type="checkbox"
          checked={cfg.enviar_ao_alocar ?? false}
          onChange={(e) =>
            onChange({ ...cfg, enviar_ao_alocar: e.target.checked })
          }
        />
        Autorizar envio automático ao concluir a alocação
      </label>
      <p className="text-xs text-gray-500">
        Realocações só enviam avisos quando você marcar a opção e confirmar. Os
        avaliadores afetados também recebem e-mail.
      </p>
      <label className="flex gap-2 items-center text-sm font-medium">
        <input
          type="checkbox"
          checked={cfg.email.ativo}
          onChange={(e) => email({ ativo: e.target.checked })}
        />
        Enviar por e-mail (SMTP)
      </label>
      {cfg.email.ativo && (
        <div className="border rounded-xl p-3 space-y-3 bg-white">
          <Campo titulo="Serviço de e-mail">
            <select
              className={classe}
              value={cfg.email.provedor ?? "smtp"}
              onChange={(e) =>
                email(
                  e.target.value === "gmail"
                    ? {
                        provedor: "gmail",
                        host: "smtp.gmail.com",
                        porta: 587,
                        usuario: cfg.email.remetente,
                      }
                    : { provedor: "smtp" },
                )
              }
            >
              <option value="gmail">Gmail / Google Workspace</option>
              <option value="smtp">Outro serviço SMTP</option>
            </select>
          </Campo>
          <Campo titulo="E-mail remetente">
            <input
              className={classe}
              type="email"
              placeholder="remetente"
              value={cfg.email.remetente}
              onChange={(e) =>
                email({
                  remetente: e.target.value,
                  ...(cfg.email.provedor === "gmail"
                    ? { usuario: e.target.value }
                    : {}),
                })
              }
            />
          </Campo>
          <Campo
            titulo={
              cfg.email.provedor === "gmail"
                ? "Senha de aplicativo do Google"
                : "Senha SMTP"
            }
          >
            <input
              className={classe}
              type="password"
              autoComplete="new-password"
              placeholder={
                cfg.email.senha_definida
                  ? "Senha salva — deixe vazio para manter"
                  : "senha"
              }
              value={cfg.email.senha}
              onChange={(e) => email({ senha: e.target.value })}
            />
          </Campo>
          {cfg.email.provedor === "gmail" && (
            <p className="text-xs text-gray-600">
              Ative a verificação em duas etapas e{" "}
              <a
                className="text-blue-700 underline"
                href="https://myaccount.google.com/apppasswords"
                target="_blank"
                rel="noreferrer"
              >
                gere uma senha de aplicativo no Google
              </a>
              . Cole essa senha aqui. A conta precisa permitir senhas de
              aplicativo.
            </p>
          )}
          <details open={cfg.email.provedor !== "gmail"}>
            <summary className="text-sm cursor-pointer">
              Configuração SMTP avançada
            </summary>
            <div className="grid sm:grid-cols-2 gap-2 mt-2">
              <Campo titulo="Servidor SMTP">
                <input
                  className={classe}
                  placeholder="servidor SMTP"
                  value={cfg.email.host}
                  onChange={(e) => email({ host: e.target.value })}
                />
              </Campo>
              <Campo titulo="Porta SMTP">
                <input
                  className={classe}
                  type="number"
                  min={1}
                  max={65535}
                  placeholder="porta"
                  value={cfg.email.porta}
                  onChange={(e) => email({ porta: Number(e.target.value) })}
                />
              </Campo>
              <Campo titulo="Usuário SMTP">
                <input
                  className={classe}
                  placeholder="usuário"
                  value={cfg.email.usuario}
                  onChange={(e) => email({ usuario: e.target.value })}
                />
              </Campo>
            </div>
          </details>
        </div>
      )}
      <label className="flex gap-2 items-center text-sm font-medium">
        <input
          type="checkbox"
          checked={cfg.whatsapp.ativo}
          onChange={(e) => whats({ ativo: e.target.checked })}
        />
        Enviar por WhatsApp
      </label>
      {cfg.whatsapp.ativo && (
        <div className="border rounded-xl p-3 space-y-3 bg-white">
          <Campo titulo="Provedor WhatsApp">
            <select
              className={classe}
              value={cfg.whatsapp.provedor ?? "meta"}
              onChange={(e) => whats({ provedor: e.target.value })}
            >
              <option value="meta">WhatsApp Business (Meta)</option>
              <option value="webhook">
                Webhook de outro provedor (avançado)
              </option>
            </select>
          </Campo>
          {(cfg.whatsapp.provedor ?? "meta") === "meta" ? (
            <>
              <ol className="list-decimal pl-5 text-xs text-gray-600 space-y-1">
                <li>
                  Abra sua{" "}
                  <a
                    href="https://developers.facebook.com/apps/"
                    target="_blank"
                    rel="noreferrer"
                    className="underline text-blue-700"
                  >
                    conta de WhatsApp Business na Meta
                  </a>
                  .
                </li>
                <li>
                  Em WhatsApp → Configuração da API, copie o ID do número e o
                  token de acesso.
                </li>
                <li>
                  Escolha um modelo de mensagem aprovado para enviar lembretes e
                  mudanças de entrevista.
                </li>
              </ol>
              <Campo titulo="ID do número na Meta">
                <input
                  className={classe}
                  value={cfg.whatsapp.phone_id ?? ""}
                  onChange={(e) => whats({ phone_id: e.target.value })}
                />
              </Campo>
              <Campo titulo="Modelo aprovado na Meta">
                <input
                  className={classe}
                  placeholder="aviso_entrevista"
                  value={cfg.whatsapp.template ?? ""}
                  onChange={(e) => whats({ template: e.target.value })}
                />
              </Campo>
              <Campo titulo="Idioma do modelo">
                <input
                  className={classe}
                  value={cfg.whatsapp.idioma ?? "pt_BR"}
                  onChange={(e) => whats({ idioma: e.target.value })}
                />
              </Campo>
              <Campo titulo="Variáveis do modelo, na ordem">
                <input
                  className={classe}
                  placeholder="nome, data, horario, mesa"
                  value={(cfg.whatsapp.variaveis_template ?? []).join(", ")}
                  onChange={(e) =>
                    whats({
                      variaveis_template: e.target.value
                        .split(",")
                        .map((v) => v.trim())
                        .filter(Boolean),
                    })
                  }
                />
              </Campo>
              <p className="text-xs text-gray-500">
                O WhatsApp usa o texto do modelo aprovado pela Meta com estas
                variáveis. A mensagem livre acima vale para e-mail e webhook.
                Use um token permanente na conta de produção.
              </p>
            </>
          ) : (
            <Campo titulo="URL do webhook">
              <input
                className={classe}
                placeholder="URL do webhook (POST JSON com to e message)"
                value={cfg.whatsapp.url}
                onChange={(e) => whats({ url: e.target.value })}
              />
            </Campo>
          )}
          <Campo titulo="Token de acesso">
            <input
              className={classe}
              type="password"
              autoComplete="new-password"
              placeholder={
                cfg.whatsapp.token_definido
                  ? "Token salvo — deixe vazio para manter"
                  : "token"
              }
              value={cfg.whatsapp.token}
              onChange={(e) => whats({ token: e.target.value })}
            />
          </Campo>
          <p className="text-xs text-gray-500">
            Use a coluna Telefone/WhatsApp da planilha. Matrícula de 9 dígitos
            não é usada como telefone. Telefones brasileiros com DDD recebem o
            código 55.
          </p>
        </div>
      )}
      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          disabled={ocupado}
          className="border rounded-lg px-3 py-2 text-sm disabled:opacity-50"
          onClick={testar}
        >
          Testar conexão
        </button>
        <button
          type="button"
          disabled={ocupado}
          className="border rounded-lg px-3 py-2 text-sm disabled:opacity-50"
          onClick={verPrevia}
        >
          Ver prévia da mensagem
        </button>
      </div>
      {Object.entries(status).map(([k, v]) => (
        <p key={k} role="status" className="text-sm bg-blue-50 p-3 rounded-lg">
          {k}: {v}
        </p>
      ))}
      {previa && (
        <div className="bg-blue-50 rounded-lg p-3 text-sm whitespace-pre-wrap">
          <p className="font-semibold mb-1">
            Prévia — nenhuma mensagem enviada
          </p>
          {previa}
          {destinosPrevia.map((a, i) => (
            <p key={i} className="mt-2 text-xs">
              {a.canal}: {a.erro || a.destino}
            </p>
          ))}
        </div>
      )}
    </div>
  );
}
function Campo({ titulo, children }: { titulo: string; children: ReactNode }) {
  return (
    <label className="block text-sm text-gray-700">
      {titulo}
      {children}
    </label>
  );
}
