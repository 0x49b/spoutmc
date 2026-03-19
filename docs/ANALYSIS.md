# Code Analysis Report

## Security Issues

### Critical
1.  **API Response Data Leakage (Sensitive Fields Exposure)**
    *   **File:** `internal/webserver/api/v1/user/user.go`
    *   **Issue:** The `getUsers` function returns `[]models.User` directly to the client. The `models.User` struct (defined in `internal/models/user.go`) contains the `Password` field (which stores the bcrypt hash). Exposing password hashes is a critical security vulnerability.
    *   **Recommendation:** Use a DTO (Data Transfer Object) like `UserResponse` (which is already defined) to map only safe fields before returning the response.

2.  **Arbitrary File Write (Potential RCE)**
    *   **File:** `internal/webserver/api/v1/server/server.go`
    *   **Issue:** The `updateServerFileHandler` allows writing content to any file within the server directory. While it checks `filepath.HasPrefix(fullPath, serverDir)`, there are no restrictions on *which* files can be modified. An attacker with access to this endpoint could overwrite the server binary, startup scripts, or libraries to achieve Remote Code Execution (RCE) when the server restarts.
    *   **Recommendation:** Implement a whitelist of allowed file extensions (e.g., `.properties`, `.yml`, `.json`, `.toml`, `.txt`) or specific filenames that can be edited. Block editing of executable files (e.g., `.jar`, `.sh`).

### High
3.  **Global Lock in API Handlers (DoS Vector)**
    *   **File:** `internal/webserver/api/v1/user/user.go`
    *   **Issue:** A global `lock` (sync.Mutex) is used in `getUser` and `getUsers` handlers (`lock.Lock()` / `defer lock.Unlock()`). This serializes all requests to these endpoints, meaning only one user fetch can be processed at a time for the entire server. This makes the application highly susceptible to Denial of Service (DoS) attacks and severely limits throughput.
    *   **Recommendation:** Remove the global lock. GORM's `*gorm.DB` is thread-safe.

4.  **Insecure Container Management**
    *   **File:** `cmd/spoutmc/main.go`
    *   **Issue:** `cleanupContainersNotInConfig` automatically deletes containers that are not present in the current configuration. If the configuration fails to load or is accidentally empty, this could lead to data loss (wiping out all game servers).
    *   **Recommendation:** Implement a safeguard/confirmation mechanism or "dry-run" check before deleting containers during startup.

5.  **Frontend Authentication Mock Usage in Production**
    *   **File:** `web/src/store/authStore.ts`
    *   **Issue:** The frontend authentication store (`authStore.ts`) relies entirely on mock data (`mockLogin`, `mockUsers`) and does not communicate with the backend API.
    *   **Impact:** The frontend is effectively disconnected from the backend's authentication system. Real users cannot log in, and permissions are not enforced against the backend.
    *   **Recommendation:** Refactor `authStore.ts` to make real HTTP requests to the backend (`/api/v1/user/...`) instead of using mocks.

### Medium
6.  **Hardcoded Secrets/Configuration**
    *   **File:** `internal/security/password.go`
    *   **Issue:** Password hashing uses a hardcoded bcrypt cost of 14. While 14 is secure, it is computationally expensive. If the server is under heavy load, this could contribute to DoS.
    *   **Recommendation:** Make the cost configurable or verify if 14 is appropriate for the target hardware.
    *   **File:** `internal/webserver/webserver.go`
    *   **Issue:** CORS configuration explicitly allows `http://localhost:3000` and `http://localhost:5173`. In a production environment, this should be configurable to allow the actual domain of the frontend.

7.  **Insecure Token Storage (Frontend)**
    *   **File:** `web/src/store/authStore.ts`
    *   **Issue:** Authentication tokens are stored in `localStorage` (`localStorage.setItem('auth_token', token)`).
    *   **Impact:** Tokens stored in `localStorage` are accessible to any JavaScript running on the page, making them vulnerable to Cross-Site Scripting (XSS) attacks.
    *   **Recommendation:** Store tokens in `httpOnly` cookies, or if using tokens, keep them in memory and use a refresh token flow (with the refresh token in an `httpOnly` cookie).

## Performance Issues

1.  **Inefficient Polling in SSE (Scalability)**
    *   **File:** `internal/webserver/api/v1/server/server.go` (`streamServers`)
    *   **Issue:** The `streamServers` handler creates a new `time.Ticker` for *each* connected client. Inside the loop, it calls `docker.GetNetworkContainers()` and `docker.GetContainerStats(container.ID)`.
    *   **Impact:** If 100 users connect, the server will query the Docker daemon 100 times every 2 seconds (plus stats calls for every container for every user). This is O(N*M) where N is clients and M is servers. This will likely overwhelm the Docker daemon and the application.
    *   **Recommendation:** Implement a broadcast pattern. A single background goroutine should poll Docker periodically and broadcast the results to all connected SSE clients.

2.  **Blocking External HTTP Calls**
    *   **File:** `internal/webserver/api/v1/user/user.go` (`createUser`)
    *   **Issue:** `createUser` calls `getMojangData`, which performs a synchronous HTTP GET request to `https://playerdb.co`.
    *   **Impact:** If the external service is slow or down, the API request will hang, potentially exhausting available goroutines/threads if many users try to register simultaneously.
    *   **Recommendation:** Use a context with a timeout for the external API call.

3.  **Image Processing in Request Path**
    *   **File:** `internal/webserver/api/v1/user/user.go` (`createUser`)
    *   **Issue:** Image processing (`processor.ProcessSkin`) happens synchronously within the request handler. Image processing is CPU intensive.
    *   **Recommendation:** Offload image processing to a background job/worker queue if possible, or ensure it's heavily rate-limited.

## Coding Principles & Best Practices

1.  **Architecture & Layering**
    *   **Observation:** The codebase mixes HTTP handling, business logic, and database access within the handlers (e.g., `internal/webserver/api/v1/user/user.go`).
    *   **Recommendation:** Adopt a Service/Repository pattern. Move DB logic to a repository layer and business logic (like calling Mojang API) to a service layer. The HTTP handlers should only deal with request parsing and response formatting.

2.  **Error Handling**
    *   **Observation:** Many errors are logged and then a generic 500 error is returned.
    *   **Recommendation:** While hiding internal details is good for security, the logging could be more structured, and specific error types (e.g., "User already exists") should return appropriate 4xx status codes.

3.  **Configuration Management**
    *   **Observation:** "velocity.toml", "server.properties" filenames are hardcoded strings in multiple places.
    *   **Recommendation:** Define these as constants in a configuration or constants package to avoid typos and ease maintenance.

4.  **Concurrency Control**
    *   **Observation:** The use of `sync.Mutex` (`lock`) in `internal/webserver/api/v1/server/server.go` and `user.go` suggests an attempt to manage concurrency, but it's applied too broadly (global lock).
    *   **Recommendation:** Use fine-grained locking if necessary, but rely on the database's transaction isolation and atomic operations where possible.

5.  **Testing**
    *   **Observation:** Tests for `internal/docker` are noted to be long-running and time out.
    *   **Recommendation:** Mock the Docker client interface to unit test the logic without needing a running Docker daemon. Keep integration tests separate.
