// Display cleanup never changes the stored title or the media's filesystem path.
export function mediaTitle(value: string, preserveYear = false): string {
  let title = value.replace(/\[[^\]]*\]/g, '');
  if (!preserveYear) title = title.replace(/\(\d{4}\)/g, '');
  return title.replace(/\s+/g, ' ').trim();
}
