// Fix what the server couldn't work out about one file, or set the same
// thing on several at once (#105). Single mode starts from what is already
// known; bulk mode starts blank and sends only what is filled in.
import { useEffect, useState } from 'react';
import {
  lookupTMDbItem,
  getCertifications,
  getGenres,
  updateLibraryItem,
  updateLibraryItems,
  type Genre,
  type LibraryItemPatch,
  type UnmatchedItem,
} from '../../api/client';
import InfoModal from './InfoModal';
import MultiSelectDropdown from './MultiSelectDropdown';

type Certs = { movie: string[]; tv: string[] };

const toInt = (s: string): number | null => (s.trim() === '' ? null : Number(s));
const isInt = (s: string) => s.trim() === '' || /^-?\d+$/.test(s.trim());
const sameSet = (a: string[], b: string[]) => a.length === b.length && a.every((v) => b.includes(v));

export default function EditItemsDialog({ items, onClose, onSaved }: { items: UnmatchedItem[]; onClose: () => void; onSaved: (updated: number) => void }) {
  const single = items.length === 1;
  const first = items[0];

  const [tmdbId, setTMDbId] = useState(single && first.tmdbId ? String(first.tmdbId) : '');
  const [genres, setGenres] = useState<Genre[]>([]);
  const [certs, setCerts] = useState<Certs>({ movie: [], tv: [] });
  const [mediaType, setMediaType] = useState<string>(single ? first.mediaType : '');
  const [title, setTitle] = useState(single ? first.title : '');
  const [seriesTitle, setSeriesTitle] = useState(single ? (first.seriesTitle ?? '') : '');
  const [year, setYear] = useState(single && first.year ? String(first.year) : '');
  const [season, setSeason] = useState(single && first.season !== null ? String(first.season) : '');
  const [episode, setEpisode] = useState(single && first.episode !== null ? String(first.episode) : '');
  const [genreIds, setGenreIds] = useState<string[]>(single ? (first.genres ?? []) : []);
  const [cert, setCert] = useState(single ? (first.certification ?? '') : '');
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    getGenres()
      .then(setGenres)
      .catch(() => undefined);
    getCertifications()
      .then(setCerts)
      .catch(() => undefined);
  }, []);

  const allSameType = items.every((i) => i.mediaType === first.mediaType);
  const effectiveType = mediaType || (allSameType ? first.mediaType : 'mixed');
  const showEpisodeFields = effectiveType === 'episode' || effectiveType === 'mixed';

  const buildPatch = (): LibraryItemPatch | string => {
    for (const [label, v] of [
      ['Year', year],
      ['Season', season],
      ['Episode', episode],
    ] as const) {
      if (!isInt(v)) return `${label} must be a whole number.`;
    }
    const patch: LibraryItemPatch = {};
    if (single) {
      if (mediaType !== first.mediaType) patch.mediaType = mediaType as 'movie' | 'episode';
      if (title.trim() !== first.title) patch.title = title.trim();
      if (mediaType === 'episode' && seriesTitle.trim() === '') return 'A TV Episode needs a show name.';
      if (mediaType === 'episode' && seriesTitle.trim() !== (first.seriesTitle ?? '')) patch.seriesTitle = seriesTitle.trim() || null;
      if (year.trim() !== (first.year ? String(first.year) : '')) patch.year = toInt(year);
      if (mediaType === 'episode') {
        if (season.trim() !== (first.season !== null ? String(first.season) : '')) patch.season = toInt(season);
        if (episode.trim() !== (first.episode !== null ? String(first.episode) : '')) patch.episode = toInt(episode);
      }
      if (!sameSet(genreIds, first.genres ?? [])) patch.genres = genreIds;
      if (cert !== (first.certification ?? '')) patch.certification = cert || null;
      if (title.trim() === '') return 'A title is required.';
    } else {
      if (mediaType) patch.mediaType = mediaType as 'movie' | 'episode';
      if (seriesTitle.trim()) patch.seriesTitle = seriesTitle.trim();
      if (year.trim()) patch.year = toInt(year);
      if (season.trim()) patch.season = toInt(season);
      if (genreIds.length > 0) patch.genres = genreIds;
      if (cert) patch.certification = cert;
    }
    return Object.keys(patch).length === 0 ? 'Nothing has changed.' : patch;
  };

  const handleSave = async () => {
    const patch = buildPatch();
    if (typeof patch === 'string') {
      setError(patch);
      return;
    }
    setSaving(true);
    setError(null);
    try {
      const result = single
        ? await updateLibraryItem(first.id, patch)
        : await updateLibraryItems(
            items.map((i) => i.id),
            patch,
          );
      onSaved(result.updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save.');
      setSaving(false);
    }
  };

  const genreOptions = genres.map((g) => ({ value: g.id, label: g.label }));

  return (
    <InfoModal open title={single ? 'Edit File' : `Edit ${items.length} files`} onClose={onClose}>
      {single ? (
        <p className="admin-section-hint edit-items-path">{first.path}</p>
      ) : (
        <p className="admin-section-hint">Fill in only what should apply to all {items.length} files. Empty fields leave each file as it is.</p>
      )}

      {single && (
        <div className="admin-field-row">
          <label htmlFor="tmdb-id">TMDb ID ({first.mediaType === 'episode' ? 'Show' : 'Movie'})</label>
          <input id="tmdb-id" type="number" min={1} value={tmdbId} onChange={(e) => setTMDbId(e.target.value)} />
          <button
            className="admin-btn"
            disabled={saving || !/^[1-9]\d*$/.test(tmdbId)}
            onClick={async () => {
              setSaving(true);
              setError(null);
              try {
                const result = await lookupTMDbItem(first.id, Number(tmdbId));
                onSaved(result.updated);
              } catch (e) {
                setError((e as Error).message);
                setSaving(false);
              }
            }}
          >
            Look Up and Apply
          </button>
          <p className="admin-section-hint">Applies TMDb metadata immediately. For TV, updates this show’s episodes. Unsaved edits below are not applied.</p>
        </div>
      )}
      <div className="admin-field-row">
        <label>Type</label>
        <select value={mediaType} onChange={(e) => setMediaType(e.target.value)}>
          {!single && <option value="">Leave Unchanged</option>}
          <option value="movie">Movie</option>
          <option value="episode">TV Episode</option>
        </select>
      </div>
      {single && (
        <div className="admin-field-row">
          <label>Title</label>
          <input type="text" value={title} maxLength={200} onChange={(e) => setTitle(e.target.value)} />
        </div>
      )}
      {showEpisodeFields && (
        <>
          <div className="admin-field-row">
            <label>Show Name</label>
            <input
              type="text"
              value={seriesTitle}
              maxLength={200}
              placeholder={single ? '' : 'Same Show For Every File'}
              onChange={(e) => setSeriesTitle(e.target.value)}
            />
          </div>
          <div className="admin-field-row">
            <label>Season</label>
            <input type="text" inputMode="numeric" value={season} onChange={(e) => setSeason(e.target.value)} />
          </div>
          {single && (
            <div className="admin-field-row">
              <label>Episode</label>
              <input type="text" inputMode="numeric" value={episode} onChange={(e) => setEpisode(e.target.value)} />
            </div>
          )}
        </>
      )}
      <div className="admin-field-row">
        <label>Year</label>
        <input type="text" inputMode="numeric" value={year} maxLength={4} onChange={(e) => setYear(e.target.value)} />
      </div>
      <div className="admin-field-row">
        <label>Genres</label>
        <MultiSelectDropdown
          label="Genres"
          placeholder={single ? 'No Genres' : 'Leave Unchanged'}
          options={genreOptions}
          selected={genreIds}
          onChange={setGenreIds}
        />
      </div>
      <div className="admin-field-row">
        <label>Rating</label>
        <select value={cert} onChange={(e) => setCert(e.target.value)}>
          <option value="">{single ? 'Not Rated (Automatic)' : 'Leave Unchanged'}</option>
          {(effectiveType === 'movie' || effectiveType === 'mixed') && (
            <optgroup label="Movie Ratings">
              {certs.movie.map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </optgroup>
          )}
          {(effectiveType === 'episode' || effectiveType === 'mixed') && (
            <optgroup label="TV Ratings">
              {certs.tv.map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </optgroup>
          )}
        </select>
      </div>
      <p className="admin-section-hint">
        Genres and ratings you set here are kept over automatic lookups. Correcting the title, show or year makes the next refresh look the file up again.
      </p>

      {error && <p className="profile-field-error">{error}</p>}
      <div className="admin-table-actions">
        <button className="admin-btn" onClick={() => void handleSave()} disabled={saving}>
          {saving ? 'Saving…' : 'Save'}
        </button>
        <button className="admin-btn admin-btn-secondary" onClick={onClose} disabled={saving}>
          Cancel
        </button>
      </div>
    </InfoModal>
  );
}
