// Re-export from service modules for backward compatibility
export { ApiError, extractErrorMessage } from "../services/error-handler";
export * as songApi from "../services/songs-api";
export * as votingApi from "../services/voting-api";
export * as authApi from "../services/auth-api";
export * as adminApi from "../services/admin-api";
export * as juryApi from "../services/jury-api";
export * as resultsApi from "../services/results-api";

// Import individual exports for convenience
import * as songApiModule from "../services/songs-api";
import * as votingApiModule from "../services/voting-api";
import * as authApiModule from "../services/auth-api";
import * as adminApiModule from "../services/admin-api";
import * as juryApiModule from "../services/jury-api";
import * as resultsApiModule from "../services/results-api";

// Re-export the api object for backward compatibility
export const api = {
  getSongs: songApiModule.getSongs,
  getVotes: votingApiModule.getVotes,
  getVoteState: votingApiModule.getVoteState,
  submitVote: votingApiModule.submitVote,
  getResults: resultsApiModule.getResults,
  getCountries: resultsApiModule.getCountries,
  getContestCurrent: resultsApiModule.getContestCurrent,
  authLogin: authApiModule.authLogin,
  authVerify: authApiModule.authVerify,
  login: authApiModule.login,
  logout: authApiModule.logout,
  session: authApiModule.session,
  adminOpen: adminApiModule.adminOpen,
  adminClose: adminApiModule.adminClose,
  adminDeleteVotes: adminApiModule.adminDeleteVotes,
  adminAddCountry: adminApiModule.adminAddCountry,
  adminAddArtist: adminApiModule.adminAddArtist,
  adminAddSong: adminApiModule.adminAddSong,
  adminStartContest: adminApiModule.adminStartContest,
  adminAdvanceContest: adminApiModule.adminAdvanceContest,
  juryVote: juryApiModule.juryVote,
  getJuryVoteState: juryApiModule.getJuryVoteState,
  getStats: resultsApiModule.getStats,
};
