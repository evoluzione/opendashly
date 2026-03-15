export type LoginPayload = {
  username: string;
  password: string;
};

export type ChangePasswordPayload = {
  newPassword: string;
  currentPassword?: string;
};

export function buildLoginPayload(username: string, password: string): LoginPayload {
  return { username, password };
}

export function buildChangePasswordPayload(
  newPassword: string,
  currentPassword?: string
): ChangePasswordPayload {
  if (!currentPassword) {
    return { newPassword };
  }
  return { newPassword, currentPassword };
}

export function buildFirstLoginPasswordPayload(newPassword: string): { newPassword: string } {
  return { newPassword };
}
