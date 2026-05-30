import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ProfileView } from './ProfileView';
import type { Post, ProfileDetails } from '../types';

vi.mock('../../wailsjs/runtime/runtime', () => ({
  BrowserOpenURL: vi.fn(),
}));

const profile = (overrides: Partial<ProfileDetails> = {}): ProfileDetails => ({
  pubkey: 'deadbeef1234567890',
  displayName: 'Alice',
  avatarURL: '',
  bio: '',
  updatedAt: 1700000000,
  ...overrides,
});

const post = (overrides: Partial<Post> = {}): Post => ({
  id: 'p-1',
  pubkey: 'deadbeef1234567890',
  title: 'Profile Post',
  bodyPreview: 'body',
  contentCid: '',
  imageCid: '',
  thumbCid: '',
  imageMime: '',
  imageSize: 0,
  imageWidth: 0,
  imageHeight: 0,
  score: 0,
  timestamp: Math.floor(Date.now() / 1000),
  zone: 'public',
  subId: 'general',
  visibility: 'normal',
  isPinned: false,
  pinnedAt: 0,
  isLocked: false,
  lockedAt: 0,
  ...overrides,
});

describe('ProfileView', () => {
  it('renders the display name and pubkey', () => {
    render(
      <ProfileView
        profile={profile({ displayName: 'Alice', pubkey: 'pk-12345' })}
        posts={[]}
        profiles={{}}
        isOwnProfile={false}
        onEditProfile={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );
    expect(screen.getByText('Alice')).toBeInTheDocument();
    expect(screen.getByText('pk-12345')).toBeInTheDocument();
  });

  it('falls back to truncated pubkey when displayName is empty', () => {
    render(
      <ProfileView
        profile={profile({ displayName: '', pubkey: 'abcdef0123456789' })}
        posts={[]}
        profiles={{}}
        isOwnProfile={false}
        onEditProfile={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );
    expect(screen.getByText('abcdef01')).toBeInTheDocument();
  });

  it('falls back to "User" when no profile is provided', () => {
    render(
      <ProfileView
        profile={null}
        posts={[]}
        profiles={{}}
        isOwnProfile={false}
        onEditProfile={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );
    expect(screen.getByText('User')).toBeInTheDocument();
    expect(screen.getByText('Unknown pubkey')).toBeInTheDocument();
  });

  it('renders the avatar image when avatarURL is present', () => {
    render(
      <ProfileView
        profile={profile({ displayName: 'Alice', avatarURL: 'https://x/a.png' })}
        posts={[]}
        profiles={{}}
        isOwnProfile={false}
        onEditProfile={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );
    const img = screen.getByAltText('Alice') as HTMLImageElement;
    expect(img.src).toBe('https://x/a.png');
  });

  it('shows the bio when set', () => {
    render(
      <ProfileView
        profile={profile({ bio: 'I build things.' })}
        posts={[]}
        profiles={{}}
        isOwnProfile={false}
        onEditProfile={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );
    expect(screen.getByText('I build things.')).toBeInTheDocument();
  });

  it('shows "Add a bio" hint for own profile when bio is empty', () => {
    render(
      <ProfileView
        profile={profile({ bio: '' })}
        posts={[]}
        profiles={{}}
        isOwnProfile={true}
        onEditProfile={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );
    expect(screen.getByText(/Add a bio from settings/)).toBeInTheDocument();
  });

  it('shows "No bio yet." for others when bio is empty', () => {
    render(
      <ProfileView
        profile={profile({ bio: '' })}
        posts={[]}
        profiles={{}}
        isOwnProfile={false}
        onEditProfile={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );
    expect(screen.getByText('No bio yet.')).toBeInTheDocument();
  });

  it('renders Edit Profile button only for own profile', () => {
    const { rerender } = render(
      <ProfileView
        profile={profile()}
        posts={[]}
        profiles={{}}
        isOwnProfile={false}
        onEditProfile={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );
    expect(screen.queryByRole('button', { name: 'Edit Profile' })).not.toBeInTheDocument();

    rerender(
      <ProfileView
        profile={profile()}
        posts={[]}
        profiles={{}}
        isOwnProfile={true}
        onEditProfile={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );
    expect(screen.getByRole('button', { name: 'Edit Profile' })).toBeInTheDocument();
  });

  it('clicking Edit Profile fires onEditProfile', () => {
    const onEditProfile = vi.fn();
    render(
      <ProfileView
        profile={profile()}
        posts={[]}
        profiles={{}}
        isOwnProfile={true}
        onEditProfile={onEditProfile}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Edit Profile' }));
    expect(onEditProfile).toHaveBeenCalledTimes(1);
  });

  it('shows "This is you" stat for own profile', () => {
    render(
      <ProfileView
        profile={profile()}
        posts={[]}
        profiles={{}}
        isOwnProfile={true}
        onEditProfile={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );
    expect(screen.getByText('This is you')).toBeInTheDocument();
  });

  it('shows "Peer profile" stat for someone else', () => {
    render(
      <ProfileView
        profile={profile()}
        posts={[]}
        profiles={{}}
        isOwnProfile={false}
        onEditProfile={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );
    expect(screen.getByText('Peer profile')).toBeInTheDocument();
  });

  it('shows post count and renders empty state when no posts', () => {
    render(
      <ProfileView
        profile={profile()}
        posts={[]}
        profiles={{}}
        isOwnProfile={false}
        onEditProfile={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );
    expect(screen.getByText('0 visible')).toBeInTheDocument();
    expect(screen.getByText('No public posts yet.')).toBeInTheDocument();
  });

  it('renders one PostCard per post and reflects count', () => {
    render(
      <ProfileView
        profile={profile()}
        posts={[
          post({ id: 'p1', title: 'First Post' }),
          post({ id: 'p2', title: 'Second Post' }),
          post({ id: 'p3', title: 'Third Post' }),
        ]}
        profiles={{}}
        isOwnProfile={false}
        onEditProfile={vi.fn()}
        onPostClick={vi.fn()}
        onUpvote={vi.fn()}
      />,
    );

    expect(screen.getByText('First Post')).toBeInTheDocument();
    expect(screen.getByText('Second Post')).toBeInTheDocument();
    expect(screen.getByText('Third Post')).toBeInTheDocument();
    expect(screen.getByText('3 visible')).toBeInTheDocument();
  });
});
