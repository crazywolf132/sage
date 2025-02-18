package cmd

import (
	"os"

	"github.com/crazywolf132/sage/internal/config"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/crazywolf132/sage/internal/update"
	"github.com/crazywolf132/sage/internal/version"
	"github.com/spf13/cobra"
)

// Commands that don't require GitHub functionality
var noGitHubCommands = map[string]bool{
	"config":     true,
	"completion": true,
	"help":       true,
	"version":    true,
}

var rootCmd = &cobra.Command{
	Use:   "sage",
	Short: "Burning away Git complexity",
	Long: `Sage is a modern CLI tool that simplifies Git workflows and enhances productivity.

It provides intuitive commands for common Git operations and adds powerful features like:
• Smart commit messages with AI assistance
• Streamlined PR workflows and status checks
• Easy branch synchronization
• GitHub integration
• Interactive UI elements for better user experience
• Shell completion for commands and branch names

Run 'sage help' to see all available commands or 'sage <command> --help' for detailed information about a specific command.

To enable shell completion:
• Bash:  source <(sage completion bash)
• Zsh:   source <(sage completion zsh)
• Fish:  sage completion fish | source
• PowerShell: sage completion powershell | Out-String | Invoke-Expression`,
	Version:       version.Get(),
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Load config (global + local) once
		if err := config.LoadAllConfigs(); err != nil {
			ui.Warnf("Failed to load config: %v\n", err)
		}

		// Only sync git config features if we're in a git repository
		g := git.NewShellGit()
		inRepo, _ := g.IsRepo()
		if inRepo {
			if err := config.SyncGitConfigFeatures(); err != nil {
				ui.Warnf("Failed to sync git config features: %v\n", err)
			}
		}

		// Check for updates using the public GitHub API
		_ = update.CheckForUpdatesPublic(version.Get())
	},
}

func init() {
	rootCmd.SetUsageTemplate(ui.ColorHeadings(rootCmd.UsageTemplate()))

	// Add completion command
	rootCmd.AddCommand(completionCmd)
}

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for Sage commands.

To load completions:

Bash:
  $ source <(sage completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ sage completion bash > /etc/bash_completion.d/sage
  # macOS:
  $ sage completion bash > /usr/local/etc/bash_completion.d/sage

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ sage completion zsh > "${fpath[1]}/_sage"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ sage completion fish | source

  # To load completions for each session, execute once:
  $ sage completion fish > ~/.config/fish/completions/sage.fish

PowerShell:
  PS> sage completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> sage completion powershell > sage.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			_ = cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			_ = cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			_ = cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			_ = cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}
