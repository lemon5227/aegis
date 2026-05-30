import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { DiscoverView } from './DiscoverView';
import type { Sub } from '../types';

const sub = (overrides: Partial<Sub> = {}): Sub => ({
  id: 'rust',
  title: 'Rust Community',
  description: 'Talk about Rust',
  createdAt: 1,
  ...overrides,
});

describe('DiscoverView', () => {
  it('renders the page title', () => {
    render(
      <DiscoverView
        subs={[]}
        subscribedSubIds={new Set()}
        onSubClick={vi.fn()}
        onToggleSubscription={vi.fn()}
      />,
    );
    expect(screen.getByText('Discover All Sub-communities')).toBeInTheDocument();
  });

  it('renders empty state when no subs', () => {
    render(
      <DiscoverView
        subs={[]}
        subscribedSubIds={new Set()}
        onSubClick={vi.fn()}
        onToggleSubscription={vi.fn()}
      />,
    );
    expect(screen.getByText(/No sub-communities found/)).toBeInTheDocument();
  });

  it('renders one card per sub with id, title, description', () => {
    render(
      <DiscoverView
        subs={[
          sub({ id: 'rust', title: 'Rust', description: 'rusty' }),
          sub({ id: 'go', title: 'Go', description: 'goey' }),
        ]}
        subscribedSubIds={new Set()}
        onSubClick={vi.fn()}
        onToggleSubscription={vi.fn()}
      />,
    );
    expect(screen.getByText('rust')).toBeInTheDocument();
    expect(screen.getByText('Rust')).toBeInTheDocument();
    expect(screen.getByText('rusty')).toBeInTheDocument();
    expect(screen.getByText('go')).toBeInTheDocument();
  });

  it('shows "No description" when title is empty', () => {
    render(
      <DiscoverView
        subs={[sub({ id: 'empty', title: '' })]}
        subscribedSubIds={new Set()}
        onSubClick={vi.fn()}
        onToggleSubscription={vi.fn()}
      />,
    );
    expect(screen.getByText('No description')).toBeInTheDocument();
  });

  it('clicking the card body fires onSubClick(subId)', () => {
    const onSubClick = vi.fn();
    render(
      <DiscoverView
        subs={[sub({ id: 'click-me' })]}
        subscribedSubIds={new Set()}
        onSubClick={onSubClick}
        onToggleSubscription={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByText('click-me'));
    expect(onSubClick).toHaveBeenCalledWith('click-me');
  });

  it('clicking the Subscribe button fires onToggleSubscription(subId) and does NOT bubble to onSubClick', () => {
    const onSubClick = vi.fn();
    const onToggleSubscription = vi.fn();
    render(
      <DiscoverView
        subs={[sub({ id: 'sub-x' })]}
        subscribedSubIds={new Set()}
        onSubClick={onSubClick}
        onToggleSubscription={onToggleSubscription}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Subscribe' }));
    expect(onToggleSubscription).toHaveBeenCalledWith('sub-x');
    expect(onSubClick).not.toHaveBeenCalled();
  });

  it('shows "Subscribed" state when sub is in subscribedSubIds', () => {
    render(
      <DiscoverView
        subs={[sub({ id: 'rust' })]}
        subscribedSubIds={new Set(['rust'])}
        onSubClick={vi.fn()}
        onToggleSubscription={vi.fn()}
      />,
    );
    // The 'Subscribed' badge text appears twice — once as a pill, once on the
    // button. Use getAllByText to assert at least one rendering.
    const subscribedLabels = screen.getAllByText('Subscribed');
    expect(subscribedLabels.length).toBeGreaterThanOrEqual(1);
  });

  it('clicking the "Subscribed" button toggles off the subscription', () => {
    const onToggleSubscription = vi.fn();
    render(
      <DiscoverView
        subs={[sub({ id: 'rust' })]}
        subscribedSubIds={new Set(['rust'])}
        onSubClick={vi.fn()}
        onToggleSubscription={onToggleSubscription}
      />,
    );
    // The button label includes a check icon plus text. Find by role + accessible name.
    const buttons = screen.getAllByRole('button');
    const subscribedBtn = buttons.find((b) => b.textContent?.includes('Subscribed'));
    expect(subscribedBtn).toBeDefined();
    fireEvent.click(subscribedBtn!);
    expect(onToggleSubscription).toHaveBeenCalledWith('rust');
  });
});
