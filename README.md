# Sage

Sage is a lightweight command-line tool that wraps Git with friendlier commands, safety checks, and shortcuts. It's built in Go, currently maintained by a single developer (me!), and aims to make Git workflows less stressful—especially when collaborating or juggling multiple branches.

## Why Sage Exists

Working with Git can sometimes feel overwhelming. Even experienced developers occasionally run into mysterious conflicts, forget to stash changes before switching branches, or accidentally force-push over someone else's work. Meanwhile, new contributors might struggle with memorizing the right commands or fear messing up the repository history.

I built Sage because I was tired of hearing (and feeling) this frustration—and I wanted a helper tool that didn't hide Git's power, but still made common tasks (like branching, committing, pushing, and creating pull requests) simpler and safer.

## How Sage Helps

* **Safer Operations**: Before destructive actions (like force pushes), Sage prompts you for confirmation, and automatically creates backup refs so you can recover if something goes wrong.
* **Simple Commands**: Instead of juggling git checkout -b, git pull, git push -u, etc., you can do things like `sage start feature/my-branch --push` to handle it all in one go.
* **Undo Functionality**: Run `sage undo` to revert the last Sage-driven action—like aborting a merge or rolling back a commit—without digging around in Git's man pages.
* **Pull Request Integration**: `sage pr create --title "Add feature"` automatically handles authentication and talks to GitHub for you. You can list, merge, or check out pull requests right from the CLI.
* **Config & Extensibility**: You can tweak Sage's behavior (like your default branch, whether to rebase or merge) via configuration files or environment variables.

## Getting Started

1. Install Go (1.20+ recommended).
2. Clone this repo:

```bash
git clone https://github.com/crazywolf132/sage.git
cd sage
```

3. Build:

```bash
go build -o sage
```

4. Install Sage to your PATH (optional, for convenience):

```bash
go install
```

5. Check it:

```bash
sage --help
sage version
```

## Basic Usage

### Start a new branch

```bash
sage start feature/my-branch --push
```

Creates feature/my-branch from your default branch (e.g., main), pulls latest updates, and pushes it up to GitHub.

### Commit changes

```bash
sage commit "Implement new feature"
```

Stages all changes and commits them with a single line command.

### Push changes

```bash
sage push
```

Pushes your current branch to origin. If --force is needed, Sage will prompt you for confirmation, then create a backup ref just in case.

### Undo last operation

```bash
sage undo
```

Rolls back your most recent commit, merge, or rebase.

### Pull request operations

* Create a PR:

```bash
sage pr create --title "Add feature" --body "Implements a new feature"
```

* List PRs:

```bash
sage pr list --state open
```

* Checkout a PR branch:

```bash
sage pr checkout 42
```

* Merge a PR:

```bash
sage pr merge 42 --method squash
```

## Future Growth

Even though I'm just one person maintaining this right now, I see a lot of potential for Sage:

* **More Git Host Support**: Integrations with GitLab, Bitbucket, or self-hosted Git services.
* **Enhanced Undo**: A detailed operation log to revert more than just the last commit or merge.
* **Interactive Conflict Resolution**: Potential for a TUI or guided conflict resolution flow.
* **Plugin System**: Let teams extend Sage with custom commands or checks.
* **Optional Lint/Checks**: Pre-commit hooks, code checks, or commit message style enforcement.

I hope to grow this into a stable, community-driven project where Git novices and veterans alike can feel more confident in their daily workflows.

## Contributing & Feedback

I welcome all issues, ideas, and pull requests. If you run into a bug or have a feature request, please open an issue. This project is something I work on in my spare time, so replies may not be immediate—but I'll do my best to keep up.

Some ways you can help:

* **Open an Issue**: Report bugs or suggest improvements.
* **Submit a Pull Request**: If you fix something or add a feature, I'd love to see it.
* **Share Your Workflow**: Hearing how you use Sage (or what's blocking you) helps guide development.

## License

MIT License. Feel free to use Sage for your own projects, modify it, or share it, as long as you follow the license terms.

---

Thanks for checking out Sage!
If it saves you time or frustration, feel free to star the repo or spread the word. I appreciate any feedback as I continue improving this tool.