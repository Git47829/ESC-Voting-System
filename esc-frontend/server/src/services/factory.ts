import type { AuthService, ContestService, VotingService, ResultsService } from "./interfaces.js";
import { ProductionAuthService } from "./auth-service.js";
import { ProductionContestService } from "./contest-service.js";
import { ProductionVotingService } from "./voting-service.js";
import { ProductionResultsService } from "./results-service.js";

interface ServiceContainer {
  authService: AuthService;
  contestService: ContestService;
  votingService: VotingService;
  resultsService: ResultsService;
}

let serviceContainer: ServiceContainer | null = null;

export function initializeServices(): ServiceContainer {
  const container: ServiceContainer = {
    authService: new ProductionAuthService(),
    contestService: new ProductionContestService(),
    votingService: new ProductionVotingService(),
    resultsService: new ProductionResultsService()
  };

  serviceContainer = container;
  return serviceContainer;
}

export function getServices(): ServiceContainer {
  if (!serviceContainer) {
    return initializeServices();
  }
  return serviceContainer;
}

export function getAuthService(): AuthService {
  return getServices().authService;
}

export function getContestService(): ContestService {
  return getServices().contestService;
}

export function getVotingService(): VotingService {
  return getServices().votingService;
}

export function getResultsService(): ResultsService {
  return getServices().resultsService;
}
