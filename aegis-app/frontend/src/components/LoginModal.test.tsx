import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { LoginModal } from './LoginModal';

describe('LoginModal', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders nothing when isOpen=false', () => {
    const { container } = render(
      <LoginModal
        isOpen={false}
        onClose={vi.fn()}
        onCreateIdentity={vi.fn()}
        onActivateIdentity={vi.fn()}
        onLoadIdentity={vi.fn()}
        onImportMnemonic={vi.fn()}
      />,
    );
    expect(container).toBeEmptyDOMElement();
  });

  it('renders the welcome screen with three actions when isOpen=true', () => {
    render(
      <LoginModal
        isOpen={true}
        onClose={vi.fn()}
        onCreateIdentity={vi.fn()}
        onActivateIdentity={vi.fn()}
        onLoadIdentity={vi.fn()}
        onImportMnemonic={vi.fn()}
      />,
    );
    expect(screen.getByText('Welcome to Aegis')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Create New Identity' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Load Existing Identity' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Import from Mnemonic' })).toBeInTheDocument();
  });

  it('clicking "Load Existing Identity" calls onLoadIdentity', () => {
    const onLoadIdentity = vi.fn();
    render(
      <LoginModal
        isOpen={true}
        onClose={vi.fn()}
        onCreateIdentity={vi.fn()}
        onActivateIdentity={vi.fn()}
        onLoadIdentity={onLoadIdentity}
        onImportMnemonic={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Load Existing Identity' }));
    expect(onLoadIdentity).toHaveBeenCalledTimes(1);
  });

  it('clicking "Import from Mnemonic" switches to import mode', () => {
    render(
      <LoginModal
        isOpen={true}
        onClose={vi.fn()}
        onCreateIdentity={vi.fn()}
        onActivateIdentity={vi.fn()}
        onLoadIdentity={vi.fn()}
        onImportMnemonic={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Import from Mnemonic' }));
    expect(screen.getByText('Import Identity')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('word1 word2 word3 ...')).toBeInTheDocument();
  });

  it('Import button is disabled until mnemonic input has content', () => {
    render(
      <LoginModal
        isOpen={true}
        onClose={vi.fn()}
        onCreateIdentity={vi.fn()}
        onActivateIdentity={vi.fn()}
        onLoadIdentity={vi.fn()}
        onImportMnemonic={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Import from Mnemonic' }));

    const importButton = screen.getByRole('button', { name: 'Import' });
    expect(importButton).toBeDisabled();

    fireEvent.change(screen.getByPlaceholderText('word1 word2 word3 ...'), {
      target: { value: 'word1 word2 word3' },
    });
    expect(importButton).not.toBeDisabled();
  });

  it('successful import calls onImportMnemonic and closes modal', async () => {
    const onImportMnemonic = vi.fn().mockResolvedValue(undefined);
    const onClose = vi.fn();

    render(
      <LoginModal
        isOpen={true}
        onClose={onClose}
        onCreateIdentity={vi.fn()}
        onActivateIdentity={vi.fn()}
        onLoadIdentity={vi.fn()}
        onImportMnemonic={onImportMnemonic}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Import from Mnemonic' }));
    fireEvent.change(screen.getByPlaceholderText('word1 word2 word3 ...'), {
      target: { value: '  twelve magic words go here ...  ' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Import' }));

    await waitFor(() => {
      expect(onImportMnemonic).toHaveBeenCalledWith('twelve magic words go here ...');
      expect(onClose).toHaveBeenCalledTimes(1);
    });
  });

  it('failed import keeps the modal open and shows error', async () => {
    const onImportMnemonic = vi.fn().mockRejectedValue(new Error('bad mnemonic'));
    const onClose = vi.fn();

    render(
      <LoginModal
        isOpen={true}
        onClose={onClose}
        onCreateIdentity={vi.fn()}
        onActivateIdentity={vi.fn()}
        onLoadIdentity={vi.fn()}
        onImportMnemonic={onImportMnemonic}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Import from Mnemonic' }));
    fireEvent.change(screen.getByPlaceholderText('word1 word2 word3 ...'), {
      target: { value: 'wrong words' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Import' }));

    await waitFor(() => {
      expect(screen.getByText(/Import failed/)).toBeInTheDocument();
      expect(onClose).not.toHaveBeenCalled();
    });
  });

  it('Back button in import mode returns to selection', () => {
    render(
      <LoginModal
        isOpen={true}
        onClose={vi.fn()}
        onCreateIdentity={vi.fn()}
        onActivateIdentity={vi.fn()}
        onLoadIdentity={vi.fn()}
        onImportMnemonic={vi.fn()}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Import from Mnemonic' }));
    fireEvent.click(screen.getByRole('button', { name: 'Back' }));
    expect(screen.getByText('Welcome to Aegis')).toBeInTheDocument();
  });

  it('clicking "Create New Identity" advances to backup mode showing the mnemonic words', async () => {
    const onCreateIdentity = vi.fn().mockResolvedValue({
      publicKey: 'pubkey-12345',
      mnemonic: 'apple banana cherry date elder fig grape honey ice juice kiwi lemon',
    });

    render(
      <LoginModal
        isOpen={true}
        onClose={vi.fn()}
        onCreateIdentity={onCreateIdentity}
        onActivateIdentity={vi.fn()}
        onLoadIdentity={vi.fn()}
        onImportMnemonic={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Create New Identity' }));

    await waitFor(() => {
      expect(screen.getByText('Backup Mnemonic')).toBeInTheDocument();
      expect(screen.getByText(/^1\. apple/)).toBeInTheDocument();
      expect(screen.getByText(/^12\. lemon/)).toBeInTheDocument();
    });
  });

  it('Continue button is disabled until the backup checkbox is checked', async () => {
    const onCreateIdentity = vi.fn().mockResolvedValue({
      publicKey: 'pk',
      mnemonic: 'a b c d e f g h i j k l',
    });

    render(
      <LoginModal
        isOpen={true}
        onClose={vi.fn()}
        onCreateIdentity={onCreateIdentity}
        onActivateIdentity={vi.fn()}
        onLoadIdentity={vi.fn()}
        onImportMnemonic={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Create New Identity' }));

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Continue' })).toBeInTheDocument();
    });

    expect(screen.getByRole('button', { name: 'Continue' })).toBeDisabled();

    fireEvent.click(screen.getByRole('checkbox'));
    expect(screen.getByRole('button', { name: 'Continue' })).not.toBeDisabled();
  });

  it('clicking Continue (after backup) calls onActivateIdentity with the new identity', async () => {
    const identity = { publicKey: 'pk-active', mnemonic: 'a b c d e f g h i j k l' };
    const onCreateIdentity = vi.fn().mockResolvedValue(identity);
    const onActivateIdentity = vi.fn().mockResolvedValue(undefined);
    const onClose = vi.fn();

    render(
      <LoginModal
        isOpen={true}
        onClose={onClose}
        onCreateIdentity={onCreateIdentity}
        onActivateIdentity={onActivateIdentity}
        onLoadIdentity={vi.fn()}
        onImportMnemonic={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Create New Identity' }));
    await waitFor(() => screen.getByRole('button', { name: 'Continue' }));

    fireEvent.click(screen.getByRole('checkbox'));
    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

    await waitFor(() => {
      expect(onActivateIdentity).toHaveBeenCalledWith(identity);
      expect(onClose).toHaveBeenCalledTimes(1);
    });
  });

  it('failed create identity surfaces an error and stays on selection', async () => {
    const onCreateIdentity = vi.fn().mockResolvedValue(null);

    render(
      <LoginModal
        isOpen={true}
        onClose={vi.fn()}
        onCreateIdentity={onCreateIdentity}
        onActivateIdentity={vi.fn()}
        onLoadIdentity={vi.fn()}
        onImportMnemonic={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Create New Identity' }));

    await waitFor(() => {
      expect(screen.getByText(/Failed to create identity/)).toBeInTheDocument();
      expect(screen.queryByText('Backup Mnemonic')).not.toBeInTheDocument();
    });
  });
});
