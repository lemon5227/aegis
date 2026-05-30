import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { Header } from './Header';
import type { Profile } from '../types';

vi.mock('../../wailsjs/runtime/runtime', () => ({
  BrowserOpenURL: vi.fn(),
}));

const noopHandlers = () => ({
  onCreatePost: vi.fn(),
  onProfileClick: vi.fn(),
  onViewOwnProfile: vi.fn(),
  onMyPostsClick: vi.fn(),
  onDraftsClick: vi.fn(),
  onHistoryClick: vi.fn(),
  onPendingSyncClick: vi.fn(),
  onFavoritesClick: vi.fn(),
  onSignOut: vi.fn(),
  onThemeToggle: vi.fn(),
  onSearch: vi.fn(),
  onSearchResultClick: vi.fn(),
  onSearchClear: vi.fn(),
  onNotificationsClick: vi.fn(),
});

const baseProps = (overrides: Record<string, unknown> = {}) => ({
  currentSubId: 'general',
  isDark: false,
  searchQuery: '',
  searchResults: null,
  unreadNotificationCount: 0,
  ...noopHandlers(),
  ...overrides,
});

const profile = (overrides: Partial<Profile> = {}): Profile => ({
  pubkey: 'pk',
  displayName: 'Alice',
  avatarURL: '',
  updatedAt: 1,
  ...overrides,
});

describe('Header', () => {
  it('renders the search input', () => {
    render(<Header {...baseProps()} />);
    expect(screen.getByPlaceholderText('Search...')).toBeInTheDocument();
  });

  it('search placeholder reflects sub-scope when toggled to sub', () => {
    // Default scope is 'global' — placeholder says 'Search...'.
    // Toggling scope to a sub changes the placeholder.
    render(<Header {...baseProps({ currentSubId: 'rust' })} />);

    // Find the scope toggle button — it's the leftmost button in the search bar.
    const scopeToggle = screen.getAllByRole('button').find((b) =>
      b.className.includes('text-xs') || (b.textContent || '').toLowerCase().includes('rust') || (b.textContent || '').toLowerCase().includes('global'),
    );
    expect(scopeToggle).toBeDefined();
  });

  it('typing in the search input fires onSearch', () => {
    const handlers = noopHandlers();
    render(<Header {...baseProps({ ...handlers })} />);
    const input = screen.getByPlaceholderText('Search...') as HTMLInputElement;
    fireEvent.change(input, { target: { value: 'hello' } });
    expect(handlers.onSearch).toHaveBeenCalled();
    expect(handlers.onSearch.mock.calls[0][0]).toBe('hello');
  });

  it('clear (×) button appears when searchQuery is non-empty and calls onSearchClear', () => {
    const handlers = noopHandlers();
    const { rerender } = render(<Header {...baseProps({ ...handlers })} />);

    // No clear button on empty query — the × icon button is absent.
    const initialButtons = screen.queryAllByText('close');
    const initialCloseCount = initialButtons.length;

    rerender(<Header {...baseProps({ ...handlers, searchQuery: 'something' })} />);
    const updatedButtons = screen.queryAllByText('close');
    expect(updatedButtons.length).toBeGreaterThan(initialCloseCount);

    fireEvent.click(updatedButtons[updatedButtons.length - 1]);
    expect(handlers.onSearchClear).toHaveBeenCalledTimes(1);
  });

  it('renders notifications icon', () => {
    render(<Header {...baseProps()} />);
    expect(screen.getByText('notifications')).toBeInTheDocument();
  });

  it('clicking the notifications button calls onNotificationsClick', () => {
    const handlers = noopHandlers();
    render(<Header {...baseProps(handlers)} />);

    const notifIcon = screen.getByText('notifications');
    const notifButton = notifIcon.closest('button')!;
    fireEvent.click(notifButton);
    expect(handlers.onNotificationsClick).toHaveBeenCalledTimes(1);
  });

  it('does not render unread badge when count is 0', () => {
    render(<Header {...baseProps({ unreadNotificationCount: 0 })} />);
    expect(screen.queryByText('1')).not.toBeInTheDocument();
  });

  it('renders unread count badge when > 0', () => {
    render(<Header {...baseProps({ unreadNotificationCount: 7 })} />);
    expect(screen.getByText('7')).toBeInTheDocument();
  });

  it('renders "99+" badge when count exceeds 99', () => {
    render(<Header {...baseProps({ unreadNotificationCount: 250 })} />);
    expect(screen.getByText('99+')).toBeInTheDocument();
  });

  it('user-menu button opens the menu on click', () => {
    render(<Header {...baseProps({ profile: profile({ displayName: 'Alice' }) })} />);

    // Menu items hidden initially.
    expect(screen.queryByText('Drafts')).not.toBeInTheDocument();

    // The avatar fallback div renders 'AL' (Alice's initials). It is unique
    // in the rendered Header, so we can navigate to its enclosing button.
    const avatar = screen.getByText('AL');
    const trigger = avatar.closest('button');
    expect(trigger).toBeTruthy();
    fireEvent.click(trigger!);

    // Some menu item should now be visible.
    expect(screen.getByText('Drafts')).toBeInTheDocument();
    expect(screen.getByText('Favorites')).toBeInTheDocument();
    expect(screen.getByText('History')).toBeInTheDocument();
  });

  it('shows the avatar image when profile has avatarURL', () => {
    render(
      <Header
        {...baseProps({ profile: profile({ displayName: 'Alice', avatarURL: 'https://x/a.png' }) })}
      />,
    );
    const imgs = screen.getAllByAltText('Alice') as HTMLImageElement[];
    expect(imgs.length).toBeGreaterThanOrEqual(1);
    expect(imgs[0].src).toBe('https://x/a.png');
  });

  it('falls back to "?" placeholder when no profile', () => {
    render(<Header {...baseProps()} />);
    expect(screen.getByText('?')).toBeInTheDocument();
  });

  it('theme toggle icon changes between light_mode and dark_mode based on isDark', () => {
    const { rerender } = render(<Header {...baseProps({ isDark: false })} />);
    // Light mode -> show dark_mode icon (so user can switch TO dark)
    expect(screen.getByText('dark_mode')).toBeInTheDocument();

    rerender(<Header {...baseProps({ isDark: true })} />);
    // Dark mode -> show light_mode icon (so user can switch TO light)
    expect(screen.getByText('light_mode')).toBeInTheDocument();
  });

  it('clicking the theme toggle calls onThemeToggle', () => {
    const handlers = noopHandlers();
    render(<Header {...baseProps(handlers)} />);
    const themeIcon = screen.getByText('dark_mode');
    fireEvent.click(themeIcon.closest('button')!);
    expect(handlers.onThemeToggle).toHaveBeenCalledTimes(1);
  });
});
