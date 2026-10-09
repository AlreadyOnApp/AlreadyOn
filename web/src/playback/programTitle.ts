// Presentation only: metadata and filenames remain unchanged. Count Unicode
// characters, not UTF-16 code units, when applying the player's title limit.
export function cleanPlayerTitle(value: string): string {
  const clean = value
    .replace(/\.[a-z0-9]{2,4}$/i, match => /\.(mkv|mp4|avi|mov|m4v|ts|webm)$/i.test(match) ? '' : match)
    .replace(/\[[^\]]*\]|\{[^}]*\}/g, ' ')
    .replace(/\b(?:480p|720p|1080[pi]|2160p|remux|bluray|blu-ray|web-dl|webrip|hdtv|x26[45]|h26[45]|hevc|eac3|aac|dts)\b.*$/i, '')
    .replace(/[._]/g, ' ')
    .replace(/[^\p{L}\p{M}\p{N}\s,:'-]/gu, '')
    .replace(/\s+/g, ' ').trim();
  return Array.from(clean).slice(0, 50).join('').trimEnd();
}
