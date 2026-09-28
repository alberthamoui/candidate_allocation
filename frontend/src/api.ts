// Chave de sessão no sessionStorage
const SESSION_KEY = 'allocation_session_id';

export function getSessionId(): string | null {
  return sessionStorage.getItem(SESSION_KEY);
}

export function setSessionId(id: string): void {
  sessionStorage.setItem(SESSION_KEY, id);
}

export function clearSessionId(): void {
  sessionStorage.removeItem(SESSION_KEY);
}

// Wrapper de fetch que injeta X-Session-Id automaticamente
async function apiFetch(path: string, options: RequestInit = {}): Promise<Response> {
  const id = getSessionId();
  const headers: Record<string, string> = {
    ...(options.headers as Record<string, string>),
  };
  if (id) headers['X-Session-Id'] = id;
  return fetch(path, { ...options, headers });
}

// Mensagem para quando a sessão some: no plano gratuito do Render o servidor
// dorme sem uso e as sessões (em memória) se perdem.
export const MSG_SESSAO_EXPIRADA =
  'Sua sessão expirou (o servidor reinicia depois de um tempo sem uso). Recomece enviando a planilha.';

async function checkOk(res: Response): Promise<any> {
  const data = await res.json().catch(() => ({}));
  if (res.status === 401) throw new Error(MSG_SESSAO_EXPIRADA);
  if (!res.ok) throw new Error(data.error ?? `HTTP ${res.status}`);
  return data;
}

// ==================================================
// ============= TIPOS COMPARTILHADOS ===============
// ==================================================

export interface MappingItem {
  nomeColuna: string;
  indice: number;
  variavel: string;
}

export interface ProgressEvent {
  step: string;
  pct: number;
  tentativa?: number;
  total?: number;
  score?: number;
  done?: boolean;
  result?: AlocacaoResponse;
  error?: string;
}

export interface AlocacaoResponse {
  mesas: MesaResult[];
  total_alocados: number;
  nao_alocados_info: CandidatoResultado[];
  pontuacao: number;
  qualidade: ItemQualidade[];
}

export interface MesaResult {
  id: number;
  dia_id: number;
  dia_nome: string; // horário
  descricao: string;
  candidatos: CandidatoResultado[];
  avaliadores: AvaliadorResultado[];
}

export interface CandidatoResultado {
  id: number;
  nome: string;
  email_insper: string;
  curso: string;
  semestre: number;
  opcoes: string[]; // horários na ordem de preferência
  opcao: number; // em qual opção ficou (1 = 1ª); 0 se não alocado
  nao_posso: string[]; // avaliadores
  prefiro_nao: string[];
  conflitos: string[]; // avaliadores "prefiro não" que ficaram na mesa dele
}

export interface AvaliadorResultado {
  id: number;
  nome: string;
  email: string;
  sigla: string;
  nao_posso: string[]; // candidatos
  prefiro_nao: string[];
}

export interface ItemQualidade {
  codigo: string;
  titulo: string;
  descricao: string;
  valor: number;
  tom: "bom" | "neutro" | "atencao" | "ruim";
  candidatos: number[]; // ids destacados ao clicar
}

export interface ParametrosAlocacao {
  mesas_por_horario: number;
  min_pessoas_por_mesa: number;
  max_pessoas_por_mesa: number;
  avaliadores_por_mesa: number;
  criterios: CriterioAlocacao[];
}

export type TipoCriterio = 'misturar' | 'agrupar' | 'maximo' | 'minimo' | 'um_de_cada';

// Regra opcional sobre curso/semestre (ver criterios.go)
export interface CriterioAlocacao {
  tipo: TipoCriterio;
  coluna: 'curso' | 'semestre';
  valores: string[];
  limite: number; // maximo/minimo
  peso: number; // pontos por unidade de desvio
}

export interface ValorColuna {
  valor: string;
  quantidade: number;
}

export interface HorarioCapacidade {
  descricao: string;
  interessados: number;
  primeira_opcao: number;
}

export interface CapacidadeResponse {
  parametros: ParametrosAlocacao;
  candidatos: number;
  avaliadores: number;
  mesas_por_horario: number;
  capacidade_por_horario: number;
  capacidade_total: number;
  max_alocaveis: number;
  horarios: HorarioCapacidade[];
  avisos: string[];
  valores_colunas: Record<string, ValorColuna[]>;
}

// Parâmetros escolhidos na tela de parâmetros; ficam no sessionStorage para
// a tela de resultado usar (e sobreviver a um recarregamento).
const PARAMS_KEY = 'allocation_params';

