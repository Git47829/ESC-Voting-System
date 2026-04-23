import type { AuthService, ContestService, VotingService, ResultsService } from "./interfaces.js";
import { ProductionAuthService, MockAuthService } from "./auth-service.js";
import { ProductionContestService, MockContestService } from "./contest-service.js";
import { ProductionVotingService, MockVotingService } from "./voting-service.js";
import { ProductionResultsService, MockResultsService } from "./results-service.js";
import { isMockMode } from "../config.js";

interface ServiceContainer {
  authService: AuthService;
  contestService: ContestService;
  votingService: VotingService;
  resultsService: ResultsService;
}

let serviceContainer: ServiceContainer | null = null;

export function initializeServices(): ServiceContainer {
  const mockMode = isMockMode();

  const container: ServiceContainer = {
    authService: mockMode ? new MockAuthService() : new ProductionAuthService(),
    contestService: mockMode ? new MockContestService() : new ProductionContestService(),
    votingService: mockMode ? new MockVotingService() : new ProductionVotingService(),
    resultsService: mockMode ? new MockResultsService() : new ProductionResultsService()
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
