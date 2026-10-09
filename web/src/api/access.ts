// The browser signs in with a session cookie the server sets (HttpOnly, so
// scripts never see it), and every same-origin request carries it. Playback
// URLs contain only scoped short-lived tokens issued by the server.
export async function apiFetch(input: string, init?: RequestInit): Promise<Response> {
  const url = new URL(input, window.location.origin);
  if (url.origin !== window.location.origin) throw new Error('Management requests must use the application origin.');
  const response = await globalThis.fetch(url, init);
  // Only sign-in rejections carry this header; playback 401s do not.
  if (response.status === 401 && response.headers.get('WWW-Authenticate') === 'Bearer') window.dispatchEvent(new Event('alreadyon-access-expired'));
  // An owner required a new password while this browser was signed in.
  if (response.status === 403) {
    const body = await response.clone().json().catch(() => null);
    if (body?.code === 'password_change_required') window.dispatchEvent(new Event('alreadyon-password-change-required'));
  }
  return response;
}

// Ends this browser's session and returns to the sign-in page.
export async function signOut(): Promise<void> {
  await globalThis.fetch('/api/auth/logout', { method: 'POST' }).catch(() => undefined);
  window.location.assign('/');
}
