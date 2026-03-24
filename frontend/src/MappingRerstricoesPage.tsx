import { useNavigate } from "react-router-dom";
import { BuildRestricoesWithMapping } from "../wailsjs/go/main/App";
import MappingEditorPage from "./MappingEditorPage";
import type { MappingFieldInfo, MappingItem } from "./importTypes";

interface MappingRestricoesPageProps {
	mapping: MappingItem[] | null;
	setMapping: (items: MappingItem[]) => void;
	fieldInfos: MappingFieldInfo[];
	setRestricoes: (data: any) => void;
}

export default function MappingRestricoesPage({
	mapping,
	setMapping,
	fieldInfos,
	setRestricoes,
}: MappingRestricoesPageProps) {
	const navigate = useNavigate();

	async function handleConfirm(items: MappingItem[]) {
		const restricoes = await BuildRestricoesWithMapping(items);
		setRestricoes(restricoes);
		navigate("/verifyRestricoes");
	}

	return (
		<MappingEditorPage
			title="Mapeamento de Restricoes"
			description="Mapeie apenas os campos centrais da aba de restricoes. Colunas nao utilizadas ficam disponiveis para remapeamento."
			mapping={mapping}
			setMapping={setMapping}
			fieldInfos={fieldInfos}
			onConfirm={handleConfirm}
			confirmLabel="Revisar restricoes"
			allowExtraFields={false}
		/>
	);
}
