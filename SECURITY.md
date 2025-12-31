# Security Policy

## Supported Versions

We take security seriously and provide security updates for the following versions:

| Version | Supported          | Status |
| ------- | ------------------ | ------ |
| 0.1.x   | :white_check_mark: | Alpha - active development |

### Support Timeline

- **Current version (0.x)**: Active development, security patches as needed
- **Older versions**: Please upgrade to latest

## Reporting a Vulnerability

We appreciate responsible disclosure of security vulnerabilities. Please do not create public GitHub issues for security vulnerabilities.

### How to Report

**For security vulnerabilities, please email:** security@sdbx.one (or create a private security advisory)

Alternatively, use GitHub's private vulnerability reporting:
1. Go to the repository's Security tab
2. Click "Report a vulnerability"
3. Fill out the private advisory form

### What to Include

Please include the following information:

1. **Description** - Clear description of the vulnerability
2. **Impact** - What can an attacker achieve?
3. **Affected Versions** - Which versions are vulnerable?
4. **Reproduction** - Step-by-step instructions to reproduce
5. **Proof of Concept** - Code, screenshots, or logs (if applicable)
6. **Suggested Fix** - If you have a solution (optional)
7. **Disclosure Timeline** - Your expected timeline for public disclosure

### Example Report

```
Subject: SQL Injection in Config Management

Description:
The config set command does not properly sanitize user input,
allowing SQL injection through the key parameter.

Impact:
An attacker with local access could read/modify configuration
database contents.

Affected Versions:
v0.1.0-alpha

Steps to Reproduce:
1. Run: sdbx config set "key'; DROP TABLE secrets;--" value
2. Configuration database is corrupted

Suggested Fix:
Use parameterized queries or input validation

Disclosure Timeline:
Prefer 90 days for fix before public disclosure
```

## Response Process

1. **Acknowledgment** - Within 48 hours of report
2. **Initial Assessment** - Within 5 business days
3. **Fix Development** - Timeline depends on severity
4. **Security Advisory** - Published with fix
5. **Public Disclosure** - After fix is released

### Severity Levels

| Severity | Response Time | Description |
|----------|--------------|-------------|
| Critical | 24-48 hours | Remote code execution, authentication bypass |
| High | 3-5 days | Privilege escalation, data exposure |
| Medium | 1-2 weeks | Information disclosure, DoS |
| Low | Best effort | Minor issues with limited impact |

## Security Best Practices

### For Users

#### 1. Keep SDBX Updated
```bash
# Check current version
sdbx version

# Update to latest version
curl -fsSL https://github.com/maiko/SDBX/releases/latest/download/install.sh | bash
```

#### 2. Secure Your Secrets
- Never commit secrets to version control
- Rotate secrets regularly: `sdbx secrets rotate`
- Use strong passwords (12+ characters, mixed case, numbers, symbols)
- Enable 2FA in Authelia for additional security

#### 3. VPN Configuration
- Always use VPN for torrent downloads
- Verify kill-switch is working: `sdbx doctor`
- Use trusted VPN providers with P2P support
- Regularly check for IP leaks

#### 4. Network Security
- Use Cloudflare Tunnel or firewall for exposure
- Never expose services directly to internet without authentication
- Use subdomain routing for better isolation
- Enable Fail2Ban for brute force protection (if available)

#### 5. Access Control
- Use strong Authelia passwords
- Enable 2FA for all users
- Regularly review user access
- Use separate accounts for different users
- Monitor access logs: `sdbx logs authelia`

#### 6. System Hardening
- Run with non-root user (PUID/PGID)
- Keep Docker updated
- Use minimal host OS (Ubuntu Server, Debian)
- Enable automatic security updates
- Regular backups: `sdbx backup run`

#### 7. Monitoring
- Run `sdbx doctor` regularly
- Monitor service logs for suspicious activity
- Set up alerting for failed authentication attempts
- Review Tautulli (if enabled) for unusual access patterns

### For Developers

#### 1. Code Security

```go
// ✅ Good - Use parameterized queries
stmt, err := db.Prepare("SELECT * FROM users WHERE id = ?")
result, err := stmt.Query(userID)

// ❌ Bad - SQL injection vulnerability
query := fmt.Sprintf("SELECT * FROM users WHERE id = %s", userID)
```

