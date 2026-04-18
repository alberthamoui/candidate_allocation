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

async function checkOk(res: Response): Promise<any> {
  const data = await res.json();
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
  nao_alocados_info: PessoaInfo[];
  pontuacao: number;
}

export interface MesaResult {
  id: number;
  descricao: string;
  candidatos: string[];
  avaliadores: string[];
}

export interface PessoaInfo {
  id: number;
  nome: string;
  email_insper: string;
  curso: string;
  semestre: number;
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

// Etapa 4 — alocação via SSE
// Retorna um EventSource. O caller ouve eventos até receber { done: true }.
export function startAlocacao(
  onProgress: (e: ProgressEvent) => void,
  onDone: (result: AlocacaoResponse) => void,
  onError: (msg: string) => void
): EventSource {
  const id = getSessionId();
  const url = `/api/alocar?sessionId=${encodeURIComponent(id ?? '')}`;
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

// Etapa 5 — download do Excel
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
