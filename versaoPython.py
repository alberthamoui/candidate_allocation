import sqlite3
import random
import time
from collections import defaultdict

# ==================================================
# ============== CONSTANTES ========================
# ==================================================

MESAS_POR_HORARIO        = 5
MIN_PESSOAS_POR_MESA     = 5
MAX_PESSOAS_POR_MESA     = 8
MIN_AVALIADORES_POR_MESA = 5
MAX_AVALIADORES_POR_MESA = 5
MAX_TESTES               = 100_000
MELHOR_CASO              = 50


# ==================================================
# ==================== CLASSES =====================
# ==================================================

class Mesa:
    def __init__(self, ID, DiaID, Descricao):
        self.ID = ID
        self.DiaID = DiaID
        self.Descricao = Descricao
        self.Candidatos = []
        self.Avaliadores = []


class ResultadoAlocacao:
    def __init__(self, Alocacao, Pontuacao, Alocados):
        self.Alocacao = Alocacao
        self.Pontuacao = Pontuacao
        self.Alocados = Alocados


class Avaliador:
    def __init__(self, ID, Nome, Email):
        self.ID = ID
        self.Nome = Nome
        self.Email = Email


class Horario:
    def __init__(self, ID, Descricao):
        self.ID = ID
        self.Descricao = Descricao
        self.Candidatos = []
        self.Avaliadores = []


# ==================================================
# =========== CARREGAMENTO DE DADOS DB =============
# ==================================================

def carregar_horarios(conn):
    horarios = {}
    cur = conn.cursor()
    for row in cur.execute("SELECT id, opcao FROM opcoes_horario"):
        h = Horario(row[0], row[1])
        horarios[h.ID] = h
    return horarios


def carregar_disponibilidades(conn, horarios):
    prefs = defaultdict(list)
    cur = conn.cursor()
    for pid, hid, pref in cur.execute(
        "SELECT pessoa_id, horario_id, preferencia FROM disponibilidade ORDER BY pessoa_id, preferencia ASC"
    ):
        if hid not in horarios:
            print(f"[WARN] horario_id {hid} não encontrado. Ignorando.")
            continue
        horarios[hid].Candidatos.append(pid)
        prefs[pid].append(hid)
    return prefs


def carregar_avaliadores(conn):
    avals = []
    cur = conn.cursor()
    for row in cur.execute("SELECT id, nome, email FROM avaliador"):
        avals.append(Avaliador(row[0], row[1], row[2]))
    return avals


def carregar_restricoes(conn):
    hard = defaultdict(dict)
    soft = defaultdict(dict)
    cur = conn.cursor()
    for cid, nP, pN in cur.execute("SELECT candidato_id, naoPosso, prefiroNao FROM restricoes"):
        def parse(raw, target):
            if raw is None or not raw.strip():
                return
            for sig in raw.split(","):
                sig = sig.strip().lstrip("A")
                try:
                    aid = int(sig)
                except ValueError:
                    continue
                target[aid][cid] = True
        parse(nP, hard)
        parse(pN, soft)
    return hard, soft


def gerar_mesas(hmap, avals):
    todas = []
    porDia = defaultdict(list)

    random.seed(time.time())

    for h in hmap.values():
        perm = avals[:]
        random.shuffle(perm)
        idx = 0

        for i in range(MESAS_POR_HORARIO):
            m = Mesa(h.ID * 100 + i, h.ID, f"{h.Descricao} – mesa {i+1}")

            qtd = random.randint(MIN_AVALIADORES_POR_MESA, MAX_AVALIADORES_POR_MESA)
            for _ in range(qtd):
                avID = perm[idx % len(perm)].ID
                idx += 1
                m.Avaliadores.append(avID)

            todas.append(m)
            porDia[h.ID].append(m)
    return todas, porDia


# ==================================================
# ============== ALOCAÇÃO DE PESSOAS ===============
# ==================================================

