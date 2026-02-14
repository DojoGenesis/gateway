# Tool Security Considerations

## System Tools

### run_command

**Risk Level:** HIGH

The `run_command` tool executes arbitrary shell commands on the system. This tool must be used with extreme caution as it can perform any operation that the running process has permissions to execute.

#### Security Measures Implemented

1. **Command Validation**
   - Blocks dangerous command patterns that could cause system damage
   - Patterns blocked include:
     - `rm -rf /` and `rm -rf /*` (system root deletion)
     - `mkfs` (filesystem formatting)
     - `dd if=/dev/zero` (disk wiping)
     - `:(){ :|:& };:` (fork bomb)
     - `chmod -R 777 /` or `chmod -R 666 /` (dangerous permission changes on root)

2. **Timeout Protection**
   - Default timeout: 30 seconds
   - Maximum timeout: 5 minutes (300 seconds)
   - Prevents long-running or hanging processes from consuming resources

3. **Context Cancellation**
   - Supports context cancellation for graceful shutdown
   - Commands are killed when context is cancelled

4. **Output Capture**
   - Both stdout and stderr are captured
   - Exit codes are returned for proper error handling

5. **Working Directory Isolation**
   - Commands can be run in specified working directories
   - Helps limit scope of operations

#### Known Limitations

1. **Allowlist vs Blocklist**
   - Current implementation uses a blocklist approach
   - A more secure approach would be an allowlist of permitted commands
   - Blocklist cannot catch all dangerous patterns

2. **Shell Injection**
   - The tool executes commands through a shell (sh/cmd)
   - Vulnerable to shell injection if command parameters are constructed from untrusted input
   - **CRITICAL:** Never construct commands from user input without proper sanitization

3. **Privilege Level**
   - Commands run with the same privileges as the running process
   - If the process runs as root/administrator, commands have full system access

4. **Resource Limits**
   - No CPU or memory limits enforced
   - Commands could consume excessive resources within the timeout period

#### Best Practices

1. **Never Trust User Input**
   - Validate and sanitize all inputs before constructing commands
   - Use parameterized approaches when possible
   - Prefer native Go implementations over shell commands

2. **Principle of Least Privilege**
   - Run the application with minimal necessary permissions
   - Consider using sandboxing or containerization

3. **Audit and Logging**
   - Log all command executions for security auditing
   - Include user identity, timestamp, and command executed

4. **Error Handling**
   - Always check exit codes and error messages
   - Never expose raw error messages to end users (may leak system information)

5. **Alternative Approaches**
   - Consider if the operation can be performed using Go standard library
   - For code execution, use sandboxed environments (e.g., E2B API for Python)
   - For file operations, use the file operation tools instead

#### Recommendations for Production

1. **Implement Command Allowlist**
   - Define specific allowed commands
   - Reject all other commands by default

2. **Add Resource Limits**
   - Implement CPU and memory limits using cgroups or similar
   - Limit number of concurrent command executions

3. **Enhance Audit Trail**
   - Log to secure, append-only storage
   - Include caller identity and context

4. **Consider Sandboxing**
   - Run commands in isolated containers
   - Use security frameworks like AppArmor or SELinux

5. **Rate Limiting**
   - Implement rate limiting per user/session
   - Prevent abuse and DoS attacks

---

**Last Updated:** 2026-01-31  
**Version:** 0.0.16
