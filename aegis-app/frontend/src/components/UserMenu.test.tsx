import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { UserMenu } from './UserMenu';
import type { Profile } from '../types';

const profile = (overrides: Partial<Profile> = {}): Profile => ({
  pubkey: 'deadbeef',
  displayName: 'Alice',
  avatarURL: '',
  updatedAt: 1,
  ...overrides,
});

const noopHandlers = () => ({
  onProfileClick: vi.fn(),
  onMyPostsClick: vi.fn(),
  onFavoritesClick: vi.fn(),
  onSettingsClick: vi.fn(),
  onGovernanceClick: vi.fn(),
  onSignOut: vi.fn(),
});

describe('UserMenu', () => {
  it('renders the trigger button with display name', () => {
    render(
      <UserMenu profile={profile({ displayName: 'Alice' })} isAdmin={false} {...noopHandlers()} />,
    );
    expect(screen.getByText('Alice')).toBeInTheDocument();
  });

  it('falls back to "User" when no profile provided', () => {
    render(<UserMenu isAdmin={false} {...noopHandlers()} />);
    expect(screen.getByText('User')).toBeInTheDocument();
  });

  it('does NOT render menu items before the trigger is clicked', () => {
    render(
      <UserMenu profile={profile()} isAdmin={false} {...noopHandlers()} />,
    );
    expect(screen.queryByText('Profile')).not.toBeInTheDocument();
    expect(screen.queryByText('My Posts')).not.toBeInTheDocument();
  });

  it('opens the menu on trigger click and shows account items', () => {
    render(
      <UserMenu profile={profile()} isAdmin={false} {...noopHandlers()} />,
    );
    fireEvent.click(screen.getByText('Alice'));
    expect(screen.getByText('Profile')).toBeInTheDocument();
    expect(screen.getByText('My Posts')).toBeInTheDocument();
    expect(screen.getByText('My Favorites')).toBeInTheDocument();
    expect(screen.getByText('Settings')).toBeInTheDocument();
    expect(screen.getByText('Log Out')).toBeInTheDocument();
  });

  it('hides the Governance Panel for non-admin users', () => {
    render(<UserMenu profile={profile()} isAdmin={false} {...noopHandlers()} />);
    fireEvent.click(screen.getByText('Alice'));
    expect(screen.queryByText('Governance Panel')).not.toBeInTheDocument();
  });

  it('shows the Governance Panel for admin users', () => {
    render(<UserMenu profile={profile()} isAdmin={true} {...noopHandlers()} />);
    fireEvent.click(screen.getByText('Alice'));
    expect(screen.getByText('Governance Panel')).toBeInTheDocument();
  });

  it('clicking Profile fires onProfileClick and closes menu', () => {
    const handlers = noopHandlers();
    render(<UserMenu profile={profile()} isAdmin={false} {...handlers} />);
    fireEvent.click(screen.getByText('Alice'));
    fireEvent.click(screen.getByText('Profile'));
    expect(handlers.onProfileClick).toHaveBeenCalledTimes(1);
    // Menu should close after action.
    expect(screen.queryByText('Profile')).not.toBeInTheDocument();
  });

  it('clicking My Posts fires onMyPostsClick', () => {
    const handlers = noopHandlers();
    render(<UserMenu profile={profile()} isAdmin={false} {...handlers} />);
    fireEvent.click(screen.getByText('Alice'));
    fireEvent.click(screen.getByText('My Posts'));
    expect(handlers.onMyPostsClick).toHaveBeenCalledTimes(1);
  });

  it('clicking My Favorites fires onFavoritesClick', () => {
    const handlers = noopHandlers();
    render(<UserMenu profile={profile()} isAdmin={false} {...handlers} />);
    fireEvent.click(screen.getByText('Alice'));
    fireEvent.click(screen.getByText('My Favorites'));
    expect(handlers.onFavoritesClick).toHaveBeenCalledTimes(1);
  });

  it('clicking Settings fires onSettingsClick', () => {
    const handlers = noopHandlers();
    render(<UserMenu profile={profile()} isAdmin={false} {...handlers} />);
    fireEvent.click(screen.getByText('Alice'));
    fireEvent.click(screen.getByText('Settings'));
    expect(handlers.onSettingsClick).toHaveBeenCalledTimes(1);
  });

  it('clicking Governance Panel fires onGovernanceClick (admin only)', () => {
    const handlers = noopHandlers();
    render(<UserMenu profile={profile()} isAdmin={true} {...handlers} />);
    fireEvent.click(screen.getByText('Alice'));
    fireEvent.click(screen.getByText('Governance Panel'));
    expect(handlers.onGovernanceClick).toHaveBeenCalledTimes(1);
  });

  it('clicking Log Out fires onSignOut', () => {
    const handlers = noopHandlers();
    render(<UserMenu profile={profile()} isAdmin={false} {...handlers} />);
    fireEvent.click(screen.getByText('Alice'));
    fireEvent.click(screen.getByText('Log Out'));
    expect(handlers.onSignOut).toHaveBeenCalledTimes(1);
  });

  it('renders avatar image when avatarURL is present', () => {
    render(
      <UserMenu
        profile={profile({ displayName: 'Bob', avatarURL: 'https://x/avatar.png' })}
        isAdmin={false}
        {...noopHandlers()}
      />,
    );
    const img = screen.getByAltText('Bob') as HTMLImageElement;
    expect(img.src).toBe('https://x/avatar.png');
  });

  it('clicking outside the menu closes it', () => {
    render(
      <div>
        <UserMenu profile={profile()} isAdmin={false} {...noopHandlers()} />
        <div data-testid="outside">Outside the menu</div>
      </div>,
    );

    fireEvent.click(screen.getByText('Alice'));
    expect(screen.getByText('Profile')).toBeInTheDocument();

    // Click outside (mousedown is what the listener uses).
    fireEvent.mouseDown(screen.getByTestId('outside'));
    expect(screen.queryByText('Profile')).not.toBeInTheDocument();
  });

  it('clicking the trigger toggles the menu open and closed', () => {
    render(
      <UserMenu profile={profile()} isAdmin={false} {...noopHandlers()} />,
    );
    const trigger = screen.getByText('Alice');

    fireEvent.click(trigger);
    expect(screen.getByText('Profile')).toBeInTheDocument();

    fireEvent.click(trigger);
    expect(screen.queryByText('Profile')).not.toBeInTheDocument();
  });
});
