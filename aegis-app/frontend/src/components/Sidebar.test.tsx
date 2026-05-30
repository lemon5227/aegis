import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { Sidebar } from './Sidebar';
import type { Sub } from '../types';

const sub = (overrides: Partial<Sub> = {}): Sub => ({
  id: 'general',
  title: 'General',
  description: '',
  createdAt: 1,
  ...overrides,
});

const defaultProps = {
  subs: [],
  subscribedSubs: [],
  currentSubId: 'recommended',
  onSelectSub: vi.fn(),
  onDiscoverClick: vi.fn(),
  onCreateSub: vi.fn(),
};

describe('Sidebar', () => {
  it('renders the brand and the recommended feed entry', () => {
    render(<Sidebar {...defaultProps} />);
    expect(screen.getByText('Aegis')).toBeInTheDocument();
    expect(screen.getByText('Recommended Feed')).toBeInTheDocument();
  });

  it('omits the subscriptions section when there are no subs', () => {
    render(<Sidebar {...defaultProps} subscribedSubs={[]} />);
    expect(screen.queryByText('My Subscriptions')).not.toBeInTheDocument();
  });

  it('lists each subscribed sub by id', () => {
    const subscribed = [sub({ id: 'rust' }), sub({ id: 'go' }), sub({ id: 'devops' })];
    render(<Sidebar {...defaultProps} subscribedSubs={subscribed} />);

    expect(screen.getByText('My Subscriptions')).toBeInTheDocument();
    expect(screen.getByText('rust')).toBeInTheDocument();
    expect(screen.getByText('go')).toBeInTheDocument();
    expect(screen.getByText('devops')).toBeInTheDocument();
  });

  it('clicking the recommended-feed button calls onSelectSub("recommended")', () => {
    const onSelectSub = vi.fn();
    render(<Sidebar {...defaultProps} onSelectSub={onSelectSub} />);

    fireEvent.click(screen.getByText('Recommended Feed'));
    expect(onSelectSub).toHaveBeenCalledWith('recommended');
  });

  it('clicking a sub button calls onSelectSub with that id', () => {
    const onSelectSub = vi.fn();
    render(
      <Sidebar
        {...defaultProps}
        onSelectSub={onSelectSub}
        subscribedSubs={[sub({ id: 'rust' })]}
      />,
    );

    fireEvent.click(screen.getByText('rust'));
    expect(onSelectSub).toHaveBeenCalledWith('rust');
  });

  it('clicking "Discover More" calls onDiscoverClick', () => {
    const onDiscoverClick = vi.fn();
    render(<Sidebar {...defaultProps} onDiscoverClick={onDiscoverClick} />);

    fireEvent.click(screen.getByText('Discover More'));
    expect(onDiscoverClick).toHaveBeenCalledTimes(1);
  });

  it('clicking "Create Sub" calls onCreateSub', () => {
    const onCreateSub = vi.fn();
    render(<Sidebar {...defaultProps} onCreateSub={onCreateSub} />);

    fireEvent.click(screen.getByText('Create Sub'));
    expect(onCreateSub).toHaveBeenCalledTimes(1);
  });

  it('renders an unread indicator for subs in unreadSubs (non-selected)', () => {
    const subscribed = [sub({ id: 'has-unread' }), sub({ id: 'no-unread' })];
    const unreadSubs = new Set(['has-unread']);

    const { container } = render(
      <Sidebar
        {...defaultProps}
        subscribedSubs={subscribed}
        unreadSubs={unreadSubs}
        currentSubId="recommended"
      />,
    );

    // The unread indicator is a small red dot — assert there's exactly one.
    const dots = container.querySelectorAll('.bg-red-500');
    expect(dots.length).toBe(1);
  });

  it('hides the unread dot for the currently-selected sub', () => {
    const { container } = render(
      <Sidebar
        {...defaultProps}
        subscribedSubs={[sub({ id: 'selected' })]}
        unreadSubs={new Set(['selected'])}
        currentSubId="selected"
      />,
    );

    const dots = container.querySelectorAll('.bg-red-500');
    expect(dots.length).toBe(0);
  });
});
