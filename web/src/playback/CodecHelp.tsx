import { useEffect, useId, useRef } from 'react';

// Browser APIs report unsupported formats, not whether an OS package is missing.
export const BROWSER_CODEC_MESSAGE = 'The stream could not play. The browser does not support the required video or audio codecs.';

// Native dialog supplies focus trapping, Escape dismissal and top-layer display,
// including when the player is fullscreen. Keep it inside the fullscreen subtree.
export function CodecHelp({ onClose, nativeServer = false, missingCodecs = [] }: { onClose: () => void; nativeServer?: boolean; missingCodecs?: string[] }) {
  const dialog = useRef<HTMLDialogElement>(null);
  const title = useId();
  const explanation = useId();
  useEffect(() => {
    const element = dialog.current!;
    const trigger = document.activeElement;
    element.showModal();
    return () => {
      element.close();
      if (trigger instanceof HTMLElement && trigger.isConnected) trigger.focus();
    };
  }, []);

  return (
    <dialog ref={dialog} className="codec-help" aria-labelledby={title} aria-describedby={explanation} onCancel={onClose}
      onKeyDown={(event) => {
        event.stopPropagation();
        // Keep Tab inside the help instead of moving into browser chrome.
        if (event.key !== 'Tab') return;
        const items = event.currentTarget.querySelectorAll<HTMLElement>('button:not(:disabled), a[href]');
        const first = items[0];
        const last = items[items.length - 1];
        if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus(); }
        else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
      }} onKeyUp={(event) => event.stopPropagation()}>
      <div className="codec-help-header">
        <h2 id={title}>{nativeServer ? 'Set Up Local Server Playback' : 'Help With Browser Playback'}</h2>
        <button type="button" onClick={onClose} aria-label="Close playback help" autoFocus>Close</button>
      </div>
      {nativeServer ? <>
        <p id={explanation}>
          Your local development server runs outside the AlreadyOn Docker image and uses FFmpeg installed on the server computer.
          That FFmpeg installation cannot decode this title{missingCodecs.length > 0 ? ` (${missingCodecs.join(', ')})` : ''}.
          The official Docker image includes the required media packages; native development needs those packages installed locally.
          Follow the instructions for the computer running the server, not the device viewing this page. Installing browser codecs will not fix this error.
        </p>
        <section>
          <h3>Windows</h3>
          <ol>
            <li>Install a full FFmpeg build using <code>winget install --id Gyan.FFmpeg --exact</code>, or use a Windows build linked from the official FFmpeg download page.</li>
            <li>Ensure its <code>bin</code> directory is on the server process's PATH and includes both <code>ffmpeg</code> and <code>ffprobe</code>. Reopen your terminal or VS Code after changing PATH.</li>
            <li>If the server runs in WSL, install FFmpeg inside that Linux distribution using its instructions below instead.</li>
          </ol>
          <p><a href="https://ffmpeg.org/download.html" target="_blank" rel="noopener noreferrer">FFmpeg Downloads</a></p>
        </section>
        <section>
          <h3>macOS</h3>
          <ol>
            <li>With Homebrew installed, run <code>brew install ffmpeg</code>. For an existing Homebrew installation of FFmpeg, run <code>brew upgrade ffmpeg</code>.</li>
            <li>Ensure the server's terminal can run both <code>ffmpeg</code> and <code>ffprobe</code> from the Homebrew installation.</li>
          </ol>
          <p><a href="https://brew.sh/" target="_blank" rel="noopener noreferrer">Install Homebrew</a>
            {' · '}<a href="https://formulae.brew.sh/formula/ffmpeg" target="_blank" rel="noopener noreferrer">Homebrew FFmpeg</a></p>
        </section>
        <section>
          <h3>Ubuntu / Debian-based Linux</h3>
          <ol>
            <li>On the server computer, run <code>sudo apt update</code>, then <code>sudo apt install ffmpeg</code>.</li>
            <li>If FFmpeg is unavailable on Ubuntu, enable the Universe repository through Software &amp; Updates, refresh the package list, and install it again.</li>
          </ol>
          <p><a href="https://ffmpeg.org/download.html" target="_blank" rel="noopener noreferrer">FFmpeg Distribution Packages</a>
            {' · '}<a href="https://wiki.debian.org/ffmpeg" target="_blank" rel="noopener noreferrer">Debian FFmpeg Installation</a></p>
        </section>
        <section>
          <h3>Fedora / Red Hat / CentOS-based Linux</h3>
          <ol>
            <li>Follow RPM Fusion's configuration guide to enable its Free and Nonfree repositories for your exact distribution and release.</li>
            <li>On Fedora, replace the limited build with <code>sudo dnf swap ffmpeg-free ffmpeg --allowerasing</code>. Review the proposed package changes before accepting.</li>
            <li>For Red Hat or CentOS-based Enterprise Linux, follow the matching RPM Fusion instructions, including EPEL prerequisites, then install full FFmpeg using its multimedia guide. Do not use Fedora repository packages.</li>
          </ol>
          <p><a href="https://rpmfusion.org/Configuration" target="_blank" rel="noopener noreferrer">RPM Fusion Setup By Distribution</a>
            {' · '}<a href="https://rpmfusion.org/Howto/Multimedia" target="_blank" rel="noopener noreferrer">Multimedia Codec Installation</a></p>
        </section>
        <p className="codec-help-note">Check <code>ffmpeg -hide_banner -codecs</code> for a D (decoding) flag beside the missing formats. Restart the local server after installation; on the Linux development setup, rerun <code>make preview</code>. Decoder availability is checked once per server process. Native Windows and macOS development are not currently validated.</p>
      </> : <>
      <p id={explanation}>
        Your browser cannot play the video or audio format offered by this stream. It needs a decoder, also called a codec, to turn that
        format into pictures and sound. Some browsers use operating-system codecs; others include their own. Missing or unsupported codecs
        can cause this error, though damaged media can also fail decoding. Follow the instructions for the computer running your browser,
        then fully quit and reopen the browser. Installing browser support does not repair a missing decoder on the AlreadyOn server.
      </p>
      <section>
        <h3>Windows</h3>
        <ol>
          <li>Install available Windows updates and update your browser.</li>
          <li>On Windows N editions, open Settings, find Optional features, and install the Media Feature Pack. Restart Windows if prompted.</li>
          <li>For other missing formats, use Microsoft's guide to find the matching Microsoft Store codec extension. An extension's availability in your browser depends on the browser and hardware.</li>
        </ol>
        <p><a href="https://support.microsoft.com/en-us/windows/apps/windowsmediaplayer/codecs-faq" target="_blank" rel="noopener noreferrer">Microsoft codec installation guide</a>
          {' · '}<a href="https://support.microsoft.com/en-us/windows/experience/platform-variants/media-feature-pack-for-windows-n" target="_blank" rel="noopener noreferrer">Media Feature Pack for N editions</a></p>
      </section>
      <section>
        <h3>macOS</h3>
        <ol>
          <li>Open System Settings → General → Software Update and install available updates. Safari and Apple's media support update with macOS.</li>
          <li>If you use Firefox or Chrome, update that browser too. Try the same video in Safari if your usual browser still cannot play it.</li>
          <li>macOS has no separate general-purpose codec pack to install for Safari. Format support also depends on your Mac's hardware; use conversion if the format remains unsupported.</li>
        </ol>
        <p><a href="https://support.apple.com/en-us/102665" target="_blank" rel="noopener noreferrer">Update Safari</a>
          {' · '}<a href="https://support.apple.com/en-us/116944" target="_blank" rel="noopener noreferrer">Apple media compatibility</a></p>
      </section>
      <section>
        <h3>Ubuntu / Debian-based Linux</h3>
        <ol>
          <li>Update your browser and refresh the package list with <code>sudo apt update</code>.</li>
          <li>For a native browser on Ubuntu, install the common media packages with <code>sudo apt install ffmpeg ubuntu-restricted-extras</code>. Review any license prompts.</li>
          <li>On Debian, use <code>sudo apt install ffmpeg</code>. The Ubuntu-specific restricted-extras package does not apply to Debian.</li>
          <li>For Snap or Flatpak browsers, update the browser and its runtime through its software store; host packages alone may not be available inside that browser.</li>
        </ol>
        <p><a href="https://help.ubuntu.com/community/RestrictedFormats" target="_blank" rel="noopener noreferrer">Ubuntu restricted formats</a>
          {' · '}<a href="https://wiki.debian.org/ffmpeg" target="_blank" rel="noopener noreferrer">Debian FFmpeg installation</a></p>
      </section>
      <section>
        <h3>Fedora / Red Hat / CentOS-based Linux</h3>
        <ol>
          <li>Update your browser and system packages through the software updater.</li>
          <li>For a native browser, follow RPM Fusion's configuration guide for your exact distribution and release, then its multimedia guide to install full codec support.</li>
          <li>Fedora and Enterprise Linux (Red Hat/CentOS) use different repository setup steps. Follow the matching branch of the guide, including any EPEL prerequisites for Enterprise Linux.</li>
          <li>If your browser is a Flatpak, update its app and runtime in the software store. Installing host RPM codecs alone may not update the sandboxed browser.</li>
        </ol>
        <p><a href="https://rpmfusion.org/Configuration" target="_blank" rel="noopener noreferrer">RPM Fusion setup by distribution</a>
          {' · '}<a href="https://rpmfusion.org/Howto/Multimedia" target="_blank" rel="noopener noreferrer">Multimedia codec installation</a></p>
      </section>
      <p className="codec-help-note">Guides open in a new tab. After updating, restart your browser. You can also try another browser.</p>
      </>}
    </dialog>
  );
}
