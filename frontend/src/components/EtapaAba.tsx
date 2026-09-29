import { ReactNode, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ArrowLeftIcon, ExclamationTriangleIcon } from "@heroicons/react/24/outline";
import { Aviso, BotaoPrimario, BotaoSecundario } from "./ui";

// Tela intermediária que lê uma aba da planilha já enviada (avaliadores ou
// restrições) e segue para o mapeamento dela.
export default function EtapaAba({
	icone,
	titulo,
	descricao,
	campos,
	rotuloBotao,
	processar,
	proximaRota,
	erroPrefixo,
}: {
	icone: ReactNode;
	titulo: string;
	descricao: ReactNode;
	campos: string[];
	rotuloBotao: string;
	processar: () => Promise<void>;
	proximaRota: string;
	erroPrefixo: string;
}) {
	const navigate = useNavigate();
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState<string | null>(null);

	async function handleProcess() {
		setLoading(true);
		setError(null);
		try {
			await processar();
			navigate(proximaRota);
		} catch (err: any) {
			setError(`${erroPrefixo} Detalhe: ${err.message}`);
		} finally {
			setLoading(false);
		}
	}

	return (
		<div className="mx-auto flex min-h-[60vh] w-full max-w-lg flex-col justify-center">
			<div className="painel p-8">
				<div className="flex h-11 w-11 items-center justify-center rounded-[var(--radius-lg)] bg-[var(--accent-soft)] text-[var(--accent)]">
					{icone}
				</div>
				<h1 className="mt-4 text-2xl font-semibold">{titulo}</h1>
				<p className="mt-2 text-sm leading-6 text-[var(--text-muted)]">{descricao}</p>

				<div className="mt-5">
					<div className="rotulo-caps mb-2">Colunas esperadas</div>
					<div className="flex flex-wrap gap-1.5">
						{campos.map((c) => (
							<span key={c} className="rounded-[var(--radius)] border border-[var(--line)] bg-[var(--bg-subtle)] px-2 py-0.5 text-xs">
								{c}
							</span>
						))}
					</div>
				</div>

				{error && (
					<Aviso tom="perigo" icone={<ExclamationTriangleIcon className="h-4 w-4" />} className="mt-5">
						{error}
					</Aviso>
				)}

				<div className="mt-6 flex flex-col-reverse gap-2 sm:flex-row">
					<BotaoSecundario onClick={() => navigate(-1)} className="sm:w-32">
						<ArrowLeftIcon className="h-4 w-4" />
						Voltar
					</BotaoSecundario>
					<BotaoPrimario onClick={handleProcess} disabled={loading} className="flex-1">
						{loading ? "Processando..." : rotuloBotao}
					</BotaoPrimario>
				</div>
			</div>
		</div>
	);
}
