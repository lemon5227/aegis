import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { CreateSubModal } from './CreateSubModal';

describe('CreateSubModal', () => {
  it('renders nothing when isOpen=false', () => {
    const { container } = render(
      <CreateSubModal isOpen={false} onClose={vi.fn()} onCreate={vi.fn()} />,
    );
    expect(container).toBeEmptyDOMElement();
  });

  it('renders the form when isOpen=true', () => {
    render(<CreateSubModal isOpen={true} onClose={vi.fn()} onCreate={vi.fn()} />);
    expect(screen.getByText('Create Sub')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('e.g. golang, tech, music')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('e.g. Go Programming')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('What is this community about?')).toBeInTheDocument();
  });

  it('Create button is disabled until subId has content', () => {
    render(<CreateSubModal isOpen={true} onClose={vi.fn()} onCreate={vi.fn()} />);
    const createButton = screen.getByRole('button', { name: 'Create' });
    expect(createButton).toBeDisabled();

    fireEvent.change(screen.getByPlaceholderText('e.g. golang, tech, music'), {
      target: { value: 'golang' },
    });
    expect(createButton).not.toBeDisabled();
  });

  it('subId input strips uppercase and non-alphanumeric chars on type', () => {
    render(<CreateSubModal isOpen={true} onClose={vi.fn()} onCreate={vi.fn()} />);
    const input = screen.getByPlaceholderText('e.g. golang, tech, music') as HTMLInputElement;
    fireEvent.change(input, { target: { value: 'My-Sub-NAME 123!' } });
    expect(input.value).toBe('mysubname123');
  });

  it('clicking Cancel calls onClose', () => {
    const onClose = vi.fn();
    render(<CreateSubModal isOpen={true} onClose={onClose} onCreate={vi.fn()} />);
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('clicking Create with valid input fires onCreate with trimmed values', async () => {
    const onCreate = vi.fn().mockResolvedValue(undefined);
    const onClose = vi.fn();

    render(<CreateSubModal isOpen={true} onClose={onClose} onCreate={onCreate} />);

    fireEvent.change(screen.getByPlaceholderText('e.g. golang, tech, music'), {
      target: { value: 'golang' },
    });
    fireEvent.change(screen.getByPlaceholderText('e.g. Go Programming'), {
      target: { value: '  Go Programming  ' },
    });
    fireEvent.change(screen.getByPlaceholderText('What is this community about?'), {
      target: { value: '  All things Go.  ' },
    });

    fireEvent.click(screen.getByRole('button', { name: 'Create' }));

    await waitFor(() => {
      expect(onCreate).toHaveBeenCalledWith('golang', 'Go Programming', 'All things Go.');
      expect(onClose).toHaveBeenCalledTimes(1);
    });
  });

  it('failed create surfaces error message and keeps modal open', async () => {
    const onCreate = vi.fn().mockRejectedValue(new Error('sub already exists'));
    const onClose = vi.fn();

    render(<CreateSubModal isOpen={true} onClose={onClose} onCreate={onCreate} />);

    fireEvent.change(screen.getByPlaceholderText('e.g. golang, tech, music'), {
      target: { value: 'taken' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Create' }));

    await waitFor(() => {
      expect(screen.getByText('sub already exists')).toBeInTheDocument();
      expect(onClose).not.toHaveBeenCalled();
    });
  });

  it('failed create with non-Error reason falls back to a default error message', async () => {
    const onCreate = vi.fn().mockRejectedValue(undefined);

    render(<CreateSubModal isOpen={true} onClose={vi.fn()} onCreate={onCreate} />);

    fireEvent.change(screen.getByPlaceholderText('e.g. golang, tech, music'), {
      target: { value: 'fail' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Create' }));

    await waitFor(() => {
      expect(screen.getByText('Failed to create sub.')).toBeInTheDocument();
    });
  });
});
