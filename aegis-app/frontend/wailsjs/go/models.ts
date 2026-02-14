export namespace main {
	
	export class Identity {
	    mnemonic: string;
	    publicKey: string;
	
	    static createFrom(source: any = {}) {
	        return new Identity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mnemonic = source["mnemonic"];
	        this.publicKey = source["publicKey"];
	    }
	}

}

