import type { PlaybackSession } from './types';
import { cleanPlayerTitle } from './programTitle';

// A matching tune session supplies station and episode labels; hide them while tuning.
export function ProgramOverlay({ session }: { session?: PlaybackSession }) {
  if (!session) return null;
  const program = session.program;
  const episode = program?.mediaType === 'episode';
  const title = cleanPlayerTitle(program?.title ?? '');
  const series = episode ? cleanPlayerTitle(program.seriesTitle ?? '') : '';
  const episodeLabel = episode && program.episodeNumber !== undefined
    ? `S${String(program.seasonNumber ?? 0).padStart(2, '0')}E${String(program.episodeNumber).padStart(2, '0')} - ` : '';
  return (
    <div className="playback-program" aria-label="Now Playing">
      <div className="playback-program-station">
        <span className="playback-program-number">{session.channelNumber === undefined ? '' : String(session.channelNumber).padStart(3, '0')}</span>
        <span className="playback-program-channel">{session.channelName}</span>
      </div>
      <div className={'playback-program-info' + (episode ? ' is-episode' : '')}>
        {episode && <span className="playback-program-series">{series}</span>}
        <span className="playback-program-title">{episodeLabel}{title}</span>
      </div>
    </div>
  );
}
