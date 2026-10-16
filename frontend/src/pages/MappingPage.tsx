import { useNavigate } from "react-router-dom";
import { useState, useEffect, useRef } from "react";
import { ArrowLeftIcon, Bars3Icon, ExclamationTriangleIcon } from "@heroicons/react/24/outline";
import { nomeCampo } from "../nomesCampos";
import { Aviso, BarraAcoes, BotaoPrimario, BotaoSecundario, CabecalhoPagina, cn, Secao, Selo } from "../components/ui";

interface MappingItem {
	nomeColuna: string;
	indice: number;
	variavel: string;
}
interface MappingPageProps {
	mapping: MappingItem[] | null;
	/** Recebe o mapeamento atual e devolve o resultado da leitura. */
	buildFn: (items: MappingItem[]) => Promise<any>;
	/** Chamado com o resultado antes de ir para a próxima tela. */
	onSuccess: (result: any) => Promise<void> | void;
	/** Rota seguinte. */
	nextRoute: string;
	titulo: string;
	descricao: string;
	rotuloConfirmar: string;
}

export default function MappingPage({
	mapping,
	buildFn,
	onSuccess,
	nextRoute,
	titulo,
	descricao,
	rotuloConfirmar,
}: MappingPageProps) {
	const navigate = useNavigate();
	const [loading, setLoading] = useState(false);
	const [erro, setErro] = useState<string | null>(null);
	const dragActiveRef = useRef<boolean>(false);
	const [draggedIndex, setDraggedIndex] = useState<number | null>(null);
	const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);
	const [items, setItems] = useState<MappingItem[]>([]);

	useEffect(() => {
		setItems(mapping ?? []);
	}, []);

	// Rolagem automática enquanto arrasta perto das bordas
	useEffect(() => {
		const handleAutoScroll = (e: DragEvent) => {
			if (!dragActiveRef.current) return;
			const threshold = 80;
			const scrollSpeed = 6;
			if (e.clientY < threshold) {
				window.scrollBy(0, -scrollSpeed);
			} else if (e.clientY > window.innerHeight - threshold) {
				window.scrollBy(0, scrollSpeed);
			}
		};
		window.addEventListener("dragover", handleAutoScroll);
		return () => window.removeEventListener("dragover", handleAutoScroll);
	}, []);

	function onDragStart(e: React.DragEvent<HTMLDivElement>, index: number) {
		setDraggedIndex(index);
		dragActiveRef.current = true;

		const ghost = document.createElement("div");
		ghost.textContent = items[index].nomeColuna || "sem coluna";
		Object.assign(ghost.style, {
			padding: "8px 14px",
			background: "#1a1a1a",
			color: "#fff",
			borderRadius: "6px",
			fontSize: "13px",
			fontWeight: "500",
			position: "absolute",
			top: "-1000px",
		});
		document.body.appendChild(ghost);
		e.dataTransfer.setDragImage(ghost, 20, 16);
		setTimeout(() => document.body.removeChild(ghost), 0);
	}

	function onDragOver(e: React.DragEvent<HTMLDivElement>, index: number) {
		e.preventDefault();
		setDragOverIndex(index);
	}

	function onDragLeave(e: React.DragEvent<HTMLDivElement>) {
		if (e.currentTarget.contains(e.relatedTarget as Node)) return;
		setDragOverIndex(null);
	}

	function onDrop(e: React.DragEvent<HTMLDivElement>, dropIndex: number) {
		e.preventDefault();
		if (draggedIndex === null) return;
		const novo = items.map((i) => ({ ...i }));
		[novo[draggedIndex].nomeColuna, novo[dropIndex].nomeColuna] = [novo[dropIndex].nomeColuna, novo[draggedIndex].nomeColuna];
		[novo[draggedIndex].indice, novo[dropIndex].indice] = [novo[dropIndex].indice, novo[draggedIndex].indice];
		setItems(novo);
		onDragEnd();
	}

	function onDragEnd() {
		setDraggedIndex(null);
		setDragOverIndex(null);
		dragActiveRef.current = false;
	}

	async function onConfirm() {
		if (loading) return;
		setLoading(true);
		setErro(null);
		try {
			const result = await buildFn(items);
			await onSuccess(result);
			navigate(nextRoute);
		} catch (err: any) {
			setErro(err?.message ?? String(err));
		} finally {
			setLoading(false);
		}
	}

	const comColuna = items.filter((i) => i.nomeColuna).length;

	return (
		<div className="space-y-6">
			<CabecalhoPagina titulo={titulo} descricao={descricao} />

			<Secao
				titulo="Campos"
				ajuda="Arraste a coluna de um campo e solte sobre outro campo: as duas colunas trocam de lugar."
				acoes={
					<Selo tom={comColuna === items.length ? "sucesso" : "atencao"}>
						{comColuna} de {items.length} campos com coluna
					</Selo>
				}
			>
				<div className="hidden grid-cols-[1fr_1.2fr] gap-4 px-4 pb-2 md:grid">
					<span className="rotulo-caps">Campo do sistema</span>
					<span className="rotulo-caps">Coluna da planilha</span>
				</div>
				<div className="space-y-2">
					{items.map((item, index) => (
						<div
							key={index}
							data-testid="linha-mapeamento"
							data-variavel={item.variavel}
							className={cn(
								"grid grid-cols-1 items-center gap-2 rounded-[var(--radius-lg)] border px-4 py-3 transition-colors md:grid-cols-[1fr_1.2fr] md:gap-4",
								dragOverIndex === index ? "border-[var(--accent)] bg-[var(--accent-soft)]" : "border-[var(--line)] bg-white"
							)}
						>
							<div className="min-w-0">
								<div className="text-sm font-medium">{nomeCampo(item.variavel)}</div>
								<div className="font-mono text-[11px] text-[var(--text-light)]">{item.variavel}</div>
							</div>
							<div
								draggable
								onDragStart={(e) => onDragStart(e, index)}
								onDragOver={(e) => onDragOver(e, index)}
								onDragLeave={onDragLeave}
								onDrop={(e) => onDrop(e, index)}
								onDragEnd={onDragEnd}
								data-testid="coluna"
								className={cn(
									"flex cursor-grab items-center justify-between gap-3 rounded-[var(--radius)] border px-3 py-2 transition-colors active:cursor-grabbing",
									item.nomeColuna
										? "border-[var(--line-strong)] bg-[var(--bg-subtle)] hover:border-[var(--accent)]"
										: "border-dashed border-[rgba(245,166,35,0.5)] bg-[var(--warning-soft)]"
								)}
							>
								{item.nomeColuna ? (
									<span className="truncate text-sm font-medium">{item.nomeColuna}</span>
								) : (
									<span className="text-sm italic text-[var(--warning)]">sem coluna</span>
								)}
								<Bars3Icon className="h-4 w-4 flex-shrink-0 text-[var(--text-light)]" />
							</div>
						</div>
					))}
				</div>
			</Secao>

			{erro && (
				<Aviso tom="perigo" icone={<ExclamationTriangleIcon className="h-4 w-4" />}>
					{erro}
				</Aviso>
			)}

			<BarraAcoes dica="Confira as colunas antes de continuar.">
				<BotaoSecundario onClick={() => navigate(-1)}>
					<ArrowLeftIcon className="h-4 w-4" />
					Voltar
				</BotaoSecundario>
				<BotaoPrimario onClick={onConfirm} disabled={loading}>
					{loading ? "Processando..." : rotuloConfirmar}
				</BotaoPrimario>
			</BarraAcoes>
		</div>
	);
}