export function getParametrosSalvos(): ParametrosAlocacao | null {
  try {
    const raw = sessionStorage.getItem(PARAMS_KEY);
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

export function salvarParametros(p: ParametrosAlocacao): void {
  sessionStorage.setItem(PARAMS_KEY, JSON.stringify(p));
}

function paramsQuery(p: ParametrosAlocacao | null): string {
  if (!p) return '';
  // números como texto; os critérios vão como JSON
  return new URLSearchParams(
    Object.entries(p).map(([k, v]) => [k, typeof v === 'object' ? JSON.stringify(v) : String(v)])
  ).toString();
}

// ==================================================
// =================== ENDPOINTS ====================
// ==================================================

// Etapa 1: upload do arquivo + criação de sessão
export async function upload(
  file: File,
  nOpcoes: number,
  emailDomain: string
): Promise<MappingItem[]> {
  const form = new FormData();
  form.append('file', file);
  form.append('nOpcoes', String(nOpcoes));
  form.append('emailDomain', emailDomain);

  const res = await fetch('/api/upload', { method: 'POST', body: form });
  const data = await checkOk(res);
  setSessionId(data.sessionId);
  return data.mapping;
}

// Etapa 1 — mapeamento de candidatos
export async function buildUsuarios(items: MappingItem[]): Promise<any> {
  const res = await apiFetch('/api/build-usuarios', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(items),
  });
  return checkOk(res);
}

export async function saveUsuarios(data: any[]): Promise<void> {
  const res = await apiFetch('/api/save-usuarios', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  await checkOk(res);
}

// Etapa 2 — avaliadores
export async function suggestMappingAvaliador(): Promise<MappingItem[]> {
  const res = await apiFetch('/api/suggest-avaliador', { method: 'POST' });
  return checkOk(res);
}

export async function buildAvaliadores(items: MappingItem[]): Promise<any[]> {
  const res = await apiFetch('/api/build-avaliadores', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(items),
  });
  return checkOk(res);
}

export async function saveAvaliadores(data: any[]): Promise<void> {
  const res = await apiFetch('/api/save-avaliadores', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  await checkOk(res);
}

// Etapa 3 — restrições
export async function suggestMappingRestricao(): Promise<MappingItem[]> {
  const res = await apiFetch('/api/suggest-restricao', { method: 'POST' });
  return checkOk(res);
}

export async function buildRestricoes(items: MappingItem[]): Promise<any[]> {
  const res = await apiFetch('/api/build-restricoes', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(items),
  });
  return checkOk(res);
}

export async function saveRestricoes(data: any[]): Promise<void> {
  const res = await apiFetch('/api/save-restricoes', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  await checkOk(res);
}

// Etapa 4 — prévia do que cabe com os parâmetros (sem parâmetros: padrões)
export async function getCapacidade(p: ParametrosAlocacao | null): Promise<CapacidadeResponse> {
  const q = paramsQuery(p);
  return checkOk(await apiFetch('/api/capacidade' + (q ? '?' + q : '')));
}

// Etapa 5 — alocação via SSE
// Retorna um EventSource. O caller ouve eventos até receber { done: true }.
export function startAlocacao(
  params: ParametrosAlocacao | null,
  onProgress: (e: ProgressEvent) => void,
  onDone: (result: AlocacaoResponse) => void,
  onError: (msg: string) => void
): EventSource {
  const id = getSessionId();
  const q = paramsQuery(params);
  const url = `/api/alocar?sessionId=${encodeURIComponent(id ?? '')}` + (q ? '&' + q : '');
  const es = new EventSource(url);

  es.onmessage = (ev) => {
    const data: ProgressEvent = JSON.parse(ev.data);
    if (data.error) {
      es.close();
      onError(data.error);
      return;
    }
    if (data.done && data.result) {
      es.close();
      onDone(data.result);
      return;
    }
    onProgress(data);
  };

  es.onerror = () => {
    es.close();
    onError('Conexão com o servidor perdida durante a alocação.');
  };

  return es;
}

// Etapa 6 — download do Excel
export function downloadExcel(): void {
  const id = getSessionId();
  const url = `/api/export?sessionId=${encodeURIComponent(id ?? '')}`;
  const a = document.createElement('a');
  a.href = url;
  a.download = 'alocacao.xlsx';
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
}

// Reset — encerra a sessão no servidor e limpa o storage
export async function resetSession(): Promise<void> {
  await apiFetch('/api/session', { method: 'DELETE' });
  clearSessionId();
}

// Versão — branch e commit que o servidor está rodando
export interface VersaoInfo {
  branch: string;
  commit: string;
  modificado: boolean;
  desatualizado: boolean;
}

export async function getVersao(): Promise<VersaoInfo> {
  return checkOk(await fetch('/api/versao'));
}
