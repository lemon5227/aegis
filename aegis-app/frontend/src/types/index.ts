export interface Sub {
  id: string;
  title: string;
  description: string;
  createdAt: number;
}

export interface PostIndex {
  id: string;
  pubkey: string;
  title: string;
  bodyPreview: string;
  contentCid: string;
  imageCid: string;
  thumbCid: string;
  imageMime: string;
  imageSize: number;
  imageWidth: number;
  imageHeight: number;
  score: number;
  timestamp: number;
  zone: string;
  subId: string;
  visibility: string;
}

export interface Profile {
  pubkey: string;
  displayName: string;
  avatarURL: string;
  bio?: string;
  updatedAt: number;
}

export interface ProfileDetails extends Profile {
  bio: string;
}

export interface Identity {
  mnemonic: string;
  publicKey: string;
}

export interface GovernanceAdmin {
  adminPubkey: string;
  role: string;
  active: boolean;
}

export interface Comment {
  id: string;
  postId: string;
  parentId: string;
  pubkey: string;
  body: string;
  attachments?: CommentAttachment[];
  score: number;
  timestamp: number;
}

export interface CommentAttachment {
  kind: string;
  ref: string;
  mime?: string;
  width?: number;
  height?: number;
  sizeBytes?: number;
}

export interface ModerationLog {
  id: number;
  targetPubkey: string;
  action: string;
  sourceAdmin: string;
  timestamp: number;
  reason: string;
  result: string;
}

export interface ModerationState {
  targetPubkey: string;
  action: string;
  sourceAdmin: string;
  timestamp: number;
  reason: string;
}

export interface AntiEntropyStats {
  syncRequestsSent: number;
  syncRequestsReceived: number;
  syncResponsesReceived: number;
  syncSummariesReceived: number;
  indexInsertions: number;
  blobFetchAttempts: number;
  blobFetchSuccess: number;
  blobFetchFailures: number;
  lastSyncAt: number;
  lastRemoteSummaryTs: number;
  lastObservedSyncLagSec: number;
}

export interface P2PStatus {
  started: boolean;
  peerId: string;
  listenAddrs: string[];
  announceAddrs: string[];
  connectedPeers: string[];
  topic: string;
}

export interface EntityOpRecord {
  opId: string;
  entityType: string;
  entityId: string;
  opType: string;
  authorPubkey: string;
  lamport: number;
  timestamp: number;
  schemaVersion: number;
  authScope: string;
  payloadJson: string;
}

export interface TombstoneGCResult {
  scannedPosts: number;
  deletedPosts: number;
  scannedComments: number;
  deletedComments: number;
}

export interface ForumMessage {
  id: string;
  pubkey: string;
  title: string;
  body: string;
  contentCid: string;
  imageCid: string;
  thumbCid: string;
  imageMime: string;
  imageSize: number;
  imageWidth: number;
  imageHeight: number;
  content: string;
  score: number;
  timestamp: number;
  sizeBytes: number;
  zone: string;
  subId: string;
  isProtected: number;
  visibility: string;
}

export interface Post extends PostIndex {
  authorProfile?: Profile;
}

export type PostComposerMode = 'text' | 'link';

export interface CreatePostInput {
  title: string;
  body: string;
  imageBase64?: string;
  imageMime?: string;
  externalImageURL?: string;
  mode: PostComposerMode;
  linkURL?: string;
}

export interface PostDraftSummary {
  kind: 'post';
  storageKey: string;
  subId: string;
  authorPublicKey: string;
  title: string;
  body: string;
  linkURL: string;
  externalImageURL: string;
  mode: PostComposerMode;
  updatedAt: number;
}

export interface CommentDraftSummary {
  kind: 'comment';
  storageKey: string;
  postId: string;
  authorPublicKey: string;
  body: string;
  replyToId: string | null;
  updatedAt: number;
}

export type DraftSummary = PostDraftSummary | CommentDraftSummary;

export interface RecentlyViewedEntry {
  postId: string;
  viewedAt: number;
}

export type PendingSyncActionKind =
  | 'post-create'
  | 'post-edit'
  | 'post-delete'
  | 'comment-create'
  | 'comment-edit'
  | 'comment-delete'
  | 'post-vote'
  | 'comment-vote'
  | 'profile-publish';

export interface PendingSyncAction {
  id: string;
  kind: PendingSyncActionKind;
  entityId: string;
  summary: string;
  createdAt: number;
}

export type SortMode = 'hot' | 'new' | 'top-day' | 'top-week' | 'top-month' | 'top-all';
export type Theme = 'light' | 'dark';

export interface FeedStreamItem {
  post: ForumMessage;
  reason: string;
  isSubscribed: boolean;
  recommendationScore: number;
}

export interface FeedStream {
  items: FeedStreamItem[];
  algorithm: string;
  generatedAt: number;
}

export interface SubStats {
  subId: string;
  subscriberCount: number;
  postCount: number;
  activeAuthors: number;
  recentPosts24h: number;
  createdAt: number;
}

export interface Notification {
  id: string;
  type: string;
  sourcePubkey: string;
  targetEntityId: string;
  targetType: string;
  postId: string;
  isRead: boolean;
  createdAt: number;
}

export interface NotificationPage {
  items: Notification[];
  nextCursor: string;
}
