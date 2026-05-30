import { describe, expect, it } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import { useToasts } from './useToasts';

describe('useToasts', () => {
  it('starts with an empty list', () => {
    const { result } = renderHook(() => useToasts());
    expect(result.current.toasts).toEqual([]);
  });

  it('addToast appends a new toast and returns its id', () => {
    const { result } = renderHook(() => useToasts());

    let id = '';
    act(() => {
      id = result.current.addToast({
        type: 'info',
        title: 'Hello',
        message: 'World',
      });
    });

    expect(typeof id).toBe('string');
    expect(id.length).toBeGreaterThan(0);
    expect(result.current.toasts).toHaveLength(1);
    expect(result.current.toasts[0]).toMatchObject({
      id,
      type: 'info',
      title: 'Hello',
      message: 'World',
    });
  });

  it('preserves insertion order across multiple addToast calls', () => {
    const { result } = renderHook(() => useToasts());
    act(() => {
      result.current.addToast({ type: 'info', title: 'First', message: '' });
    });
    act(() => {
      result.current.addToast({ type: 'success', title: 'Second', message: '' });
    });
    act(() => {
      result.current.addToast({ type: 'warning', title: 'Third', message: '' });
    });

    expect(result.current.toasts.map((t) => t.title)).toEqual([
      'First',
      'Second',
      'Third',
    ]);
  });

  it('removeToast drops the matching id', () => {
    const { result } = renderHook(() => useToasts());

    let firstId = '';
    let secondId = '';
    act(() => {
      firstId = result.current.addToast({ type: 'info', title: 'A', message: '' });
      secondId = result.current.addToast({ type: 'info', title: 'B', message: '' });
    });
    expect(result.current.toasts).toHaveLength(2);

    act(() => {
      result.current.removeToast(firstId);
    });
    expect(result.current.toasts).toHaveLength(1);
    expect(result.current.toasts[0].id).toBe(secondId);
  });

  it('removeToast for an unknown id is a no-op', () => {
    const { result } = renderHook(() => useToasts());
    act(() => {
      result.current.addToast({ type: 'info', title: 'Kept', message: '' });
    });

    act(() => {
      result.current.removeToast('does-not-exist');
    });
    expect(result.current.toasts).toHaveLength(1);
  });

  it('returns stable function references across renders (useCallback)', () => {
    const { result, rerender } = renderHook(() => useToasts());
    const firstAdd = result.current.addToast;
    const firstRemove = result.current.removeToast;

    rerender();

    expect(result.current.addToast).toBe(firstAdd);
    expect(result.current.removeToast).toBe(firstRemove);
  });

  it('generates unique ids for each call', () => {
    const { result } = renderHook(() => useToasts());
    const ids = new Set<string>();
    act(() => {
      for (let i = 0; i < 50; i++) {
        ids.add(result.current.addToast({ type: 'info', title: '', message: '' }));
      }
    });
    expect(ids.size).toBe(50); // no collisions across 50 calls
  });
});
