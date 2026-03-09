# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- GitHub Actions CI/CD pipeline
- CONTRIBUTING.md with development guide
- SECURITY.md with vulnerability reporting process
- Issue and PR templates
- Competitive comparison in README

## [0.1.0] - 2025-01-15

### Added
- Core event capture engine (INPUT / MODEL / RETRIEVAL / TOOL_CALL / OUTPUT / POST_EDIT)
- Merkle tree certificate generation and verification
- Three evidence levels: internal, compliance (TSA), legal (blockchain)
- OpenAI-compatible proxy API (`/api/v1/chat/completions`)
- Minimal disclosure proofs with zero-knowledge verification
- Python SDK with async support and OpenAI drop-in wrapper
- Go, JavaScript, Java SDK (initial release)
- Standalone CLI verifier for offline certificate verification
- React console with event explorer and certificate viewer
- Docker Compose deployment with PostgreSQL, Redis, MinIO
- Ethereum/Polygon smart contracts for certificate anchoring
- Behavior fingerprinting engine
- DAG-based event causality tracking
- JWT authentication and API key middleware
- Rate limiting and CORS support
