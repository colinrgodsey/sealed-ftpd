# 3. Virtual File System (VFS)

Create an abstraction layer in Go to interact with the SQLite database as if it were a file system. This would include functions for creating, reading, updating, and deleting files and directories.

**Key Constraint:** The VFS implementation must enforce the strict **10MB file size limit**. Write operations exceeding this should fail.

## Testing

Unit tests will be written for the VFS implementation to ensure that all file and directory operations (create, read, update, delete) work correctly, including verifying that files > 10MB cannot be written.
