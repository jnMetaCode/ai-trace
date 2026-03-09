# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability, please report it responsibly.

### How to Report

**DO NOT** open a public GitHub issue for security vulnerabilities.

Instead, please email: **security@aitrace.cc**

Include:
1. Description of the vulnerability
2. Steps to reproduce
3. Potential impact
4. Suggested fix (if any)

### Response Timeline

- **Initial Response**: Within 48 hours
- **Status Update**: Within 7 days
- **Fix Timeline**: Depends on severity
  - Critical: 24-48 hours
  - High: 7 days
  - Medium: 30 days
  - Low: Next release

### What to Expect

1. **Acknowledgment**: We'll confirm receipt of your report
2. **Investigation**: We'll investigate and validate the issue
3. **Fix Development**: We'll develop and test a fix
4. **Disclosure**: We'll coordinate disclosure with you
5. **Credit**: We'll credit you in the security advisory (if desired)

## Security Design Principles

### API Key Handling

- **Never Stored**: Upstream API keys are passed through, never persisted
- **Memory Only**: Keys exist only in request context
- **TLS Required**: All production deployments must use HTTPS

### Cryptographic Standards

- **Signatures**: Ed25519 for all digital signatures
- **Hashing**: SHA-256 for Merkle trees and content hashing
- **TLS**: TLS 1.2+ required for all connections

### Access Control

- **API Key Authentication**: All API endpoints require authentication
- **Rate Limiting**: Token bucket algorithm prevents abuse
- **CORS**: Configurable cross-origin policies

### Data Protection

- **Audit Trails**: All operations are logged
- **Tamper-proof Storage**: L2 evidence uses WORM storage
- **Encryption at Rest**: Database encryption recommended

## Security Best Practices

### For Operators

1. **Use Strong API Keys**
   ```bash
   openssl rand -hex 32
   ```

2. **Enable TLS**
   - Use valid SSL certificates
   - Configure TLS 1.2+ only

3. **Network Security**
   - Firewall rules to restrict access
   - Internal network for database/Redis

4. **Regular Updates**
   - Keep dependencies updated
   - Monitor security advisories

### For Developers

1. **Input Validation**
   - All inputs are validated
   - SQL injection prevention via parameterized queries

2. **Output Encoding**
   - Proper JSON encoding
   - XSS prevention headers

3. **Dependency Management**
   - Regular dependency audits
   - Minimal dependency footprint

## Security Checklist

### Before Production

- [ ] Changed all default passwords
- [ ] Configured firewall rules
- [ ] Enabled HTTPS/TLS
- [ ] Set up API keys
- [ ] Enabled rate limiting
- [ ] Configured audit logging
- [ ] Set up monitoring alerts
- [ ] Performed security scan

### Ongoing

- [ ] Regular security updates
- [ ] Log review
- [ ] Access audit
- [ ] Penetration testing (annual)

## Compliance

AI-Trace is designed to help organizations meet:

- **SOC 2**: Audit trail requirements
- **GDPR**: Data processing accountability
- **HIPAA**: Audit controls (when properly configured)
- **Financial Regulations**: Decision audit trails

## Contact

- Security Issues: security@aitrace.cc
- General Support: support@aitrace.cc
