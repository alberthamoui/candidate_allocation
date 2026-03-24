import { useNavigate } from "react-router-dom";
import { BuildUsuariosWithMapping } from "../wailsjs/go/main/App";
import MappingEditorPage from "./MappingEditorPage";
import type { MappingDraft, MappingFieldInfo, MappingItem } from "./importTypes";

interface MappingPageProps {
	mapping: MappingDraft[] | null;
	setMapping: (items: MappingDraft[]) => void;
	fieldInfos: MappingFieldInfo[];
	setUsers: (data: any) => void;
	setDuplicatas: (data: any) => void;
	setDuplicateFields: (data: string[]) => void;
}

export default function MappingPage({
	mapping,
	setMapping,
	fieldInfos,
	setUsers,
	setDuplicatas,
	setDuplicateFields,
}: MappingPageProps) {
	const navigate = useNavigate();

	async function handleConfirm(items: MappingItem[]) {
		const { usuarios, duplicates, duplicateFields } =
			await BuildUsuariosWithMapping(items);
		setUsers(usuarios);
		setDuplicatas(duplicates);
		setDuplicateFields(duplicateFields ?? []);
		navigate("/verify");
	}

	return (
		<MappingEditorPage
			title="Mapeamento de Candidatos"
			description="Arraste as colunas do arquivo para as variaveis correspondentes e ajuste os campos extras antes da revisao."
			mapping={mapping}
			setMapping={setMapping}
			fieldInfos={fieldInfos}
			onConfirm={handleConfirm}
			confirmLabel="Revisar candidatos"
		/>
	);
}
