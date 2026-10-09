// Scanned files no channel airs, with why, so an admin can fill in what is
// missing (#104). Sits at the bottom of the Media tab.
import { useCallback, useEffect, useState } from 'react';
import { getUnmatched, refreshSchedule, type UnmatchedItem } from '../../api/client';
import EditItemsDialog from './EditItemsDialog';
import './AdminShared.css';

const PAGE_SIZE = 50;

const fileName = (path: string) => path.split('/').pop() || path;

export default function UnmatchedContentCard() {
  const [page, setPage] = useState(0);
  const [total, setTotal] = useState<number | null>(null);
  const [items, setItems] = useState<UnmatchedItem[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Map<number, UnmatchedItem>>(new Map());
  const [editing, setEditing] = useState<UnmatchedItem[] | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const [reload, setReload] = useState(0);

  const load = useCallback(() => setReload((n) => n + 1), []);

  useEffect(() => {
    let cancelled = false;
    const controller = new AbortController();
    getUnmatched(PAGE_SIZE, page * PAGE_SIZE, controller.signal)
      .then((r) => {
        if (cancelled) return;
        // Everything on this page was fixed: step back to the last page that still has rows.
        if (r.items.length === 0 && r.total > 0 && page > 0) {
          setPage(Math.max(0, Math.ceil(r.total / PAGE_SIZE) - 1));
          return;
        }
        setItems(r.items);
        setTotal(r.total);
        setError(null);
      })
      .catch((err: Error) => {
        if (!cancelled) setError(err.message);
      });
    return () => {
      cancelled = true;
      controller.abort();
    };
  }, [page, reload]);

  // A selection can span pages, so whole items are kept, not just ids.
  const toggle = (item: UnmatchedItem) =>
    setSelected((prev) => {
      const next = new Map(prev);
      if (next.has(item.id)) next.delete(item.id);
      else next.set(item.id, item);
      return next;
    });

  const allOnPage = items.length > 0 && items.every((i) => selected.has(i.id));
  const toggleAll = () =>
    setSelected((prev) => {
      const next = new Map(prev);
      for (const i of items) {
        if (allOnPage) next.delete(i.id);
        else next.set(i.id, i);
      }
      return next;
    });

  const selectedCount = selected.size;

  const handleSaved = (updated: number) => {
    setEditing(null);
    setSelected(new Map());
    setNotice(`Saved ${updated} ${updated === 1 ? 'file' : 'files'}. Apply the changes to look them up again and put them on channels.`);
    load();
  };

  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      const r = await refreshSchedule();
      setNotice(`Changes applied: rebuilt ${r.channelsRefreshed} channel lineup(s).`);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to refresh the schedule');
    } finally {
      setRefreshing(false);
    }
  };

  if (total === null && !error) return (
    <>
      <hr className="admin-divider unmatched-break" />
      <div className="admin-card">
        <h3>Unmatched Content</h3>
        <p className="admin-section-hint" role="status">Loading unmatched media…</p>
      </div>
    </>
  );

  if (total === 0 && !notice) {
    return (
      <>
        <hr className="admin-divider unmatched-break" />
        <div className="admin-card">
          <h3>Unmatched Content</h3>
          <p className="admin-section-hint">Every scanned file is on a channel.</p>
        </div>
      </>
    );
  }

  const pageCount = Math.max(1, Math.ceil((total ?? 0) / PAGE_SIZE));
  const from = page * PAGE_SIZE + 1;
  const to = page * PAGE_SIZE + items.length;

  return (
    <>
      <hr className="admin-divider unmatched-break" />
      <div className="admin-card">
        <h3>Unmatched Content</h3>
        <p className="admin-section-hint">
          These scanned files aren't on any channel, usually because their genres, year or rating are missing or don't fit a channel. Correct what's wrong, then
          refresh the schedule.
        </p>
        {error && <p className="profile-field-error">{error}</p>}
        {notice && (
          <p className="admin-section-hint unmatched-notice">
            {notice}{' '}
            <button className="admin-btn" onClick={() => void handleRefresh()} disabled={refreshing}>
              {refreshing ? 'Applying…' : 'Apply Changes'}
            </button>
          </p>
        )}

        {items.length > 0 && (
          <>
            <div className="admin-table-actions" style={{ marginBottom: 12 }}>
              <button className="admin-btn" disabled={selectedCount < 2} onClick={() => setEditing([...selected.values()])}>
                Edit selected{selectedCount > 0 ? ` (${selectedCount})` : ''}
              </button>
              {selectedCount === 1 && <span className="admin-section-hint">Select one more file, or use that row's Edit button.</span>}
            </div>
            <table className="admin-table">
              <thead>
                <tr>
                  <th>
                    <input type="checkbox" checked={allOnPage} onChange={toggleAll} aria-label="Select All Files On This Page" />
                  </th>
                  <th>File</th>
                  <th>Why It Isn't On A Channel</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {items.map((i) => (
                  <tr key={i.id}>
                    <td>
                      <input type="checkbox" checked={selected.has(i.id)} onChange={() => toggle(i)} aria-label={`Select ${fileName(i.path)}`} />
                    </td>
                    <td>
                      <div>{fileName(i.path)}</div>
                      <div className="admin-section-hint unmatched-path">{i.path}</div>
                      {i.isManual && <span className="admin-badge admin-badge-on">Edited</span>}
                    </td>
                    <td>{i.reason}</td>
                    <td className="admin-table-actions">
                      <button className="admin-btn admin-btn-secondary" onClick={() => setEditing([i])}>
                        Edit
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <div className="admin-table-actions unmatched-pager">
              <span className="admin-section-hint">
                Showing {from}-{to} of {total}
              </span>
              <button className="admin-btn admin-btn-secondary" disabled={page === 0} onClick={() => setPage((p) => p - 1)}>
                Previous
              </button>
              <button className="admin-btn admin-btn-secondary" disabled={page + 1 >= pageCount} onClick={() => setPage((p) => p + 1)}>
                Next
              </button>
            </div>
          </>
        )}

        {editing && <EditItemsDialog items={editing} onClose={() => setEditing(null)} onSaved={handleSaved} />}
      </div>
    </>
  );
}
