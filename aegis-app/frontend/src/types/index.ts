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

export type SortMode = 'hot' | 'new';
export type Theme = 'light' | 'dark';
