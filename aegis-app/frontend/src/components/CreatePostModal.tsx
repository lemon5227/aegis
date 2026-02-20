import { useState, useRef } from 'react';

interface CreatePostModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreate: (title: string, body: string, imageBase64?: string, imageMime?: string, externalImageURL?: string) => Promise<void> | void;
}

export function CreatePostModal({ isOpen, onClose, onCreate }: CreatePostModalProps) {
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [imageBase64, setImageBase64] = useState('');
  const [imageMime, setImageMime] = useState('');
  const [imagePreview, setImagePreview] = useState('');
  const [externalImageURL, setExternalImageURL] = useState('');
  const [imageBusy, setImageBusy] = useState(false);
  const [imageMessage, setImageMessage] = useState('');
  const [submitBusy, setSubmitBusy] = useState(false);
  const [submitMessage, setSubmitMessage] = useState('');
  const fileInputRef = useRef<HTMLInputElement>(null);

  if (!isOpen) return null;

  const handleSubmit = async () => {
    if (!title.trim()) return;
    setSubmitBusy(true);
    setSubmitMessage('');
    try {
      await onCreate(title.trim(), body.trim(), imageBase64.trim(), imageMime.trim(), externalImageURL.trim());
      setTitle('');
      setBody('');
      setImageBase64('');
      setImageMime('');
      setImagePreview('');
      setExternalImageURL('');
      onClose();
    } catch (error: any) {
      setSubmitMessage(error?.message || 'Failed to publish post.');
    } finally {
      setSubmitBusy(false);
    }
  };

  const handleImageSelect = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    if (!file.type.startsWith('image/')) {
      setImageMessage('Please choose an image file.');
      event.target.value = '';
      return;
    }

    const readFileAsDataURL = (input: File): Promise<string> => {
      return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(typeof reader.result === 'string' ? reader.result : '');
        reader.onerror = () => reject(new Error('Failed to read image'));
        reader.readAsDataURL(input);
      });
    };

    const loadImageFromDataURL = (dataURL: string): Promise<HTMLImageElement> => {
      return new Promise((resolve, reject) => {
        const image = new Image();
        image.onload = () => resolve(image);
        image.onerror = () => reject(new Error('Failed to decode image'));
        image.src = dataURL;
      });
    };

    const canvasToBlob = (canvas: HTMLCanvasElement, type: string, quality: number): Promise<Blob> => {
      return new Promise((resolve, reject) => {
        canvas.toBlob((blob) => {
          if (!blob) {
            reject(new Error('Failed to convert image'));
            return;
          }
          resolve(blob);
        }, type, quality);
      });
    };

    const compressPostImage = async (input: File): Promise<{ mime: string; base64: string; preview: string }> => {
      const MAX_SOURCE_BYTES = 8 * 1024 * 1024;
      const MAX_DIMENSION = 1600;
      const MAX_OUTPUT_BYTES = 420 * 1024;

      if (input.size > MAX_SOURCE_BYTES) {
        throw new Error('Image too large (>8MB). Please use a smaller file.');
      }

      const sourceDataURL = await readFileAsDataURL(input);
      const image = await loadImageFromDataURL(sourceDataURL);
      const scale = Math.min(1, MAX_DIMENSION / Math.max(image.width, image.height));
      const targetWidth = Math.max(1, Math.round(image.width * scale));
      const targetHeight = Math.max(1, Math.round(image.height * scale));

      const canvas = document.createElement('canvas');
      canvas.width = targetWidth;
      canvas.height = targetHeight;
      const context = canvas.getContext('2d');
      if (!context) {
        throw new Error('Canvas unavailable');
      }
      context.drawImage(image, 0, 0, targetWidth, targetHeight);

      const qualityCandidates = [0.9, 0.82, 0.74, 0.66, 0.58, 0.5, 0.42];
      for (const quality of qualityCandidates) {
        const blob = await canvasToBlob(canvas, 'image/jpeg', quality);
        if (blob.size <= MAX_OUTPUT_BYTES) {
          const dataURL = await readFileAsDataURL(new File([blob], 'post.jpg', { type: 'image/jpeg' }));
          const marker = ';base64,';
          const index = dataURL.indexOf(marker);
          if (index <= 0) {
            throw new Error('Image encode failed');
          }
          return {
            mime: 'image/jpeg',
            base64: dataURL.slice(index + marker.length),
            preview: dataURL,
          };
        }
      }

      throw new Error('Could not compress enough. Try a smaller image or use external URL.');
    };

    try {
      setImageBusy(true);
      setImageMessage('Compressing image...');
      const result = await compressPostImage(file);
      setImageMime(result.mime);
      setImageBase64(result.base64);
      setImagePreview(result.preview);
      setImageMessage('Image ready. It will be uploaded with the post.');
    } catch (error: any) {
      setImageMessage(error?.message || 'Failed to process image.');
      setImageMime('');
      setImageBase64('');
      setImagePreview('');
    } finally {
      setImageBusy(false);
      event.target.value = '';
    }
  };

  const clearImage = () => {
    setImageBase64('');
    setImageMime('');
    setImagePreview('');
    setImageMessage('');
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const clearExternalImage = () => {
    setExternalImageURL('');
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
                disabled={imageBusy}
                className="w-full text-sm text-warm-text-secondary dark:text-slate-400 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-medium file:bg-warm-accent file:text-white file:cursor-pointer file:transition-colors"
              />
              {imageMessage && (
                <p className="mt-2 text-xs text-warm-text-secondary dark:text-slate-300">{imageMessage}</p>
              )}
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

            <div className="mt-3">
              <label className="block text-xs font-medium text-warm-text-secondary dark:text-slate-400 mb-2">
                Or use external image URL (recommended for lower network storage pressure)
              </label>
              <div className="flex gap-2">
                <input
                  type="url"
                  value={externalImageURL}
                  onChange={(e) => setExternalImageURL(e.target.value)}
                  placeholder="https://example.com/image.jpg"
                  className="flex-1 px-3 py-2 text-sm rounded-lg border border-warm-border dark:border-border-dark bg-warm-bg dark:bg-background-dark text-warm-text-primary dark:text-white focus:ring-2 focus:ring-warm-accent focus:border-transparent outline-none"
                />
                <button
                  onClick={clearExternalImage}
                  className="px-3 py-2 text-xs font-medium text-warm-text-secondary dark:text-slate-300 rounded-lg border border-warm-border dark:border-border-dark hover:bg-warm-bg dark:hover:bg-background-dark"
                >
                  Clear
                </button>
              </div>
              <p className="mt-1 text-xs text-warm-text-secondary dark:text-slate-400">
                If both local file and external URL are set, local file is used for post media.
              </p>
            </div>
          </div>
        </div>
        
        <div className="flex justify-end gap-3 p-4 border-t border-warm-border dark:border-border-dark sticky bottom-0 bg-warm-card dark:bg-surface-dark">
          <button
            onClick={onClose}
            disabled={submitBusy}
            className="px-4 py-2 text-sm font-medium text-warm-text-secondary hover:text-warm-text-primary transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={() => void handleSubmit()}
            disabled={!title.trim() || submitBusy}
            className="px-4 py-2 bg-warm-accent hover:bg-warm-accent-hover text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {submitBusy ? 'Posting...' : 'Post'}
          </button>
        </div>
        {submitMessage && (
          <div className="px-4 pb-3 text-xs text-red-500">{submitMessage}</div>
        )}
      </div>
    </div>
  );
}
