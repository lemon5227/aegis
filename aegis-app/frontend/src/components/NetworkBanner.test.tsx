import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { NetworkBanner } from './NetworkBanner';
import type { NetworkHealthSnapshot } from '../lib/networkHealth';
import type { PendingSyncAction } from '../types';

const health = (overrides: Partial<NetworkHealthSnapshot> = {}): NetworkHealthSnapshot => ({
  level: 'healthy',
  label: 'Up To Date',
  summary: 'Everything looks current.',
  peerCount: 1,
  lagSeconds: 0,
  lastSyncAt: 1700000000,
  lastRemoteSummaryTs: 1700000000,
  blobFailureRate: 0,
  ...overrides,
});

const action = (overrides: Partial<PendingSyncAction> = {}): PendingSyncAction => ({
  id: 'a-1',
  kind: 'post-create',
  entityId: 'p-1',
  summary: '',
  createdAt: 1,
  ...overrides,
});

describe('NetworkBanner', () => {
  it('renders nothing when healthy and no pending actions', () => {
    const { container } = render(
      <NetworkBanner
        health={health({ level: 'healthy' })}
        pendingSyncActions={[]}
      />,
    );
    expect(container).toBeEmptyDOMElement();
  });

  it('renders when offline even without pending actions', () => {
    render(
      <NetworkBanner
        health={health({ level: 'offline', label: 'Offline', summary: 'You are offline' })}
        pendingSyncActions={[]}
      />,
    );
    expect(screen.getByText('Offline')).toBeInTheDocument();
    expect(screen.getByText('You are offline')).toBeInTheDocument();
  });

  it('renders when degraded', () => {
    render(
      <NetworkBanner
        health={health({ level: 'degraded', label: 'Working Offline', summary: 'No peers' })}
        pendingSyncActions={[]}
      />,
    );
    expect(screen.getByText('Working Offline')).toBeInTheDocument();
  });

  it('renders when syncing', () => {
    render(
      <NetworkBanner
        health={health({ level: 'syncing', label: 'Syncing', summary: 'Catching up' })}
        pendingSyncActions={[]}
      />,
    );
    expect(screen.getByText('Syncing')).toBeInTheDocument();
  });

  it('renders when healthy but pending actions exist', () => {
    render(
      <NetworkBanner
        health={health({ level: 'healthy', label: 'Up To Date' })}
        pendingSyncActions={[action(), action({ id: 'a-2' })]}
      />,
    );
    expect(screen.getByText('Up To Date')).toBeInTheDocument();
    expect(screen.getByText('2 saved')).toBeInTheDocument();
    expect(screen.getByText(/safely saved/)).toBeInTheDocument();
  });

  it('does not show the saved-count badge when level is non-healthy', () => {
    render(
      <NetworkBanner
        health={health({ level: 'degraded' })}
        pendingSyncActions={[action()]}
      />,
    );
    // Badge is gated on level === 'healthy'.
    expect(screen.queryByText('1 saved')).not.toBeInTheDocument();
  });

  it('clicking "Sync Now" calls onSyncNow', () => {
    const onSyncNow = vi.fn();
    render(
      <NetworkBanner
        health={health({ level: 'syncing', label: 'Syncing' })}
        pendingSyncActions={[]}
        onSyncNow={onSyncNow}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Sync Now' }));
    expect(onSyncNow).toHaveBeenCalledTimes(1);
  });

  it('disables "Sync Now" while busy and shows alternate label', () => {
    render(
      <NetworkBanner
        health={health({ level: 'syncing', label: 'Syncing' })}
        pendingSyncActions={[]}
        busy
      />,
    );
    const btn = screen.getByRole('button', { name: 'Syncing...' });
    expect(btn).toBeDisabled();
  });

  it('clicking "Network" calls onOpenNetworkSettings', () => {
    const onOpenNetworkSettings = vi.fn();
    render(
      <NetworkBanner
        health={health({ level: 'offline', label: 'Offline' })}
        pendingSyncActions={[]}
        onOpenNetworkSettings={onOpenNetworkSettings}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Network' }));
    expect(onOpenNetworkSettings).toHaveBeenCalledTimes(1);
  });

  it('renders the right material icon per level', () => {
    const cases: Array<{ level: NetworkHealthSnapshot['level']; icon: string }> = [
      { level: 'offline', icon: 'wifi_off' },
      { level: 'degraded', icon: 'sync_problem' },
      { level: 'syncing', icon: 'sync' },
    ];

    cases.forEach(({ level, icon }) => {
      const { unmount } = render(
        <NetworkBanner
          health={health({ level, label: level })}
          pendingSyncActions={[]}
        />,
      );
      expect(screen.getByText(icon)).toBeInTheDocument();
      unmount();
    });
  });
});
