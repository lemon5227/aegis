import { CreatePostInput, PostComposerMode } from '../types';

const LINK_POST_PREFIX = 'Aegis-Link:';

export type ParsedPostContent = {
  mode: PostComposerMode;
  body: string;
  preview: string;
  linkURL: string;
  linkHostname: string;
};

function normalizeLineEndings(value: string): string {
  return value.replace(/\r\n/g, '\n');
}

function getHostname(value: string): string {
  try {
    return new URL(value).hostname.replace(/^www\./, '');
  } catch {
    return '';
  }
}

function truncate(value: string, maxLength: number): string {
  if (value.length <= maxLength) {
    return value;
  }
  return `${value.slice(0, Math.max(0, maxLength - 1)).trimEnd()}…`;
}

export function buildPostBody(input: Pick<CreatePostInput, 'mode' | 'body' | 'linkURL'>): string {
  const normalizedBody = normalizeLineEndings(input.body.trim());
  if (input.mode !== 'link') {
    return normalizedBody;
  }

  const normalizedURL = (input.linkURL || '').trim();
  if (!normalizedURL) {
    return normalizedBody;
  }
  if (!normalizedBody) {
    return `${LINK_POST_PREFIX} ${normalizedURL}`;
  }
  return `${LINK_POST_PREFIX} ${normalizedURL}\n\n${normalizedBody}`;
}

export function parsePostContent(rawBody: string, fallbackPreview = ''): ParsedPostContent {
  const normalizedBody = normalizeLineEndings(rawBody || '').trim();
  const fallback = normalizeLineEndings(fallbackPreview || '').trim();
  const lines = normalizedBody.split('\n');
  const firstLine = lines[0]?.trim() || '';

  if (!firstLine.startsWith(LINK_POST_PREFIX)) {
    const preview = fallback || truncate(normalizedBody, 140);
    return {
      mode: 'text',
      body: normalizedBody,
      preview,
      linkURL: '',
      linkHostname: '',
    };
  }

  const linkURL = firstLine.slice(LINK_POST_PREFIX.length).trim();
  const commentary = lines.slice(1).join('\n').trim();
  const preview = commentary || fallback.replace(firstLine, '').trim() || getHostname(linkURL) || linkURL;

  return {
    mode: 'link',
    body: commentary,
    preview: truncate(preview, 140),
    linkURL,
    linkHostname: getHostname(linkURL),
  };
}

export function isValidExternalURL(value: string): boolean {
  try {
    const parsed = new URL(value);
    return parsed.protocol === 'http:' || parsed.protocol === 'https:';
  } catch {
    return false;
  }
}
