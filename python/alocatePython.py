#!/usr/bin/env python3
# -*- coding: utf-8 -*-

# ==================================================
# ============== IMPORTS E CONSTANTES ==============
# ==================================================

import sqlite3
import random
import time
import math
from dataclasses import dataclass, field
from typing import Dict, List, Tuple, Optional
import itertools
import sys

import re

MESAS_POR_HORARIO        = 5
MIN_PESSOAS_POR_MESA     = 5
MAX_PESSOAS_POR_MESA     = 8
MIN_AVALIADORES_POR_MESA = 5
MAX_AVALIADORES_POR_MESA = 5
MAX_TESTES               = 100_000
MELHOR_CASO              = 50

# ==================================================
# ==================== STRUCTS =====================
# ==================================================

@dataclass
class Mesa:
    ID: int                    # único (ex.: 301 = quarta-mesa1)
    DiaID: int                 # 1=segunda, 2=terça, ...
    Descricao: str             # "quarta – mesa 2"
    Candidatos: List[int] = field(default_factory=list)  # já alocados
    Avaliadores: List[int] = field(default_factory=list)

@dataclass
class ResultadoAlocacao:
    Alocacao: Dict[int, int]
    Pontuacao: int
    Alocados: int

@dataclass
class Avaliador:
    ID: int
    Nome: str
    Email: str

@dataclass
class Horario:
    ID: int
    Descricao: str
    Candidatos: List[int] = field(default_factory=list)
    Avaliadores: List[int] = field(default_factory=list)

@dataclass
class horarioInfo:
    H: Horario
    Pessoas: List[int]

def fatorialBig(n: int) -> int:
    # Apenas por paridade com o Go (não é usado no fluxo principal)
    if n < 0:
        raise ValueError("n deve ser não-negativo")
    return math.factorial(n)

# ==================================================
# =========== CARREGAMENTO DE DADOS DB ============
# ==================================================

def carregarHorarios(conn: sqlite3.Connection) -> Dict[int, Horario]:
    horarios: Dict[int, Horario] = {}
    cur = conn.cursor()
    cur.execute("SELECT id, opcao FROM opcoes_horario")
    for row in cur.fetchall():
        hid, desc = row
        horarios[hid] = Horario(ID=hid, Descricao=str(desc), Candidatos=[], Avaliadores=[])
    return horarios

def carregarDisponibilidades(conn: sqlite3.Connection, horarios: Dict[int, Horario]) -> Dict[int, List[int]]:
    prefs: Dict[int, List[int]] = {}
    cur = conn.cursor()
    cur.execute("""
        SELECT pessoa_id, horario_id, preferencia
        FROM disponibilidade
        ORDER BY pessoa_id, preferencia ASC
    """)
    for pid, hid, pref in cur.fetchall():
        h = horarios.get(hid)
        if h is None:
            # Log equivalente ao Go (apenas print simples)
            print(f"[WARN] horario_id {hid} não encontrado na tabela de horários. Ignorando.")
            continue
        h.Candidatos.append(pid)
        prefs.setdefault(pid, []).append(hid)
    return prefs

def carregarAvaliadores(conn: sqlite3.Connection) -> List[Avaliador]:
    cur = conn.cursor()
    cur.execute("SELECT id, nome, email FROM avaliador")
    avals: List[Avaliador] = []
    for row in cur.fetchall():
        aid, nome, email = row
        avals.append(Avaliador(ID=aid, Nome=str(nome), Email=str(email)))
    return avals

def carregarRestricoes(conn: sqlite3.Connection) -> Tuple[Dict[int, Dict[int, bool]], Dict[int, Dict[int, bool]]]:
    """
    Retorna (hard, soft) onde as chaves de primeiro nível são AvaliadorID e
    o valor é um dict de {CandidatoID: True}.
    """
    cur = conn.cursor()
    cur.execute("SELECT candidato_id, naoPosso, prefiroNao FROM restricoes")
    hard: Dict[int, Dict[int, bool]] = {}
    soft: Dict[int, Dict[int, bool]] = {}

    def parse(raw: Optional[str], target: Dict[int, Dict[int, bool]], cid: int):
        if raw is None or str(raw).strip() == "":
            return
        for sig in str(raw).split(","):
            sig = sig.strip()
            # remove prefixo 'A' se existir (ex.: "A12")
            sig = re.sub(r"^A", "", sig, flags=re.IGNORECASE)
            try:
                aid = int(sig)
            except ValueError:
                continue
            target.setdefault(aid, {})[cid] = True

    for cid, nP, pN in cur.fetchall():
        parse(nP, hard, cid)
        parse(pN, soft, cid)

    return hard, soft

# ==================================================
# =============== GERAÇÃO DE MESAS =================
# ==================================================

