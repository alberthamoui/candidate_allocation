import {
  createContext,
  ReactNode,
  useContext,
  useEffect,
  useState,
} from "react";
import {
  AlocacaoResponse,
  CandidatoResultado,
  ConfigAvisos,
  LoteAvisos,
  MesaResult,
  ResultadoOperacao,
  SugestaoRealocacao,
  avaliar,
  enviarAvisos,
  getAvisos,
  getHistoricoAvisos,
  getLote,
  getSessionId,
  getSugestoes,
  realocar,
  repetirFalhas,
  salvarAvisos,
} from "../api";
import AvisosForm from "./AvisosForm";
import { Aviso, BotaoPrimario, BotaoSecundario, Modal, Secao } from "./ui";

type Alvo =
  | { tipo: "pessoa"; pessoa: CandidatoResultado }
  | { tipo: "mesa"; mesa: MesaResult }
  | { tipo: "dia"; de: string };
type Destinatarios = { pessoa?: string; mesa_id?: number; nome: string };
type Acoes = {
  mover: (a: Alvo) => void;
  avisar: (a: Destinatarios) => void;
  avaliar: (id: number, r: string) => void;
  ocupado: boolean;
};
const Contexto = createContext<Acoes | null>(null);
export function AcoesPessoa({
  pessoa,
  alocado = true,
}: {
  pessoa: CandidatoResultado;
  alocado?: boolean;
}) {
  const a = useContext(Contexto);
  if (!a) return null;
  return (
    <div className="flex flex-wrap items-center gap-2 px-1.5 pb-2 text-xs">
      <button
        className="text-[var(--accent)] underline"
        disabled={a.ocupado}
        onClick={() => a.mover({ tipo: "pessoa", pessoa })}
        aria-label={`Realocar ${pessoa.nome}`}
      >
        Realocar
      </button>
      <select
        className="rounded border border-[var(--line)] bg-white py-1"
        aria-label={`Avaliação de ${pessoa.nome}`}
        disabled={a.ocupado}
        value={pessoa.avaliacao ?? ""}
        onChange={(e) => a.avaliar(pessoa.id, e.target.value)}
      >
        <option value="">Avaliação pendente</option>
        <option value="aprovado">Passou</option>
        <option value="reprovado">Não passou</option>
      </select>
      {alocado && (
        <button
          className="text-[var(--accent)] underline"
          disabled={a.ocupado}
          onClick={() =>
            a.avisar({ pessoa: String(pessoa.id), nome: pessoa.nome })
          }
        >
          Avisar
        </button>
      )}
    </div>
  );
}
export function AcoesMesa({ mesa }: { mesa: MesaResult }) {
  const a = useContext(Contexto);
  if (!a) return null;
  return (
    <div className="flex flex-wrap gap-3 text-xs">
      <button
        disabled={a.ocupado}
        className="text-[var(--accent)] underline"
        onClick={() => a.mover({ tipo: "mesa", mesa })}
      >
        Realocar mesa
      </button>
      <button
        disabled={a.ocupado}
        className="text-[var(--accent)] underline"
        onClick={() =>
          a.avisar({
            mesa_id: mesa.id,
            nome: `participantes da mesa ${mesa.id}`,
          })
        }
      >
        Avisar mesa
      </button>
    </div>
  );
}
export function AcoesHorario({ horario }: { horario: string }) {
  const a = useContext(Contexto);
  return a ? (
    <button
      disabled={a.ocupado}
      className="text-xs text-[var(--accent)] underline"
      onClick={() => a.mover({ tipo: "dia", de: horario })}
    >
      Realocar horário ou dia
    </button>
  ) : null;
}

