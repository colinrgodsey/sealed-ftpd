# 6. Write FTP Commands

Implement write-related FTP commands like `STOR`, `DELE`, `MKD`, `RMD`.

**Constraint:** The `STOR` command implementation must handle the **10MB limit** gracefully, returning an appropriate FTP error code if the limit is exceeded.

## Testing

Unit tests will be written for each write-related FTP command (`STOR`, `DELE`, `MKD`, `RMD`) to ensure they function as expected, specifically testing that oversized uploads are rejected.
