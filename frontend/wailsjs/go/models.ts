export namespace main {
	
	export class DecodeOptions {
	    parseDelimited: boolean;
	    maxDepth: number;
	    maxFields: number;
	    maxBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new DecodeOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.parseDelimited = source["parseDelimited"];
	        this.maxDepth = source["maxDepth"];
	        this.maxFields = source["maxFields"];
	        this.maxBytes = source["maxBytes"];
	    }
	}
	export class DecodeRequest {
	    input: string;
	    inputEncoding: string;
	    parseDelimited: boolean;
	    maxDepth: number;
	    maxFields: number;
	    maxBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new DecodeRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.input = source["input"];
	        this.inputEncoding = source["inputEncoding"];
	        this.parseDelimited = source["parseDelimited"];
	        this.maxDepth = source["maxDepth"];
	        this.maxFields = source["maxFields"];
	        this.maxBytes = source["maxBytes"];
	    }
	}
	export class ValueVariant {
	    candidateType: string;
	    displayValue: string;
	    description?: string;
	    confidence?: string;
	
	    static createFrom(source: any = {}) {
	        return new ValueVariant(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.candidateType = source["candidateType"];
	        this.displayValue = source["displayValue"];
	        this.description = source["description"];
	        this.confidence = source["confidence"];
	    }
	}
	export class Part {
	    byteRange: number[];
	    index: number;
	    fieldNumber: number;
	    wireType: number;
	    typeName: string;
	    rawHex: string;
	    value: ValueVariant[];
	    children?: Part[];
	
	    static createFrom(source: any = {}) {
	        return new Part(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.byteRange = source["byteRange"];
	        this.index = source["index"];
	        this.fieldNumber = source["fieldNumber"];
	        this.wireType = source["wireType"];
	        this.typeName = source["typeName"];
	        this.rawHex = source["rawHex"];
	        this.value = this.convertValues(source["value"], ValueVariant);
	        this.children = this.convertValues(source["children"], Part);
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
	export class DecodeResult {
	    parts: Part[];
	    leftover: string;
	    error?: string;
	    warnings?: string[];
	    inputSize: number;
	
	    static createFrom(source: any = {}) {
	        return new DecodeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.parts = this.convertValues(source["parts"], Part);
	        this.leftover = source["leftover"];
	        this.error = source["error"];
	        this.warnings = source["warnings"];
	        this.inputSize = source["inputSize"];
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
	export class OpenFileResult {
	    path: string;
	    size: number;
	    cancelled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new OpenFileResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.size = source["size"];
	        this.cancelled = source["cancelled"];
	    }
	}
	
	export class SaveFileResult {
	    path: string;
	    cancelled: boolean;
	    format: string;
	
	    static createFrom(source: any = {}) {
	        return new SaveFileResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.cancelled = source["cancelled"];
	        this.format = source["format"];
	    }
	}

}