export default function OperacoesAgenda({
  result,
  onResult,
  children,
}: {
  result: AlocacaoResponse;
  onResult: (r: AlocacaoResponse) => void;
  children: ReactNode;
}) {
  const [alvo, setAlvo] = useState<Alvo | null>(null);
  const [destino, setDestino] = useState("");
  const [mesaID, setMesaID] = useState(0);
  const [origem, setOrigem] = useState("");
  const [avisarAoMover, setAvisarAoMover] = useState(false);
  const [sugestoes, setSugestoes] = useState<SugestaoRealocacao[]>([]);
  const [destinatarios, setDestinatarios] = useState<Destinatarios | null>(
    null,
  );
  const [cfg, setCfg] = useState<ConfigAvisos | null>(null);
  const [canais, setCanais] = useState(false);
  const [erro, setErro] = useState("");
  const [mensagem, setMensagem] = useState("");
  const [ocupado, setOcupado] = useState(false);
  const [historico, setHistorico] = useState(false);
  const [lotes, setLotes] = useState<LoteAvisos[]>([]);
  const [proximo, setProximo] = useState<number>();
  const [detalhe, setDetalhe] = useState<LoteAvisos | null>(null);
  const [loteAtivo, setLoteAtivo] = useState<LoteAvisos | null>(null);
  useEffect(() => {
    getAvisos()
      .then(setCfg)
      .catch((e) => setErro(e.message));
  }, []);
  // Consulta somente o resumo do envio em andamento, nunca o histórico completo.
  useEffect(() => {
    if (!loteAtivo || loteAtivo.pendentes === 0) return;
    let cancelado = false;
    let timer: ReturnType<typeof setTimeout>;
    async function atualizar() {
      try {
        const l = await getLote(loteAtivo!.id, true);
        if (cancelado) return;
        setLoteAtivo(l);
        if (l.pendentes > 0) timer = setTimeout(atualizar, 1500);
      } catch (e: any) {
        if (!cancelado) setErro(e.message);
      }
    }
    timer = setTimeout(atualizar, 500);
    return () => {
      cancelado = true;
      clearTimeout(timer);
    };
  }, [loteAtivo?.id]);
  // Um lote criado no servidor pode ter acabado antes da primeira consulta.
  async function receber(op: ResultadoOperacao) {
    if (op.resultado) onResult(op.resultado);
    setMensagem(op.resumo);
    if (op.lote_id) setLoteAtivo(await getLote(op.lote_id, true));
  }
  async function executar(fn: () => Promise<ResultadoOperacao>) {
    setOcupado(true);
    setErro("");
    try {
      await receber(await fn());
      return true;
    } catch (e: any) {
      setErro(e.message);
      return false;
    } finally {
      setOcupado(false);
    }
  }
  async function mover(a: Alvo) {
    setAlvo(a);
    setDestino("");
    setMesaID(0);
    setOrigem(a.tipo === "dia" ? a.de : "");
    setAvisarAoMover(false);
    setSugestoes([]);
    setErro("");
    if (a.tipo === "pessoa") {
      try {
        setSugestoes(await getSugestoes(a.pessoa.id));
      } catch (e: any) {
        setErro(e.message);
      }
    }
  }
  async function confirmarMovimento() {
    if (!alvo) return;
    const pedido =
      alvo.tipo === "pessoa"
        ? {
            tipo: "pessoa" as const,
            pessoa: String(alvo.pessoa.id),
            horario: destino,
            mesa_id: mesaID,
          }
        : alvo.tipo === "mesa"
          ? { tipo: "mesa" as const, mesa_id: alvo.mesa.id, para: destino }
          : { tipo: "dia" as const, de: origem, para: destino };
    if (await executar(() => realocar({ ...pedido, avisar: avisarAoMover })))
      setAlvo(null);
  }
  async function carregarHistorico(antes = 0) {
    setErro("");
    try {
      const p = await getHistoricoAvisos(antes);
      setLotes((v) => (antes ? [...v, ...p.lotes] : p.lotes));
      setProximo(p.proximo);
    } catch (e: any) {
      setErro(e.message);
    }
  }
  async function fecharCanais() {
    if (ocupado) return;
    setCanais(false);
    try {
      setCfg(await getAvisos());
    } catch (e: any) {
      setErro(e.message);
    }
  }
  async function gravarCanais() {
    if (!cfg) return;
    setOcupado(true);
    setErro("");
    try {
      await salvarAvisos(cfg);
      setCfg(await getAvisos());
      setMensagem("Canais e mensagem salvos.");
      setCanais(false);
    } catch (e: any) {
      setErro(e.message);
    } finally {
      setOcupado(false);
    }
  }
  const ligados = !!cfg?.ativo && (cfg.email.ativo || cfg.whatsapp.ativo);
  return (
    <Contexto.Provider
      value={{
        mover,
        avisar: setDestinatarios,
        avaliar: (id, r) => {
          void executar(() => avaliar([{ id, resultado: r }]));
        },
        ocupado,
      }}
    >
      <Secao
        titulo="Operar a agenda"
        subtitulo="Use as ações de cada pessoa, mesa ou horário. As mudanças aparecem também na CLI e no Excel."
      >
        <div className="flex flex-wrap gap-3">
          <BotaoSecundario onClick={() => setCanais(true)}>
            Configurar canais
          </BotaoSecundario>
          <BotaoSecundario
            disabled={ocupado}
            onClick={() =>
              setDestinatarios({ nome: "todos os participantes alocados" })
            }
          >
            Avisar todos
          </BotaoSecundario>
          <BotaoSecundario
            onClick={() => {
              setHistorico((v) => !v);
              if (!historico) void carregarHistorico();
            }}
          >
            Histórico de envios
          </BotaoSecundario>
        </div>
        <p className="mt-3 text-xs text-[var(--text-muted)]">
          {ligados
            ? "Canais habilitados: " +
              [
                cfg?.email.ativo ? "e-mail" : "",
                cfg?.whatsapp.ativo ? "WhatsApp" : "",
              ]
                .filter(Boolean)
                .join(" e ")
            : "Avisos desligados. Configure um canal para enviar."}
        </p>
        <details className="mt-3 text-xs">
          <summary className="cursor-pointer">
            Usar esta agenda pela CLI
          </summary>
          <pre className="mt-2 overflow-auto rounded bg-[var(--bg-subtle)] p-3">{`alocador resultado -servidor ${location.origin} -sessao ${getSessionId()}`}</pre>
          <p>
            O ID dá acesso à agenda. Compartilhe somente com administradores
            autorizados.
          </p>
        </details>
      </Secao>
      {erro && (
        <div role="alert">
          <Aviso tom="perigo">{erro}</Aviso>
        </div>
      )}
      {mensagem && (
        <div role="status">
          <Aviso tom="sucesso">{mensagem}</Aviso>
        </div>
      )}
      {loteAtivo && (
        <Aviso
          tom={loteAtivo.falhas || loteAtivo.incertos ? "atencao" : "info"}
        >
          <p role="status">
            Envio: {loteAtivo.enviados} aceitos pelo provedor ·{" "}
            {loteAtivo.falhas} falhas · {loteAtivo.incertos} incertos ·{" "}
            {loteAtivo.pendentes} pendentes.
          </p>
          <button
            className="underline text-sm"
            onClick={() => {
              setHistorico(true);
              void carregarHistorico();
            }}
          >
            Ver detalhes e resolver falhas
          </button>
        </Aviso>
      )}
      {(result.alertas ?? []).length > 0 && (
        <Aviso tom="atencao">
          <details>
            <summary className="cursor-pointer">
              {result.alertas!.length} alertas na agenda — revisar
            </summary>
            <ul className="mt-2 list-disc pl-5">
              {result.alertas!.map((s, i) => (
                <li key={i}>{s}</li>
              ))}
            </ul>
          </details>
        </Aviso>
      )}
      {historico && (
        <Secao
          titulo="Histórico de envios"
          subtitulo="Aceito significa recebido pelo provedor. Envios incertos precisam de conferência antes de um novo envio."
        >
          <BotaoSecundario onClick={() => void carregarHistorico()}>
            Atualizar histórico
          </BotaoSecundario>
          {lotes.length === 0 && (
            <p className="mt-3 text-sm">Nenhum envio registrado.</p>
          )}
          <ul className="mt-3 divide-y divide-[var(--line)]">
            {lotes.map((l) => (
              <li key={l.id} className="py-3 text-sm">
                <div>
                  {new Date(l.criado).toLocaleString("pt-BR")} · {l.status}
                </div>
                <div>
                  {l.enviados} aceitos · {l.falhas} falhas · {l.incertos}{" "}
                  incertos · {l.pendentes} pendentes
                </div>
                <div className="flex gap-3 mt-2">
                  <button
                    className="underline"
                    onClick={() =>
                      getLote(l.id)
                        .then(setDetalhe)
                        .catch((e) => setErro(e.message))
                    }
                  >
                    Ver mensagens
                  </button>
                  {l.falhas > 0 && !l.repetido_por && l.pendentes === 0 && (
                    <button
                      className="underline"
                      disabled={ocupado}
                      onClick={async () => {
                        if (await executar(() => repetirFalhas(l.id)))
                          void carregarHistorico();
                      }}
                    >
                      Repetir falhas confirmadas
                    </button>
                  )}
                </div>
              </li>
            ))}
          </ul>
          {proximo && (
            <BotaoSecundario onClick={() => void carregarHistorico(proximo)}>
              Carregar mais envios
            </BotaoSecundario>
          )}
        </Secao>
      )}
      {children}
      {alvo && (
        <Modal
          titulo={
            alvo.tipo === "pessoa"
              ? `Realocar ${alvo.pessoa.nome}`
              : alvo.tipo === "mesa"
                ? `Realocar mesa ${alvo.mesa.id}`
                : "Realocar horário ou dia inteiro"
          }
          onFechar={() => !ocupado && setAlvo(null)}
          acoes={
            <>
              <BotaoSecundario disabled={ocupado} onClick={() => setAlvo(null)}>
                Cancelar
              </BotaoSecundario>
              <BotaoPrimario
                disabled={ocupado || (!destino.trim() && !mesaID)}
                onClick={confirmarMovimento}
              >
                {ocupado ? "Salvando..." : "Confirmar realocação"}
              </BotaoPrimario>
            </>
          }
        >
          <div className="space-y-4">
            {alvo.tipo === "pessoa" && (
              <>
                <p className="text-sm">
                  Sugestões ordenadas por vagas, preferências, equipe e
                  conflitos. Você pode escolher qualquer mesa ou informar um
                  novo horário.
                </p>
                <div className="space-y-2">
                  {sugestoes.slice(0, 3).map((s, i) => (
                    <button
                      key={s.mesa_id}
                      className="block w-full rounded border border-[var(--line)] p-3 text-left text-sm"
                      onClick={() => {
                        setMesaID(s.mesa_id);
                        setDestino("");
                      }}
                      aria-pressed={mesaID === s.mesa_id}
                    >
                      <b>
                        {i === 0 ? "Melhor opção: " : ""}
                        {s.descricao}
                      </b>
                      <span className="block">{s.motivos.join(" · ")}</span>
                      {s.alertas.length > 0 && (
                        <span className="block text-[var(--warning)]">
                          {s.alertas.join(" · ")}
                        </span>
                      )}
                    </button>
                  ))}
                </div>
                <label className="block">
                  <span className="rotulo">Mesa de destino</span>
                  <select
                    className="campo"
                    value={mesaID}
                    onChange={(e) => {
                      setMesaID(Number(e.target.value));
                      setDestino("");
                    }}
                  >
                    <option value={0}>Definir outro horário</option>
                    {result.mesas.map((m) => (
                      <option key={m.id} value={m.id}>
                        {m.descricao} ({m.candidatos.length} pessoas)
                      </option>
                    ))}
                  </select>
                </label>
              </>
            )}
            {alvo.tipo === "dia" && (
              <>
                <label className="block">
                  <span className="rotulo">
                    Origem: horário completo ou somente o dia
                  </span>
                  <input
                    className="campo"
                    value={origem}
                    onChange={(e) => setOrigem(e.target.value)}
                  />
                </label>
                <p className="text-sm">
                  Para mover o dia inteiro, deixe somente a data na origem e no
                  destino. Os intervalos de horário serão preservados.
                </p>
              </>
            )}
            {!mesaID && (
              <label className="block">
                <span className="rotulo">
                  Novo {alvo.tipo === "dia" ? "dia ou horário" : "horário"}
                </span>
                <input
                  className="campo"
                  value={destino}
                  onChange={(e) => setDestino(e.target.value)}
                  placeholder="29/10/2026 14:00-16:00"
                />
              </label>
            )}
            <p className="text-xs text-[var(--text-muted)]">
              A escolha manual é livre. Se houver excesso de capacidade ou
              conflitos, a agenda exibirá alertas.
            </p>
            <label className="flex gap-2 text-sm">
              <input
                type="checkbox"
                disabled={!ligados}
                checked={avisarAoMover}
                onChange={(e) => setAvisarAoMover(e.target.checked)}
              />
              Autorizar envio aos participantes afetados após salvar
            </label>
            {!ligados && (
              <p className="text-xs">
                Configure os canais para avisar por e-mail ou WhatsApp.
              </p>
            )}
          </div>
        </Modal>
      )}
      {destinatarios && (
        <Modal
          titulo="Confirmar envio"
          onFechar={() => !ocupado && setDestinatarios(null)}
          acoes={
            <>
              <BotaoSecundario
                disabled={ocupado}
                onClick={() => setDestinatarios(null)}
              >
                Cancelar
              </BotaoSecundario>
              <BotaoPrimario
                disabled={ocupado || !ligados}
                onClick={async () => {
                  if (await executar(() => enviarAvisos(destinatarios)))
                    setDestinatarios(null);
                }}
              >
                Autorizar envio
              </BotaoPrimario>
            </>
          }
        >
          <p>
            Enviar a mensagem configurada para {destinatarios.nome}? Os
            avaliadores das mesas afetadas também recebem e-mail.
          </p>
          {!ligados && (
            <p className="mt-3">
              Ative um canal em “Configurar canais” primeiro.
            </p>
          )}
        </Modal>
      )}
      {canais && (
        <Modal
          titulo="Canais e mensagem"
          onFechar={() => void fecharCanais()}
          acoes={
            <>
              <BotaoSecundario
                disabled={ocupado}
                onClick={() => void fecharCanais()}
              >
                Cancelar
              </BotaoSecundario>
              <BotaoPrimario disabled={ocupado || !cfg} onClick={gravarCanais}>
                Salvar canais
              </BotaoPrimario>
            </>
          }
        >
          {cfg ? (
            <AvisosForm cfg={cfg} onChange={setCfg} />
          ) : (
            <p>Carregando configuração...</p>
          )}
        </Modal>
      )}
      {detalhe && (
        <Modal titulo="Mensagens do envio" onFechar={() => setDetalhe(null)}>
          <div className="space-y-3">
            {(detalhe.avisos ?? []).map((a, i) => (
              <article key={i} className="border-b pb-3 text-sm">
                <b>{a.nome}</b> · {a.canal} · {a.status}
                <p>{a.destino}</p>
                <p className="whitespace-pre-wrap">{a.mensagem}</p>
                {a.erro && <p className="text-[var(--danger)]">{a.erro}</p>}
              </article>
            ))}
          </div>
        </Modal>
      )}
    </Contexto.Provider>
  );
}
