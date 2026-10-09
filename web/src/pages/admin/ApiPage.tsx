// Real backend (GET/POST/DELETE /api/admin/api-keys — see
// server/internal/auth.APIKey and server/internal/httpapi/apikeys.go).
//
// Keys are for scripts and tools; the web UI signs in with the server account
// instead. Every management route accepts a key as a bearer token.
import { useEffect, useRef, useState } from "react";
import { getApiKeys, createApiKey, revokeApiKey, type ApiKey } from "../../api/client";
import "./AdminShared.css";

const SWAGGER_UI_PATH = "/api/docs/";

export default function ApiPage() {
  const [keys, setKeys] = useState<ApiKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [nameDraft, setNameDraft] = useState("");
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [newRawKey, setNewRawKey] = useState<string | null>(null);
  const [confirmRevokeId, setConfirmRevokeId] = useState<number | null>(null);
  const [copied, setCopied] = useState(false);
  const [copyError, setCopyError] = useState<string | null>(null);
  const keyInput = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!newRawKey) return;
    keyInput.current?.focus();
    keyInput.current?.select();
  }, [newRawKey]);

  const fetchKeys = () =>
    getApiKeys()
      .then((data) => {
        setKeys(data);
        setLoadError(null);
      })
      .catch((err: Error) => setLoadError(err.message))
      .finally(() => setLoading(false));

  useEffect(() => {
    fetchKeys();
  }, []);

  const handleGenerate = async () => {
    if (nameDraft.trim() === "") return;
    setCreating(true);
    setCreateError(null);
    try {
      const created = await createApiKey(nameDraft.trim());
      setCopied(false);
      setCopyError(null);
      setNewRawKey(created.key);
      setNameDraft("");
      setKeys((prev) => [{ id: created.id, name: created.name, keyPrefix: created.keyPrefix, createdAt: created.createdAt }, ...prev]);
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : "Failed to generate API key");
    } finally {
      setCreating(false);
    }
  };

  const handleRevoke = async (id: number) => {
    if (confirmRevokeId !== id) {
      setConfirmRevokeId(id);
      return;
    }
    try {
      await revokeApiKey(id);
      setConfirmRevokeId(null);
      setKeys((prev) => prev.filter((k) => k.id !== id));
    } catch (err) {
      setLoadError(err instanceof Error ? err.message : "Failed to revoke API key");
    }
  };

  const handleCopy = async () => {
    if (!newRawKey) return;
    setCopied(false);
    setCopyError(null);
    keyInput.current?.focus();
    keyInput.current?.select();
    try {
      // LAN HTTP lacks the Clipboard API. Keep the selected-input fallback
      // synchronous with the click so browsers can accept the user gesture.
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(newRawKey);
      } else if (!document.execCommand("copy")) {
        throw new Error("Copy failed");
      }
      setCopied(true);
    } catch {
      setCopyError("Couldn't copy automatically. Use your browser's Copy command on the selected key.");
    }
  };

  return (
    <div className="admin-section">
      <div className="admin-card">
        <h3>Live API Docs</h3>
        <a href={SWAGGER_UI_PATH} target="_blank" rel="noreferrer" className="admin-btn">
          Open Swagger UI ↗
        </a>
      </div>

      <div className="admin-card">
        <h3>API Keys</h3>
        <p className="admin-section-hint">Generate a key to authenticate scripts or other clients directly against the server API.</p>

        {loadError && <p className="profile-field-error">{loadError}</p>}

        {newRawKey && (
          <div className="admin-card" style={{ borderColor: "var(--accent)" }}>
            <p className="admin-section-hint">
              <strong>Copy this key now — it won't be shown again.</strong>
            </p>
            <div className="admin-field-row">
              <input ref={keyInput} aria-label="New API key" type="text" readOnly value={newRawKey} onFocus={(e) => e.currentTarget.select()} />
              <button type="button" className="admin-btn admin-btn-secondary" onClick={handleCopy}>
                {copied ? "Copied!" : "Copy"}
              </button>
              <button
                type="button"
                className="admin-btn"
                onClick={() => {
                  setNewRawKey(null);
                  setCopied(false);
                  setCopyError(null);
                }}
              >
                Done
              </button>
            </div>
            {copyError && <p className="profile-field-error" role="alert">{copyError}</p>}
          </div>
        )}

        <div className="admin-field-row api-key-create-row">
          <label>New Key Name</label>
          <input
            type="text"
            value={nameDraft}
            placeholder="e.g. Living room Apple TV"
            onChange={(e) => setNameDraft(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleGenerate()}
          />
          <button type="button" className="admin-btn" onClick={handleGenerate} disabled={creating || nameDraft.trim() === ""}>
            {creating ? "Generating…" : "Generate"}
          </button>
        </div>
        {createError && <p className="profile-field-error">{createError}</p>}

        {loading ? (
          <p className="admin-section-hint">Loading…</p>
        ) : (
          <table className="admin-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Key</th>
                <th>Created</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {keys.map((k) => (
                <tr key={k.id}>
                  <td>{k.name}</td>
                  <td>{k.keyPrefix}…</td>
                  <td>{new Date(k.createdAt).toLocaleString()}</td>
                  <td className="admin-table-actions">
                    <button
                      type="button"
                      className={"admin-btn admin-btn-secondary" + (confirmRevokeId === k.id ? " admin-btn-danger" : "")}
                      onClick={() => handleRevoke(k.id)}
                    >
                      {confirmRevokeId === k.id ? "Confirm revoke?" : "Revoke"}
                    </button>
                  </td>
                </tr>
              ))}
              {keys.length === 0 && (
                <tr>
                  <td colSpan={4} className="admin-section-hint">
                    No API keys yet.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
