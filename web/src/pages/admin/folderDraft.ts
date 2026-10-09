// A folder or pattern list being edited. null means "use the server's
// built-in default" (sent to the API as null); an array is a custom list,
// even an empty one.
export type ListDraft = string[] | null;

export function draftFromSaved(source: 'default' | 'admin', savedValues: string[]): ListDraft {
  return source === 'admin' ? [...savedValues] : null;
}

// What the list currently shows: the custom list, or the defaults.
export function draftValues(draft: ListDraft, defaults: string[]): string[] {
  return draft ?? defaults;
}

function sameList(a: string[], b: string[]): boolean {
  return a.length === b.length && a.every((v, i) => v === b[i]);
}

// A custom list identical to the defaults is just the default.
export function normalizeDraft(next: string[], defaults: string[]): ListDraft {
  return sameList(next, defaults) ? null : next;
}

// Whether saving would change anything on the server.
export function draftChanged(draft: ListDraft, savedValues: string[], savedSource: 'default' | 'admin'): boolean {
  if (draft === null) return savedSource === 'admin';
  if (savedSource === 'default') return true;
  return !sameList(draft, savedValues);
}
