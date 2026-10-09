import type { Snapshot } from './types';

// Explicitly approved device-local measurements. No credentials belong here.
export const PLAYBACK_CACHE_KEY = 'alreadyon.playbackCapabilities';
export const CACHE_TTL = 24 * 60 * 60 * 1000;
export const CACHE_VERSION = 4;
let memory: Snapshot | null = null;

export function readSnapshot(): Snapshot | null {
  try {
    const raw = localStorage.getItem(PLAYBACK_CACHE_KEY);
    if (raw && raw.length < 64000) memory = JSON.parse(raw) as Snapshot;
  } catch { /* Private browsing can retain only the in-memory snapshot. */ }
  const s = memory;
  if (!s || s.version !== CACHE_VERSION || s.browser !== navigator.userAgent || !Number.isFinite(s.capturedAt) || s.capturedAt > Date.now() ||
      !s.capabilities || !Array.isArray(s.capabilities.video) || s.capabilities.video.length > 64 || !Array.isArray(s.capabilities.audio) || s.capabilities.audio.length > 32 ||
      !Array.isArray(s.capabilities.containers) || typeof s.capabilities.hls !== 'boolean') return null;
  if (!s.capabilities.containers.every(c => typeof c === 'string')) return null;
  if (s.capabilities.video.some(v => !v || typeof v.codec !== 'string' || typeof v.profile !== 'string' || typeof v.pixelFormat !== 'string' || !Number.isFinite(v.level) || !Number.isFinite(v.fps) || !Number.isFinite(v.width) || !Number.isFinite(v.height) || !Number.isFinite(v.bitrate))) return null;
  if (s.capabilities.audio.some(a => !a || typeof a.codec !== 'string' || !Number.isFinite(a.channels) || !Number.isFinite(a.sampleRate) || !Number.isFinite(a.bitrate))) return null;
  if (s.capabilities.video.some(v =>
    [v.container, v.colorTransfer, v.hdrMetadata, v.codecTag].some(value => value != null && typeof value !== 'string') ||
    [v.dolbyVisionProfile, v.dolbyVisionLevel].some(value => value != null && (!Number.isInteger(value) || value < 0)))) return null;
  if (s.capabilities.audio.some(a =>
    [a.container, a.profile].some(value => value != null && typeof value !== 'string') ||
    [a.spatial, a.hls].some(value => value != null && typeof value !== 'boolean'))) return null;
  // A damaged network entry only discards the measurement, not codec results.
  const n = s.network;
  if (n && (typeof n.origin !== 'string' || !Number.isFinite(n.capturedAt) || !Number.isFinite(n.mbps) || n.mbps <= 0 || (n.latencyMs != null && !Number.isFinite(n.latencyMs)))) delete s.network;
  return s;
}
export function writeSnapshot(snapshot: Snapshot) {
  memory = snapshot;
  try { localStorage.setItem(PLAYBACK_CACHE_KEY, JSON.stringify(snapshot)); } catch { /* Memory fallback. */ }
}

export function invalidateSnapshot() {
  memory = null;
  try { localStorage.removeItem(PLAYBACK_CACHE_KEY); } catch { /* Memory-only mode. */ }
}