def gerarMesas(hmap: Dict[int, Horario], avals: List[Avaliador]) -> Tuple[List[Mesa], Dict[int, List[Mesa]]]:
    todas: List[Mesa] = []
    porDia: Dict[int, List[Mesa]] = {}

    random.seed(int(time.time()))

    for h in hmap.values():
        # embaralha avaliadores apenas 1x por dia
        perm = list(avals)
        random.shuffle(perm)

        idx = 0  # índice circular em perm

        for i in range(MESAS_POR_HORARIO):
            m = Mesa(
                ID=h.ID * 100 + i,
                DiaID=h.ID,
                Descricao=f"{h.Descricao} – mesa {i+1}",
                Candidatos=[],
                Avaliadores=[],
            )

            qtd = random.randint(MIN_AVALIADORES_POR_MESA, MAX_AVALIADORES_POR_MESA)
            for _ in range(qtd):
                avID = perm[idx % len(perm)].ID
                idx += 1
                m.Avaliadores.append(avID)

            todas.append(m)
            porDia.setdefault(h.ID, []).append(m)

    return todas, porDia

# ==================================================
# ========= PRÉ-PROCESSAMENTO DE HORÁRIOS =========
# ==================================================

def filtrarHorariosValidos(horarios: Dict[int, Horario]) -> List[Horario]:
    valid: List[Horario] = []
    for h in horarios.values():
        if len(h.Candidatos) >= MIN_PESSOAS_POR_MESA:
            valid.append(h)
    return valid

def sortHorariosPorCandidatos(hs: List[Horario]) -> None:
    hs.sort(key=lambda h: len(h.Candidatos))

# ==================================================
# ============== ALOCAÇÃO DE PESSOAS ===============
# ==================================================

def conflitante(avIDs: List[int], pid: int,
                hard: Dict[int, Dict[int, bool]],
                soft: Dict[int, Dict[int, bool]]) -> Tuple[bool, bool]:
    hardBlock = False
    softTouch = False
    for av in avIDs:
        if hard.get(av, {}).get(pid, False):
            return True, True
        if soft.get(av, {}).get(pid, False):
            softTouch = True
    return hardBlock, softTouch

def podeAvaliar(avID: int, pid: int, restr: Dict[int, Dict[int, bool]]) -> bool:
    return not restr.get(avID, {}).get(pid, False)

def podeAlocarNoHorario(h: Horario, pid: int, restr: Dict[int, Dict[int, bool]]) -> bool:
    for av in h.Avaliadores:
        if not podeAvaliar(av, pid, restr):
            return False
    return True

def fazerAlocacaoMesas(
    mesas: List[Mesa],
    porDia: Dict[int, List[Mesa]],
    prefs: Dict[int, List[int]],
    hard: Dict[int, Dict[int, bool]],
    soft: Dict[int, Dict[int, bool]],
) -> ResultadoAlocacao:

    aloc: Dict[int, int] = {}
    ocupado: Dict[int, int] = {}
    alocados: Dict[int, bool] = {}
    pontos = 0

    # Itera pelos níveis de preferência (0..3)
    for nivel in range(4):
        for pid, pref in prefs.items():
            if alocados.get(pid, False) or len(pref) <= nivel:
                continue
            dia = pref[nivel]

            melhor: Optional[Mesa] = None
            for m in porDia.get(dia, []):
                if ocupado.get(m.ID, 0) >= MAX_PESSOAS_POR_MESA:
                    continue
                hardBlock, softTouch = conflitante(m.Avaliadores, pid, hard, soft)
                if hardBlock:
                    continue

                if melhor is None:
                    melhor = m  # prioriza mesa SEM softTouch
                else:
                    _, melhorSoftTouch = conflitante(melhor.Avaliadores, pid, hard, soft)
                    if not softTouch and melhorSoftTouch:
                        melhor = m

            if melhor is None:
                continue

            # aloca na melhor mesa encontrada
            ocupado[melhor.ID] = ocupado.get(melhor.ID, 0) + 1
            melhor.Candidatos.append(pid)
            alocados[pid] = True
            aloc[pid] = melhor.ID
            pontos += nivel

    # remove mesas que ficaram abaixo do mínimo
    for m in mesas:
        if ocupado.get(m.ID, 0) > 0 and ocupado.get(m.ID, 0) < MIN_PESSOAS_POR_MESA:
            for pid in list(m.Candidatos):
                if aloc.get(pid) == m.ID:
                    del aloc[pid]
                    if pid in alocados:
                        del alocados[pid]
            m.Candidatos = []
            ocupado[m.ID] = 0

    return ResultadoAlocacao(Alocacao=aloc, Pontuacao=pontos, Alocados=len(aloc))

# ==================================================
# ============= IMPRESSÃO DOS RESULTADOS ===========
# ==================================================

def imprimirAlocacao(aloc: Dict[int, int], horarios: Dict[int, Horario]) -> int:
    print("\n---- ALOCAÇÃO FINAL ----")
    alSet: Dict[int, bool] = {}

    for pid, hid in aloc.items():
        h = horarios[hid]
        print(f"Pessoa {pid} -> {h.Descricao} (horário ID {h.ID})")
        alSet[pid] = True

    # calcula totais únicos
    totalSet: Dict[int, bool] = {}
    for h in horarios.values():
        for pid in h.Candidatos:
            totalSet[pid] = True

    nao: List[int] = [pid for pid in totalSet.keys() if not alSet.get(pid, False)]
    print(f"\n---- NÃO ALOCADOS ({len(nao)}) ----\n{nao}")
    return len(totalSet)

