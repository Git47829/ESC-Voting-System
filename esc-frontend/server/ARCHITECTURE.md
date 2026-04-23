/**
 * ESC VOTING SYSTEM - SERVER ARCHITECTURE GUIDE
 * 
 * This guide explains the refactored server architecture following SOLID principles.
 */

// ============================================================================
// SERVICE ARCHITECTURE
// ============================================================================

/**
 * The server uses a service factory pattern to abstract business logic from routes.
 * This enables dependency injection and easy switching between mock/production implementations.
 * 
 * SERVICE INTERFACES (src/services/interfaces.ts):
 * - AuthService: Authentication, login, token verification
 * - ContestService: Contests, countries, songs management
 * - VotingService: Public and jury voting, vote state
 * - ResultsService: Vote aggregation and statistics
 * 
 * IMPLEMENTATIONS:
 * - Production services make API calls to upstream services
 * - Mock services use in-memory mock data
 * 
 * USAGE IN ROUTES:
 * 
 *   import { getAuthService, getVotingService } from "../services/factory.js";
 *   
 *   const authService = getAuthService();
 *   const votingService = getVotingService();
 *   
 *   // Services are automatically swapped based on isMockMode()
 *   const result = await authService.loginWithToken(role, token);
 *   const voteResult = await votingService.castPublicVote(songId, points);
 */

// ============================================================================
// MOCK DATA ARCHITECTURE
// ============================================================================

/**
 * Mock data is split into focused services following Single Responsibility Principle:
 * 
 * MockSongDataService (src/mock/song-data-service.ts):
 *   - Song CRUD operations
 *   - Vote casting for songs
 *   - Voting state management
 * 
 * MockContestDataService (src/mock/contest-data-service.ts):
 *   - Contest lifecycle (start, advance)
 *   - Country management
 * 
 * MockVoteDataService (src/mock/vote-data-service.ts):
 *   - Vote result aggregation
 *   - Rankings and statistics
 * 
 * MockDataService (src/mock/index.ts):
 *   - Facade that coordinates all mock services
 *   - Maintains backward compatibility with original API
 */

// ============================================================================
// MIDDLEWARE ORGANIZATION
// ============================================================================

/**
 * Middleware is organized by concern:
 * 
 * src/middleware/auth.ts:
 *   - requireRole(role): Guards routes requiring specific user role
 * 
 * src/middleware/csrf.ts:
 *   - csrfProtection: CSRF token validation middleware
 *   - csrfTokenEndpoint: Endpoint to get/generate CSRF tokens
 * 
 * src/middleware/error-handler.ts:
 *   - errorHandler: Global error handling for all unhandled errors
 *                   (includes Axios error detection)
 * 
 * src/routes/health.ts:
 *   - healthCheck: Health check endpoint for monitoring
 */

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

/**
 * Common utilities are centralized in src/utils/helpers.ts:
 * 
 * authHeaders(session): Creates authorization headers from session
 * decodeVoteStateCookie(value): Decodes vote state from cookie
 * toInt(value, fallback): Safe integer conversion
 * normalizeYoutubeUrl(url): Normalizes various YouTube URL formats
 * getJuryVoteState(session): Gets or initializes jury vote state
 * juryPointValues: Constant array of allowed jury point values
 * 
 * IMPORT:
 *   import { authHeaders, toInt, ... } from "../utils/helpers.js";
 *   // or use barrel export:
 *   import { toInt } from "../utils/index.js";
 */

// ============================================================================
// SESSION STATE STRUCTURE
// ============================================================================

/**
 * SessionData is now composed of focused interfaces:
 * 
 * AuthSession:
 *   - role: 'admin' | 'jury'
 *   - token: JWT or auth token
 *   - email: User email (for email-based auth)
 *   - pendingEmail, pendingRole, pendingPassword: OTP flow state
 * 
 * VoteSession:
 *   - voteState: { votesRemaining, votesCast }
 *   - juryVoteState: { token, votesCast }
 * 
 * SecuritySession:
 *   - csrfToken: CSRF token for form submissions
 * 
 * SessionData extends all three for complete session state.
 * This composition approach makes it clear which parts of session
 * each middleware/route depends on.
 */

// ============================================================================
// ADDING NEW ENDPOINTS
// ============================================================================

/**
 * When adding a new endpoint:
 * 
 * 1. Add to appropriate service interface (src/services/interfaces.ts)
 * 2. Implement in both Production and Mock service classes
 * 3. Add route handler that calls the service
 * 4. Services handle all business logic, routes handle HTTP
 * 
 * EXAMPLE:
 *   // 1. Add to interface
 *   export interface ContestService {
 *     getSchedule(): Promise<Schedule>;
 *   }
 *   
 *   // 2. Implement
 *   class ProductionContestService {
 *     async getSchedule() {
 *       const response = await upstream.get("/schedule");
 *       return response.data.payload;
 *     }
 *   }
 *   
 *   // 3. Add route
 *   apiRouter.get("/schedule", asyncHandler(async (_req, res) => {
 *     try {
 *       const schedule = await contestService.getSchedule();
 *       res.json({ payload: schedule });
 *     } catch (error) {
 *       res.status(502).json({ error: "Failed to fetch schedule" });
 *     }
 *   }));
 */

// ============================================================================
// KEY FILES REFERENCE
// ============================================================================

/**
 * Entry Point:
 *   src/index.ts - App initialization, middleware setup, routing
 * 
 * Routes:
 *   src/routes/api.ts - Main API endpoints
 *   src/routes/health.ts - Health check
 * 
 * Services:
 *   src/services/interfaces.ts - Service contracts
 *   src/services/factory.ts - Service creation & DI
 *   src/services/{auth,voting,contest,results}-service.ts
 * 
 * Mock Data:
 *   src/mock/index.ts - Main mock service
 *   src/mock/{song,contest,vote}-data-service.ts - Focused mock services
 * 
 * Middleware:
 *   src/middleware/{auth,csrf,error-handler}.ts
 * 
 * Configuration:
 *   src/config.ts - Config parsing and validation
 *   src/session.d.ts - Session type definitions
 *   src/types.ts - Domain type definitions
 *   src/upstream.ts - Upstream HTTP client
 * 
 * Utilities:
 *   src/utils/helpers.ts - Common utility functions
 */

export {};
