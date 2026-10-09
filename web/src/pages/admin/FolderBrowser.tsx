import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { browseFolders, type FolderBrowserListing } from "../../api/client";
import "./FolderBrowser.css";

export default function FolderBrowser({ title, onSelect, onClose }: {
  title: string;
  onSelect: (path: string) => void;
  onClose: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [attempt, setAttempt] = useState(0);
  const [path, setPath] = useState("/data/media/");
  const [address, setAddress] = useState(path);
  const [listing, setListing] = useState<FolderBrowserListing | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const element = dialog.current!;
    const opener = document.activeElement as HTMLElement | null;
    element.showModal();
    return () => { element.close(); opener?.focus(); };
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    browseFolders(path, controller.signal).then((result) => {
      if (controller.signal.aborted) return;
      setListing(result);
      setLoading(false);
    }).catch((err: Error) => {
      if (controller.signal.aborted) return;
      setError(err.message);
      setLoading(false);
    });
    return () => controller.abort();
  }, [path, attempt]);

  function navigate(next: string) {
    setAttempt((value) => value + 1);
    setListing(null);
    setError(null);
    setLoading(true);
    setAddress(next);
    setPath(next);
  }
  const parent = listing ? listing.parent : path.replace(/\/+$/, "").replace(/\/[^/]*$/, "") || "/";

  return createPortal(
    <dialog ref={dialog} className="folder-browser" aria-labelledby="folder-browser-title"
      onCancel={(event) => { event.preventDefault(); onClose(); }} onClick={(event) => { if (event.target === event.currentTarget) onClose(); }}>
      <div className="folder-browser-content">
        <div className="admin-modal-header">
          <h3 id="folder-browser-title">Add folder — {title}</h3>
          <button type="button" className="admin-modal-close" aria-label="Close folder browser" onClick={onClose}>×</button>
        </div>
        <p>Browse folders on the server, then select the folder you want to add.</p>
        <form className="folder-browser-address" onSubmit={(event) => { event.preventDefault(); navigate(address.trim()); }}>
          <button type="button" className="admin-btn admin-btn-secondary" disabled={!parent || path === "/"} onClick={() => navigate(parent)}>↑ Up</button>
          <input aria-label="Folder path" value={address} onChange={(event) => setAddress(event.target.value)} required pattern="/.*" />
          <button className="admin-btn admin-btn-secondary" type="submit">Go</button>
        </form>
        <div className="folder-browser-list" aria-busy={loading}>
          {loading && <p role="status">Loading folders…</p>}
          {error && <p role="alert" className="profile-field-error">{error}</p>}
          {listing?.folders.map((folder) => (
            <button type="button" className="folder-browser-entry" key={folder.path} onClick={() => navigate(folder.path)}>
              <span aria-hidden="true">📁</span><span>{folder.name}</span><span aria-hidden="true">›</span>
            </button>
          ))}
          {listing?.folders.length === 0 && <p>No subfolders. You can select this folder.</p>}
        </div>
        <div className="folder-browser-selection">Selected folder: <strong>{listing?.path ?? path}</strong></div>
        <div className="folder-browser-actions">
          <button type="button" className="admin-btn admin-btn-secondary" onClick={onClose}>Cancel</button>
          <button type="button" className="admin-btn" disabled={loading || !listing} onClick={() => { if (listing) onSelect(listing.path); }}>Select Folder</button>
        </div>
      </div>
    </dialog>, document.body,
  );
}
