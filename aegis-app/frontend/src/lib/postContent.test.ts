import { describe, expect, it } from 'vitest';
import { buildPostBody, isValidExternalURL, parsePostContent } from './postContent';

describe('isValidExternalURL', () => {
  it('accepts http URLs', () => {
    expect(isValidExternalURL('http://example.com/x.png')).toBe(true);
  });

  it('accepts https URLs', () => {
    expect(isValidExternalURL('https://example.com/x.png')).toBe(true);
  });

  it('rejects javascript:', () => {
    expect(isValidExternalURL('javascript:alert(1)')).toBe(false);
  });

  it('rejects data: URIs', () => {
    expect(isValidExternalURL('data:image/png;base64,xyz')).toBe(false);
  });

  it('rejects ftp', () => {
    expect(isValidExternalURL('ftp://example.com/file.txt')).toBe(false);
  });

  it('rejects unparseable strings', () => {
    expect(isValidExternalURL('not a url')).toBe(false);
    expect(isValidExternalURL('')).toBe(false);
  });
});

describe('buildPostBody (text mode)', () => {
  it('trims and normalizes line endings for plain text', () => {
    const got = buildPostBody({ mode: 'text', body: '  hello\r\nworld  ', linkURL: '' });
    expect(got).toBe('hello\nworld');
  });

  it('returns empty string for whitespace-only body', () => {
    const got = buildPostBody({ mode: 'text', body: '   ', linkURL: '' });
    expect(got).toBe('');
  });
});

describe('buildPostBody (link mode)', () => {
  it('prefixes link with the Aegis-Link sentinel when no body', () => {
    const got = buildPostBody({
      mode: 'link',
      body: '',
      linkURL: 'https://example.com/article',
    });
    expect(got).toBe('Aegis-Link: https://example.com/article');
  });

  it('combines link line and body with a blank separator', () => {
    const got = buildPostBody({
      mode: 'link',
      body: 'My commentary',
      linkURL: 'https://example.com/x',
    });
    expect(got).toBe('Aegis-Link: https://example.com/x\n\nMy commentary');
  });

  it('falls back to plain body when linkURL is blank', () => {
    const got = buildPostBody({
      mode: 'link',
      body: 'just a thought',
      linkURL: '   ',
    });
    expect(got).toBe('just a thought');
  });
});

describe('parsePostContent (text)', () => {
  it('classifies normal body as text mode', () => {
    const parsed = parsePostContent('hello world');
    expect(parsed.mode).toBe('text');
    expect(parsed.body).toBe('hello world');
    expect(parsed.linkURL).toBe('');
    expect(parsed.preview).toBe('hello world');
  });

  it('truncates long previews to 140 chars', () => {
    const long = 'a'.repeat(200);
    const parsed = parsePostContent(long);
    expect(parsed.preview.length).toBeLessThanOrEqual(140);
    expect(parsed.preview.endsWith('…')).toBe(true);
  });

  it('uses fallback preview when provided', () => {
    const parsed = parsePostContent('full body content', 'short summary');
    expect(parsed.preview).toBe('short summary');
  });

  it('handles empty input', () => {
    const parsed = parsePostContent('');
    expect(parsed.mode).toBe('text');
    expect(parsed.body).toBe('');
    expect(parsed.preview).toBe('');
  });
});

describe('parsePostContent (link)', () => {
  it('extracts link URL from the Aegis-Link sentinel', () => {
    const parsed = parsePostContent('Aegis-Link: https://example.com/post');
    expect(parsed.mode).toBe('link');
    expect(parsed.linkURL).toBe('https://example.com/post');
    expect(parsed.linkHostname).toBe('example.com');
  });

  it('strips the leading www. from the hostname', () => {
    const parsed = parsePostContent('Aegis-Link: https://www.example.com/');
    expect(parsed.linkHostname).toBe('example.com');
  });

  it('uses the body as preview when commentary is present', () => {
    const parsed = parsePostContent(
      'Aegis-Link: https://example.com/x\n\nLook at this article.',
    );
    expect(parsed.body).toBe('Look at this article.');
    expect(parsed.preview).toBe('Look at this article.');
  });

  it('falls back to hostname when no commentary', () => {
    const parsed = parsePostContent('Aegis-Link: https://example.com/path');
    expect(parsed.preview).toBe('example.com');
  });

  it('falls back to URL itself when hostname extraction fails', () => {
    const parsed = parsePostContent('Aegis-Link: not-a-real-url');
    expect(parsed.linkURL).toBe('not-a-real-url');
    // hostname extraction returns '' for unparseable URLs;
    // preview should fall back to the raw URL itself.
    expect(parsed.preview).toBe('not-a-real-url');
  });

  it('normalizes CRLF before parsing', () => {
    const parsed = parsePostContent(
      'Aegis-Link: https://example.com/\r\n\r\nCommentary line',
    );
    expect(parsed.mode).toBe('link');
    expect(parsed.body).toBe('Commentary line');
  });
});

describe('buildPostBody + parsePostContent round-trip', () => {
  it('round-trips a text post', () => {
    const built = buildPostBody({ mode: 'text', body: 'plain text post', linkURL: '' });
    const parsed = parsePostContent(built);
    expect(parsed.mode).toBe('text');
    expect(parsed.body).toBe('plain text post');
  });

  it('round-trips a link post with commentary', () => {
    const built = buildPostBody({
      mode: 'link',
      body: 'commentary',
      linkURL: 'https://example.com/x',
    });
    const parsed = parsePostContent(built);
    expect(parsed.mode).toBe('link');
    expect(parsed.linkURL).toBe('https://example.com/x');
    expect(parsed.body).toBe('commentary');
  });

  it('round-trips a bare link', () => {
    const built = buildPostBody({
      mode: 'link',
      body: '',
      linkURL: 'https://example.com/',
    });
    const parsed = parsePostContent(built);
    expect(parsed.mode).toBe('link');
    expect(parsed.linkURL).toBe('https://example.com/');
    expect(parsed.body).toBe('');
  });
});
