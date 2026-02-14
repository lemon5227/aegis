import { useState } from 'react';
import './App.css';
import { GenerateIdentity } from "../wailsjs/go/main/App";

function App() {
    const [mnemonic, setMnemonic] = useState('');
    const [publicKey, setPublicKey] = useState('');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);

    async function createIdentity() {
        setLoading(true);
        setError('');

        try {
            const wailsBridge = (window as any)?.go?.main?.App;
            if (!wailsBridge || typeof wailsBridge.GenerateIdentity !== 'function') {
                throw new Error('未检测到 Wails Runtime。请在项目根目录运行: wails dev');
            }

            const identity = await GenerateIdentity();
            setMnemonic(identity.mnemonic);
            setPublicKey(identity.publicKey);
        } catch (exception) {
            setError(String(exception));
        } finally {
            setLoading(false);
        }
    }

    return (
        <div id="App">
            <h1>Aegis Phase 1</h1>
            <p className="result">创建本地身份（12词助记词 + 公钥）</p>
            <div className="input-box">
                <button className="btn" onClick={createIdentity} disabled={loading}>
                    {loading ? 'Generating...' : 'Create Identity'}
                </button>
            </div>

            {mnemonic && (
                <div className="panel">
                    <h3>Mnemonic</h3>
                    <p>{mnemonic}</p>
                </div>
            )}

            {publicKey && (
                <div className="panel">
                    <h3>Public Key</h3>
                    <p>{publicKey}</p>
                </div>
            )}

            {error && <p className="error">{error}</p>}
        </div>
    );
}

export default App;
