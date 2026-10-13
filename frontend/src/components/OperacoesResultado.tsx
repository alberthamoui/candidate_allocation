import { ReactNode, useEffect, useState } from "react";
import {
  AlocacaoResponse,
  AvisoGerado,
  ConfigAvisos,
  LoteAvisos,
  ResultadoOperacao,
  SugestaoRealocacao,
  avaliar,
  configAvisosVazia,
  enviarAvisos,
  getAvisos,
  getHistoricoAvisos,
  getSessionId,
  getSugestoes,
  realocar,
  repetirFalhas,
  salvarAvisos,
} from "../api";
import AvisosForm from "./AvisosForm";

type Aba = "imprevisto" | "pessoa" | "avaliacao" | "avisos";
const abas: { id: Aba; rotulo: string }[] = [
  { id: "imprevisto", rotulo: "Imprevisto" },
  { id: "pessoa", rotulo: "Uma pessoa" },
  { id: "avaliacao", rotulo: "Avaliação" },
  { id: "avisos", rotulo: "Avisos" },
];
const campo =
  "mt-1 w-full border border-gray-300 rounded-lg p-2 text-sm bg-white";
const botao =
  "px-4 py-2 rounded-xl bg-blue-600 text-white text-sm font-semibold disabled:opacity-50";
