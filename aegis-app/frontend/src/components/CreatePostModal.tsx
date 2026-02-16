import { useState, useRef } from 'react';

interface CreatePostModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreate: (title: string, body: string) => void;
}

export function CreatePostModal({ isOpen, onClose, onCreate }: CreatePostModalProps) {
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [imageBase64, setImageBase64] = useState('');
  const [imageMime, setImageMime] = useState('');
  const [imagePreview, setImagePreview] = useState('');
  const fileInputRef = useRef<HTMLInputElement>(null);

  if (!isOpen) return null;

  const handleSubmit = () => {
    if (!title.trim()) return;
    onCreate(title.trim(), body.trim());
    setTitle('');
    setBody('');
    setImageBase64('');
    setImageMime('');
    setImagePreview('');
    onClose();
  };

  const handleImageSelect = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    const mime = file.type || 'image/jpeg';
    const reader = new FileReader();
    reader.onload = () => {
      const result = reader.result as string;
      const marker = ';base64,';
      const index = result.indexOf(marker);
      if (index > 0) {
        setImageMime(mime);
        setImageBase64(result.slice(index + marker.length));
        setImagePreview(result);
      }
    };
    reader.readAsDataURL(file);
  };

  const clearImage = () => {
    setImageBase64('');
    setImageMime('');
    setImagePreview('');
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative bg-warm-card dark:bg-surface-dark rounded-xl shadow-2xl w-full max-w-lg border border-warm-border dark:border-border-dark max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between p-4 border-b border-warm-border dark:border-border-dark sticky top-0 bg-warm-card dark:bg-surface-dark">
          <h2 className="text-lg font-bold text-warm-text-primary dark:text-white">Create Post</h2>
          <button onClick={onClose} className="text-warm-text-secondary hover:text-warm-text-primary">
            <span className="material-icons">close</span>
          </button>
        </div>
        
        <div className="p-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-warm-text-primary dark:text-white mb-2">
              Title *
            </label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Enter a descriptive title"
              className="w-full px-4 py-2.5 rounded-lg border border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark text-warm-text-primary dark:text-white focus:ring-2 focus:ring-warm-accent focus:border-transparent outline-none"
            />
          </div>
          
          <div>
            <label className="block text-sm font-medium text-warm-text-primary dark:text-white mb-2">
              Content
            </label>
            <textarea
              value={body}
              onChange={(e) => setBody(e.target.value)}
              placeholder="What's on your mind?"
              rows={6}
              className="w-full px-4 py-2.5 rounded-lg border border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark text-warm-text-primary dark:text-white focus:ring-2 focus:ring-warm-accent focus:border-transparent outline-none resize-none"
            />
          </div>
          
          <div>
            <label className="block text-sm font-medium text-warm-text-primary dark:text-white mb-2">
              Image (optional)
            </label>
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*"
              onChange={handleImageSelect}
              className="w-full text-sm text-warm-text-secondary dark:text-slate-400 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-medium file:bg-warm-accent file:text-white file:cursor-pointer file:transition-colors"
            />
            {imagePreview && (
              <div className="mt-3 relative inline-block">
                <img 
                  src={imagePreview} 
                  alt="Preview" 
                  className="max-h-40 rounded-lg border border-warm-border dark:border-border-dark"
                />
                <button
                  onClick={clearImage}
                  className="absolute -top-2 -right-2 bg-red-500 text-white rounded-full p-1 hover:bg-red-600"
                >
                  <span className="material-icons text-sm">close</span>
                </button>
              </div>
            )}
          </div>
        </div>
        
        <div className="flex justify-end gap-3 p-4 border-t border-warm-border dark:border-border-dark sticky bottom-0 bg-warm-card dark:bg-surface-dark">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-warm-text-secondary hover:text-warm-text-primary transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={!title.trim()}
            className="px-4 py-2 bg-warm-accent hover:bg-warm-accent-hover text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Post
          </button>
        </div>
      </div>
    </div>
  );
}
