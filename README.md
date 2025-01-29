# Sage 🪴

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
* **Smart Recovery**: `sage undo` gives you a clean way to reverse your last operation.
* **PR Magic**: Create and manage pull requests right from your terminal. No more context-switching to GitHub!
* **Your Tool, Your Rules**: Customize Sage to work how you want. Because everyone has their preferred Git workflow.

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

## Basic Usage 🚀

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

### Undo that thing you just did

```bash
sage undo
```
We all make mistakes. This one's got your back.

### PR stuff made easy

```bash
# Create a PR
sage pr create --title "🚀 Add awesome feature" --body "Trust me, this is good"

# See what's cooking
sage pr list --state open

# Check out someone's PR
sage pr checkout 42

# Merge it in
sage pr merge 42 --method squash
```

## Future Growth

Even though I'm just one person maintaining this right now, I see a lot of potential for Sage:

* **More Git Host Support**: Integrations with GitLab, Bitbucket, or self-hosted Git services.
* **Enhanced Undo**: A detailed operation log to revert more than just the last commit or merge.
* **Interactive Conflict Resolution**: Potential for a TUI or guided conflict resolution flow.
* **Plugin System**: Let teams extend Sage with custom commands or checks.
* **Optional Lint/Checks**: Pre-commit hooks, code checks, or commit message style enforcement.

I hope to grow this into a stable, community-driven project where developers can feel more confident in their daily workflows.

## Contributing & Feedback

I welcome all issues, ideas, and pull requests. If you run into a bug or have a feature request, please open an issue. This project is something I work on in my spare time, so replies may not be immediate—but I'll do my best to keep up.

Some ways you can help:

* **Open an Issue**: Report bugs or suggest improvements.
* **Submit a Pull Request**: If you fix something or add a feature, I'd love to see it.
* **Share Your Workflow**: Hearing how you use Sage (or what's blocking you) helps guide development.

## License

MIT License. Feel free to use Sage for your own projects, modify it, or share it, as long as you follow the license terms.

---

If Sage saves you from even one tedious Git task, my mission is accomplished! Star the repo if you like it, and feel free to spread the word to your fellow developers. 

Happy coding! 🎉