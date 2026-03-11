import { useEffect, useState } from 'react';
import { DraftSummary } from '../types';
import { listDrafts, removeDraft } from '../lib/drafts';

interface DraftsViewProps {
  authorPublicKey?: string;
  refreshToken?: number;
  onOpenPostDraft: (subId: string) => void;
  onOpenCommentDraft: (postId: string) => void;
}

function formatUpdatedAt(timestamp: number): string {
  if (!timestamp) {
    return 'Unknown update time';
  }
  return new Date(timestamp).toLocaleString();
}

function getDraftTitle(draft: DraftSummary): string {
  if (draft.kind === 'post') {
    return draft.title.trim() || 'Untitled post draft';
  }
  return draft.replyToId ? 'Reply draft' : 'Comment draft';
}

function getDraftMeta(draft: DraftSummary): string {
  if (draft.kind === 'post') {
    return `r/${draft.subId} · ${draft.mode === 'link' ? 'Link post' : 'Text post'}`;
  }
  return `Post ${draft.postId.slice(0, 8)}${draft.replyToId ? ` · replying to ${draft.replyToId.slice(0, 8)}` : ''}`;
}

export function DraftsView({ authorPublicKey, refreshToken = 0, onOpenPostDraft, onOpenCommentDraft }: DraftsViewProps) {
  const [drafts, setDrafts] = useState<DraftSummary[]>([]);

  useEffect(() => {
    setDrafts(listDrafts(authorPublicKey));
  }, [authorPublicKey, refreshToken]);

  const handleDeleteDraft = (storageKey: string) => {
    removeDraft(storageKey);
    setDrafts((current) => current.filter((draft) => draft.storageKey !== storageKey));
  };

  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-6">
      <div className="mx-auto max-w-4xl space-y-6">
        <section className="rounded-3xl border border-warm-border dark:border-border-dark bg-warm-card dark:bg-surface-dark p-6">
          <p className="text-xs uppercase tracking-[0.18em] text-warm-text-secondary dark:text-slate-400">Drafts</p>
          <h1 className="mt-2 text-3xl font-bold text-warm-text-primary dark:text-white">Saved Drafts</h1>
          <p className="mt-3 text-sm text-warm-text-secondary dark:text-slate-300">
            Local drafts are autosaved per identity. Open one to continue writing, or delete it to clean up unfinished work.
          </p>
        </section>

        {drafts.length === 0 ? (
          <div className="rounded-3xl border border-warm-border dark:border-border-dark bg-warm-card dark:bg-surface-dark py-16 text-center text-warm-text-secondary dark:text-slate-400">
            <span className="material-icons-outlined text-4xl mb-3">drafts</span>
            <p>No drafts saved yet.</p>
          </div>
        ) : (
          <div className="space-y-4">
            {drafts.map((draft) => (
              <article
                key={draft.storageKey}
                className="rounded-2xl border border-warm-border dark:border-border-dark bg-warm-card dark:bg-surface-dark p-5"
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="min-w-0">
                    <div className="text-xs uppercase tracking-[0.16em] text-warm-text-secondary dark:text-slate-400">
                      {draft.kind === 'post' ? 'Post Draft' : 'Comment Draft'}
                    </div>
                    <h2 className="mt-1 text-lg font-semibold text-warm-text-primary dark:text-white">
                      {getDraftTitle(draft)}
                    </h2>
                    <div className="mt-1 text-sm text-warm-text-secondary dark:text-slate-400">
                      {getDraftMeta(draft)}
                    </div>
                    <div className="mt-3 line-clamp-3 whitespace-pre-wrap break-words text-sm text-warm-text-secondary dark:text-slate-300">
                      {draft.kind === 'post'
                        ? draft.body || draft.linkURL || draft.externalImageURL || 'No body content yet.'
                        : draft.body || 'No comment content yet.'}
                    </div>
                    <div className="mt-3 text-xs text-warm-text-secondary dark:text-slate-400">
                      Updated {formatUpdatedAt(draft.updatedAt)}
                    </div>
                  </div>

                  <div className="flex shrink-0 items-center gap-2">
                    <button
                      onClick={() => {
                        if (draft.kind === 'post') {
                          onOpenPostDraft(draft.subId);
                          return;
                        }
                        onOpenCommentDraft(draft.postId);
                      }}
                      className="rounded-lg bg-warm-accent px-4 py-2 text-sm font-medium text-white hover:bg-warm-accent-hover"
                    >
                      Resume
                    </button>
                    <button
                      onClick={() => handleDeleteDraft(draft.storageKey)}
                      className="rounded-lg border border-warm-border dark:border-border-dark px-4 py-2 text-sm font-medium text-warm-text-secondary dark:text-slate-300 hover:bg-warm-bg dark:hover:bg-background-dark"
                    >
                      Delete
                    </button>
                  </div>
                </div>
              </article>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
