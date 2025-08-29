package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/johnstilia/commitron/pkg/ai"
	"github.com/johnstilia/commitron/pkg/config"
	"github.com/johnstilia/commitron/pkg/git"
	"github.com/spf13/cobra"
)

// Command-specific flags
var dryRun bool
var force bool

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a commit message using AI",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if we're in a git repository
		if !git.IsGitRepo() {
			return fmt.Errorf("\033[1;31m❌ Not a git repository\033[0m")
		}

		// Use specified config file or default
		var cfg *config.Config
		var err error
		if configPath != "" {
			cfg, err = config.LoadConfigFromPath(configPath)
			if err != nil {
				return fmt.Errorf("\033[1;31m❌ Error loading configuration from %s: %w\033[0m", configPath, err)
			}
		} else {
			cfg, err = config.LoadConfig()
			if err != nil {
				return fmt.Errorf("\033[1;31m❌ Error loading configuration: %w\033[0m", err)
			}
		}

		// Get staged files
		stagedFiles, err := git.GetStagedFiles()
		if err != nil {
			return fmt.Errorf("\033[1;31m❌ Error getting staged files: %w\033[0m", err)
		}

		if len(stagedFiles) == 0 {
			return fmt.Errorf("\033[1;31m❌ No staged files found. Stage files using 'git add' before running commitron\033[0m")
		}

		// Get changes content for context
		changes, err := git.GetStagedChanges()
		if err != nil {
			return fmt.Errorf("\033[1;31m❌ Error getting staged changes: %w\033[0m", err)
		}

		// Generate commit message using AI
		fmt.Println("\033[1;36m🤖 Analyzing changes...\033[0m")
		message, err := ai.GenerateCommitMessage(cfg, stagedFiles, changes)
		if err != nil {
			return fmt.Errorf("\033[1;31m❌ Error generating commit message: %w\033[0m", err)
		}

		// In dry run mode, just display the message without committing
		if dryRun {
			fmt.Println("\n\033[38;5;244m🔍 Dry run completed. No commit was created.\033[0m")
			return nil
		}

		// Create the commit with the confirmed message
		fmt.Print("\n\033[1;36m💾 Creating commit... \033[0m")
		err = git.Commit(message)
		if err != nil {
			fmt.Println("\033[1;31m❌ failed\033[0m")
			return fmt.Errorf("\033[1;31m❌ Error: %w\033[0m", err)
		}
		fmt.Println("\033[1;32m✓ complete\033[0m")

		return nil
	},
}

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine config path
		var targetPath string
		if configPath != "" {
			targetPath = configPath
		} else {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("\033[1;31m❌ Error getting home directory: %w\033[0m", err)
			}
			targetPath = filepath.Join(homeDir, ".commitronrc")
		}

		// Check if config file already exists
		if _, err := os.Stat(targetPath); err == nil && !force {
			return fmt.Errorf("\033[1;31m❌ Configuration file already exists at %s (use --force to overwrite)\033[0m", targetPath)
		}

		// Create example config
		if err := config.SaveExampleConfig(targetPath); err != nil {
			return fmt.Errorf("\033[1;31m❌ Error creating configuration file: %w\033[0m", err)
		}

		fmt.Println("\n\033[1;32m✓ Configuration Ready\033[0m")
		fmt.Printf("\n  📁 File created at: \033[38;5;76m%s\033[0m\n", targetPath)
		fmt.Println("\n  \033[38;5;252mEdit this file to configure your AI provider and settings.\033[0m")
		return nil
	},
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("\n\033[1;36mcommitron v0.1.3\033[0m")
		fmt.Println("\n  \033[38;5;252m🤖 AI-powered commit message generator\033[0m")
		fmt.Println("\n  \033[38;5;244mBuilt with ❤️ using Go\033[0m")
	},
}

func init() {
	// Add flags to generate command
	generateCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Preview the commit message without creating a commit")

	// Add flags to init command
	initCmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing configuration file")
}
