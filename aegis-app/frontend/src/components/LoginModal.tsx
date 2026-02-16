import { useState } from 'react';

interface LoginModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreateIdentity: () => void;
  onLoadIdentity: () => void;
  onImportMnemonic: (mnemonic: string) => void;
}

export function LoginModal({ isOpen, onClose, onCreateIdentity, onLoadIdentity, onImportMnemonic }: LoginModalProps) {
  const [mnemonicInput, setMnemonicInput] = useState('');
  const [mode, setMode] = useState<'select' | 'create' | 'import'>('select');

  if (!isOpen) return null;

  const handleImport = () => {
    if (!mnemonicInput.trim()) return;
    onImportMnemonic(mnemonicInput.trim());
    setMnemonicInput('');
    onClose();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative bg-warm-card dark:bg-surface-dark rounded-xl shadow-2xl w-full max-w-md border border-warm-border dark:border-border-dark">
        <div className="flex items-center justify-between p-4 border-b border-warm-border dark:border-border-dark">
          <h2 className="text-lg font-bold text-warm-text-primary dark:text-white">
            {mode === 'select' ? 'Welcome to Aegis' : mode === 'create' ? 'Create Identity' : 'Import Identity'}
          </h2>
          <button onClick={onClose} className="text-warm-text-secondary hover:text-warm-text-primary">
            <span className="material-icons">close</span>
          </button>
        </div>
        
        <div className="p-6">
          {mode === 'select' ? (
            <div className="space-y-4">
              <p className="text-sm text-warm-text-secondary dark:text-slate-400 text-center mb-6">
                Join the decentralized forum. Create a new identity or import an existing one.
              </p>
              <button
                onClick={onCreateIdentity}
                className="w-full py-3 bg-warm-accent hover:bg-warm-accent-hover text-white font-medium rounded-lg transition-colors"
              >
                Create New Identity
              </button>
              <button
                onClick={onLoadIdentity}
                className="w-full py-3 bg-warm-card dark:bg-surface-lighter border border-warm-border dark:border-border-dark text-warm-text-primary dark:text-white font-medium rounded-lg hover:bg-warm-sidebar dark:hover:bg-border-dark transition-colors"
              >
                Load Existing Identity
              </button>
              <div className="relative my-4">
                <div className="absolute inset-0 flex items-center">
                  <div className="w-full border-t border-warm-border dark:border-border-dark"></div>
                </div>
                <div className="relative flex justify-center text-xs">
                  <span className="px-2 bg-warm-card dark:bg-surface-dark text-warm-text-secondary">or</span>
                </div>
              </div>
              <button
                onClick={() => setMode('import')}
                className="w-full py-2 text-sm text-warm-text-secondary hover:text-warm-accent transition-colors"
              >
                Import from Mnemonic
              </button>
            </div>
          ) : mode === 'import' ? (
            <div className="space-y-4">
              <p className="text-sm text-warm-text-secondary dark:text-slate-400">
                Enter your 12-24 word mnemonic phrase to restore your identity.
              </p>
              <textarea
                value={mnemonicInput}
                onChange={(e) => setMnemonicInput(e.target.value)}
                placeholder="word1 word2 word3 ..."
                rows={4}
                className="w-full px-4 py-2.5 rounded-lg border border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark text-warm-text-primary dark:text-white focus:ring-2 focus:ring-warm-accent focus:border-transparent outline-none resize-none"
              />
              <div className="flex gap-3">
                <button
                  onClick={() => setMode('select')}
                  className="flex-1 py-2 text-sm font-medium text-warm-text-secondary hover:text-warm-text-primary transition-colors"
                >
                  Back
                </button>
                <button
                  onClick={handleImport}
                  disabled={!mnemonicInput.trim()}
                  className="flex-1 py-2 bg-warm-accent hover:bg-warm-accent-hover text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50"
                >
                  Import
                </button>
              </div>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}
