import { fetchJson } from "./http-client";
import type { Song } from "../types";

export const getSongs = async (): Promise<Song[]> => {
  const data = await fetchJson<{ payload: Song[] }>("/api/songs");
  return data.payload;
};
