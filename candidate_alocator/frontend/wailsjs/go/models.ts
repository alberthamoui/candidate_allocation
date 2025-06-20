export namespace main {
	
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

