![Sage Banner](docs/sage.jpg)

Hey there! Welcome to Sage - your friendly neighborhood Git companion. Think of it as a smart wrapper around Git that helps you streamline your workflow.

## Why Did I Build This? 🤔

Let's be real - Git is powerful, but sometimes its workflow can be streamlined. I built this because:
- I wanted to automate repetitive Git workflows
- Switching contexts between terminal and browser for PR management was tedious
- Merge conflicts were taking up too much of my day
- I knew there had to be a faster way to handle common Git tasks

So I built Sage to make my life easier, and hopefully yours too!

## What Makes Sage Cool? ✨

* **Workflow Automation**: Sage handles common Git operations with smart defaults and built-in best practices.
* **Simple Commands**: Instead of typing `git checkout -b feature/branch && git pull origin main && git push -u origin feature/branch`, just do `sage start feature/branch`. Your fingers will thank you.
* **Time Travel (Kind of)**: A super detailed undo system that lets you track and revert operations with precision.
* **PR Magic**: Create and manage pull requests right from your terminal. No more context-switching to GitHub!
* **Branch Wizardry**: Smart syncing with automatic stash handling and conflict detection.
* **AI Helper** 🤖: Optional AI features for commit messages and PR content (needs OpenAI API key).

## Getting Started 🚀

### Requirements
- Go 1.20 or later (required for module support)
- Git (obviously!)
- GitHub account for PR features

### Installation Options

1. Quick Install (recommended):
```bash
go install github.com/crazywolf132/sage@latest
```

2. Build from source:
```bash
git clone https://github.com/crazywolf132/sage.git
cd sage
go build
./sage -v
```

3. Verify installation:
```bash
sage --help
sage -v
```

## Basic Usage 🛠️

### Start a new branch
```bash
sage start feature/awesome-stuff
```
Boom! New branch created, latest updates pulled, and pushed to GitHub. All in one go.

### Commit your masterpiece
```bash
sage commit "Add that thing that does the stuff"
```
Stages and commits everything. No more `git add .` followed by `git commit -m` dance.

### Push it real good
```bash
sage push
```
Pushes your work to origin. If you need --force, Sage will make sure you don't shoot yourself in the foot.

### Oops! (Undo System) 🔄
```bash
# See what you've been up to
sage undo --history

# Take back that last thing you did
sage undo

# Undo something specific
sage undo --id <operation-id>

# Preview before you undo (safety first!)
sage undo --preview

# Find exactly what you need to undo
sage undo --category commit --group branch
```

### PR stuff made easy
```bash
# Create a PR
sage pr create --title "🚀 Add awesome feature" --body "Trust me, this is good"

# See what's cooking
sage pr list

# Check out someone's PR
sage pr checkout 42

# See what reviewers are saying
sage pr todos 42

# Merge it in
sage pr merge 42 --method squash
```

## Setting Things Up ⚙️

