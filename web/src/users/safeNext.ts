// Parse before comparing origins: browsers remove tabs/newlines from URLs,
// which can turn an apparently local path into a protocol-relative URL.
export function safeNext(search: string, origin: string): string | null {
  const next = new URLSearchParams(search).get('next');
  if (!next || !next.startsWith('/') || next.startsWith('//') || next.includes('\\')) return null;
  try {
    const target = new URL(next, origin);
    return target.origin === origin ? target.href : null;
  } catch {
    return null;
  }
}
