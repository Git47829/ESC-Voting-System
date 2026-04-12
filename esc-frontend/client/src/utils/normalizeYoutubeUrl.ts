export const normalizeYoutubeUrl = (url: string): string => {
  if (!url) return url;
  const embedMatch = /youtube\.com\/embed\/([A-Za-z0-9_-]{11})/.exec(url);
  if (embedMatch) return `https://www.youtube.com/embed/${embedMatch[1]}`;
  const shortMatch = /youtu\.be\/([A-Za-z0-9_-]{11})/.exec(url);
  if (shortMatch) return `https://www.youtube.com/embed/${shortMatch[1]}`;
  const watchMatch = /youtube\.com\/(?:watch\?(?:.*&)?v=|shorts\/)([A-Za-z0-9_-]{11})/.exec(url);
  if (watchMatch) return `https://www.youtube.com/embed/${watchMatch[1]}`;
  return url;
};

