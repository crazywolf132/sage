# Sage Roadmap

This document lays out the direction and aspirations for Sage. As a single-maintainer project, progress will depend on available time and community contributions. If you'd like to help or suggest ideas, feel free to open an issue or pull request.

## Short-Term (0–3 Months)

### 1. Better Undo Mechanism
- Track a history of Sage commands (commit, push, merge, etc.) in a simple log to allow multiple undo steps, not just the last operation.

### 2. Improved Tests & CI
- Add unit tests for critical commands (commit, push, undo), plus integration tests using temporary Git repositories.
- Set up a continuous integration pipeline (GitHub Actions or similar) to ensure new commits don't break core functionality.

### 3. Enhanced GitHub Integration
- Add flags for PR subcommands to set reviewers, assignees, or labels (e.g., `--reviewers @alice,@bob`).
- Allow searching/filtering PRs by label or author.

### 4. Documentation Updates
- Expand the README and wiki to include detailed usage examples, known workarounds, and best practices.
- Provide a quick "Getting Started" guide for each major feature (start, commit, push, PR create, etc.).

## Medium-Term (3–6 Months)

### 1. Interactive Conflict Resolution
- Develop a minimal TUI or guided approach for merges and rebases that have conflicts—e.g., show conflicting files, allow the user to pick a resolution, and continue with one command.

### 2. Plugin/Extension System
- Design a way to register custom commands or hooks, so teams can integrate their own checks, commands, or pre-push steps without forking the entire codebase.

### 3. Support More Git Providers
- Add optional configuration or subcommands for Bitbucket, GitLab, etc. Possibly detect the remote host automatically and adapt PR commands accordingly.

### 4. Refined Undo & Safety
- Provide explicit logging so a user can do `sage history` to see recent actions and revert a specific step.
- Store temporary references for merges/rebases in a safer or more standardized way.

## Long-Term (6+ Months)

### 1. Full "Wizard" Mode
- Provide an optional interactive flow for new users: a step-by-step guide through starting a branch, committing, pushing, and opening a PR, with descriptions of what's happening behind the scenes.

### 2. Deeper Policy/Checks Integration
- Let users define rules (e.g., "Always require JIRA ticket number in commit messages," "Run unit tests before pushing," etc.).
- Possibly integrate with local or remote CI to block merges that don't pass checks.

### 3. Enhanced Cross-Platform Support
- Test thoroughly on Windows, macOS, and Linux to handle differences in path usage, environment variables, ANSI color handling, etc.
- Provide fallback or detection logic if certain OS-level features aren't available.

### 4. Community Growth & Governance
- Encourage more contributors to become maintainers, ensuring long-term stability.
- Possibly set up a Slack/Discord channel for users to ask questions or share tips.

## Feedback & Contribution

This roadmap is a living document, not a strict timeline. If you have ideas or feel something should be prioritized, open an issue or join an existing discussion. The main goal of Sage is to make Git usage simpler and safer—so any features that further this mission are welcome.

Thank you for your support and interest in Sage!