function diaDe(h: string) {
  return h.match(/^(.*?)\s+\d{1,2}[-h:]/)?.[1] ?? h.split(" ")[0];
}
export default function OperacoesResultado({
  result,
  onResultado,
}: {
  result: AlocacaoResponse;
  onResultado: (r: AlocacaoResponse) => void;
}) {
  const [aba, setAba] = useState<Aba>("imprevisto");
  const [cfg, setCfg] = useState<ConfigAvisos>(configAvisosVazia());
  const [de, setDe] = useState("");
  const [escopo, setEscopo] = useState("horario");
  const [paraDia, setParaDia] = useState("");
  const [mesaId, setMesaId] = useState(result.mesas[0]?.id ?? 0);
  const [paraMesa, setParaMesa] = useState("");
  const [pessoaId, setPessoaId] = useState(0);
  const [mesaDestino, setMesaDestino] = useState(0);
  const [horarioPessoa, setHorarioPessoa] = useState("");
  const [busca, setBusca] = useState("");
  const [avisar, setAvisar] = useState(false);
  const [resumo, setResumo] = useState("");
  const [avisos, setAvisos] = useState<AvisoGerado[]>([]);
  const [historico, setHistorico] = useState<LoteAvisos[]>([]);
  const [sugestoes, setSugestoes] = useState<SugestaoRealocacao[]>([]);
  const [erro, setErro] = useState("");
  const [ocupado, setOcupado] = useState(false);
  const mesas = result.mesas ?? [];
  const horarios = Array.from(new Set(mesas.map((m) => m.dia_nome)));
  const dias =
    escopo === "dia" ? Array.from(new Set(horarios.map(diaDe))) : horarios;
  const pessoas = [
    ...mesas.flatMap((m) =>
      m.candidatos.map((c) => ({ ...c, onde: m.dia_nome })),
    ),
    ...(result.nao_alocados_info ?? []).map((c) => ({
      ...c,
      onde: "sem mesa",
    })),
  ];
  const pessoasVisiveis = pessoas.filter((p) =>
    `${p.nome} ${p.email_insper} ${p.email_pessoal ?? ""}`
      .toLocaleLowerCase()
      .includes(busca.toLocaleLowerCase()),
  );
  const podeAvisar = cfg.ativo && (cfg.email.ativo || cfg.whatsapp.ativo);
  const enviando = historico.some((l) => l.status === "enviando");
  useEffect(() => {
    getAvisos()
      .then((c) => {
        setCfg(c);
        setAvisar(c.ativo && (c.email.ativo || c.whatsapp.ativo));
      })
      .catch((e) => setErro(e.message));
  }, []);
  useEffect(() => {
    if (!dias.includes(de)) setDe(dias[0] ?? "");
  }, [result, escopo]);
  useEffect(() => {
    if (!pessoas.some((p) => p.id === pessoaId))
      setPessoaId(pessoas[0]?.id ?? 0);
    if (!mesas.some((m) => m.id === mesaId)) setMesaId(mesas[0]?.id ?? 0);
  }, [result]);
  useEffect(() => {
    let valido = true;
    if (aba === "pessoa" && pessoaId)
      getSugestoes(pessoaId)
        .then((s) => {
          if (valido) setSugestoes(s);
        })
        .catch((e) => {
          if (valido) setErro(e.message);
        });
    return () => {
      valido = false;
    };
  }, [aba, pessoaId, result]);
  useEffect(() => {
    let ativo = true;
    const atualizar = () =>
      getHistoricoAvisos()
        .then((h) => {
          if (ativo) setHistorico(h);
        })
        .catch((e) => {
          if (ativo) setErro(e.message);
        });
    atualizar();
    const timer = setInterval(atualizar, 2500);
    return () => {
      ativo = false;
      clearInterval(timer);
    };
  }, []);
  async function executar(acao: () => Promise<ResultadoOperacao>) {
    setOcupado(true);
    setErro("");
    setResumo("");
    setAvisos([]);
    try {
      const op = await acao();
      setResumo(op.resumo);
      setAvisos(op.avisos ?? []);
      if (op.resultado) onResultado(op.resultado);
      setHistorico(await getHistoricoAvisos());
    } catch (e: any) {
      setErro(e.message);
    } finally {
      setOcupado(false);
    }
  }
  function escolher(s: SugestaoRealocacao) {
    setMesaDestino(s.mesa_id);
    setHorarioPessoa(s.horario);
  }
  const afetadas = mesas.filter((m) =>
    escopo === "dia" ? diaDe(m.dia_nome) === de : m.dia_nome === de,
  );
  const qtd = afetadas.reduce((n, m) => n + m.candidatos.length, 0);
  return (
    <section className="bg-white rounded-2xl shadow-lg border border-gray-100 p-6 space-y-4">
      <div className="flex flex-wrap gap-2">
        {abas.map((a) => (
          <button
            type="button"
            key={a.id}
            aria-pressed={aba === a.id}
            className={`px-4 py-2 rounded-full text-sm font-semibold ${aba === a.id ? "bg-blue-600 text-white" : "bg-gray-100 text-gray-700"}`}
            onClick={() => setAba(a.id)}
          >
            {a.rotulo}
          </button>
        ))}
      </div>
      <p className="text-xs text-gray-600">
        Realocação manual é livre. As sugestões ajudam a escolher; os alertas
        mostram exceções. A lista, os indicadores e o Excel acompanham a agenda
        atual.
      </p>
      {!!result.alertas?.length && (
        <details className="bg-amber-50 text-amber-900 p-3 rounded-lg">
          <summary className="cursor-pointer font-medium text-sm">
            {result.alertas.length} alerta(s) da agenda manual
          </summary>
          <ul className="list-disc pl-5 mt-2 text-sm space-y-1">
            {result.alertas.map((a, i) => (
              <li key={i}>{a}</li>
            ))}
          </ul>
        </details>
      )}
      {aba === "imprevisto" && (
        <div className="space-y-4">
          <Rotulo texto="O que mover?">
            <select
              aria-label="O que mover?"
              className={campo}
              value={escopo}
              onChange={(e) => setEscopo(e.target.value)}
            >
              <option value="horario">Um intervalo de horário</option>
              <option value="dia">
                Um dia inteiro, preservando os horários
              </option>
            </select>
          </Rotulo>
          <div className="grid sm:grid-cols-3 gap-3">
            <Rotulo texto="Dia atual">
              <select
                className={campo}
                value={de}
                onChange={(e) => setDe(e.target.value)}
              >
                {dias.map((d) => (
                  <option key={d}>{d}</option>
                ))}
              </select>
            </Rotulo>
            <Rotulo texto="Novo dia">
              <input
                className={campo}
                placeholder={
                  escopo === "dia" ? "sexta ou 23/10/2026" : "sexta 14-16"
                }
                value={paraDia}
                onChange={(e) => setParaDia(e.target.value)}
              />
            </Rotulo>
            <div className="flex items-end">
              <button
                type="button"
                className={botao}
                disabled={ocupado || !paraDia.trim()}
                onClick={() => {
                  if (
                    !window.confirm(
                      `Mover ${afetadas.length} mesa(s) e ${qtd} candidato(s) de "${de}" para "${paraDia}"?${avisar ? " Enviar avisos aos candidatos e avaliadores afetados." : ""}`,
                    )
                  )
                    return;
                  executar(() =>
                    realocar({ tipo: "dia", de, para: paraDia, avisar }),
                  );
                }}
              >
                Mover o dia
              </button>
            </div>
          </div>
          <p className="text-sm text-gray-600">
            {afetadas.length} mesa(s), {qtd} candidato(s).{" "}
            {escopo === "dia"
              ? "Todos os intervalos desse dia serão preservados."
              : "Somente o intervalo selecionado será alterado."}
          </p>
          <div className="grid sm:grid-cols-3 gap-3">
            <Rotulo texto="Mesa">
              <select
                className={campo}
                value={mesaId}
                onChange={(e) => setMesaId(Number(e.target.value))}
              >
                {mesas.map((m) => (
                  <option value={m.id} key={m.id}>
                    #{m.id} · {m.descricao}
                  </option>
                ))}
              </select>
            </Rotulo>
            <Rotulo texto="Destino da mesa">
              <input
                className={campo}
                value={paraMesa}
                onChange={(e) => setParaMesa(e.target.value)}
                placeholder="quinta 10-12"
              />
            </Rotulo>
            <div className="flex items-end">
              <button
                type="button"
                className={botao}
                disabled={ocupado || !paraMesa.trim()}
                onClick={() => {
                  if (
                    !window.confirm(
                      `Mover a mesa para "${paraMesa}"?${avisar ? " Avisar candidatos e avaliadores." : ""}`,
                    )
                  )
                    return;
                  executar(() =>
                    realocar({
                      tipo: "mesa",
                      mesa_id: mesaId,
                      para: paraMesa,
                      avisar,
                    }),
                  );
                }}
              >
                Mover a mesa
              </button>
            </div>
          </div>
          <label className="flex gap-2 items-center text-sm">
            <input
              type="checkbox"
              checked={avisar}
              disabled={!podeAvisar}
              onChange={(e) => setAvisar(e.target.checked)}
            />
            Avisar todo mundo por e-mail e WhatsApp
          </label>
        </div>
      )}
      {(aba === "pessoa" || aba === "avaliacao") && (
        <Rotulo texto="Buscar pessoa por nome ou e-mail">
          <input
            className={campo}
            type="search"
            value={busca}
            onChange={(e) => setBusca(e.target.value)}
            placeholder="Digite o nome ou e-mail"
          />
        </Rotulo>
      )}
      {aba === "pessoa" && (
        <div className="space-y-4">
          <Rotulo texto="Pessoa">
            <select
              className={campo}
              value={pessoaId}
              onChange={(e) => {
                setPessoaId(Number(e.target.value));
                setMesaDestino(0);
                setHorarioPessoa("");
              }}
            >
              {pessoasVisiveis.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.nome} — {p.onde}
                </option>
              ))}
            </select>
          </Rotulo>
          <div className="space-y-2">
            <h3 className="font-semibold text-sm">
              Melhores destinos sugeridos
            </h3>
            <p className="text-xs text-gray-500">
              Ordenados por restrições, avaliadores, vagas e preferência. Você
              pode escolher qualquer mesa ou um horário novo.
            </p>
            <div className="grid md:grid-cols-3 gap-2">
              {sugestoes.slice(0, 3).map((s, i) => (
                <button
                  type="button"
                  key={s.mesa_id}
                  onClick={() => escolher(s)}
                  className={`text-left border rounded-xl p-3 text-sm ${mesaDestino === s.mesa_id ? "border-blue-500 bg-blue-50" : "border-gray-200"}`}
                >
                  <span className="block font-semibold">
                    {i + 1}º · {s.descricao}
                  </span>
                  <span className="block text-xs text-green-800 mt-1">
                    {s.motivos.join(" · ")}
                  </span>
                  {!!s.alertas.length && (
                    <span className="block text-xs text-amber-800 mt-1">
                      {s.alertas.join(" · ")}
                    </span>
                  )}
                </button>
              ))}
            </div>
          </div>
          <Rotulo texto="Mesa de destino">
            <select
              className={campo}
              value={mesaDestino}
              onChange={(e) => {
                const id = Number(e.target.value);
                setMesaDestino(id);
                if (id)
                  setHorarioPessoa(
                    mesas.find((m) => m.id === id)?.dia_nome ?? "",
                  );
              }}
            >
              <option value={0}>Escolher horário manualmente</option>
              {sugestoes.map((s) => (
                <option key={s.mesa_id} value={s.mesa_id}>
                  #{s.mesa_id} · {s.descricao} ·{" "}
                  {s.alertas.length
                    ? `${s.alertas.length} alerta(s)`
                    : "sem conflitos"}
                </option>
              ))}
            </select>
          </Rotulo>
          <Rotulo texto="Novo horário">
            <input
              className={campo}
              value={horarioPessoa}
              onChange={(e) => {
                setHorarioPessoa(e.target.value);
                setMesaDestino(0);
              }}
              placeholder="sexta 8-10"
              list="horarios-conhecidos"
            />
            <datalist id="horarios-conhecidos">
              {horarios.map((d) => (
                <option key={d} value={d} />
              ))}
            </datalist>
          </Rotulo>
          {mesaDestino === 0 &&
            horarioPessoa &&
            !horarios.includes(horarioPessoa) && (
              <p className="text-sm text-amber-800 bg-amber-50 p-3 rounded-lg">
                Este horário cria uma mesa manual sem avaliadores. Você pode
                continuar; a agenda mostrará essa pendência.
              </p>
            )}
          {!!sugestoes.find((s) => s.mesa_id === mesaDestino)?.alertas
            .length && (
            <p className="text-sm text-amber-800 bg-amber-50 p-3 rounded-lg">
              {sugestoes
                .find((s) => s.mesa_id === mesaDestino)
                ?.alertas.join(" · ")}
            </p>
          )}
          <label className="flex gap-2 items-center text-sm">
            <input
              type="checkbox"
              checked={avisar}
              disabled={!podeAvisar}
              onChange={(e) => setAvisar(e.target.checked)}
            />
            Avisar esta pessoa
          </label>
          <button
            type="button"
            className={botao}
            disabled={
              ocupado || !pessoaId || (!horarioPessoa.trim() && !mesaDestino)
            }
            onClick={() => {
              const nome = pessoas.find((p) => p.id === pessoaId)?.nome;
              if (
                avisar &&
                !window.confirm(
                  `Mover ${nome} para ${horarioPessoa} e enviar avisos?`,
                )
              )
                return;
              executar(() =>
                realocar({
                  tipo: "pessoa",
                  pessoa: String(pessoaId),
                  horario: horarioPessoa,
                  mesa_id: mesaDestino || undefined,
                  avisar,
                }),
              );
            }}
          >
            Trocar horário
          </button>
        </div>
      )}
      {(aba === "imprevisto" || aba === "pessoa") && !podeAvisar && (
        <p className="text-sm text-gray-500">
          Para enviar avisos, configure um canal na aba Avisos.
        </p>
      )}
      {aba === "avaliacao" && (
        <div>
          <p className="text-sm text-gray-600 mb-3">
            Marque quem passou. A avaliação é salva e aparece no Excel.
          </p>
          <ul className="divide-y border rounded-xl max-h-96 overflow-y-auto">
            {pessoasVisiveis.map((p) => (
              <li
                key={p.id}
                className="flex flex-wrap justify-between items-center gap-2 p-3"
              >
                <span className="text-sm">
                  {p.nome}
                  <span className="block text-xs text-gray-500">{p.onde}</span>
                </span>
                <div className="flex gap-1">
                  <Nota
                    ativo={p.avaliacao === "aprovado"}
                    ocupado={ocupado}
                    onClick={() =>
                      executar(() =>
                        avaliar([{ id: p.id, resultado: "aprovado" }]),
                      )
                    }
                  >
                    Passou
                  </Nota>
                  <Nota
                    ativo={p.avaliacao === "reprovado"}
                    ocupado={ocupado}
                    onClick={() =>
                      executar(() =>
                        avaliar([{ id: p.id, resultado: "reprovado" }]),
                      )
                    }
                  >
                    Não passou
                  </Nota>
                  {p.avaliacao && (
                    <Nota
                      ativo={false}
                      ocupado={ocupado}
                      onClick={() =>
                        executar(() =>
                          avaliar([{ id: p.id, resultado: "pendente" }]),
                        )
                      }
                    >
                      Limpar
                    </Nota>
                  )}
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}
      {aba === "avisos" && (
        <div className="space-y-4">
          <AvisosForm cfg={cfg} onChange={setCfg} />
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              className={botao}
              disabled={ocupado}
              onClick={() =>
                executar(async () => {
                  await salvarAvisos(cfg);
                  setCfg(await getAvisos());
                  return {
                    resumo: "Configuração de avisos salva.",
                    avisos: [],
                  };
                })
              }
            >
              Salvar mensagem
            </button>
            <button
              type="button"
              className={botao}
              disabled={ocupado || enviando || !podeAvisar}
              onClick={() => {
                if (
                  !window.confirm(
                    "Enviar avisos para todos os candidatos alocados e seus avaliadores?",
                  )
                )
                  return;
                executar(async () => {
                  await salvarAvisos(cfg);
                  return enviarAvisos();
                });
              }}
            >
              Enviar para todos
            </button>
          </div>
          <p className="text-xs text-gray-500">
            Para usar a mesma alocação na CLI:{" "}
            <code className="break-all">
              server mesas -servidor {window.location.origin} -sessao{" "}
              {getSessionId()}
            </code>
          </p>
        </div>
      )}
      {erro && (
        <p
          role="alert"
          className="bg-red-50 text-red-700 p-3 rounded-lg text-sm"
        >
          {erro}
        </p>
      )}
      {resumo && (
        <p
          role="status"
          className="bg-green-50 text-green-800 p-3 rounded-lg text-sm"
        >
          {resumo}
        </p>
      )}
      {avisos.map((a, i) => (
        <p key={i} className="text-sm">
          {a.nome}: {a.enviado ? "aceito pelo provedor" : a.erro}
        </p>
      ))}
      {!!historico.length && (
        <details open={enviando} className="border rounded-xl p-3">
          <summary className="cursor-pointer font-semibold text-sm">
            Histórico de avisos {enviando ? "· envio em andamento" : ""}
          </summary>
          <div className="space-y-3 mt-3">
            {historico.map((l) => (
              <div key={l.id} className="border rounded-lg p-3 text-sm">
                <p>
                  {new Date(l.criado).toLocaleString("pt-BR")} · {l.status}
                </p>
                <p>
                  {l.enviados} aceito(s) pelo provedor · {l.falhas} falha(s) ·{" "}
                  {l.avisos.length} aviso(s)
                </p>
                {l.status !== "enviando" &&
                  !l.repetido_por &&
                  l.avisos.some((a) => !a.enviado) && (
                    <button
                      type="button"
                      className="mt-2 underline text-blue-700"
                      disabled={ocupado || enviando}
                      onClick={() => executar(() => repetirFalhas(l.id))}
                    >
                      Repetir somente falhas
                    </button>
                  )}
                <details className="mt-2">
                  <summary className="cursor-pointer">
                    Ver destinatários e falhas
                  </summary>
                  <ul className="max-h-48 overflow-y-auto">
                    {l.avisos.map((a, i) => (
                      <li className="py-1" key={i}>
                        {a.nome} · {a.canal} ·{" "}
                        {a.enviado
                          ? "aceito pelo provedor"
                          : a.erro || "aguardando"}
                      </li>
                    ))}
                  </ul>
                </details>
              </div>
            ))}
          </div>
        </details>
      )}
    </section>
  );
}
function Rotulo({ texto, children }: { texto: string; children: ReactNode }) {
  return (
    <label className="block text-sm font-medium text-gray-700">
      {texto}
      {children}
    </label>
  );
}
function Nota({
  ativo,
  ocupado,
  onClick,
  children,
}: {
  ativo: boolean;
  ocupado: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      aria-pressed={ativo}
      disabled={ocupado}
      onClick={onClick}
      className={`text-xs px-3 py-2 rounded-full ${ativo ? "bg-blue-600 text-white" : "bg-gray-100 text-gray-700"}`}
    >
      {children}
    </button>
  );
}
