import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { ArrowLeftIcon, CheckCircleIcon, MagnifyingGlassIcon, XMarkIcon } from "@heroicons/react/24/outline";
import { UserCard } from "../components/UserCard";
import { saveUsuarios } from "../api";
import { Aviso, BarraAcoes, BotaoPrimario, BotaoSecundario, CabecalhoPagina, Metrica, Modal, Secao, Selo } from "../components/ui";

interface ErrorItem {
	field: string;
	msg: string;
}

interface Usuario {
	[key: string]: any;
}

interface UserWrapper {
	erros: ErrorItem[];
	usuario: Usuario;
}

interface VerifyUserPageProps {
	usuarios: Record<number, UserWrapper>;
	duplicates: number[][];
}

export default function VerifyUserPage({
	usuarios,
	duplicates,
}: VerifyUserPageProps) {
	const makeEditableCopy = () =>
		Object.fromEntries(
			Object.entries(usuarios ?? {}).map(([id, u]) => [id, { ...u.usuario }])
		);

	const navigate = useNavigate();

	const [editedUsers, setEditedUsers] = useState<Record<number, Usuario>>(
		makeEditableCopy()
	);
	const [dupGroups, setDupGroups] = useState<number[][]>(duplicates ?? []);
	const [acceptedIds, setAcceptedIds] = useState<Set<number>>(new Set());
	const [errorMsg, setErrorMsg] = useState<string | null>(null);
	const [saveSuccess, setSaveSuccess] = useState(false);
	const [busca, setBusca] = useState("");
	const [soErros, setSoErros] = useState(false);

	const getGroup = (id: number) =>
		dupGroups.find((g) => g.includes(id)) || [];

	function handleCellChange(
		userId: number,
		field: string,
		value: string | number
	) {
		setEditedUsers((prev) => ({
			...prev,
			[userId]: { ...prev[userId], [field]: value },
		}));
	}
	const flattenDup = () => dupGroups.flat();
	const isDuplicate = (id: number) => flattenDup().includes(id);

	function acceptOne(group: number[], idAccepted: number) {
		setAcceptedIds((s) => new Set(s).add(idAccepted));
		const others = group.filter((id) => id !== idAccepted);

		setEditedUsers((prev) => {
			const nxt = { ...prev };
			others.forEach((id) => delete nxt[id]);
			return nxt;
		});
		setDupGroups((prev) => prev.filter((g) => g !== group));
	}

	function acceptAll(group: number[]) {
		const keys = ["cpf", "email_pessoal", "email_insper"];
		const seen = new Map<string, number>();
		for (const id of group) {
			const usr = editedUsers[id];
			for (const k of keys) {
				const v = usr[k];
				if (v && seen.has(`${k}_${v}`)) {
					setErrorMsg(
						`Não é possível aceitar todos: campo "${k}" duplicado entre IDs ${seen.get(
							`${k}_${v}`
						)} e ${id}`
					);
					return;
				}
				seen.set(`${k}_${v}`, id);
			}
		}
		setAcceptedIds((s) => {
			const n = new Set(s);
			group.forEach((id) => n.add(id));
			return n;
		});
		setDupGroups((prev) => prev.filter((g) => g !== group));
	}

	function deleteUser(userId: number) {
		setEditedUsers((prev) => {
			const next = { ...prev };
			delete next[userId];
			return next;
		});
		setAcceptedIds((prev) => {
			const next = new Set(prev);
			next.delete(userId);
			return next;
		});
		setDupGroups((prev) =>
			prev
				.map((group) => group.filter((id) => id !== userId))
				.filter((group) => group.length > 1)
		);
	}

	function rejectAll(group: number[]) {
		setEditedUsers((prev) => {
			const n = { ...prev };
			group.forEach((id) => delete n[id]);
			return n;
		});
		setDupGroups((prev) => prev.filter((g) => g !== group));
		setAcceptedIds((s) => {
			const n = new Set(s);
			group.forEach((id) => n.delete(id));
			return n;
		});
	}

	async function saveCandidates() {
		if (dupGroups.length > 0) {
			setErrorMsg(
				"Não é possível salvar enquanto houver usuários duplicados. Resolva todos os conflitos primeiro."
			);
			return;
		}

		const usuariosParaSalvar = Object.entries(editedUsers).map(
			([, user]) => ({
				timestamp: user.timestamp || "",
				nome: user.nome || "",
				cpf: user.cpf || "",
				numero: user.numero || "",
				telefone: user.telefone || "",
				semestre: user.semestre || "",
				curso: user.curso || "",
				email_insper: user.email_insper || "",
				email_pessoal: user.email_pessoal || "",
				opcoes: user.opcoes || [],
			})
		);

		try {
			await saveUsuarios(usuariosParaSalvar);
			setSaveSuccess(true);
			setTimeout(() => navigate("/upload-avaliador"), 1500);
		} catch (err) {
			setErrorMsg("Erro ao salvar candidatos: " + err);
		}
	}

	const renderUserCard = (userId: number, extraBtn?: React.ReactNode) => {
		const user = editedUsers[userId];
		const errors = usuarios[userId]?.erros ?? [];

		return (
			<UserCard
				userId={userId}
				user={user}
				errors={errors}
				onDelete={deleteUser}
				onCellChange={handleCellChange}
				extraBtn={extraBtn}
			/>
		);
	};

	const unicos = Object.keys(editedUsers).map(Number).filter((id) => !isDuplicate(id));
	const comErro = (id: number) => (usuarios?.[id]?.erros?.length ?? 0) > 0;
	const nErros = unicos.filter(comErro).length;
	const termo = busca.trim().toLowerCase();
	const visiveis = unicos.filter(
		(id) =>
			(!soErros || comErro(id)) &&
			(!termo || JSON.stringify(editedUsers[id]).toLowerCase().includes(termo))
	);
	const bloqueado = dupGroups.length > 0 || saveSuccess;

	return (
		<div className="space-y-6">
			{errorMsg && (
				<Modal titulo="Não foi possível salvar" onFechar={() => setErrorMsg(null)} acoes={<BotaoPrimario onClick={() => setErrorMsg(null)}>Entendido</BotaoPrimario>}>
					{errorMsg}
				</Modal>
			)}

			<CabecalhoPagina
				titulo="Revisão de candidatos"
				descricao="Confira os dados lidos da planilha. Clique em qualquer campo para corrigir. Duplicidades precisam ser resolvidas antes de salvar."
			/>

			<div className="grid grid-cols-2 gap-3 md:grid-cols-4">
				<Metrica rotulo="Candidatos" valor={Object.keys(editedUsers).length} />
				<Metrica rotulo="Sem conflito" valor={unicos.length} tom="sucesso" />
				<Metrica rotulo="Com erro de validação" valor={nErros} tom={nErros ? "atencao" : "neutro"} ajuda="Erros não impedem salvar, mas vale corrigir." />
				<Metrica rotulo="Grupos duplicados" valor={dupGroups.length} tom={dupGroups.length ? "perigo" : "neutro"} />
			</div>

			{saveSuccess && (
				<Aviso tom="sucesso" icone={<CheckCircleIcon className="h-4 w-4" />}>
					Candidatos salvos! Indo para avaliadores...
				</Aviso>
			)}

			{dupGroups.map((group, idx) => (
				<Secao
					key={idx}
					titulo={`Grupo duplicado #${idx + 1}`}
					subtitulo={`Mesmo CPF ou email nos candidatos ${group.join(", ")}. Aceite um, aceite todos (se forem pessoas diferentes) ou recuse o grupo.`}
					className="border-[rgba(245,166,35,0.5)]"
					acoes={
						<>
							<BotaoSecundario onClick={() => acceptAll(group)}>
								<CheckCircleIcon className="h-4 w-4" />
								Aceitar Todos
							</BotaoSecundario>
							<BotaoSecundario onClick={() => rejectAll(group)}>
								<XMarkIcon className="h-4 w-4" />
								Recusar Todos
							</BotaoSecundario>
						</>
					}
				>
					<div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
						{group.map((id) =>
							renderUserCard(
								id,
								<BotaoPrimario className="w-full" onClick={() => acceptOne(group, id)}>
									<CheckCircleIcon className="h-4 w-4" />
									Aceitar Este Usuário
								</BotaoPrimario>
							)
						)}
					</div>
				</Secao>
			))}

			<Secao
				titulo="Candidatos"
				acoes={
					<>
						<label className="relative">
							<MagnifyingGlassIcon className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--text-light)]" />
							<input
								type="search"
								aria-label="Buscar candidato"
								placeholder="Buscar..."
								value={busca}
								onChange={(e) => setBusca(e.target.value)}
								className="campo w-56 pl-8"
							/>
						</label>
						<button
							type="button"
							aria-pressed={soErros}
							onClick={() => setSoErros(!soErros)}
							className={soErros ? "btn-primario" : "btn-secundario"}
						>
							Só com erro <Selo tom={nErros ? "atencao" : "neutro"}>{nErros}</Selo>
						</button>
					</>
				}
			>
				{visiveis.length === 0 ? (
					<p className="py-6 text-center text-sm text-[var(--text-muted)]">Nenhum candidato com esses filtros.</p>
				) : (
					<div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">{visiveis.map((id) => renderUserCard(id))}</div>
				)}
			</Secao>

			<BarraAcoes dica={dupGroups.length > 0 ? `Resolva ${dupGroups.length} grupo(s) duplicado(s) para salvar.` : `${Object.keys(editedUsers).length} candidatos serão salvos.`}>
				<BotaoSecundario onClick={() => navigate(-1)}>
					<ArrowLeftIcon className="h-4 w-4" />
					Voltar
				</BotaoSecundario>
				<BotaoPrimario onClick={saveCandidates} disabled={bloqueado}>
					<CheckCircleIcon className="h-4 w-4" />
					Salvar Candidatos
				</BotaoPrimario>
			</BarraAcoes>
		</div>
	);
}
