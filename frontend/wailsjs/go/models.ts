export namespace main {
	
	export class PessoaInfo {
	    id: number;
	    nome: string;
	    email_insper: string;
	    curso: string;
	    semestre: number;
	
	    static createFrom(source: any = {}) {
	        return new PessoaInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nome = source["nome"];
	        this.email_insper = source["email_insper"];
	        this.curso = source["curso"];
	        this.semestre = source["semestre"];
	    }
	}
	export class MesaResult {
	    id: number;
	    descricao: string;
	    candidatos: string[];
	    avaliadores: string[];
	
	    static createFrom(source: any = {}) {
	        return new MesaResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.descricao = source["descricao"];
	        this.candidatos = source["candidatos"];
	        this.avaliadores = source["avaliadores"];
	    }
	}
	export class AlocacaoResponse {
	    mesas: MesaResult[];
	    total_alocados: number;
	    nao_alocados_info: PessoaInfo[];
	    pontuacao: number;
	
	    static createFrom(source: any = {}) {
	        return new AlocacaoResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mesas = this.convertValues(source["mesas"], MesaResult);
	        this.total_alocados = source["total_alocados"];
	        this.nao_alocados_info = this.convertValues(source["nao_alocados_info"], PessoaInfo);
	        this.pontuacao = source["pontuacao"];
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
	export class AvaliadorInfo {
	    nome: string;
	    email: string;
	    sigla: string;
	
	    static createFrom(source: any = {}) {
	        return new AvaliadorInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nome = source["nome"];
	        this.email = source["email"];
	        this.sigla = source["sigla"];
	    }
	}
	export class ErrorEntry {
	    field: string;
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
	export class MappingItem {
	    nomeColuna: string;
	    indice: number;
	    variavel: string;
	
	    static createFrom(source: any = {}) {
	        return new MappingItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nomeColuna = source["nomeColuna"];
	        this.indice = source["indice"];
	        this.variavel = source["variavel"];
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
	export class Usuario {
	    timestamp: string;
	    nome: string;
	    cpf: string;
	    numero: string;
	    semestre: string;
	    curso: string;
	    email_insper: string;
	    email_pessoal: string;
	    opcoes: string[];
	
	    static createFrom(source: any = {}) {
	        return new Usuario(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.nome = source["nome"];
	        this.cpf = source["cpf"];
	        this.numero = source["numero"];
	        this.semestre = source["semestre"];
	        this.curso = source["curso"];
	        this.email_insper = source["email_insper"];
	        this.email_pessoal = source["email_pessoal"];
	        this.opcoes = source["opcoes"];
	    }
	}
	export class ValidationResult {
	    erros: ErrorEntry[];
	    usuario: Usuario;
	
	    static createFrom(source: any = {}) {
	        return new ValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.erros = this.convertValues(source["erros"], ErrorEntry);
	        this.usuario = this.convertValues(source["usuario"], Usuario);
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
	
	    static createFrom(source: any = {}) {
	        return new UsuariosResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.usuarios = this.convertValues(source["usuarios"], ValidationResult, true);
	        this.duplicates = source["duplicates"];
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