def conflitante(avIDs, pid, hard, soft):
    hardBlock = False
    softTouch = False
    for av in avIDs:
        if pid in hard[av]:
            return True, True
        if pid in soft[av]:
            softTouch = True
    return hardBlock, softTouch


def fazer_alocacao_mesas(mesas, porDia, prefs, hard, soft):
    aloc = {}
    ocupado = defaultdict(int)
    alocados = set()
    pontos = 0

    for nivel in range(4):
        for pid, pref in prefs.items():
            if pid in alocados or len(pref) <= nivel:
                continue
            dia = pref[nivel]

            melhor = None
            for m in porDia[dia]:
                if ocupado[m.ID] >= MAX_PESSOAS_POR_MESA:
                    continue
                hardBlock, softTouch = conflitante(m.Avaliadores, pid, hard, soft)
                if hardBlock:
                    continue

                if melhor is None:
                    melhor = m
                else:
                    _, melhorSoft = conflitante(melhor.Avaliadores, pid, hard, soft)
                    if not softTouch and melhorSoft:
                        melhor = m

            if melhor is None:
                continue

            ocupado[melhor.ID] += 1
            melhor.Candidatos.append(pid)
            alocados.add(pid)
            aloc[pid] = melhor.ID
            pontos += nivel

    # remove mesas com poucos candidatos
    for m in mesas:
        if 0 < ocupado[m.ID] < MIN_PESSOAS_POR_MESA:
            for pid in m.Candidatos:
                aloc.pop(pid, None)
                alocados.discard(pid)
            m.Candidatos = []
            ocupado[m.ID] = 0

    return ResultadoAlocacao(aloc, pontos, len(aloc))


# ==================================================
# ============== IMPRESSÃO DOS RESULTADOS ==========
# ==================================================

def imprimir_alocacao_mesas(aloc, mesas, prefs):
    print("\n---- ALOCAÇÃO FINAL ----")
    alocados = set()
    for pid, mid in aloc.items():
        print(f"Pessoa {pid} -> {mesas[mid].Descricao} (Mesa {mid})")
        alocados.add(pid)

    totalSet = set(prefs.keys())
    nao = sorted(totalSet - alocados)

    print(f"\n---- NÃO ALOCADOS ({len(nao)}) ----\n{nao}")
    return len(totalSet)


def imprimir_mesas_preenchidas(mesas, aloc, total):
    print(f"\n---- MESAS PREENCHIDAS ----\nPessoas únicas com disponibilidade: {total}\n")
    for m in sorted(mesas, key=lambda x: (x.DiaID, x.ID)):
        if not m.Candidatos:
            continue
        print(f"{m.Descricao} ({len(m.Candidatos)} candidatos) – {m.Candidatos} | Avaliadores: {m.Avaliadores}")


# ==================================================
# ===================== MAIN =======================
# ==================================================

def alocar(conn):
    print("---- INICIANDO ALOCAÇÃO ----")
    avals = carregar_avaliadores(conn)
    hard, soft = carregar_restricoes(conn)
    horarios = carregar_horarios(conn)
    prefs = carregar_disponibilidades(conn, horarios)

    print("---- DADOS CARREGADOS ----")

    mesas, porDia = gerar_mesas(horarios, avals)

    print("\n---- MESAS GERADAS ----")
    for m in mesas:
        print(f"Mesa {m.ID} → {m.Descricao} | Avaliadores: {m.Avaliadores}")

    start = time.time()
    res = fazer_alocacao_mesas(mesas, porDia, prefs, hard, soft)
    mapMesa = {m.ID: m for m in mesas}

    total = imprimir_alocacao_mesas(res.Alocacao, mapMesa, prefs)
    imprimir_mesas_preenchidas(mesas, res.Alocacao, total)

    print(f"\nTempo total de execução: {time.time()-start:.3f}s")


if __name__ == "__main__":
    conn = sqlite3.connect("insper.db")
    alocar(conn)
    conn.close()
