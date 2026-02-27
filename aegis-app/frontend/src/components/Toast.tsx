import { useState, useEffect, useCallback } from 'react';

export interface ToastMessage {
  id: string;
  title: string;
  message: string;
  type: 'info' | 'success' | 'warning' | 'error';
  duration?: number;
  onClick?: () => void;
}

interface ToastContainerProps {
  toasts: ToastMessage[];
  onClose: (id: string) => void;
}

export function ToastContainer({ toasts, onClose }: ToastContainerProps) {
  return (
    <div className="fixed bottom-6 right-6 z-[100] flex flex-col gap-3 pointer-events-none">
      {toasts.map((toast) => (
        <Toast key={toast.id} toast={toast} onClose={() => onClose(toast.id)} />
      ))}
    </div>
  );
}

function Toast({ toast, onClose }: { toast: ToastMessage; onClose: () => void }) {
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    // Small delay to trigger entry animation
    const timer = requestAnimationFrame(() => setIsVisible(true));
    return () => cancelAnimationFrame(timer);
  }, []);

  useEffect(() => {
    if (toast.duration === 0) return;
    const duration = toast.duration || 5000;
    const timer = setTimeout(() => {
      setIsVisible(false);
      setTimeout(onClose, 300); // Wait for exit animation
    }, duration);
    return () => clearTimeout(timer);
  }, [toast, onClose]);

  const getIcon = () => {
    switch (toast.type) {
      case 'success': return 'check_circle';
      case 'warning': return 'warning';
      case 'error': return 'error';
      default: return 'info';
    }
  };

  const getColors = () => {
    switch (toast.type) {
      case 'success': return 'bg-green-500 dark:bg-green-600 text-white';
      case 'warning': return 'bg-amber-500 dark:bg-amber-600 text-white';
      case 'error': return 'bg-red-500 dark:bg-red-600 text-white';
      default: return 'bg-warm-accent text-white';
    }
  };

  return (
    <div
      onClick={() => {
        if (toast.onClick) toast.onClick();
        onClose();
      }}
      className={`
        pointer-events-auto cursor-pointer
        flex items-start gap-3 p-4 rounded-xl shadow-lg border border-white/10
        backdrop-blur-md transition-all duration-300 transform
        max-w-sm w-80
        ${getColors()}
        ${isVisible ? 'translate-x-0 opacity-100' : 'translate-x-full opacity-0'}
      `}
    >
      <span className="material-icons-round mt-0.5">{getIcon()}</span>
      <div className="flex-1 min-w-0">
        <h4 className="font-bold text-sm leading-tight mb-1">{toast.title}</h4>
        <p className="text-xs opacity-90 leading-relaxed line-clamp-2">{toast.message}</p>
      </div>
      <button
        onClick={(e) => {
          e.stopPropagation();
          onClose();
        }}
        className="opacity-70 hover:opacity-100 transition-opacity"
      >
        <span className="material-icons-round text-sm">close</span>
      </button>
    </div>
  );
}

export function useToasts() {
  const [toasts, setToasts] = useState<ToastMessage[]>([]);

  const addToast = useCallback((toast: Omit<ToastMessage, 'id'>) => {
    const id = Math.random().toString(36).substring(2, 9);
    setToasts((prev) => [...prev, { ...toast, id }]);
    return id;
  }, []);

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  return { toasts, addToast, removeToast };
}
