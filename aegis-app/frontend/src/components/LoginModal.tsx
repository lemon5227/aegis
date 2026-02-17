import { useEffect, useState } from 'react';
import { Identity } from '../types';

interface LoginModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreateIdentity: () => Promise<Identity | null>;
  onActivateIdentity: (identity: Identity) => Promise<void> | void;
  onLoadIdentity: () => void;
  onImportMnemonic: (mnemonic: string) => Promise<void> | void;
}

export function LoginModal({ isOpen, onClose, onCreateIdentity, onActivateIdentity, onLoadIdentity, onImportMnemonic }: LoginModalProps) {
  const [mnemonicInput, setMnemonicInput] = useState('');
  const [mode, setMode] = useState<'select' | 'import' | 'backup'>('select');
  const [createdIdentity, setCreatedIdentity] = useState<Identity | null>(null);
  const [hasBackedUp, setHasBackedUp] = useState(false);
  const [busy, setBusy] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');

  useEffect(() => {
    if (!isOpen) {
      setMnemonicInput('');
      setMode('select');
      setCreatedIdentity(null);
      setHasBackedUp(false);
      setBusy(false);
      setErrorMessage('');
    }
  }, [isOpen]);

  if (!isOpen) return null;

  const handleImport = async () => {
    if (!mnemonicInput.trim()) return;
    setBusy(true);
    setErrorMessage('');
    try {
      await onImportMnemonic(mnemonicInput.trim());
      setMnemonicInput('');
      onClose();
    } catch (error) {
      console.error('Import identity failed:', error);
      setErrorMessage('Import failed. Please verify the mnemonic and try again.');
    } finally {
      setBusy(false);
    }
  };

  const handleCreate = async () => {
    setBusy(true);
    setErrorMessage('');
    try {
      const identity = await onCreateIdentity();
      if (!identity || !identity.mnemonic) {
        setErrorMessage('Failed to create identity. Please try again.');
        return;
      }
      setCreatedIdentity(identity);
      setHasBackedUp(false);
      setMode('backup');
    } catch (error) {
      console.error('Create identity failed:', error);
      setErrorMessage('Failed to create identity. Please try again.');
    } finally {
      setBusy(false);
    }
  };

  const handleDownloadMnemonic = () => {
    if (!createdIdentity) return;
    const fileContent = [
      'Aegis Identity Backup',
      '',
      `Public Key: ${createdIdentity.publicKey}`,
      `Mnemonic: ${createdIdentity.mnemonic}`,
      '',
      'WARNING: Keep this file offline and private. Anyone with this mnemonic can control your account.',
    ].join('\n');

    const blob = new Blob([fileContent], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = `aegis-mnemonic-${createdIdentity.publicKey.slice(0, 8)}.txt`;
    document.body.appendChild(anchor);
    anchor.click();
    document.body.removeChild(anchor);
    URL.revokeObjectURL(url);
  };

  const handleContinueWithIdentity = async () => {
    if (!createdIdentity || !hasBackedUp) return;
    setBusy(true);
    setErrorMessage('');
    try {
      await onActivateIdentity(createdIdentity);
      onClose();
    } catch (error) {
      console.error('Activate identity failed:', error);
      setErrorMessage('Failed to continue with this identity. Please retry.');
    } finally {
      setBusy(false);
    }
  };

  const mnemonicWords = createdIdentity?.mnemonic?.trim().split(/\s+/) || [];

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative bg-warm-card dark:bg-surface-dark rounded-xl shadow-2xl w-full max-w-md border border-warm-border dark:border-border-dark">
        <div className="flex items-center justify-between p-4 border-b border-warm-border dark:border-border-dark">
            <h2 className="text-lg font-bold text-warm-text-primary dark:text-white">
              {mode === 'select' ? 'Welcome to Aegis' : mode === 'import' ? 'Import Identity' : 'Backup Mnemonic'}
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
                onClick={handleCreate}
                disabled={busy}
                className="w-full py-3 bg-warm-accent hover:bg-warm-accent-hover text-white font-medium rounded-lg transition-colors"
              >
                {busy ? 'Creating...' : 'Create New Identity'}
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
              {errorMessage && (
                <p className="text-sm text-red-600 dark:text-red-400 text-center">{errorMessage}</p>
              )}
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
                  disabled={busy}
                  className="flex-1 py-2 text-sm font-medium text-warm-text-secondary hover:text-warm-text-primary transition-colors"
                >
                  Back
                </button>
                <button
                  onClick={handleImport}
                  disabled={!mnemonicInput.trim() || busy}
                  className="flex-1 py-2 bg-warm-accent hover:bg-warm-accent-hover text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50"
                >
                  {busy ? 'Importing...' : 'Import'}
                </button>
              </div>
              {errorMessage && (
                <p className="text-sm text-red-600 dark:text-red-400 text-center">{errorMessage}</p>
              )}
            </div>
          ) : (
            <div className="space-y-4">
              <p className="text-sm text-warm-text-secondary dark:text-slate-400">
                Please save these 12 words now. This is the only way to recover your identity.
              </p>
              <div className="grid grid-cols-2 gap-2 rounded-lg border border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark p-3">
                {mnemonicWords.map((word, index) => (
                  <div key={`${index}-${word}`} className="font-mono text-sm text-warm-text-primary dark:text-white">
                    {index + 1}. {word}
                  </div>
                ))}
              </div>
              <button
                onClick={handleDownloadMnemonic}
                className="w-full py-2 text-sm font-medium text-warm-accent bg-warm-accent/10 border border-transparent rounded-lg hover:bg-warm-accent/20 transition-colors"
              >
                Download Mnemonic File
              </button>
              <label className="flex items-start gap-2 text-sm text-warm-text-primary dark:text-white">
                <input
                  type="checkbox"
                  checked={hasBackedUp}
                  onChange={(e) => setHasBackedUp(e.target.checked)}
                  className="mt-0.5"
                />
                <span>I have safely backed up my mnemonic phrase.</span>
              </label>
              <div className="flex gap-3">
                <button
                  onClick={() => setMode('select')}
                  disabled={busy}
                  className="flex-1 py-2 text-sm font-medium text-warm-text-secondary hover:text-warm-text-primary transition-colors disabled:opacity-50"
                >
                  Back
                </button>
                <button
                  onClick={handleContinueWithIdentity}
                  disabled={!hasBackedUp || busy}
                  className="flex-1 py-2 bg-warm-accent hover:bg-warm-accent-hover text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50"
                >
                  {busy ? 'Continuing...' : 'Continue'}
                </button>
              </div>
              {errorMessage && (
                <p className="text-sm text-red-600 dark:text-red-400 text-center">{errorMessage}</p>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
