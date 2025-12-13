# Step 12: Stress Testing

This step involves creating and running a dedicated stress test to verify the server's stability and performance under high load, specifically targeting the requirement of supporting "several hundred concurrent users".

## Goals

1.  **High Concurrency Simulation:**
    *   Simulate 10000 concurrent users connecting to the server.
    *   Each simulated user should perform a sequence of standard FTP operations (e.g., Login, List, Upload small file, Download, Delete).

2.  **Stability Verification:**
    *   Ensure the server does not crash or deadlock under this load.
    *   Verify that error rates (e.g., connection timeouts, database locks) remain within acceptable limits.
    *   Monitor resource usage (memory/CPU) if possible/applicable during the test run.

3.  **Database Performance:**
    *   Confirm that SQLite WAL mode handles the concurrent write/read load effectively.
    *   Tune connection pooling settings if bottlenecks are observed.

## Implementation Details

*   Create a new test file `tests/stress_test.go` (or a standalone tool in `cmd/stress-test`).
*   Use goroutines to spawn simulated clients.
*   The test should be configurable for the number of users and duration/iterations.
*   **Note:** This test might be resource-intensive and should be run with care.

## Deliverables

*   A reproducible stress test suite/script.
*   Report of the stress test results (success rate, average time, any errors encountered) in the implementation notes.
*   Any necessary code adjustments to `pkg/db` or `pkg/vfs` resulting from the test findings.
