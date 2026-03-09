# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in AI-Trace, please report it responsibly.

**DO NOT** open a public GitHub issue for security vulnerabilities.

### How to Report

Email: **security@aitrace.cc** (or your actual security email)

Please include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Response Timeline

- **Acknowledgment**: Within 48 hours
- **Assessment**: Within 1 week
- **Fix**: Depending on severity, typically within 2-4 weeks

### Scope

In scope:
- AI-Trace server (`server/`)
- Console application (`console/`)
- SDKs (`sdk/`)
- Smart contracts (`contracts/`)
- Verifier (`verifier/`)

Out of scope:
- Third-party dependencies (report to upstream)
- Social engineering

## Supported Versions

| Version | Supported |
|---------|-----------|
| latest  | Yes       |
| < latest | No       |

## Security Best Practices

When deploying AI-Trace:
- Always change default API keys and passwords
- Use TLS/HTTPS in production
- Restrict network access to the API server
- Regularly update to the latest version
- Review the `.env.example` file for all configurable security settings
