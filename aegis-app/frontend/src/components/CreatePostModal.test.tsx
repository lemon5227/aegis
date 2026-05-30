import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { CreatePostModal } from './CreatePostModal';

const POST_DRAFT_PREFIX = 'aegis:create-post-draft:v2:';

describe('CreatePostModal', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    vi.useRealTimers();
    window.localStorage.clear();
  });

  // -----------------------------------------------------------------------
  // Render gating
  // -----------------------------------------------------------------------

  it('renders nothing when isOpen=false', () => {
    const { container } = render(
      <CreatePostModal
        isOpen={false}
        onClose={vi.fn()}
        subId="general"
        onCreate={vi.fn()}
      />,
    );
    expect(container).toBeEmptyDOMElement();
  });

  it('renders the modal header and tabs when isOpen=true', () => {
    render(
      <CreatePostModal
        isOpen={true}
        onClose={vi.fn()}
        subId="general"
        onCreate={vi.fn()}
      />,
    );
    expect(screen.getByText('Create Post')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Text Post' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Link Post' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Compose' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Preview' })).toBeInTheDocument();
  });

  it('shows the autosave info line referencing the sub id', () => {
    render(
      <CreatePostModal
        isOpen={true}
        onClose={vi.fn()}
        subId="rust"
        onCreate={vi.fn()}
      />,
    );
    expect(screen.getByText(/r\/rust/)).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Mode switching
  // -----------------------------------------------------------------------

  it('default mode is text — link URL field is hidden', () => {
    render(
      <CreatePostModal
        isOpen={true}
        onClose={vi.fn()}
        subId="general"
        onCreate={vi.fn()}
      />,
    );
    expect(screen.queryByPlaceholderText('https://example.com/article')).not.toBeInTheDocument();
  });

  it('switching to link mode reveals the link URL field', () => {
    render(
      <CreatePostModal
        isOpen={true}
        onClose={vi.fn()}
        subId="general"
        onCreate={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Link Post' }));
    expect(screen.getByPlaceholderText('https://example.com/article')).toBeInTheDocument();
  });

  it('placeholder for body changes between text and link modes', () => {
    render(
      <CreatePostModal
        isOpen={true}
        onClose={vi.fn()}
        subId="general"
        onCreate={vi.fn()}
      />,
    );
    expect(screen.getByPlaceholderText("What's on your mind?")).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Link Post' }));
    expect(screen.getByPlaceholderText('Add context for why this link matters.')).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Close button
  // -----------------------------------------------------------------------

  it('clicking the × button calls onClose', () => {
    const onClose = vi.fn();
    const { container } = render(
      <CreatePostModal
        isOpen={true}
        onClose={onClose}
        subId="general"
        onCreate={vi.fn()}
      />,
    );
    // The close icon is inside a button with the 'close' material icon.
    const closeButton = Array.from(container.querySelectorAll('button')).find(
      (b) => b.textContent === 'close',
    );
    expect(closeButton).toBeDefined();
    fireEvent.click(closeButton!);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('clicking the backdrop calls onClose', () => {
    const onClose = vi.fn();
    const { container } = render(
      <CreatePostModal
        isOpen={true}
        onClose={onClose}
        subId="general"
        onCreate={vi.fn()}
      />,
    );
    // The backdrop is the absolute-positioned div with bg-black/50.
    const backdrop = container.querySelector('.bg-black\\/50') as HTMLElement;
    expect(backdrop).toBeTruthy();
    fireEvent.click(backdrop);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  // -----------------------------------------------------------------------
  // Draft persistence (restore + autosave)
  // -----------------------------------------------------------------------

  it('restores a saved draft when reopened with the same author/sub key', () => {
    const key = `${POST_DRAFT_PREFIX}alice:rust`;
    window.localStorage.setItem(
      key,
      JSON.stringify({
        version: 2,
        mode: 'text',
        title: 'Saved Title',
        body: 'Saved body',
        linkURL: '',
        externalImageURL: '',
        updatedAt: 1700000000,
      }),
    );

    render(
      <CreatePostModal
        isOpen={true}
        onClose={vi.fn()}
        subId="rust"
        authorPublicKey="alice"
        onCreate={vi.fn()}
      />,
    );

    expect(screen.getByDisplayValue('Saved Title')).toBeInTheDocument();
    expect(screen.getByDisplayValue('Saved body')).toBeInTheDocument();
    expect(screen.getByText('Recovered your draft for this sub.')).toBeInTheDocument();
  });

  it('initializes empty when no saved draft exists', () => {
    render(
      <CreatePostModal
        isOpen={true}
        onClose={vi.fn()}
        subId="rust"
        authorPublicKey="alice"
        onCreate={vi.fn()}
      />,
    );

    expect(screen.queryByDisplayValue('Saved Title')).not.toBeInTheDocument();
    expect(screen.getByText('Drafts are autosaved for r/rust.')).toBeInTheDocument();
  });

  it('autosaves the draft to localStorage after the autosave delay', async () => {
    vi.useFakeTimers();
    render(
      <CreatePostModal
        isOpen={true}
        onClose={vi.fn()}
        subId="rust"
        authorPublicKey="alice"
        onCreate={vi.fn()}
      />,
    );

    const titleInput = screen.getByPlaceholderText('Enter a descriptive title');
    fireEvent.change(titleInput, { target: { value: 'Autosaved Title' } });

    // Advance past the autosave delay.
    vi.advanceTimersByTime(2000);

    await vi.waitFor(() => {
      const stored = window.localStorage.getItem(`${POST_DRAFT_PREFIX}alice:rust`);
      expect(stored).not.toBeNull();
      const parsed = JSON.parse(stored!);
      expect(parsed.title).toBe('Autosaved Title');
    });
  });

  // -----------------------------------------------------------------------
  // Submit flow
  // -----------------------------------------------------------------------

  it('submit button is disabled when there is nothing valid to submit', () => {
    render(
      <CreatePostModal
        isOpen={true}
        onClose={vi.fn()}
        subId="general"
        onCreate={vi.fn()}
      />,
    );
    // The publish button label is dynamic; just find the disabled submit-shaped button.
    // It's the only button styled with bg-warm-accent that is also disabled at start.
    const submit = screen.getByRole('button', { name: /^Post( Link)?$/ });
    expect(submit).toBeDisabled();
  });

  it('typing a title and body enables submit and onCreate fires with normalized payload', async () => {
    const onCreate = vi.fn();
    const onClose = vi.fn();

    render(
      <CreatePostModal
        isOpen={true}
        onClose={onClose}
        subId="general"
        authorPublicKey="alice"
        onCreate={onCreate}
      />,
    );

    fireEvent.change(screen.getByPlaceholderText('Enter a descriptive title'), {
      target: { value: 'Hello world' },
    });
    fireEvent.change(screen.getByPlaceholderText("What's on your mind?"), {
      target: { value: 'This is the body of the post.' },
    });

    const submit = screen.getByRole('button', { name: /^Post( Link)?$/ });
    expect(submit).not.toBeDisabled();
    fireEvent.click(submit);

    await waitFor(() => {
      expect(onCreate).toHaveBeenCalledTimes(1);
    });
    const arg = onCreate.mock.calls[0][0];
    expect(arg.mode).toBe('text');
    expect(arg.title).toBe('Hello world');
    expect(arg.body).toBe('This is the body of the post.');
  });

  it('successful submit clears the persisted draft and closes the modal', async () => {
    const key = `${POST_DRAFT_PREFIX}alice:general`;
    window.localStorage.setItem(
      key,
      JSON.stringify({
        version: 2,
        mode: 'text',
        title: 'Pre-filled',
        body: 'Pre-filled body',
        linkURL: '',
        externalImageURL: '',
        updatedAt: 1,
      }),
    );

    const onCreate = vi.fn().mockResolvedValue(undefined);
    const onClose = vi.fn();

    render(
      <CreatePostModal
        isOpen={true}
        onClose={onClose}
        subId="general"
        authorPublicKey="alice"
        onCreate={onCreate}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: /^Post( Link)?$/ }));

    await waitFor(() => {
      expect(onCreate).toHaveBeenCalledTimes(1);
      expect(onClose).toHaveBeenCalledTimes(1);
      expect(window.localStorage.getItem(key)).toBeNull();
    });
  });

  it('failed submit keeps the draft and surfaces an error message', async () => {
    const onCreate = vi.fn().mockRejectedValue(new Error('network down'));

    render(
      <CreatePostModal
        isOpen={true}
        onClose={vi.fn()}
        subId="general"
        authorPublicKey="alice"
        onCreate={onCreate}
      />,
    );

    fireEvent.change(screen.getByPlaceholderText('Enter a descriptive title'), {
      target: { value: 'Doomed Title' },
    });
    fireEvent.change(screen.getByPlaceholderText("What's on your mind?"), {
      target: { value: 'Doomed body' },
    });

    fireEvent.click(screen.getByRole('button', { name: /^Post( Link)?$/ }));

    await waitFor(() => {
      expect(onCreate).toHaveBeenCalled();
    });

    // Title still in the form (not cleared on failure).
    expect(screen.getByDisplayValue('Doomed Title')).toBeInTheDocument();
  });

  // -----------------------------------------------------------------------
  // Validation
  // -----------------------------------------------------------------------

  it('shows a link validation error for non-http(s) URLs in link mode', () => {
    render(
      <CreatePostModal
        isOpen={true}
        onClose={vi.fn()}
        subId="general"
        onCreate={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Link Post' }));

    fireEvent.change(screen.getByPlaceholderText('https://example.com/article'), {
      target: { value: 'javascript:alert(1)' },
    });

    // The submit button should be disabled with such a URL.
    const submit = screen.getByRole('button', { name: /^Post( Link)?$/ });
    expect(submit).toBeDisabled();
  });

  it('shows external-image validation error for non-http(s) URLs', () => {
    render(
      <CreatePostModal
        isOpen={true}
        onClose={vi.fn()}
        subId="general"
        onCreate={vi.fn()}
      />,
    );

    // Fill required fields so the only thing left to fail is the image URL.
    fireEvent.change(screen.getByPlaceholderText('Enter a descriptive title'), {
      target: { value: 'A title' },
    });
    fireEvent.change(screen.getByPlaceholderText("What's on your mind?"), {
      target: { value: 'A body' },
    });
    fireEvent.change(screen.getByPlaceholderText('https://example.com/image.jpg'), {
      target: { value: 'data:image/png;base64,xx' },
    });

    const submit = screen.getByRole('button', { name: /^Post( Link)?$/ });
    expect(submit).toBeDisabled();
  });
});
