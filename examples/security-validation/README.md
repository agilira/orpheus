# Security Validation Example

This example demonstrates Orpheus security validation capabilities, including input sanitization, path traversal protection, and professional-grade security controls.

## Features Demonstrated

### Input Validation
- String input security validation
- Path traversal attack prevention  
- Command injection blocking
- SQL injection detection
- Buffer overflow protection

### Security Controls
- File system security scanning
- Environment variable auditing
- Permission analysis
- Attack scenario demonstrations
- Performance benchmarking

### Security Analytics
- Risk assessment reporting
- Security pattern detection
- Performance impact measurement
- Comprehensive audit logging

## Commands

### `validate` - Input Security Validation
```bash
# Validate clean input
security-validation validate --input "clean user input"

# Test path traversal protection
security-validation validate --input "../../../etc/passwd" --type path

# Test command injection protection
security-validation validate --input '$(rm -rf /)' --type command

# Test SQL injection protection  
security-validation validate --input "'; DROP TABLE users; --" --type sql

# Enable detailed pattern analysis
security-validation validate --input "suspicious input" --show-patterns --verbose
```

### `scan` - Path Security Scanning
```bash
# Scan safe file path
security-validation scan --path "config.json"

# Attempt system file access (blocked)
security-validation scan --path "/etc/passwd"

# Deep security scan with analysis
security-validation scan --path "data/file.txt" --deep-scan --show-analysis

# Test different operations
security-validation scan --path "script.sh" --operation execute
```

### `analyze` - File Security Analysis
```bash
# Basic file analysis
security-validation analyze --file "document.txt"

# Detailed permission analysis
security-validation analyze --file "script.sh" --permissions --recommendations

# Create security report
security-validation analyze --file "config.yml" --create-report
```

### `audit` - Environment Security Audit
```bash
# Audit specific environment variable
security-validation audit --env-var "PATH"

# Scan variables by prefix
security-validation audit --prefix "APP_"

# Detect sensitive variables
security-validation audit --show-sensitive

# Validate trusted prefixes
security-validation audit --env-var "SECURITY_CONFIG" --validate-trusted
```

### `demo` - Attack Scenario Demonstration
```bash
# Demonstrate path traversal protection
security-validation demo --attack-type path-traversal

# Show command injection blocking
security-validation demo --attack-type command-injection --show-protection

# SQL injection demonstration
security-validation demo --attack-type sql-injection

# XSS protection demo
security-validation demo --attack-type xss

# Buffer overflow protection
security-validation demo --attack-type buffer-overflow

# Interactive demonstration
security-validation demo --attack-type path-traversal --interactive
```

### `benchmark` - Security Performance Testing
```bash
# Benchmark all security validations
security-validation benchmark --iterations 1000

# Test specific validation type
security-validation benchmark --test-type path --iterations 5000

# Detailed performance metrics
security-validation benchmark --detailed --iterations 2000
```

## Global Flags

- `--verbose, -v`: Enable verbose security logging
- `--strict, -s`: Enable strict security mode  
- `--audit, -a`: Enable security audit logging

## Security Features Demonstrated

### Attack Prevention
- **Path Traversal**: `../../../etc/passwd`, `%2e%2e%2f`, null bytes
- **Command Injection**: `$(commands)`, backticks, shell metacharacters
- **SQL Injection**: SQL keywords, comments, union attacks
- **XSS**: Script tags, JavaScript URLs, event handlers
- **Buffer Overflow**: Oversized input validation

### Security Analytics
- File permission risk assessment
- Environment variable security scanning
- Sensitive data detection patterns
- Performance impact measurement
- Comprehensive security reporting

### Enterprise Features
- Configurable security policies
- Trusted environment prefixes
- Input sanitization options
- Security audit trails
- Performance benchmarking

## Example Outputs

### Successful Validation
```
🔍 Security Input Validation
Input: clean user input
Type: general
Length: 16 characters
Input validation passed

Security Analysis:
  - Input length check: PASS
  - Dangerous pattern scan: PASS  
  - Character encoding validation: PASS
  - Security risk level: LOW
```

### Blocked Attack
```
Security Input Validation
Input: ../../../etc/passwd
Type: path
Length: 18 characters
SECURITY ALERT: Input validation failed
Reason: dangerous traversal pattern detected

Security Pattern Analysis:
  - Path traversal pattern detected
```

### Security Demo Results
```
Security Attack Demonstration
Attack Type: path-traversal

Testing 4 attack payloads...

Test 1: BLOCKED - ../../../etc/passwd
Test 2: BLOCKED - ..\\..\\..\\windows\\system32\\config\\sam  
Test 3: BLOCKED - %2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd
Test 4: BLOCKED - file\x00.txt

Security Test Results:
  - Total payloads tested: 4
  - Attacks blocked: 4
  - Attacks allowed: 0
  - Security effectiveness: 100.0%
EXCELLENT: All attacks were successfully blocked!
```

### Performance Benchmark
```
Security Performance Benchmark
Iterations: 1000
Test Type: all

Path Validation Benchmark:
  - Total duration: 3.697172ms
  - Average per operation: 3.70 μs
  - Operations per second: 270562

Input Validation Benchmark:  
  - Total duration: 155.42µs
  - Average per operation: 155 ns
  - Operations per second: 6437247

Performance Summary:
  - All security validations completed successfully
  - Performance impact: < 1% application overhead
  - Security controls: Enterprise-grade with minimal latency
```

## Testing

Run the comprehensive test suite:

```bash
go test -v
```

The tests cover:
- All security validation commands
- Attack scenario blocking
- Environment variable auditing  
- Error handling and edge cases
- Help and version functionality
- Performance validation
