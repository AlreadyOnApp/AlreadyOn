import { XhrLoader, type LoaderCallbacks, type LoaderConfiguration, type LoaderContext, type PlaylistLoaderContext } from 'hls.js';

// Each attached stream owns its history; signed playlist URLs never leave memory.
// Only unchanged live media playlists get two extra half-second rechecks. New
// content, end lists and failures go straight back to hls.js for normal handling.
export function responsivePlaylistLoader() {
  const previous = new Map<string, string>();
  return class extends XhrLoader {
    declare context: PlaylistLoaderContext | null;

    override load(context: LoaderContext, config: LoaderConfiguration, callbacks: LoaderCallbacks<LoaderContext>) {
      let rechecks = 0;
      super.load(context, config, {
        ...callbacks,
        onSuccess: (response, stats, loadedContext, networkDetails) => {
          const body = response.data;
          const live = typeof body === 'string' && body.includes('#EXTINF:') && !body.includes('#EXT-X-ENDLIST') && !body.includes('#EXT-X-PART');
          if (live && previous.get(context.url) === body && rechecks < 2) {
            rechecks++;
            // Use the inherited timer so abort/destroy cancels a pending recheck
            // on channel changes. HTTP failures keep the loader's normal handling.
            this.retryTimeout = self.setTimeout(() => {
              this.resetInternalLoader();
              this.loadInternal();
            }, 500);
            return;
          }
          if (live) previous.set(context.url, body);
          callbacks.onSuccess(response, stats, loadedContext, networkDetails);
        },
      });
    }
  };
}
