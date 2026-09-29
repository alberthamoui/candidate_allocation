import { NoSymbolIcon } from "@heroicons/react/24/outline";
import { suggestMappingRestricao } from "../api";
import EtapaAba from "../components/EtapaAba";

interface UploadRestricaoProps {
	setMappingRestricao: (data: any) => void;
}

export default function UploadRestricao({ setMappingRestricao }: UploadRestricaoProps) {
	return (
		<EtapaAba
			icone={<NoSymbolIcon className="h-6 w-6" />}
			titulo="Restrições"
			descricao={
				<>
					Os avaliadores foram salvos. Agora as restrições serão lidas da <b>terceira aba</b> da mesma planilha:
					quais avaliadores não podem (ou preferem não) avaliar cada candidato.
				</>
			}
			campos={["Candidato", "Não posso (siglas)", "Prefiro não (siglas)"]}
			rotuloBotao="Processar Restrições"
			processar={async () => setMappingRestricao(await suggestMappingRestricao())}
			proximaRota="/mapping-restricao"
			erroPrefixo="Erro ao processar restrições. Confira se o arquivo tem a aba de restrições."
		/>
	);
}
