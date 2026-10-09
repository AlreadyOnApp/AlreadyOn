import type { TimeFormat } from './GuideSettingsContext';

export function formatClock(minutesSinceMidnight: number, format: TimeFormat): string {
  const totalMinutes = ((minutesSinceMidnight % (24 * 60)) + 24 * 60) % (24 * 60);
  const h24 = Math.floor(totalMinutes / 60);
  const mm = String(totalMinutes % 60).padStart(2, '0');

  if (format === '24h') {
    return `${String(h24).padStart(2, '0')}:${mm}`;
  }

  const period = h24 < 12 ? 'am' : 'pm';
  let h12 = h24 % 12;
  if (h12 === 0) h12 = 12;
  const hh = format === '12h-padded' ? String(h12).padStart(2, '0') : String(h12);
  return `${hh}:${mm}${period}`;
}

export function formatRange(startMinutes: number, endMinutes: number, format: TimeFormat): string {
  return `${formatClock(startMinutes, format)} - ${formatClock(endMinutes, format)}`;
}
