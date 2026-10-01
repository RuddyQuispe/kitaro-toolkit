export namespace config {
	
	export class Config {
	    theme: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	    }
	}

}

export namespace history {
	
	export class Entry {
	    id: string;
	    toolId: string;
	    toolName: string;
	    kind: string;
	    input?: string;
	    left?: string;
	    right?: string;
	    output: string;
	    options?: Record<string, string>;
	    inputPreview?: string;
	    leftPreview?: string;
	    rightPreview?: string;
	    outputPreview?: string;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.toolId = source["toolId"];
	        this.toolName = source["toolName"];
	        this.kind = source["kind"];
	        this.input = source["input"];
	        this.left = source["left"];
	        this.right = source["right"];
	        this.output = source["output"];
	        this.options = source["options"];
	        this.inputPreview = source["inputPreview"];
	        this.leftPreview = source["leftPreview"];
	        this.rightPreview = source["rightPreview"];
	        this.outputPreview = source["outputPreview"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
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