def imprimirHorariosPreenchidos(horarios: Dict[int, Horario], aloc: Dict[int, int], total: int) -> None:
    print(f"\n---- HORÁRIOS PREENCHIDOS ----\nCandidatos totais: {total}\n")

    m: Dict[int, List[int]] = {}
    for pid, hid in aloc.items():
        m.setdefault(hid, []).append(pid)

    preenchidos: List[horarioInfo] = []
    for h in horarios.values():
        ps = m.get(h.ID, [])
        if len(ps) > 0:
            preenchidos.append(horarioInfo(H=h, Pessoas=ps))

    preenchidos.sort(key=lambda inf: len(inf.Pessoas), reverse=True)

    for inf in preenchidos:
        print(f"Horário {inf.H.ID} ({inf.H.Descricao}): {len(inf.Pessoas)} pessoas – {inf.Pessoas} | Avaliadores: {inf.H.Avaliadores}")

def imprimirAlocacaoMesas(aloc: Dict[int, int], mesas: Dict[int, Mesa], prefs: Dict[int, List[int]]) -> int:
    print("\n---- ALOCAÇÃO FINAL ----")
    alocados: Dict[int, bool] = {}

    for pid, mid in aloc.items():
        print(f"Pessoa {pid} -> {mesas[mid].Descricao} (Mesa {mid})")
        alocados[pid] = True

    # total único de pessoas que possuem preferência registrada
    totalSet: Dict[int, bool] = {pid: True for pid in prefs.keys()}

    nao: List[int] = [pid for pid in totalSet.keys() if not alocados.get(pid, False)]
    print(f"\n---- NÃO ALOCADOS ({len(nao)}) ----\n{nao}")
    return len(totalSet)

def imprimirMesasPreenchidas(mesas: List[Mesa], aloc: Dict[int, int], total: int) -> None:
    print(f"\n---- MESAS PREENCHIDAS ----\nPessoas únicas com disponibilidade: {total}\n")

    # Mapa de prioridade dos dias
    dias = {
        "segunda": 1,
        "terca": 2,
        "terça": 2,
        "quarta": 3,
        "quinta": 4,
        "sexta": 5,
    }

    def getDiaEMesa(desc: str) -> Tuple[int, int]:
        partes = desc.split("–")
        if len(partes) < 2:
            return 999, 999
        dia = partes[0].strip()
        numMesa = 999
        # procura por "mesa <n>"
        m = re.search(r"mesa\s+(\d+)", partes[1], flags=re.IGNORECASE)
        if m:
            try:
                numMesa = int(m.group(1))
            except ValueError:
                numMesa = 999
        prioridade = dias.get(dia.lower(), 999)
        return prioridade, numMesa

    mesas_sorted = [m for m in mesas]
    mesas_sorted.sort(key=lambda m: getDiaEMesa(m.Descricao))

    for m in mesas_sorted:
        if len(m.Candidatos) == 0:
            continue
        print(f"{m.Descricao} ({len(m.Candidatos)} candidatos) – {m.Candidatos} | Avaliadores: {m.Avaliadores}")

# ==================================================
# ===================== MAIN =======================
# ==================================================

def Alocar(conn: sqlite3.Connection) -> None:
    print("---- INICIANDO ALOCAÇÃO ----")
    # carrega dados do banco
    avals = carregarAvaliadores(conn)
    hard, soft = carregarRestricoes(conn)
    horarios = carregarHorarios(conn)
    prefs = carregarDisponibilidades(conn, horarios)

    print("---- DADOS CARREGADOS ----")

    # gera MESAS (painéis) p/ cada dia
    mesas_list, porDia = gerarMesas(horarios, avals)
    print("\n---- MESAS GERADAS ----")
    for m in mesas_list:
        print(f"Mesa {m.ID} → {m.Descricao} | Avaliadores: {m.Avaliadores}")
    print("-" * 60)

    # alocação
    start = time.time()
    res = fazerAlocacaoMesas(mesas_list, porDia, prefs, hard, soft)

    # índice mesaID -> Mesa
    mapMesa: Dict[int, Mesa] = {m.ID: m for m in mesas_list}

    # impressão
    total = imprimirAlocacaoMesas(res.Alocacao, mapMesa, prefs)
    imprimirMesasPreenchidas(mesas_list, res.Alocacao, total)

    print(f"\nTempo total de execução: {time.time() - start:.6f}s")

if __name__ == "__main__":
    # Execução direta via CLI:
    #   python alocate.py caminho_para_db.sqlite
    # se não for informado, usa 'casoTeste.db' no diretório atual
    db_path = sys.argv[1] if len(sys.argv) > 1 else "casoTeste.db"
    try:
        conn = sqlite3.connect(db_path)
    except Exception as e:
        print(f"Erro ao conectar no banco '{db_path}': {e}")
        sys.exit(1)
    with conn:
        Alocar(conn)
