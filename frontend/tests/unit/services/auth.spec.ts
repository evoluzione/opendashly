import { describe, expect, it, vi, beforeEach } from 'vitest';
import { changePassword } from '../../../src/services/auth';
import { apiRequest } from '../../../src/services/api';

vi.mock('../../../src/services/api', () => ({
  apiRequest: vi.fn()
}));

describe('auth service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('changePassword include currentPassword quando presente', async () => {
    const payload = {
      user: {
        id: 'u1',
        username: 'admin',
        role: 'admin',
        isDisabled: false
      },
      mustChangePassword: false
    };

    vi.mocked(apiRequest).mockResolvedValue(payload);

    const result = await changePassword('new-secret', 'old-secret');

    expect(result).toEqual(payload);
    expect(apiRequest).toHaveBeenCalledWith('/api/auth/change-password', {
      method: 'POST',
      body: JSON.stringify({
        newPassword: 'new-secret',
        currentPassword: 'old-secret'
      })
    });
  });

  it('changePassword non include currentPassword quando assente', async () => {
    const payload = {
      user: {
        id: 'u1',
        username: 'admin',
        role: 'admin',
        isDisabled: false
      },
      mustChangePassword: false
    };

    vi.mocked(apiRequest).mockResolvedValue(payload);

    await changePassword('new-secret');

    expect(apiRequest).toHaveBeenCalledWith('/api/auth/change-password', {
      method: 'POST',
      body: JSON.stringify({
        newPassword: 'new-secret'
      })
    });
  });
});
