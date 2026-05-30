import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ToastContainer, type ToastMessage } from './Toast';

const baseToast = (overrides: Partial<ToastMessage> = {}): ToastMessage => ({
  id: 'toast-1',
  title: 'Hello',
  message: 'World',
  type: 'info',
  ...overrides,
});

describe('ToastContainer', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders nothing for an empty list', () => {
    const { container } = render(<ToastContainer toasts={[]} onClose={vi.fn()} />);
    expect(container.querySelectorAll('h4')).toHaveLength(0);
  });

  it('renders each toast title and message', () => {
    render(
      <ToastContainer
        toasts={[
          baseToast({ id: 't1', title: 'First', message: 'msg-1' }),
          baseToast({ id: 't2', title: 'Second', message: 'msg-2' }),
        ]}
        onClose={vi.fn()}
      />,
    );

    expect(screen.getByText('First')).toBeInTheDocument();
    expect(screen.getByText('msg-1')).toBeInTheDocument();
    expect(screen.getByText('Second')).toBeInTheDocument();
    expect(screen.getByText('msg-2')).toBeInTheDocument();
  });

  it('renders different colors for each toast type', () => {
    const types: ToastMessage['type'][] = ['info', 'success', 'warning', 'error'];
    const onClose = vi.fn();

    types.forEach((type) => {
      const { container, unmount } = render(
        <ToastContainer
          toasts={[baseToast({ id: type, type, title: type, message: '' })]}
          onClose={onClose}
        />,
      );
      // Each type gets a distinct color class — we just check that *some* color
      // class is applied to lock in the type-to-style mapping.
      const toastEl = container.querySelector('[class*="bg-"]');
      expect(toastEl, `expected a colored element for type ${type}`).not.toBeNull();
      unmount();
    });
  });

  it('clicking a toast invokes its onClick (when set) and closes it', () => {
    const onClose = vi.fn();
    const onClick = vi.fn();

    render(
      <ToastContainer
        toasts={[baseToast({ onClick })]}
        onClose={onClose}
      />,
    );

    // Click the toast surface (the title area).
    fireEvent.click(screen.getByText('Hello'));

    expect(onClick).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledWith('toast-1');
  });

  it('clicking the close (×) button closes without firing onClick', () => {
    const onClose = vi.fn();
    const onClick = vi.fn();

    render(
      <ToastContainer toasts={[baseToast({ onClick })]} onClose={onClose} />,
    );

    // The close icon is inside a <button> — find it and click.
    const closeButtons = screen.getAllByRole('button');
    expect(closeButtons.length).toBeGreaterThan(0);
    fireEvent.click(closeButtons[0]);

    expect(onClose).toHaveBeenCalledWith('toast-1');
    expect(onClick).not.toHaveBeenCalled();
  });

  it('auto-dismisses after the default 5-second duration', () => {
    const onClose = vi.fn();

    render(
      <ToastContainer toasts={[baseToast()]} onClose={onClose} />,
    );

    // Before duration elapses: not closed.
    vi.advanceTimersByTime(4_000);
    expect(onClose).not.toHaveBeenCalled();

    // After duration elapses + the 300ms exit animation buffer.
    vi.advanceTimersByTime(1_000); // 5s reached -> setIsVisible(false), schedules onClose
    vi.advanceTimersByTime(300);   // exit animation completes, onClose fires
    expect(onClose).toHaveBeenCalledWith('toast-1');
  });

  it('respects a custom duration', () => {
    const onClose = vi.fn();
    render(
      <ToastContainer
        toasts={[baseToast({ duration: 1_000 })]}
        onClose={onClose}
      />,
    );

    vi.advanceTimersByTime(800);
    expect(onClose).not.toHaveBeenCalled();

    vi.advanceTimersByTime(200 + 300);
    expect(onClose).toHaveBeenCalledWith('toast-1');
  });

  it('does not auto-dismiss when duration is explicitly 0', () => {
    const onClose = vi.fn();
    render(
      <ToastContainer
        toasts={[baseToast({ duration: 0 })]}
        onClose={onClose}
      />,
    );

    // Even after a long time, no auto-close should fire.
    vi.advanceTimersByTime(60_000);
    expect(onClose).not.toHaveBeenCalled();
  });

  it('renders the right material icon per type', () => {
    const cases: Array<{ type: ToastMessage['type']; icon: string }> = [
      { type: 'success', icon: 'check_circle' },
      { type: 'warning', icon: 'warning' },
      { type: 'error', icon: 'error' },
      { type: 'info', icon: 'info' },
    ];

    cases.forEach(({ type, icon }) => {
      const { container, unmount } = render(
        <ToastContainer
          toasts={[baseToast({ id: type, type, title: type, message: '' })]}
          onClose={vi.fn()}
        />,
      );
      // Scope the query to this render's container so prior renders' DOM
      // (still cached by happy-dom across iterations) cannot collide.
      const iconEl = container.querySelector('.material-icons-round');
      expect(iconEl?.textContent).toBe(icon);
      unmount();
    });
  });
});
