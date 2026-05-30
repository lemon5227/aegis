import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { PendingSyncView } from './PendingSyncView';
import type { PendingSyncAction, PendingSyncActionKind } from '../types';

const action = (overrides: Partial<PendingSyncAction> = {}): PendingSyncAction => ({
  id: 'a-1',
  kind: 'post-create',
  entityId: 'p-1',
  summary: 'Posted "hello world"',
  createdAt: Date.parse('2026-05-30T12:00:00Z'),
  ...overrides,
});

describe('PendingSyncView', () => {
  it('renders the page title and description', () => {
    render(<PendingSyncView actions={[]} />);
    expect(screen.getByText('Saved Actions')).toBeInTheDocument();
    expect(screen.getByText('Background')).toBeInTheDocument();
  });

  it('shows the empty state when no actions', () => {
    render(<PendingSyncView actions={[]} />);
    expect(screen.getByText('No saved background actions.')).toBeInTheDocument();
  });

  it('renders one card per action with summary', () => {
    render(
      <PendingSyncView
        actions={[
          action({ id: '1', summary: 'first action' }),
          action({ id: '2', summary: 'second action' }),
        ]}
      />,
    );
    expect(screen.getByText('first action')).toBeInTheDocument();
    expect(screen.getByText('second action')).toBeInTheDocument();
  });

  it('maps each action kind to the right type label', () => {
    const cases: Array<{ kind: PendingSyncActionKind; label: string }> = [
      { kind: 'post-create', label: 'Post' },
      { kind: 'post-edit', label: 'Post' },
      { kind: 'post-delete', label: 'Post' },
      { kind: 'comment-create', label: 'Comment' },
      { kind: 'comment-edit', label: 'Comment' },
      { kind: 'comment-delete', label: 'Comment' },
      { kind: 'comment-vote', label: 'Comment' },
      { kind: 'post-vote', label: 'Reaction' },
      { kind: 'profile-publish', label: 'Profile' },
    ];

    for (const tc of cases) {
      const { unmount } = render(
        <PendingSyncView
          actions={[action({ id: tc.kind, kind: tc.kind, summary: `summary-${tc.kind}` })]}
        />,
      );
      expect(screen.getByText(tc.label)).toBeInTheDocument();
      unmount();
    }
  });

  it('clicking "Sync Now" calls onSyncNow', () => {
    const onSyncNow = vi.fn();
    render(<PendingSyncView actions={[]} onSyncNow={onSyncNow} />);
    fireEvent.click(screen.getByRole('button', { name: 'Sync Now' }));
    expect(onSyncNow).toHaveBeenCalledTimes(1);
  });

  it('renders Open / Dismiss buttons only when corresponding handlers are provided', () => {
    const { rerender } = render(<PendingSyncView actions={[action()]} />);
    expect(screen.queryByRole('button', { name: 'Open' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Dismiss' })).not.toBeInTheDocument();

    const onOpenAction = vi.fn();
    const onDismissAction = vi.fn();
    rerender(
      <PendingSyncView
        actions={[action()]}
        onOpenAction={onOpenAction}
        onDismissAction={onDismissAction}
      />,
    );
    expect(screen.getByRole('button', { name: 'Open' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Dismiss' })).toBeInTheDocument();
  });

  it('clicking Open passes the action to onOpenAction', () => {
    const onOpenAction = vi.fn();
    const a = action({ id: 'a-42', summary: 'open me' });
    render(<PendingSyncView actions={[a]} onOpenAction={onOpenAction} />);
    fireEvent.click(screen.getByRole('button', { name: 'Open' }));
    expect(onOpenAction).toHaveBeenCalledWith(a);
  });

  it('clicking Dismiss passes the id to onDismissAction', () => {
    const onDismissAction = vi.fn();
    const a = action({ id: 'a-99' });
    render(<PendingSyncView actions={[a]} onDismissAction={onDismissAction} />);
    fireEvent.click(screen.getByRole('button', { name: 'Dismiss' }));
    expect(onDismissAction).toHaveBeenCalledWith('a-99');
  });
});
