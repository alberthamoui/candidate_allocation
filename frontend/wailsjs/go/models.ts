export namespace logic {
	
	export class ErrorEntry {
	    field: number;
	    msg: string;
	
	    static createFrom(source: any = {}) {
	        return new ErrorEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.field = source["field"];
	        this.msg = source["msg"];
	    }
	}
	export class AvaliadorValidationResult {
	    erros: ErrorEntry[];
	    avaliador: types.Avaliador;
	
	    static createFrom(source: any = {}) {
	        return new AvaliadorValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.erros = this.convertValues(source["erros"], ErrorEntry);
	        this.avaliador = this.convertValues(source["avaliador"], types.Avaliador);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AvaliadoresResponse {
	    avaliadores: Record<number, AvaliadorValidationResult>;
	    duplicates: number[][];
	    duplicateFields: string[];
	
	    static createFrom(source: any = {}) {
	        return new AvaliadoresResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.avaliadores = this.convertValues(source["avaliadores"], AvaliadorValidationResult, true);
	        this.duplicates = source["duplicates"];
	        this.duplicateFields = source["duplicateFields"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ValidationResult {
	    erros: ErrorEntry[];
	    usuario: types.Candidato;
	
	    static createFrom(source: any = {}) {
	        return new ValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.erros = this.convertValues(source["erros"], ErrorEntry);
	        this.usuario = this.convertValues(source["usuario"], types.Candidato);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UsuariosResponse {
	    usuarios: Record<number, ValidationResult>;
	    duplicates: number[][];
	    duplicateFields: string[];
	
	    static createFrom(source: any = {}) {
	        return new UsuariosResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.usuarios = this.convertValues(source["usuarios"], ValidationResult, true);
	        this.duplicates = source["duplicates"];
	        this.duplicateFields = source["duplicateFields"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace types {
	
	export class NullableString {
	
	
	    static createFrom(source: any = {}) {
	        return new NullableString(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class Avaliador {
	    id: number;
	    nome: string;
	    email: string;
	    sigla: string;
	    extras: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new Avaliador(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nome = source["nome"];
	        this.email = source["email"];
	        this.sigla = source["sigla"];
	        this.extras = source["extras"];
	    }
	}
	export class Candidato {
	    timestamp: string;
	    nome: string;
	    cpf: string;
	    numero: string;
	    semestre: string;
	    curso: string;
	    email_secundario: string;
	    email_pessoal: string;
	    opcoes: string[];
	    extras: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new Candidato(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.nome = source["nome"];
	        this.cpf = source["cpf"];
	        this.numero = source["numero"];
	        this.semestre = source["semestre"];
	        this.curso = source["curso"];
	        this.email_secundario = source["email_secundario"];
	        this.email_pessoal = source["email_pessoal"];
	        this.opcoes = source["opcoes"];
	        this.extras = source["extras"];
	    }
	}
	export class MappingFieldInfo {
	    variavel: string;
	    required: boolean;
	    unique: boolean;
	    duplicate: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MappingFieldInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.variavel = source["variavel"];
	        this.required = source["required"];
	        this.unique = source["unique"];
	        this.duplicate = source["duplicate"];
	    }
	}
	export class MappingItem {
	    nomeColuna: string;
	    indice: number;
	    variavel: string;
	    includeWhenUnmapped: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MappingItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nomeColuna = source["nomeColuna"];
	        this.indice = source["indice"];
	        this.variavel = source["variavel"];
	        this.includeWhenUnmapped = source["includeWhenUnmapped"];
	    }
	}
	export class Restricao {
	    candidato: string;
	    naoPosso: string;
	    prefiroNao: string;
	
	    static createFrom(source: any = {}) {
	        return new Restricao(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.candidato = source["candidato"];
	        this.naoPosso = source["naoPosso"];
	        this.prefiroNao = source["prefiroNao"];
	    }
	}

}

