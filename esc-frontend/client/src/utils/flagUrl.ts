const alpha3ToAlpha2: Record<string, string> = {
  SWE: "se",
  DEU: "de",
  FRA: "fr",
  ITA: "it",
  ESP: "es",
  NOR: "no",
  UKR: "ua",
  NLD: "nl"
};

export const flagUrl = (countryId: string): string => {
  const normalized = countryId.trim().toUpperCase();
  const alpha2 = alpha3ToAlpha2[normalized] ?? normalized.slice(0, 2).toLowerCase();
  return `https://flagcdn.com/w80/${alpha2}.png`;
};
