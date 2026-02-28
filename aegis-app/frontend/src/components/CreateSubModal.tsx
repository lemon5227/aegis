import { useState } from 'react';

interface CreateSubModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreate: (id: string, title: string, description: string) => Promise<void> | void;
}

export function CreateSubModal({ isOpen, onClose, onCreate }: CreateSubModalProps) {
  const [subId, setSubId] = useState('');
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [submitBusy, setSubmitBusy] = useState(false);
  const [submitMessage, setSubmitMessage] = useState('');

  if (!isOpen) return null;

  const getErrorMessage = (error: unknown): string => {
    if (error instanceof Error && error.message.trim()) {
      return error.message.trim();
    }
    if (typeof error === 'string' && error.trim()) {
      return error.trim();
    }
    return 'Failed to create sub.';
  };

  const handleSubmit = async () => {
    if (!subId.trim()) return;
    setSubmitBusy(true);
    setSubmitMessage('');
    try {
      await onCreate(subId.trim(), title.trim(), description.trim());
      setSubId('');
      setTitle('');
      setDescription('');
      onClose();
    } catch (error: unknown) {
      setSubmitMessage(getErrorMessage(error));
    } finally {
      setSubmitBusy(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={() => { if (!submitBusy) onClose(); }} />
      <div className="relative bg-warm-card dark:bg-surface-dark rounded-xl shadow-2xl w-full max-w-md border border-warm-border dark:border-border-dark">
        <div className="flex items-center justify-between p-4 border-b border-warm-border dark:border-border-dark">
          <h2 className="text-lg font-bold text-warm-text-primary dark:text-white">Create Sub</h2>
          <button onClick={onClose} disabled={submitBusy} className="text-warm-text-secondary hover:text-warm-text-primary disabled:opacity-50 disabled:cursor-not-allowed">
            <span className="material-icons">close</span>
          </button>
        </div>
        
        <div className="p-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-warm-text-primary dark:text-white mb-2">
              Sub ID *
            </label>
            <input
              type="text"
              value={subId}
              onChange={(e) => setSubId(e.target.value.toLowerCase().replace(/[^a-z0-9]/g, ''))}
              placeholder="e.g. golang, tech, music"
              className="w-full px-4 py-2.5 rounded-lg border border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark text-warm-text-primary dark:text-white focus:ring-2 focus:ring-warm-accent focus:border-transparent outline-none"
            />
          </div>
          
          <div>
            <label className="block text-sm font-medium text-warm-text-primary dark:text-white mb-2">
              Title
            </label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="e.g. Go Programming"
              className="w-full px-4 py-2.5 rounded-lg border border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark text-warm-text-primary dark:text-white focus:ring-2 focus:ring-warm-accent focus:border-transparent outline-none"
            />
          </div>
          
          <div>
            <label className="block text-sm font-medium text-warm-text-primary dark:text-white mb-2">
              Description
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="What is this community about?"
              rows={3}
              className="w-full px-4 py-2.5 rounded-lg border border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark text-warm-text-primary dark:text-white focus:ring-2 focus:ring-warm-accent focus:border-transparent outline-none resize-none"
            />
          </div>
        </div>
        
        <div className="flex justify-end gap-3 p-4 border-t border-warm-border dark:border-border-dark">
          <button
            onClick={onClose}
            disabled={submitBusy}
            className="px-4 py-2 text-sm font-medium text-warm-text-secondary hover:text-warm-text-primary transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={!subId.trim() || submitBusy}
            className="px-4 py-2 bg-warm-accent hover:bg-warm-accent-hover text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {submitBusy ? 'Creating...' : 'Create'}
          </button>
        </div>
        {submitMessage && (
          <div className="px-4 pb-4 text-sm text-red-600 dark:text-red-300">
            {submitMessage}
          </div>
        )}
      </div>
    </div>
  );
}
