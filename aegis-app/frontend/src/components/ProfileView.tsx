import { Post, ProfileDetails, Profile } from '../types';
import { PostCard } from './PostCard';

interface ProfileViewProps {
  profile: ProfileDetails | null;
  posts: Array<Post & { isFavorited?: boolean }>;
  profiles: Record<string, Profile>;
  isOwnProfile: boolean;
  onEditProfile: () => void;
  onPostClick: (post: Post) => void;
  onUpvote: (postId: string) => void;
  onShare?: (post: Post) => void;
  onToggleFavorite?: (postId: string) => void;
}

function getInitials(name: string): string {
  return name.slice(0, 2).toUpperCase();
}

function formatDate(timestamp: number): string {
  if (!timestamp) return 'Unknown';
  return new Date(timestamp * 1000).toLocaleDateString();
}

export function ProfileView({
  profile,
  posts,
  profiles,
  isOwnProfile,
  onEditProfile,
  onPostClick,
  onUpvote,
  onShare,
  onToggleFavorite,
}: ProfileViewProps) {
  const displayName = profile?.displayName || profile?.pubkey?.slice(0, 8) || 'User';
  const avatarURL = profile?.avatarURL || '';
  const bio = profile?.bio || '';

  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-6">
      <div className="max-w-5xl mx-auto space-y-6">
        <section className="rounded-3xl border border-warm-border dark:border-border-dark bg-warm-card dark:bg-surface-dark overflow-hidden">
          <div className="h-32 bg-[radial-gradient(circle_at_top_left,_rgba(230,120,60,0.35),_transparent_40%),linear-gradient(135deg,_rgba(255,220,200,0.95),_rgba(255,244,238,0.9))] dark:bg-[radial-gradient(circle_at_top_left,_rgba(230,120,60,0.4),_transparent_35%),linear-gradient(135deg,_rgba(39,33,29,0.95),_rgba(27,24,22,0.98))]" />
          <div className="px-6 pb-6">
            <div className="-mt-12 flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
              <div className="flex items-end gap-4">
                {avatarURL ? (
                  <img
                    src={avatarURL}
                    alt={displayName}
                    className="w-24 h-24 rounded-3xl object-cover border-4 border-white dark:border-surface-dark shadow-lg"
                  />
                ) : (
                  <div className="w-24 h-24 rounded-3xl bg-warm-accent text-white text-3xl font-bold border-4 border-white dark:border-surface-dark shadow-lg flex items-center justify-center">
                    {getInitials(displayName)}
                  </div>
                )}
                <div className="pb-1">
                  <h1 className="text-2xl md:text-3xl font-bold text-warm-text-primary dark:text-white">{displayName}</h1>
                  <p className="text-sm text-warm-text-secondary dark:text-slate-400 font-mono mt-1">
                    {profile?.pubkey || 'Unknown pubkey'}
                  </p>
                  <p className="text-sm text-warm-text-secondary dark:text-slate-400 mt-2">
                    Joined {formatDate(profile?.updatedAt || 0)}
                  </p>
                </div>
              </div>

              {isOwnProfile && (
                <button
                  onClick={onEditProfile}
                  className="px-4 py-2 rounded-xl bg-warm-accent hover:bg-warm-accent-hover text-white text-sm font-medium shadow-sm"
                >
                  Edit Profile
                </button>
              )}
            </div>

            <div className="mt-5 grid gap-6 lg:grid-cols-[minmax(0,1fr)_260px]">
              <div className="space-y-3">
                <div>
                  <h2 className="text-xs uppercase tracking-[0.18em] text-warm-text-secondary dark:text-slate-400">About</h2>
                  <p className="mt-2 text-sm leading-7 text-warm-text-secondary dark:text-slate-300">
                    {bio || (isOwnProfile ? 'Add a bio from settings to personalize your profile.' : 'No bio yet.')}
                  </p>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3 lg:grid-cols-1">
                <div className="rounded-2xl border border-warm-border dark:border-border-dark bg-white/70 dark:bg-surface-lighter p-4">
                  <div className="text-xs uppercase tracking-[0.16em] text-warm-text-secondary dark:text-slate-400">Posts</div>
                  <div className="mt-2 text-2xl font-bold text-warm-text-primary dark:text-white">{posts.length}</div>
                </div>
                <div className="rounded-2xl border border-warm-border dark:border-border-dark bg-white/70 dark:bg-surface-lighter p-4">
                  <div className="text-xs uppercase tracking-[0.16em] text-warm-text-secondary dark:text-slate-400">Identity</div>
                  <div className="mt-2 text-sm font-semibold text-warm-text-primary dark:text-white">
                    {isOwnProfile ? 'This is you' : 'Peer profile'}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section className="space-y-3">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold uppercase tracking-[0.16em] text-warm-text-secondary dark:text-slate-400">
              Recent Posts
            </h2>
            <span className="text-xs text-warm-text-secondary dark:text-slate-400">{posts.length} visible</span>
          </div>

          {posts.length === 0 ? (
            <div className="rounded-2xl border border-warm-border dark:border-border-dark bg-warm-card dark:bg-surface-dark py-12 text-center text-warm-text-secondary dark:text-slate-400">
              <span className="material-icons-outlined text-4xl mb-3">article</span>
              <p>No public posts yet.</p>
            </div>
          ) : (
            posts.map((post) => (
              <PostCard
                key={post.id}
                post={post}
                authorProfile={profiles[post.pubkey]}
                onUpvote={onUpvote}
                onClick={onPostClick}
                onShare={onShare}
                isFavorited={post.isFavorited}
                onToggleFavorite={onToggleFavorite}
              />
            ))
          )}
        </section>
      </div>
    </div>
  );
}
