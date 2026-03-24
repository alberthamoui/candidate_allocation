export interface MappingItem {
	nomeColuna: string;
	indice: number;
	variavel: string;
	includeWhenUnmapped: boolean;
}

export interface MappingFieldInfo {
	variavel: string;
	required: boolean;
	unique: boolean;
	duplicate: boolean;
}

export interface MappingDraft extends MappingItem {
	clientId: string;
	manualExtra: boolean;
}
