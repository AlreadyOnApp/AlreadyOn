import { useEffect, useState, type FormEvent } from 'react';
import { apiFetch } from '../../api/access';

// PINs belong to the signed-in account, not a profile or another selected user.
export default function TvPairingCard() {
  const [pin, setPin] = useState('');
  const [enabled, setEnabled] = useState(false);
  const [busy, setBusy] = useState(true);
  const [message, setMessage] = useState('');
  const [fingerprint, setFingerprint] = useState('');
  useEffect(() => {
    let active = true;
    void apiFetch('/api/auth/tv-pin').then(async response => {
      if (!response.ok) throw new Error('Could not load TV pairing settings.');
      const data = await response.json() as { enabled: boolean };
      if (active) setEnabled(data.enabled);
    }).catch(error => { if (active) setMessage(String(error.message)); })
      .finally(() => { if (active) setBusy(false); });
    void apiFetch('/api/tv/trust').then(async response => {
      if (!response.ok) return;
      const data = await response.json() as { fingerprint: string };
      if (active) setFingerprint(data.fingerprint);
    }).catch(() => { /* Public trust metadata is available only on the LAN. */ });
    return () => { active = false; };
  }, []);
  async function save(event?: FormEvent) {
    event?.preventDefault();
    setBusy(true); setMessage('');
    try {
      const response = await apiFetch('/api/auth/tv-pin', {
        method: event ? 'PUT' : 'DELETE', headers: { 'Content-Type': 'application/json' },
        body: event ? JSON.stringify({ pin }) : undefined,
      });
      if (!response.ok) {
        const data = await response.json() as { error?: string };
        throw new Error(data.error || 'Could not update TV PIN.');
      }
      setEnabled(Boolean(event)); setPin('');
      setMessage(event ? 'TV PIN saved for your account.' : 'TV PIN removed. New TVs cannot sign in with it.');
    } catch (error) { setMessage(error instanceof Error ? error.message : 'Could not update TV PIN.'); }
    finally { setBusy(false); }
  }
  return <div className="admin-card">
    <h3>TV Sign-In</h3>
    <p>Set a unique 6–12 digit PIN for your account. On your TV, select this server and enter the PIN to sign in.</p>
    <p>{enabled ? 'Your TV PIN is configured.' : 'Your TV PIN is not configured.'}</p>
    <form className="tv-pairing-form" onSubmit={event => void save(event)}>
      <label htmlFor="tv-pairing-pin">{enabled ? 'Replace TV PIN' : 'TV PIN'}</label>
      <input id="tv-pairing-pin" type="password" inputMode="numeric" autoComplete="new-password"
        minLength={6} maxLength={12} pattern="[0-9]{6,12}" value={pin}
        onChange={event => setPin(event.target.value.replace(/[^0-9]/g, '').slice(0, 12))} disabled={busy} />
      <button className="admin-btn" type="submit" disabled={busy || !/^[0-9]{6,12}$/.test(pin)}>Save TV PIN</button>
      {enabled && <button className="admin-btn" type="button" disabled={busy} onClick={() => void save()}>Remove TV PIN</button>}
    </form>
    {message && <p role="status">{message}</p>}
    {fingerprint && <p>Compare this certificate fingerprint with the TV before confirming the server: <code style={{ overflowWrap: 'anywhere' }}>{fingerprint}</code></p>}
    <p>Removing or replacing a PIN prevents new sign-ins. Use Signed-In Browsers below to sign out an already paired TV.</p>
  </div>;
}