### Environment Variables
- `SAGE_GITHUB_TOKEN` or `GITHUB_TOKEN`: Your GitHub token (if you have the `gh` CLI installed and authenticated, we'll use that automatically!)
  - Required scopes: `repo`, `read:org` (for organization repos)
- `SAGE_CONFIG`: Where to keep your config
- `SAGE_OPENAI_KEY`: For AI features (totally optional)

### Quick Config
```bash
# AI Settings
sage config set ai.model gpt-4        # AI model to use
sage config set ai.base_url <url>     # Custom AI API endpoint (optional)

# Git Settings
sage config set git.default_branch main    # Default branch for operations
sage config set git.merge_method squash    # Default PR merge method

# PR Settings
sage config set pr.draft false            # Create PRs as drafts by default
sage config set pr.reviewers user1,user2  # Default PR reviewers
sage config set pr.labels feature,docs    # Default PR labels
```

### Experimental Features 🧪
Sage includes experimental features that can enhance your Git workflow. View and manage them with:
```bash
sage config experimental
```

Available experimental features:
- **Reuse Recorded Resolution (rerere)**: Git remembers how you resolved conflicts and automatically reuses those resolutions.
  ```bash
  # Enable globally (all Sage repos)
  sage config set experimental.rerere true
  # Enable for current repo only
  sage config set --local experimental.rerere true
  ```

- **Commit Graph**: Speeds up git log operations in large repositories by maintaining a commit graph.
  ```bash
  sage config set experimental.commit-graph true
  ```

- **File System Monitor**: Significantly improves `git status` performance by using OS-level file monitoring.
  ```bash
  sage config set experimental.fsmonitor true
  ```

- **Git Auto-Maintenance**: Automatically optimizes repository performance with scheduled maintenance tasks:
  - Hourly prefetch to keep your repo up-to-date
  - Automatic loose object cleanup
  - Daily reference packing
  - Incremental repack for optimal storage
  ```bash
  sage config set experimental.maintenance true
  ```

These features can be enabled globally (for all Sage repositories) or locally (per repository). Use the `--local` flag with `sage config set` to enable features for just the current repository.

### Local Storage
Sage stores its data in `.git/.sage/` in your repository:
- `undo_history.json`: Operation history for the undo system
- `config.toml`: Local repository configuration
These files are stored in your Git directory and are not committed to your repository.

### AI Features & Privacy
When using AI features:
- Commit diffs, messages, and PR content are sent to OpenAI
- Basic API key security (stored in global config only)
- Note: Currently no filtering of sensitive data - use with caution
- Consider reviewing diffs before using AI features

## Technical Details 🔧

### Error Handling
- Operations are tracked in the undo system for recovery
- Clear error messages help diagnose issues
- Failed operations can be reverted using the undo system
- State is preserved when possible during errors

### Conflict Resolution
- Automatic stash/unstash of local changes during sync
- Branch synchronization with conflict detection
- Clear reporting of conflicted files
- Status tracking during conflict resolution

### Edge Cases
- Preserves uncommitted changes via stashing
- Basic force push protection with confirmation
- Operation history for recovery
- Handles common Git scenarios

## Where We're At 🎯

Here's what's ready to roll and what we're cooking up:

✅ **Ready to Rock**
- Detailed undo system with operation tracking and selective undo
- GitHub PR management (create, list, checkout, merge)
- Branch synchronization with stash handling
- AI-powered commit messages and PR content
- Conventional commit support
- Branch cleanup with safety checks
- Operation history with filtering and preview

🔄 **In the Workshop**
- Enhanced conflict resolution tools
- More PR automation features
- Performance optimizations
- Extended documentation
- Submodule support
- Advanced branch scenarios

## Known Limitations
- Currently supports GitHub only (GitLab/Bitbucket planned)
- Large repositories might experience slower undo history loading
- AI features require internet connectivity and OpenAI API key
- No filtering of sensitive data in AI features
- Basic force push protection (confirmation only)
- Limited handling of advanced Git scenarios (submodules, detached HEAD)
- Some edge cases require manual conflict resolution
- PR features require GitHub token with appropriate scopes

## Future Growth 🌱

Even though I'm just one person maintaining this right now, I see a lot of potential for Sage:

* **More Git Host Support**: Integrations with GitLab, Bitbucket, or self-hosted Git services.
* **Interactive Conflict Resolution**: Potential for a TUI or guided conflict resolution flow.
* **Plugin System**: Let teams extend Sage with custom commands or checks.
* **Optional Lint/Checks**: Pre-commit hooks, code checks, or commit message style enforcement.

I hope to grow this into a stable, community-driven project where developers can feel more confident in their daily workflows.

## Contributing & Feedback 🤝

I welcome all issues, ideas, and pull requests. If you run into a bug or have a feature request, please open an issue. This project is something I work on in my spare time, so replies may not be immediate—but I'll do my best to keep up.

Some ways you can help:

* **Open an Issue**: Report bugs or suggest improvements.
* **Submit a Pull Request**: If you fix something or add a feature, I'd love to see it.
* **Share Your Workflow**: Hearing how you use Sage (or what's blocking you) helps guide development.

Check out [ROADMAP.md](ROADMAP.md) to see what we're planning!

## License

MIT License - See [LICENSE](LICENSE) for details.

---

If Sage saves you from even one tedious Git task, my mission is accomplished! Star the repo if you like it, and feel free to spread the word to your fellow developers. 

Happy coding! 🎉