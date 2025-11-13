# Feature Parity Master Plan: Go vs Rust Implementation

## Overview

Bring the go-mcp implementation to feature parity with the Rust edda_mcp reference implementation. This plan addresses the critical gaps identified in the comprehensive comparison between the two codebases.

## Current Status

**Complete Parity:**
- ✅ Databricks Provider (5 tools)
- ✅ IO Provider (scaffold + validate)
- ✅ Workspace Provider (6 tools)
- ✅ Deployment Provider (1 tool)
- ✅ State management (Scaffolded → Validated → Deployed)
- ✅ BLAKE3 checksums
- ✅ ENGINE_GUIDE injection

**Missing Features:**
- ❌ Dagger Sandbox (stubbed, needs full implementation)
- ❌ Trajectory Tracking (debugging and analytics)
- ❌ Bug Reporting System
- ❌ Version Check (auto-update notifications)
- ❌ Architecture improvements (state machine, validation strategies)

## Implementation Phases

### Phase 1: Dagger Sandbox Implementation ⚠️ CRITICAL
**Priority**: HIGH | **Estimated Effort**: 3-4 weeks

Implement fully functional Dagger-based containerized sandbox for secure, isolated validation.

**Status**: 🔴 Not Started
**Plan**: [feature-parity-phase-1-dagger.md](./feature-parity-phase-1-dagger.md)

**Key Deliverables**:
- Full Dagger SDK integration
- All sandbox operations implemented (exec, write, read, etc.)
- Container-based validation with file sync
- Test suite with actual containers
- Performance benchmarks

### Phase 2: Trajectory Tracking 📊
**Priority**: MEDIUM | **Estimated Effort**: 1-2 weeks

Add JSONL-based history logging for debugging and analytics.

**Status**: 🔴 Not Started
**Plan**: [feature-parity-phase-2-trajectory.md](./feature-parity-phase-2-trajectory.md)

**Key Deliverables**:
- Event capture system
- JSONL history logging
- Session metadata tracking
- Query interface (CLI command)

### Phase 3: Developer Experience Improvements 🛠️
**Priority**: MEDIUM | **Estimated Effort**: 1 week

Enhance debugging, maintenance, and update capabilities.

**Status**: 🔴 Not Started
**Plan**: [feature-parity-phase-3-devex.md](./feature-parity-phase-3-devex.md)

**Key Deliverables**:
- Bug reporting command
- Version check with auto-update notifications
- Enhanced logging capabilities
- Improved configuration validation

### Phase 4: Architecture Improvements 🏗️
**Priority**: LOW | **Estimated Effort**: 1-2 weeks

Refactor architecture to match Rust patterns where beneficial.

**Status**: 🔴 Not Started
**Plan**: [feature-parity-phase-4-architecture.md](./feature-parity-phase-4-architecture.md)

**Key Deliverables**:
- State machine with enums
- Pluggable validation strategies
- Provider registry pattern
- Error handling improvements

## Excluded Features

The following features from the Rust version are intentionally excluded:

- **Google Sheets Provider**: No current use case identified
  - Can be added later if needed
  - Would require OAuth2 integration (~2-3 weeks)

## Timeline

### Recommended Sequence

```
Sprint 1-2: Phase 1 - Dagger Sandbox        [3-4 weeks]
Sprint 3:   Phase 2 - Trajectory Tracking   [1-2 weeks]
Sprint 4:   Phase 3 - Developer Experience  [1 week]
Sprint 5:   Phase 4 - Architecture          [1-2 weeks]
                                             ___________
Total Estimated Effort:                      6-9 weeks
```

### Critical Path

The Dagger Sandbox (Phase 1) is on the critical path as it:
- Affects security and isolation of validation
- Blocks full parity with Rust version
- Is required for production-ready validation

Phases 2-4 can be implemented in parallel or reordered based on priorities.

## Success Criteria

### Overall Goals
- [ ] All Rust tools replicated in Go (excluding Google Sheets)
- [ ] Test coverage >80% for new code
- [ ] All integration tests passing
- [ ] Documentation updated (CLAUDE.md, README)
- [ ] Performance benchmarks favorable vs Rust

### Phase-Specific Criteria
Refer to individual phase plans for detailed success criteria.

## Reference Materials

### Rust Codebase Location
`/Users/fabian.jakobs/Workspaces/agent/edda/`

### Key Reference Files
- Dagger: `edda_sandbox/src/dagger.rs`
- Trajectory: `edda_mcp/src/trajectory.rs`
- Bug Report: `edda_mcp/src/yell.rs`
- Version Check: `edda_mcp/src/version_check.rs`
- State Machine: `edda_mcp/src/state.rs`

### Go Codebase Location
`/Users/fabian.jakobs/Workspaces/app-mcp/`

### Key Implementation Files
- Sandbox Interface: `pkg/sandbox/sandbox.go`
- Dagger Stub: `pkg/sandbox/dagger/dagger.go`
- Session: `pkg/session/session.go`
- Config: `pkg/config/config.go`
- Providers: `pkg/providers/*/`

## Dependencies

### Phase 1 (Dagger)
- `dagger.io/dagger` Go SDK
- Docker daemon running
- Performance testing tools

### Phase 2 (Trajectory)
- JSON serialization (stdlib)
- File rotation utilities

### Phase 3 (DevEx)
- GitHub API client
- Tar/gzip libraries (stdlib)
- Semantic version comparison

### Phase 4 (Architecture)
- No external dependencies
- Refactoring tools

## Risk Assessment

### High Risks
1. **Dagger API Differences**: Go SDK may differ from Rust
   - Mitigation: Study both SDKs, create compatibility layer

2. **Performance Overhead**: Containerization may slow validation
   - Mitigation: Benchmark, optimize, provide local fallback

3. **Docker Availability**: Not all environments have Docker
   - Mitigation: Graceful fallback to local sandbox

### Medium Risks
1. **Trajectory Disk Usage**: History logs may grow large
   - Mitigation: Implement rotation and retention policies

2. **Breaking Changes**: Refactoring may affect existing users
   - Mitigation: Semantic versioning, deprecation notices

## Testing Strategy

### For Each Phase
1. **Unit Tests**: Test components in isolation
2. **Integration Tests**: End-to-end workflow validation
3. **Security Tests**: Path validation, injection prevention
4. **Performance Tests**: Benchmarks for critical paths
5. **Compatibility Tests**: Ensure parity with Rust behavior

### Acceptance Criteria
- ✅ All tests passing (including race detector)
- ✅ Code coverage >80%
- ✅ Documentation complete
- ✅ CLAUDE.md updated
- ✅ Integration tests demonstrate feature working

## Communication

### Status Updates
Update this file after each phase completion:
- Change status indicators (🔴 → 🟡 → 🟢)
- Add completion dates
- Document any deviations from plan

### Issue Tracking
Create GitHub issues for:
- Bugs discovered during implementation
- Feature requests from testing
- Documentation gaps

## Notes

- Focus on clean, idiomatic Go code
- Maintain backward compatibility where possible
- Prioritize security and correctness over performance
- Document all design decisions and tradeoffs
- Reference Rust implementation for behavioral parity

## Version History

- **v1.0** (2025-01-08): Initial master plan created based on comprehensive codebase comparison
