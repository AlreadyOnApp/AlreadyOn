import { useEffect, useState } from 'react';
import { apiFetch } from '../../api/access';

// Only a configured flag is returned: saved PIN values never reach the browser.
export function useTvPairing() {
  const [pin, setPin] = useState('');
  const [enabled, setEnabled] = useState(false);
  const [busy, setBusy] = useState(true);
  const [error, setError] = useState('');
  useEffect(() => {
    let active = true;
    void apiFetch('/api/auth/tv-pin').then(async response => {
      if (!response.ok) throw new Error('Could not load TV pairing settings.');
      const data = await response.json() as { enabled: boolean };
      if (active) setEnabled(data.enabled);
    }).catch(error => { if (active) setError(String(error.message)); })
      .finally(() => { if (active) setBusy(false); });
    return () => { active = false; };
  }, []);
  async function update(remove = false) {
    setBusy(true); setError('');
    try {
      const response = await apiFetch('/api/auth/tv-pin', {
        method: remove ? 'DELETE' : 'PUT', headers: { 'Content-Type': 'application/json' },
        body: remove ? undefined : JSON.stringify({ pin }),
      });
      if (!response.ok) {
        const data = await response.json() as { error?: string };
        throw new Error(data.error || 'Could not update TV pairing PIN.');
      }
      setEnabled(!remove); setPin('');
    } finally { setBusy(false); }
  }
  async function remove() {
    try { await update(true); }
    catch (error) { setError(error instanceof Error ? error.message : 'Could not remove PIN.'); }
  }
  return { pin, setPin, enabled, busy, error, save: () => update(), remove };
}

