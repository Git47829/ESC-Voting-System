export const flagUrl = (countryId: string): string =>
  `https://flagcdn.com/w80/${countryId.slice(0, 2).toLowerCase()}.png`;

