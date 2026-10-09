// Active mode comes from the certificate manager, not legacy preferences.
// Keys stay in File objects and are sent only for explicit activation.
import { useEffect, useState, forwardRef, useImperativeHandle, type ReactNode } from 'react';
import { getTlsStatus, uploadTlsCertificate, restoreSelfSignedTls, saveAcmeCertificate, type TlsStatus } from '../../api/client';

export type TlsControlsHandle = { save: () => Promise<void> };
export default forwardRef<TlsControlsHandle, { onCADownloadChange: (available: boolean) => void; onDirty: (dirty: boolean) => void; ddnsHostname: string | null; actions: ReactNode }>(function TlsControls({ onCADownloadChange, onDirty, ddnsHostname, actions }, ref) {
  const [status, setStatus] = useState<TlsStatus | null>(null);
  const [hostname, setHostname] = useState('');
  const [email, setEmail] = useState('');
  const [agree, setAgree] = useState(false);
  const [mode, setMode] = useState('self-signed');
  const [files, setFiles] = useState<(File | null)[]>([null, null, null]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');
  const [inputKey, setInputKey] = useState(0);
  // The row's action column owns the download button. Never offer the
  // uploaded issuer as the generated CA while restoration is still pending.
  useEffect(() => {
    onCADownloadChange(mode === 'self-signed' && status?.mode === 'self-signed');
  }, [mode, status?.mode, onCADownloadChange]);
  useEffect(() => {
    let active = true;
    getTlsStatus()
      .then((s) => {
        if (active) {
          setStatus(s);
          setMode(s.acme?.configured ? 'acme' : s.mode);
          setAgree(s.acme?.agreementAccepted || false);
          setHostname(current => s.acme?.hostname || current);
          if (s.acme?.email) setEmail(s.acme.email);
        }
      })
      .catch(() => {
        if (active) setError('Could not load TLS status.');
      });
    const timer=window.setInterval(()=>{if(document.visibilityState==='visible') getTlsStatus().then(s=>{if(active)setStatus(s)}).catch(()=>{});},300000);
    return () => {
      window.clearInterval(timer);
      active = false;
    };
  }, []);
  const effectiveHostname = ddnsHostname === null ? hostname : ddnsHostname;
  const dirty = !!status && (mode !== (status.acme.configured ? 'acme' : status.mode) || mode === 'acme' && (agree && !status.acme.running && (status.mode !== 'acme' || status.acme.message.includes('failed')) || agree !== status.acme.agreementAccepted || email !== status.acme.email || effectiveHostname !== status.acme.hostname) || files.some(Boolean));
  useEffect(() => { onDirty(dirty && !busy); }, [dirty, busy, onDirty]);
  const row = (name: string, control: ReactNode, actions?: ReactNode) => <div className="setting-row"><div className="setting-name">{name}</div><div className="setting-control">{control}</div><div className="setting-actions">{actions}</div></div>;
  async function restore() {
    setBusy(true);
    setError('');
    setMessage('');
    try {
      const s = await restoreSelfSignedTls();
      setStatus(s);
      setMode(s.mode);
      setMessage('Self-signed certificate restored. Reconnect using a name covered by this certificate.');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not restore certificate.');
      throw e;
    } finally {
      setBusy(false);
    }
  }
  async function upload() {
    if (files.some((file) => !file)) return;
    if (files.reduce((size, file) => size + (file?.size || 0), 0) > 256 * 1024) {
      throw new Error('Combined certificate files must be no larger than 256 KiB.');
    }
    setBusy(true);
    setError('');
    setMessage('');
    try {
      const [cert, key, ca] = await Promise.all(files.map((file) => file!.text()));
      const s = await uploadTlsCertificate(cert, key, ca);
      setStatus(s);
      setMode(s.mode);
      setFiles([null, null, null]);
      setInputKey((value) => value + 1);
      setMessage('Certificate activated for UI, API and playback. Use its DNS hostname to access AlreadyOn.');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not activate certificate.');
      throw e;
    } finally {
      setBusy(false);
    }
  }
  useImperativeHandle(ref, () => ({ save: async () => {
    if (!dirty) return;
    if (mode === 'user-provided' && files.some(file => !file)) throw new Error('Choose all three certificate files before saving.');
    if (mode === 'acme') setStatus(await saveAcmeCertificate(effectiveHostname,email,agree));
    else if (mode === 'self-signed' && (status?.mode !== mode || status?.acme.configured)) await restore();
    else if (mode === 'user-provided' && files.some(Boolean)) await upload();
  } }));
  return (
    <>
      {row('TLS Certificates', <div className="setting-control-inline">
        <select
          aria-label="TLS certificate mode"
          disabled={busy || !status}
          value={mode}
          onChange={(e) => {
            setMode(e.target.value);
            setError('');
            setMessage('');
            setFiles([null, null, null]);
            setInputKey((value) => value + 1);
          }}
        >
          <option value="self-signed">Self-Signed (Default)</option>
          <option value="user-provided">User Uploaded Cert</option>
          <option value="acme">Let's Encrypt</option>
        </select>
      </div>, actions)}
      {mode === 'acme' && <>
        {ddnsHostname === null && row('Public Hostname', <input aria-label="Certificate hostname" type="text" value={hostname} onChange={e=>setHostname(e.target.value)} disabled={busy || status?.acme.running} />)}
        {row('Contact Email', <input aria-label="ACME contact email" type="email" value={email} onChange={e=>setEmail(e.target.value)} disabled={busy || status?.acme.running} />)}
        {row('Terms of Service', <a href="https://letsencrypt.org/repository/" target="_blank" rel="noreferrer">View Let's Encrypt Subscriber Agreement</a>, <button type="button" className="admin-btn" disabled={busy || status?.acme.running} onClick={()=>setAgree(!agree)}>{agree ? 'Revoke Agreement' : 'I Agree'}</button>)}
        {row('Certificate Details', <div role="status">
          {status?.acme.message.includes('failed') || status?.acme.message.includes('interrupted') ? <p className="profile-field-error">{status.acme.message} Confirm public A/AAAA records point to this server and forward WAN TCP/80 to server TCP/19902; publish 19902 in Compose.</p> : status?.mode === 'acme' ? <>Cert Hostname: {status.names.join(', ')}<br />Issued: {new Date(status.issuedAt).toLocaleDateString()}<br />Expires: {new Date(status.expiresAt).toLocaleDateString()}</> : 'No public certificate activated yet.'}
          <p>{status?.acme.running ? 'Issuance in progress…' : status?.acme.message || 'Not configured'}{status?.acme.lastAttempt ? ` · Last attempt: ${new Date(status.acme.lastAttempt*1000).toLocaleString()}` : ''}</p>
          {!agree && <p>Agreement is required for issuance and automatic renewal.</p>}
        </div>)}
        <p className="setting-note">Save Changes starts validation and production issuance. Forward public TCP/80 to this server's TCP/19902 and publish that port in Compose. Keep this rule for renewal. Status refreshes every five minutes.</p>
      </>}
      {mode === 'user-provided' && (
        <div className="tls-upload" key={inputKey}>
          <p>
            Choose PEM files: server certificate first followed by intermediates, its matching unencrypted private key, and the issuing CA bundle. Include the
            DNS names you use for UI, API and playback. PFX/P12 and binary DER are not supported.
          </p>
          {['Certificate chain (.pem/.crt)', 'Private key (.pem/.key)', 'Issuing CA bundle (.pem/.crt)'].map((label, index) => (
            <label key={label}>
              {label}
              <input
                type="file"
                disabled={busy}
                accept=".pem,.crt,.cert,.key"
                onChange={(e) => setFiles((current) => current.map((file, i) => (i === index ? e.target.files?.[0] || null : file)))}
              />
            </label>
          ))}
          <p>
            Activation changes new HTTPS connections immediately. Ensure your browser trusts the issuing CA and use a covered DNS name; localhost may stop
            working. Uploaded certificates must be renewed manually.
          </p>

        </div>
      )}
      {message && <p role="status">{message}</p>}
      {error && (
        <p role="alert" className="profile-field-error">
          {error}
        </p>
      )}
    </>
  );
});
