import { describe, it, expect, beforeEach } from 'vitest';
import { useTemplateGroupStore } from './templateGroupStore';
import type { TemplateGroup } from '../services/templateGroups';

describe('templateGroupStore', () => {
  const mockGroup: TemplateGroup = {
    id: 1,
    name: 'Work Templates',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  };

  beforeEach(() => {
    useTemplateGroupStore.setState({ groups: [], isLoading: false, error: null });
  });

  it('should initialize with empty state', () => {
    const state = useTemplateGroupStore.getState();
    expect(state.groups).toEqual([]);
    expect(state.isLoading).toBe(false);
    expect(state.error).toBeNull();
  });

  it('should set groups', () => {
    const { setGroups } = useTemplateGroupStore.getState();
    setGroups([mockGroup]);
    expect(useTemplateGroupStore.getState().groups).toEqual([mockGroup]);
  });

  it('should add group', () => {
    const { addGroup } = useTemplateGroupStore.getState();
    addGroup(mockGroup);
    expect(useTemplateGroupStore.getState().groups).toHaveLength(1);
    expect(useTemplateGroupStore.getState().groups[0]).toEqual(mockGroup);
  });

  it('should update group', () => {
    const { setGroups, updateGroup } = useTemplateGroupStore.getState();
    setGroups([mockGroup]);

    const updated = { ...mockGroup, name: 'Updated Templates' };
    updateGroup(1, updated);

    const state = useTemplateGroupStore.getState();
    expect(state.groups[0].name).toBe('Updated Templates');
  });

  it('should remove group', () => {
    const { setGroups, removeGroup } = useTemplateGroupStore.getState();
    setGroups([mockGroup]);

    removeGroup(1);

    expect(useTemplateGroupStore.getState().groups).toHaveLength(0);
  });

  it('should set loading state', () => {
    const { setLoading } = useTemplateGroupStore.getState();
    setLoading(true);
    expect(useTemplateGroupStore.getState().isLoading).toBe(true);

    setLoading(false);
    expect(useTemplateGroupStore.getState().isLoading).toBe(false);
  });

  it('should set error state', () => {
    const { setError } = useTemplateGroupStore.getState();
    setError('Test error');
    expect(useTemplateGroupStore.getState().error).toBe('Test error');

    setError(null);
    expect(useTemplateGroupStore.getState().error).toBeNull();
  });
});
