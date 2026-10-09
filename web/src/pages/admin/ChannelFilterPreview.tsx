// Debounced match feedback sits immediately before the channel save actions.
import { useEffect, useState } from 'react';
import { previewChannelFilter, type ChannelFilter } from '../../api/client';

const PREVIEW_DELAY_MS = 400;

export default function ChannelFilterPreview({ filter }: { filter: ChannelFilter }) {
  const [preview, setPreview] = useState<string | null>(null);
  // Re-count a moment after the last change so typing doesn't fire a request per key.
  useEffect(() => {
    let cancelled = false;
    const timer = window.setTimeout(() => {
      previewChannelFilter(filter)
        .then((r) => {
          if (cancelled) return;
          const shows = `${r.titles} ${r.titles === 1 ? 'show or movie' : 'shows and movies'}`;
          setPreview(r.titles === 0 ? 'Nothing in your library matches yet.' : `Matches ${shows} (${r.items} ${r.items === 1 ? 'file' : 'files'}).`);
        })
        .catch((err: Error) => {
          if (!cancelled) setPreview(err.message);
        });
    }, PREVIEW_DELAY_MS);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [filter]);

  return preview ? <p className="admin-section-hint channel-filter-preview" role="status">{preview}</p> : null;
}
