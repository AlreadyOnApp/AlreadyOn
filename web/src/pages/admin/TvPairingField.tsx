// Pairing secrets stay masked until the user explicitly reveals the entered PIN.
import { PasswordInput } from '../../components/PasswordInput';
import { useTvPairing } from './useTvPairing';

export default function TvPairingField({ pairing, disabled }: { pairing: ReturnType<typeof useTvPairing>; disabled: boolean }) {
  return <>
    <div className="admin-field-row">
      <label htmlFor="tv-pairing-pin">TV Pairing Pin</label>
      <PasswordInput id="tv-pairing-pin" visibilityLabel="PIN" inputMode="numeric" autoComplete="new-password"
        minLength={6} maxLength={12} pattern="[0-9]{6,12}" value={pairing.pin}
        placeholder={pairing.enabled ? '••••••' : undefined}
        onChange={event => pairing.setPin(event.target.value.replace(/[^0-9]/g, '').slice(0, 12))} disabled={disabled || pairing.busy} />
      {pairing.enabled && <button className="admin-btn admin-btn-secondary" type="button" disabled={disabled || pairing.busy} onClick={() => void pairing.remove()}>Remove Pin</button>}
    </div>
    {pairing.error && <p className="profile-field-error" role="alert">{pairing.error}</p>}
  </>;
}