```go
// ✅ Good - Validate input
func SetConfig(key, value string) error {
    if !isValidKey(key) {
        return errors.New("invalid key")
    }
    // ... rest of logic
}

// ❌ Bad - No validation
func SetConfig(key, value string) error {
    viper.Set(key, value)
}
```

#### 2. Secret Management
- Never log secrets
- Use crypto/rand for generation
- Clear sensitive data from memory when done
- Use secure comparison for secrets (constant-time)

```go
// ✅ Good - Constant time comparison
if subtle.ConstantTimeCompare([]byte(input), []byte(stored)) == 1 {
    // Authenticated
}

// ❌ Bad - Timing attack vulnerable
if input == stored {
    // Authenticated
}
```

#### 3. Input Validation
- Validate all user input
- Use allowlists over denylists
- Sanitize file paths to prevent traversal
- Limit input size

#### 4. Error Handling
- Never expose sensitive information in errors
- Log detailed errors internally
- Show generic errors to users

```go
// ✅ Good
if err != nil {
    log.Errorf("Failed to authenticate user %s: %v", username, err)
    return errors.New("authentication failed")
}

// ❌ Bad
if err != nil {
    return fmt.Errorf("invalid password for user %s: %v", username, err)
}
```

## Security Features

### Built-in Security

1. **Authentication & Authorization**
   - Authelia SSO with 2FA support
   - Session management
   - RBAC (Role-Based Access Control)

2. **Secret Management**
   - Automatic secret generation (crypto/rand)
   - Secure storage with proper permissions (0600)
   - Rotation support
   - Backup encryption

3. **Network Security**
   - VPN kill-switch for downloads
   - Reverse proxy (Traefik) with TLS
   - Network isolation via Docker networks
   - Port exposure controls

4. **Data Protection**
   - Encrypted backups
   - Secure file permissions
   - Non-root container execution
   - Volume isolation

### Cryptography

- **Password Hashing**: Argon2id (industry standard)
- **Secret Generation**: crypto/rand (cryptographically secure)
- **TLS**: Let's Encrypt or Cloudflare managed
- **2FA**: TOTP (Time-based One-Time Password)

## Known Security Considerations

### By Design

1. **Local System Access Required**
   - SDBX is designed for trusted environments
   - Physical/SSH access should be protected
   - Use strong SSH keys and disable password auth

2. **Docker Socket Access**
   - SDBX requires Docker access
   - This is equivalent to root access
   - Run on dedicated host or trusted environments

3. **VPN Credentials**
   - VPN passwords stored in plain text (secrets/vpn_password.txt)
   - Necessary for Gluetun to function
   - Protect secrets/ directory with file permissions
   - Use VPN-specific credentials (not your main account)

### Mitigations

- Run on dedicated hardware/VPS
- Use restrictive file permissions
- Enable disk encryption
- Regular security audits
- Monitor access logs

## Security Audit History

| Date | Auditor | Scope | Findings | Status |
|------|---------|-------|----------|--------|
| TBD  | TBD     | TBD   | TBD      | Planned |

## Compliance

### GDPR Considerations

SDBX processes personal data (usernames, IPs, etc.):
- Data minimization: Only necessary data is collected
- User rights: Users can delete their data
- Logging: Can be disabled or anonymized
- No telemetry: No data sent to external services

### PCI-DSS

SDBX does not handle payment information and is not PCI-DSS compliant. Do not use for payment processing.

## Security Resources

### External Resources

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Docker Security Best Practices](https://docs.docker.com/engine/security/)
- [Traefik Security](https://doc.traefik.io/traefik/https/tls/)
- [Authelia Security](https://www.authelia.com/overview/security/introduction/)

### Tools

- [Trivy](https://trivy.dev/) - Container vulnerability scanning
- [gosec](https://github.com/securego/gosec) - Go security checker
- [Docker Bench](https://github.com/docker/docker-bench-security) - Docker security auditing

## Hall of Fame

We appreciate security researchers who responsibly disclose vulnerabilities. Contributors will be listed here:

<!-- Security researchers who reported valid vulnerabilities -->
- TBD

## Contact

- **Security Issues**: security@sdbx.one
- **General Issues**: https://github.com/maiko/SDBX/issues
- **Discussions**: https://github.com/maiko/SDBX/discussions

---

**Thank you for helping keep SDBX secure!** 🔒
