import { useNavigate } from "react-router-dom";
import { BuildAvaliadoresWithMapping } from "../wailsjs/go/main/App";
import MappingEditorPage from "./MappingEditorPage";
import type { MappingDraft, MappingFieldInfo, MappingItem } from "./importTypes";

interface MappingAvaliadoresPageProps {
	mapping: MappingDraft[] | null;
	setMapping: (items: MappingDraft[]) => void;
	fieldInfos: MappingFieldInfo[];
	setAvaliadores: (data: any) => void;
	setDuplicatas: (data: any) => void;
	setDuplicateFields: (data: string[]) => void;
}

export default function MappingAvaliadoresPage({
	mapping,
	setMapping,
	fieldInfos,
	setAvaliadores,
	setDuplicatas,
	setDuplicateFields,
}: MappingAvaliadoresPageProps) {
	const navigate = useNavigate();

	async function handleConfirm(items: MappingItem[]) {
		const response = await BuildAvaliadoresWithMapping(items);
		setAvaliadores(response.avaliadores);
		setDuplicatas(response.duplicates);
		setDuplicateFields(response.duplicateFields ?? []);
		navigate("/verifyAvaliadores");
	}

	return (
		<MappingEditorPage
			title="Mapeamento de Avaliadores"
			description="Arraste as colunas da aba de avaliadores, confira os badges e revise antes do salvamento."
			mapping={mapping}
			setMapping={setMapping}
			fieldInfos={fieldInfos}
			onConfirm={handleConfirm}
			confirmLabel="Revisar avaliadores"
		/>
	);
}
