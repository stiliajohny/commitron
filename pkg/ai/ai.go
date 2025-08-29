package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/johnstilia/commitron/pkg/config"
	"github.com/johnstilia/commitron/pkg/ui"
)

// Template constants for different commit message formats
const (
	// Base template with common fields
	BaseTemplateJSON = `{
		"instruction": "Generate a commit message describing the changes",
		"format": {
			"max_subject_length": %d,
			"max_body_length": %d,
			"include_body": %t
		},
		"context": {
			"files": %s,
			"changes": %s
		},
		"output": {
			"format": "json",
			"subject_only": %t
		}
	}`

	// Template for conventional commits
	ConventionalCommitsJSON = `{
		"instruction": "Generate a commit message following Conventional Commits specification",
		"requirements": {
			"must_start_with_type": true,
			"must_have_subject": true,
			"format_examples": [
				"feat: add new user authentication",
				"fix(auth): resolve login timeout issue",
				"docs: update README installation steps"
			],
			"invalid_formats": [
				": description without type",
				"feature: (incorrect type name)"
			]
		},
		"convention": {
			"type": "conventional",
			"types": {
				"docs": "Documentation only changes",
				"style": "Changes that do not affect the meaning of the code (whitespace, formatting, etc)",
				"refactor": "A code change that neither fixes a bug nor adds a feature",
				"perf": "A code change that improves performance",
				"test": "Adding missing tests or correcting existing tests",
				"build": "Changes that affect the build system or external dependencies",
				"ci": "Changes to CI configuration files and scripts",
				"chore": "Other changes that don't modify source or test files",
				"revert": "Reverts a previous commit",
				"feat": "A new feature",
				"fix": "A bug fix"
			},
			"format": "type(scope): subject",
			"rules": {
				"commit_structure": "<type>[optional scope]: <description>\\n\\n[optional body]\\n\\n[optional footer(s)]",
				"breaking_change": "A commit with footer 'BREAKING CHANGE:' or with '!' after type/scope introduces a breaking API change",
				"scope_format": "A scope MAY be provided in parentheses after the type",
				"type_case": "Types MUST be lowercase",
				"description_format": "Description MUST immediately follow the colon and space",
				"body_format": "A longer commit body MUST be provided after a blank line following the description when include_body is true",
				"footer_format": "Footer MUST be separated by a blank line and follow the format 'token: value'",
				"breaking_format": "Breaking changes MUST be indicated with '!' before colon or as 'BREAKING CHANGE:' in footer"
			}
		},
		"format": {
			"max_subject_length": %d,
			"max_body_length": %d,
			"include_body": %t,
			"body_required": %t,
			"critical_note": "CRITICAL: The TOTAL combined length of 'type(scope): subject' MUST NOT exceed max_subject_length. This includes ALL characters. Keep subject extremely brief.",
			"length_examples": "Examples of good length subjects: 'fix: update validation logic', 'feat(auth): add login timeout'"
		},
		"context": {
			"files": %s,
			"changes": %s
		},
		"output": {
			"format": "json",
			"subject_only": %t,
			"response_format": {
				"type": "",
				"scope": "",
				"subject": "",
				"body": ""
			}
		}
	}`

	// Template for custom commit format
	CustomCommitJSON = `{
		"instruction": "Generate a commit message following custom template",
		"convention": {
			"type": "custom",
			"template": "%s"
		},
		"format": {
			"max_subject_length": %d,
			"max_body_length": %d,
			"include_body": %t
		},
		"context": {
			"files": %s,
			"changes": %s
		},
		"output": {
			"format": "json",
			"subject_only": %t
		}
	}`
)

// CommitTypeFormats defines the format for different commit types
var CommitTypeFormats = map[string]string{
	"":             "<commit message>",
	"conventional": "<type>(<optional scope>): <commit message>",
}

// CommitTypeDescriptions maps commit types to their descriptions for AI guidance
var CommitTypeDescriptions = map[string]string{
	"": "",
	"conventional": `Choose a type from the type-to-description JSON below that best describes the code changes:
{
  "docs": "Documentation only changes",
  "style": "Changes that do not affect the meaning of the code (whitespace, formatting, missing semi-colons, etc)",
  "refactor": "A code change that neither fixes a bug nor adds a feature",
  "perf": "A code change that improves performance",
  "test": "Adding missing tests or correcting existing tests",
  "build": "Changes that affect the build system or external dependencies",
  "ci": "Changes to CI configuration files and scripts",
  "chore": "Other changes that don't modify source or test files",
  "revert": "Reverts a previous commit",
  "feat": "A new feature",
  "fix": "A bug fix"
}`,
}

// ConventionalCommitRules contains the specification for conventional commits
const ConventionalCommitRules = `
Conventional Commits 1.0.0 Rules:

1. Commit messages MUST be structured as follows:
   <type>[optional scope]: <description>
   [optional body]
   [optional footer(s)]

2. Types:
   - fix: patches a bug (correlates with PATCH in SemVer)
   - feat: introduces a new feature (correlates with MINOR in SemVer)
   - Other types allowed: build, chore, ci, docs, style, refactor, perf, test

3. BREAKING CHANGE:
   - A commit with footer "BREAKING CHANGE:" or with "!" after type/scope introduces a breaking API change
   - Example: feat!: breaking change or feat: new feature with footer BREAKING CHANGE: description

4. Scope:
   - A scope MAY be provided in parentheses after the type: feat(parser): add ability to parse arrays

5. Format Rules:
   - Types MUST be lowercase (feat, fix, docs, etc.)
   - Description MUST immediately follow the colon and space
   - A longer commit body MUST be provided after a blank line following the description when include_body is true
   - A body is required when include_body is set to true, otherwise it is optional
   - When provided, the body must be meaningful and explain what changes were made and why
   - Footer MUST be separated by a blank line and follow the format "token: value" or "token # value"
   - Breaking changes MUST be indicated with "!" before colon or as "BREAKING CHANGE:" in footer

6. Examples:
   - fix: correct minor typos in code
   - feat(api): add ability to search by date
   - docs: correct spelling of CHANGELOG
   - feat!: send email when product is shipped (breaking change)
   - feat: add user authentication

     Implement secure user authentication with password hashing and session management.
`

// CommitMessage represents a structured commit message
type CommitMessage struct {
	Type    string `json:"type"`
	Scope   string `json:"scope"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// EnhancedFileInfo contains detailed information about a changed file
type EnhancedFileInfo struct {
	Path             string `json:"path"`              // File path
	AddedLines       int    `json:"added_lines"`       // Number of added lines
	RemovedLines     int    `json:"removed_lines"`     // Number of removed lines
	Summary          string `json:"summary"`           // Brief description of the file
	FirstLines       string `json:"first_lines"`       // First N lines of the file
	FileType         string `json:"file_type"`         // Type of the file based on extension
	PercentageChange string `json:"percentage_change"` // Percentage of the file that was changed
}

// FormatCommitMessage formats a CommitMessage into a string according to the configuration
func FormatCommitMessage(msg CommitMessage, cfg *config.Config) string {
	var result strings.Builder

	// Format the subject line according to convention
	switch cfg.Commit.Convention {
	case config.ConventionalCommits:
		if msg.Scope != "" {
			result.WriteString(fmt.Sprintf("%s(%s): %s", msg.Type, msg.Scope, msg.Subject))
		} else {
			result.WriteString(fmt.Sprintf("%s: %s", msg.Type, msg.Subject))
		}
	case config.CustomConvention:
		// For custom convention, we assume the AI has already formatted according to template
		result.WriteString(msg.Subject)
	default:
		result.WriteString(msg.Subject)
	}

	// Add body if configured and provided
	if cfg.Commit.IncludeBody && msg.Body != "" {
		result.WriteString("\n\n")
		result.WriteString(msg.Body)
	}

	return result.String()
}

// GenerateTextPrompt creates a natural language prompt for commit message generation
// This function generates a more human-readable prompt compared to the JSON template approach
func GenerateTextPrompt(cfg *config.Config, files []string, changes string) string {
	// Determine the commit convention type
	conventionType := ""
	if cfg.Commit.Convention == config.ConventionalCommits {
		conventionType = "conventional"
	}

	if cfg.AI.Debug {
		debugPrint(cfg, "TEXT PROMPT CONVENTION", conventionType)
	}

	// Build the prompt with structured information
	prompts := []string{
		"Generate a concise git commit message written in present tense for the following code changes with these specifications:",
	}

	// Add specific format requirements for conventional commits first to emphasize importance
	if cfg.Commit.Convention == config.ConventionalCommits {
		prompts = append(prompts, "YOUR RESPONSE MUST START WITH A CONVENTIONAL COMMIT TYPE. Valid types are: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert.")
		prompts = append(prompts, "Format MUST BE: type(optional-scope): subject")
		prompts = append(prompts, "Example: fix(parser): correct array parsing issue")
		prompts = append(prompts, "DO NOT START YOUR RESPONSE WITH A COLON. The type MUST come first, followed by colon.")
	}

	prompts = append(prompts, fmt.Sprintf("CRITICAL: Commit message subject MUST NOT exceed %d characters total. YOU MUST COUNT THE CHARACTERS YOURSELF AND ENSURE THE TOTAL IS UNDER %d. This is a HARD REQUIREMENT.", cfg.Commit.MaxLength, cfg.Commit.MaxLength))

	// Add body instructions based on configuration
	if cfg.Commit.IncludeBody {
		prompts = append(prompts, fmt.Sprintf("STRICT REQUIREMENT: Include a commit body that is VERY CONCISE and MUST NOT exceed %d characters. BE EXTREMELY BRIEF. DO NOT include line statistics (+/-), file lists, or raw metadata from the diff. NO fluffy descriptions. FOCUS ONLY on actual code changes in direct, technical language. AVOID unnecessary details. BE TERSE. BODY IS ABSOLUTELY REQUIRED AND MUST NOT BE EMPTY OR OMITTED.", cfg.Commit.MaxBodyLength))
	} else {
		prompts = append(prompts, "Do not include a commit body, only provide the subject line.")
	}

	prompts = append(prompts, "Exclude anything unnecessary. Your response will be passed directly into git commit.")

	// Add conventional commit rules if using that convention
	if cfg.Commit.Convention == config.ConventionalCommits {
		prompts = append(prompts, "You MUST follow these conventional commit rules:")
		prompts = append(prompts, ConventionalCommitRules)
	}

	// Add type description if using a specific convention
	if description, ok := CommitTypeDescriptions[conventionType]; ok && description != "" {
		prompts = append(prompts, description)
	}

	// Add format specification
	if format, ok := CommitTypeFormats[conventionType]; ok {
		formatExample := format
		if cfg.Commit.IncludeBody {
			formatExample += "\n\n<descriptive body explanation>"
		}
		prompts = append(prompts, fmt.Sprintf("The output response must be in format:\n%s", formatExample))
	}

	// Add specific limit instructions for conventional commits
	if cfg.Commit.Convention == config.ConventionalCommits {
		prompts = append(prompts, fmt.Sprintf("For conventional commits: CRITICAL AND MOST IMPORTANT INSTRUCTION: TOTAL length of 'type(scope): subject' MUST BE STRICTLY LESS THAN %d characters. Count all characters including type, scope, colons, spaces, and subject text. Keep subject extremely brief to ensure total length stays under %d.", cfg.Commit.MaxLength, cfg.Commit.MaxLength))
		prompts = append(prompts, fmt.Sprintf("Examples of good length subjects:\n- fix: update validation logic (%d chars)\n- feat(auth): add login timeout (%d chars)",
			len("fix: update validation logic"),
			len("feat(auth): add login timeout")))
	}

	// Add guidance for analyzing the diff
	prompts = append(prompts, `
When analyzing the code changes:
1. Pay careful attention to the actual diff content, ignoring any file structure or summaries
2. Focus on what code was added/removed/modified
3. Look for patterns across multiple files
4. Identify the primary purpose of these changes (feature, bug fix, refactor, etc.)
5. Be specific about what changed but keep it concise
`)

	// Debug prompt structure before adding file data
	if cfg.AI.Debug {
		debugPrint(cfg, "TEXT PROMPT STRUCTURE", prompts)
	}

	// Add the git diff FIRST if enabled - this is the most important contextual information
	if cfg.Context.IncludeDiff {
		// Check if changes appears to be a git diff format (from GetGitDiff function)
		if strings.Contains(changes, "diff --git") || strings.Contains(changes, "# Summary of changes") {
			prompts = append(prompts, fmt.Sprintf("\nGit Diff:\n```\n%s\n```", changes))
		} else {
			// Truncate changes if they're too long
			originalLength := len(changes)
			if len(changes) > cfg.Context.MaxContextLength {
				changes = changes[:cfg.Context.MaxContextLength] + "...[truncated]"
				if cfg.AI.Debug {
					debugPrint(cfg, "TRUNCATED CHANGES", fmt.Sprintf("Original: %d chars, Truncated: %d chars", originalLength, cfg.Context.MaxContextLength))
				}
			}
			prompts = append(prompts, fmt.Sprintf("\nDiff changes:\n```\n%s\n```", changes))
		}
	}

	// Add repository structure if enabled (as secondary context)
	if cfg.Context.IncludeRepoStructure {
		repoStructure, err := GetRepoStructure(cfg)
		if err == nil && repoStructure != "" {
			prompts = append(prompts, "\n"+repoStructure)
		}
	}

	// Gather enhanced file information if any enhanced options are enabled
	if cfg.Context.IncludeFileStats || cfg.Context.IncludeFileSummaries || cfg.Context.ShowFirstLinesOfFile > 0 {
		enhancedInfos, err := GatherEnhancedFileInfo(cfg, files)
		if err == nil && len(enhancedInfos) > 0 {
			// Add detailed file information section
			prompts = append(prompts, "\nFile changes in detail:")

			for _, info := range enhancedInfos {
				fileDetails := []string{fmt.Sprintf("* %s", info.Path)}

				// Add file type and summary
				if info.Summary != "" {
					fileDetails = append(fileDetails, fmt.Sprintf("  Type: %s - %s", strings.ToUpper(info.FileType), info.Summary))
				} else if info.FileType != "" {
					fileDetails = append(fileDetails, fmt.Sprintf("  Type: %s", strings.ToUpper(info.FileType)))
				}

				// Add change statistics
				if cfg.Context.IncludeFileStats && (info.AddedLines > 0 || info.RemovedLines > 0) {
					fileDetails = append(fileDetails, fmt.Sprintf("  Changes: +%d/-%d lines", info.AddedLines, info.RemovedLines))
					if info.PercentageChange != "" {
						fileDetails = append(fileDetails, fmt.Sprintf("  Modified: %s of file", info.PercentageChange))
					}
				}

				// Add first lines of the file if available (but not if diff is included to avoid duplication)
				if info.FirstLines != "" && !cfg.Context.IncludeDiff {
					fileDetails = append(fileDetails, fmt.Sprintf("  First %d lines:\n```\n%s\n```",
						cfg.Context.ShowFirstLinesOfFile, info.FirstLines))
				}

				prompts = append(prompts, strings.Join(fileDetails, "\n"))
			}
		}
	} else if cfg.Context.IncludeFileNames {
		// Just add the file names if detailed info is not enabled
		prompts = append(prompts, fmt.Sprintf("\nFiles changed:\n%s", strings.Join(files, "\n")))
	}

	return strings.Join(prompts, "\n")
}

// ParseCommitMessageJSON attempts to parse a JSON response into a CommitMessage struct
func ParseCommitMessageJSON(response string) (CommitMessage, error) {
	var msg CommitMessage
	var parseErr error

	// First try to extract JSON from the response if it contains other text
	jsonStr := extractJSON(response)
	if jsonStr != "" {
		// Try to unmarshal the extracted JSON
		if err := json.Unmarshal([]byte(jsonStr), &msg); err == nil {
			// Successfully parsed extracted JSON
			return msg, nil
		} else {
			parseErr = err
		}
	}

	// Next, try to unmarshal the whole response as JSON
	if err := json.Unmarshal([]byte(response), &msg); err == nil {
		// Successfully parsed whole response as JSON
		return msg, nil
	} else if parseErr == nil {
		parseErr = err
	}

	// If both JSON parsing attempts failed, try to parse as text
	extractedMsg := parseTextCommitMessage(response)

	// Check if we extracted anything meaningful
	if extractedMsg.Subject == "" && extractedMsg.Type == "" {
		// Nothing useful extracted, return error
		return extractedMsg, fmt.Errorf("failed to parse response as JSON: %v", parseErr)
	}

	// Return the text-parsed message with no error
	return extractedMsg, nil
}

// extractJSON attempts to extract a JSON object from text that might contain other content
func extractJSON(text string) string {
	// Look for JSON object start and end
	start := strings.Index(text, "{")
	if start == -1 {
		return ""
	}

	// Find matching closing brace
	depth := 1
	for end := start + 1; end < len(text); end++ {
		switch text[end] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[start : end+1]
			}
		}
	}

	return ""
}

// parseTextCommitMessage attempts to parse a plain text commit message
func parseTextCommitMessage(text string) CommitMessage {
	lines := strings.Split(text, "\n")
	msg := CommitMessage{}

	// Look for [SUBJECT] and [BODY] markers
	subjectIndex := -1
	bodyIndex := -1

	for i, line := range lines {
		if strings.Contains(line, "[SUBJECT]") {
			subjectIndex = i
		} else if strings.Contains(line, "[BODY]") {
			bodyIndex = i
		}
	}

	// Handle [SUBJECT] tag if found
	if subjectIndex >= 0 && subjectIndex < len(lines)-1 {
		subject := lines[subjectIndex+1]

		// Clean up any remaining tags
		subject = strings.TrimSpace(strings.ReplaceAll(subject, "[SUBJECT]", ""))

		// Check for conventional commit format
		if idx := strings.Index(subject, ":"); idx > 0 {
			typeScope := subject[:idx]
			msg.Subject = strings.TrimSpace(subject[idx+1:])

			// Check for scope in parentheses
			if scopeStart := strings.Index(typeScope, "("); scopeStart > 0 {
				scopeEnd := strings.Index(typeScope, ")")
				if scopeEnd > scopeStart {
					msg.Type = typeScope[:scopeStart]
					msg.Scope = typeScope[scopeStart+1 : scopeEnd]
				} else {
					msg.Type = typeScope
				}
			} else {
				msg.Type = typeScope
			}
		} else {
			msg.Subject = subject
		}
	} else if len(lines) > 0 {
		// No [SUBJECT] tag found, use first line
		subject := strings.TrimSpace(lines[0])

		// Skip any leading ":" without a type (this fixes the issue of incorrect parsing)
		if strings.HasPrefix(subject, ": ") {
			subject = strings.TrimSpace(subject[2:])
			// Apply default type since no type was provided
			msg.Type = "chore"
			msg.Subject = subject
		} else if idx := strings.Index(subject, ":"); idx > 0 {
			// Check for conventional commit format with type
			typeScope := subject[:idx]
			msg.Subject = strings.TrimSpace(subject[idx+1:])

			// Check for scope in parentheses
			if scopeStart := strings.Index(typeScope, "("); scopeStart > 0 {
				scopeEnd := strings.Index(typeScope, ")")
				if scopeEnd > scopeStart {
					msg.Type = typeScope[:scopeStart]
					msg.Scope = typeScope[scopeStart+1 : scopeEnd]
				} else {
					msg.Type = typeScope
				}
			} else {
				msg.Type = typeScope
			}
		} else {
			// No conventional format found, default to chore type
			msg.Type = "chore"
			msg.Subject = subject
		}
	}

	// Ensure we have a valid type for conventional commits
	if msg.Type == "" {
		msg.Type = "chore" // Apply default type if none found
	}

	// Handle [BODY] tag if found
	if bodyIndex >= 0 && bodyIndex < len(lines)-1 {
		bodyLines := lines[bodyIndex+1:]
		// Remove any empty lines at the start of the body
		for len(bodyLines) > 0 && strings.TrimSpace(bodyLines[0]) == "" {
			bodyLines = bodyLines[1:]
		}
		if len(bodyLines) > 0 {
			msg.Body = strings.Join(bodyLines, "\n")
		}
	} else if len(lines) > 1 {
		// No [BODY] tag found, look for body after a blank line or double newline
		var bodyLines []string
		foundEmptyLine := false

		for i := 1; i < len(lines); i++ {
			line := lines[i]

			// First check if we've found an empty line to separate subject from body
			if !foundEmptyLine && strings.TrimSpace(line) == "" {
				foundEmptyLine = true
				continue
			}

			// If we've found an empty line separator, add subsequent non-empty lines to body
			if foundEmptyLine && strings.TrimSpace(line) != "" {
				bodyLines = append(bodyLines, line)
			}
		}

		// If no empty line was found but there are more lines after first line,
		// assume lines after first are the body (especially for text prompt format)
		if !foundEmptyLine && len(lines) > 2 {
			// Skip the first line (subject) and any immediate empty line
			startIdx := 1
			if strings.TrimSpace(lines[1]) == "" {
				startIdx = 2
			}
			bodyLines = lines[startIdx:]
		}

		if len(bodyLines) > 0 {
			msg.Body = strings.Join(bodyLines, "\n")
		}
	}

	// Clean up body (remove markdown formatting or template placeholders)
	if msg.Body != "" {
		// Remove placeholder text if it appears to be template text
		if strings.Contains(strings.ToLower(msg.Body), "<descriptive body") ||
			strings.Contains(strings.ToLower(msg.Body), "explanat") ||
			strings.Contains(strings.ToLower(msg.Body), "<commit message>") ||
			strings.Contains(strings.ToLower(msg.Body), "<optional body>") {
			msg.Body = ""
		}

		// Remove markdown code block delimiters if present
		msg.Body = strings.ReplaceAll(msg.Body, "```", "")

		// Remove common template markers
		msg.Body = strings.ReplaceAll(msg.Body, "[BODY]", "")

		// Some AIs return the word "Body:" at the start - remove it
		msg.Body = strings.TrimPrefix(strings.TrimSpace(msg.Body), "Body:")
		msg.Body = strings.TrimPrefix(strings.TrimSpace(msg.Body), "body:")

		// Ensure body is properly separated from subject
		if !strings.Contains(msg.Body, "\n\n") {
			msg.Body = "\n\n" + msg.Body
		}
	}

	// Ensure body is properly trimmed
	msg.Body = strings.TrimSpace(msg.Body)

	return msg
}

// DisplayStagedFiles prints the staged files in a modern TUI format
func DisplayStagedFiles(files []string) {
	// Get current branch name
	branch := "master" // Default if we can't get the branch
	cmdBranch := exec.Command("git", "branch", "--show-current")
	branchOutput, err := cmdBranch.Output()
	if err == nil {
		branch = strings.TrimSpace(string(branchOutput))
	}

	// Get staged and modified files counts
	stagedCount := len(files)
	modifiedCount := 0
	cmdStatus := exec.Command("git", "status", "--porcelain")
	statusOutput, err := cmdStatus.Output()
	if err == nil {
		for _, line := range strings.Split(string(statusOutput), "\n") {
			if len(line) > 0 && !strings.HasPrefix(line, "??") && !strings.HasPrefix(line, " ") {
				// Count modified but not staged files
				if !strings.HasPrefix(line, "A") && !strings.HasPrefix(line, "M") {
					modifiedCount++
				}
			}
		}
	}

	// Print header with branch and status
	fmt.Printf("\n\033[1;36mcommitron\033[0m \033[38;5;244m%s\033[0m", branch)
	if stagedCount > 0 {
		fmt.Printf(" \033[1;32m●%d\033[0m", stagedCount)
	}
	if modifiedCount > 0 {
		fmt.Printf(" \033[1;33m✚%d\033[0m", modifiedCount)
	}
	fmt.Println()

	// Print staged changes section
	fmt.Println("\n\033[1;36m📦 Staged Changes\033[0m")

	// Print files with icons based on file type
	for _, file := range files {
		// Get file extension and name
		ext := strings.ToLower(filepath.Ext(file))
		if ext != "" {
			ext = ext[1:] // Remove the dot
		}
		name := filepath.Base(file)

		// Get appropriate icon
		icon := ui.GetIconForFile(name, ext)
		fmt.Printf("   \033[38;5;244m%s\033[0m %s\n", icon, file)
	}

	// Print analyzing message
	fmt.Println("\n\033[1;36m🔍 Analyzing changes...\033[0m")
}

// getFileIcon returns an appropriate icon based on file extension
func getFileIcon(file string) string {
	ext := strings.ToLower(filepath.Ext(file))
	switch ext {
	case ".go":
		return "🔵"
	case ".js", ".jsx", ".ts", ".tsx":
		return "🟡"
	case ".py":
		return "🟢"
	case ".md":
		return "📝"
	case ".yaml", ".yml", ".json":
		return "⚙️"
	case ".css", ".scss", ".sass":
		return "🎨"
	case ".html", ".htm":
		return "🌐"
	case ".sh", ".bash":
		return "🐚"
	case ".dockerfile", ".docker":
		return "🐳"
	default:
		return "📄"
	}
}

// wrapText wraps text at the specified width while preserving indentation
func wrapText(text string, width int, indent string) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	currentLine := indent + words[0]

	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = indent + word
		}
	}
	lines = append(lines, currentLine)

	return strings.Join(lines, "\n")
}

// DisplayCommitMessage shows the generated commit message with a modern UI
func DisplayCommitMessage(commitMsg string) (bool, error) {
	// Print header
	fmt.Println("\n\033[1;36m💬 Generated Commit Message\033[0m")
	fmt.Println("\033[38;5;244m────────────────────────\033[0m")

	// Display the commit message with proper formatting
	lines := strings.Split(commitMsg, "\n")
	inBody := false
	indentation := "   " // Base indentation for all lines

	for i, line := range lines {
		if line == "" {
			fmt.Println()
			if i < len(lines)-1 {
				inBody = true
			}
			continue
		}

		if inBody {
			// For body text, wrap at 80 characters
			// Check if line contains a file reference
			if strings.Contains(strings.ToLower(line), "file:") || strings.Contains(strings.ToLower(line), "files:") {
				// Extract file name if present
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					filePart := strings.TrimSpace(parts[1])
					// Try to extract file name from the text
					if strings.Contains(filePart, " ") {
						filePart = strings.Split(filePart, " ")[0]
					}
					// Get file extension and name
					ext := strings.ToLower(filepath.Ext(filePart))
					if ext != "" {
						ext = ext[1:] // Remove the dot
					}
					name := filepath.Base(filePart)
					// Get appropriate icon
					icon := ui.GetIconForFile(name, ext)
					// Replace the file name with icon + file name
					line = strings.Replace(line, filePart, icon+" "+filePart, 1)
				}
			}
			wrappedText := wrapText(line, 80, indentation)
			fmt.Printf("\033[38;5;252m%s\033[0m\n", wrappedText)
		} else {
			// For subject line, don't wrap
			fmt.Printf("%s\033[38;5;252m%s\033[0m\n", indentation, line)
		}
	}

	// Print confirmation prompt
	fmt.Println("\n\033[1;36m❓ Use this commit message?\033[0m")
	fmt.Print("\033[38;5;244m   [Y] Yes  [N] No\033[0m\n\n")

	// Get user input for confirmation
	fmt.Print("\033[1;36m> \033[0m")
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil && err.Error() != "unexpected newline" {
		return false, err
	}

	// Convert response to lowercase for easier matching
	response = strings.ToLower(response)

	// Check if the response is affirmative
	return response == "y" || response == "yes" || response == "", nil
}

// DisplayAnalysisComplete prints a completion message
func DisplayAnalysisComplete() {
	fmt.Println("\033[1;32m✓ Analysis complete\033[0m")
}

// GetGitDiff returns a more comprehensive git diff for the staged files
func GetGitDiff(files []string) (string, error) {
	var fullDiff strings.Builder

	// Add a header with overall information
	cmd := exec.Command("git", "diff", "--staged", "--stat")
	statOutput, err := cmd.Output()
	if err == nil {
		fullDiff.WriteString("# Summary of changes:\n")
		fullDiff.WriteString(string(statOutput))
		fullDiff.WriteString("\n")
	}

	// Get detailed diff for each file
	for _, file := range files {
		// Get file type for better context
		fileExt := strings.TrimPrefix(filepath.Ext(file), ".")
		if fileExt == "" {
			fileExt = "text"
		}

		// Add file header
		fullDiff.WriteString(fmt.Sprintf("\n# File: %s (type: %s)\n", file, fileExt))

		// Get the actual diff with more context lines
		cmd := exec.Command("git", "diff", "--staged", "-U10", "--", file)
		diffOutput, err := cmd.Output()
		if err == nil {
			fullDiff.WriteString(string(diffOutput))
		} else {
			fullDiff.WriteString(fmt.Sprintf("Error getting diff: %s\n", err))
		}

		fullDiff.WriteString("\n")
	}

	return fullDiff.String(), nil
}

// GenerateCommitMessage generates a commit message using the configured AI provider
func GenerateCommitMessage(cfg *config.Config, files []string, changes string) (string, error) {
	// Display staged files in TUI format if enabled
	if cfg.UI.EnableTUI {
		DisplayStagedFiles(files)
	}

	// Get more detailed git diff if requested
	var detailedDiff string
	var err error
	if cfg.Context.IncludeDiff {
		detailedDiff, err = GetGitDiff(files)
		if err == nil && detailedDiff != "" {
			// Use the detailed diff instead of the basic changes
			changes = detailedDiff
		}
	}

	// Debug: Show input data
	if cfg.AI.Debug {
		debugPrint(cfg, "INPUT FILES", files)
		debugPrint(cfg, "INPUT CHANGES", changes)
		debugPrint(cfg, "CONFIG SETTINGS", map[string]interface{}{
			"Convention":       cfg.Commit.Convention,
			"IncludeBody":      cfg.Commit.IncludeBody,
			"MaxLength":        cfg.Commit.MaxLength,
			"MaxBodyLength":    cfg.Commit.MaxBodyLength,
			"Provider":         cfg.AI.Provider,
			"Model":            cfg.AI.Model,
			"MaxContextLength": cfg.Context.MaxContextLength,
		})
	}

	var prompt string

	// Choose between JSON template approach and text prompt approach
	if cfg.Commit.Convention == config.ConventionalCommits {
		// Use the more detailed text prompt for conventional commits
		prompt = GenerateTextPrompt(cfg, files, changes)
	} else {
		// Use the JSON template approach for other conventions
		prompt = buildPrompt(cfg, files, changes)
	}

	// Debug: Show the prompt being sent to the AI
	debugPrint(cfg, "AI PROMPT", prompt)

	var rawResponse string

	// Choose the AI provider based on the configuration
	switch cfg.AI.Provider {
	case config.OpenAI:
		rawResponse, err = generateWithOpenAI(cfg, prompt)
	case config.Gemini:
		rawResponse, err = generateWithGemini(cfg, prompt)
	case config.Ollama:
		rawResponse, err = generateWithOllama(cfg, prompt)
	case config.Claude:
		rawResponse, err = generateWithClaude(cfg, prompt)
	case config.Custom:
		rawResponse, err = generateWithCustomAPI(cfg, prompt)
	default:
		return "", fmt.Errorf("unsupported AI provider: %s", cfg.AI.Provider)
	}

	if err != nil {
		debugPrint(cfg, "AI ERROR", err.Error())
		return "", err
	}

	// Display that analysis is complete
	if cfg.UI.EnableTUI {
		DisplayAnalysisComplete()
	}

	// Debug: Show the raw response from the AI
	debugPrint(cfg, "AI RESPONSE", rawResponse)

	// Parse the response into a structured CommitMessage
	commitMsg, err := ParseCommitMessageJSON(rawResponse)
	if err != nil {
		debugPrint(cfg, "PARSING ERROR", err.Error())
		// For conventional commits, ensure we have at least a type
		if cfg.Commit.Convention == config.ConventionalCommits {
			// If parsing failed but we can extract something useful from the raw text
			if strings.Contains(rawResponse, ": ") {
				parts := strings.SplitN(rawResponse, ": ", 2)
				if len(parts) == 2 {
					potential_type := strings.TrimSpace(parts[0])
					// Check if this could be a valid type
					if isValidCommitType(potential_type) {
						commitMsg.Type = potential_type
						commitMsg.Subject = strings.TrimSpace(parts[1])
						// Use the rest as body if applicable
						if cfg.Commit.IncludeBody && strings.Contains(commitMsg.Subject, "\n\n") {
							bodyParts := strings.SplitN(commitMsg.Subject, "\n\n", 2)
							if len(bodyParts) == 2 {
								commitMsg.Subject = bodyParts[0]
								commitMsg.Body = bodyParts[1]
							}
						}
						debugPrint(cfg, "MANUAL PARSING SUCCESSFUL", commitMsg)
					} else {
						// Default to a generic type
						commitMsg.Type = "chore"
						commitMsg.Subject = rawResponse
					}
				}
			} else {
				commitMsg.Type = "chore"
				commitMsg.Subject = rawResponse
			}
		} else {
			return rawResponse, nil // Fall back to raw response if parsing fails for non-conventional format
		}
	}

	// Debug: Show the parsed commit message
	debugPrint(cfg, "PARSED COMMIT", commitMsg)

	// Ensure the body is not empty if it's required
	if cfg.Commit.IncludeBody && (commitMsg.Body == "" || strings.TrimSpace(commitMsg.Body) == "") {
		// If no body was parsed, extract a reasonable body from the changes
		commitMsg.Body = generateDefaultBody(cfg, files, changes)
		debugPrint(cfg, "GENERATED DEFAULT BODY", commitMsg.Body)
	}

	// Verify message length constraints before formatting
	subjectLength := 0
	if cfg.Commit.Convention == config.ConventionalCommits && commitMsg.Type != "" {
		// For conventional commits, calculate full subject with type and scope
		if commitMsg.Scope != "" {
			subjectLength = len(commitMsg.Type) + len(commitMsg.Scope) + len(commitMsg.Subject) + 4 // +4 for "(): "
		} else {
			subjectLength = len(commitMsg.Type) + len(commitMsg.Subject) + 2 // +2 for ": "
		}
	} else {
		subjectLength = len(commitMsg.Subject)
	}

	// Check if subject exceeds max length - hard enforce the limit
	if subjectLength > cfg.Commit.MaxLength {
		// Always attempt to truncate the subject to meet the limit
		if cfg.Commit.Convention == config.ConventionalCommits && commitMsg.Type != "" {
			// Calculate maximum space available for the subject
			maxSubjectSpace := cfg.Commit.MaxLength
			if commitMsg.Scope != "" {
				maxSubjectSpace = cfg.Commit.MaxLength - len(commitMsg.Type) - len(commitMsg.Scope) - 4
			} else {
				maxSubjectSpace = cfg.Commit.MaxLength - len(commitMsg.Type) - 2
			}

			// Truncate subject if there's any space left
			if maxSubjectSpace > 3 {
				// Preserve meaning by truncating smartly - take first part of subject
				originalSubject := commitMsg.Subject
				if maxSubjectSpace < len(originalSubject) {
					// Find a good breaking point (space, comma, etc.) if possible
					breakPoint := maxSubjectSpace - 3
					for i := breakPoint; i > breakPoint-10 && i > 0; i-- {
						if originalSubject[i] == ' ' || originalSubject[i] == ',' || originalSubject[i] == ';' {
							breakPoint = i
							break
						}
					}
					commitMsg.Subject = originalSubject[:breakPoint] + "..."
				}

				// Recalculate the total length
				if commitMsg.Scope != "" {
					subjectLength = len(commitMsg.Type) + len(commitMsg.Scope) + len(commitMsg.Subject) + 4
				} else {
					subjectLength = len(commitMsg.Type) + len(commitMsg.Subject) + 2
				}
			}
		} else {
			// For non-conventional commits, just truncate the subject
			if len(commitMsg.Subject) > cfg.Commit.MaxLength {
				// Find a good breaking point (space, comma, etc.) if possible
				breakPoint := cfg.Commit.MaxLength - 3
				for i := breakPoint; i > breakPoint-10 && i > 0; i-- {
					if commitMsg.Subject[i] == ' ' || commitMsg.Subject[i] == ',' || commitMsg.Subject[i] == ';' {
						breakPoint = i
						break
					}
				}
				commitMsg.Subject = commitMsg.Subject[:breakPoint] + "..."
				subjectLength = len(commitMsg.Subject)
			}
		}

		// If still too long after truncation, force more aggressive truncation
		if subjectLength > cfg.Commit.MaxLength {
			if cfg.Commit.Convention == config.ConventionalCommits && commitMsg.Type != "" {
				// For conventional commits, preserve type and scope, but severely truncate subject
				fixedType := commitMsg.Type
				fixedScope := commitMsg.Scope

				availableSpace := cfg.Commit.MaxLength
				if fixedScope != "" {
					availableSpace = cfg.Commit.MaxLength - len(fixedType) - len(fixedScope) - 4
				} else {
					availableSpace = cfg.Commit.MaxLength - len(fixedType) - 2
				}

				// Ensure minimum subject space
				if availableSpace < 10 {
					// If necessary, truncate scope to make room for subject
					if fixedScope != "" && len(fixedScope) > 5 {
						fixedScope = fixedScope[:5]
						if fixedScope != "" {
							availableSpace = cfg.Commit.MaxLength - len(fixedType) - len(fixedScope) - 4
						} else {
							availableSpace = cfg.Commit.MaxLength - len(fixedType) - 2
						}
					}
				}

				// Create a very brief subject if needed
				if availableSpace < 10 {
					commitMsg.Subject = "update"
				} else {
					commitMsg.Subject = commitMsg.Subject[:availableSpace-3] + "..."
				}

				// Update the values
				commitMsg.Type = fixedType
				commitMsg.Scope = fixedScope

				// Recalculate final length
				if commitMsg.Scope != "" {
					subjectLength = len(commitMsg.Type) + len(commitMsg.Scope) + len(commitMsg.Subject) + 4
				} else {
					subjectLength = len(commitMsg.Type) + len(commitMsg.Subject) + 2
				}
			} else {
				// For other commits, hard truncate
				commitMsg.Subject = commitMsg.Subject[:cfg.Commit.MaxLength-3] + "..."
				subjectLength = len(commitMsg.Subject)
			}

			// Add debug entry showing we did aggressive truncation
			debugPrint(cfg, "AGGRESSIVE TRUNCATION", fmt.Sprintf("Truncated subject to length %d", subjectLength))
		}
	}

	// Check if body exceeds max length when body is included
	if cfg.Commit.IncludeBody && len(commitMsg.Body) > cfg.Commit.MaxBodyLength {
		// Truncate the body to the maximum allowed length
		commitMsg.Body = commitMsg.Body[:cfg.Commit.MaxBodyLength-3] + "..."
		debugPrint(cfg, "TRUNCATED BODY", commitMsg.Body)
	}

	// Validate against conventional commit rules if needed
	if cfg.Commit.Convention == config.ConventionalCommits {
		if err := validateConventionalCommit(commitMsg, cfg); err != nil {
			debugPrint(cfg, "CONVENTIONAL COMMIT VALIDATION ERROR", err.Error())
			// Try to fix common issues
			commitMsg = fixConventionalCommitIssues(commitMsg)

			// Re-validate after fixing
			if err := validateConventionalCommit(commitMsg, cfg); err != nil && cfg.Commit.IncludeBody && (commitMsg.Body == "" || strings.TrimSpace(commitMsg.Body) == "") {
				// If the body is still empty, add a minimal body
				commitMsg.Body = generateDefaultBody(cfg, files, changes)
				debugPrint(cfg, "ADDED DEFAULT BODY", commitMsg.Body)
			}
		}
	}

	// Format the message according to the configuration
	formattedMessage := FormatCommitMessage(commitMsg, cfg)

	// Debug: Show the final formatted message
	debugPrint(cfg, "FINAL COMMIT MESSAGE", formattedMessage)

	// Display the commit message in TUI format and ask for confirmation if enabled
	if cfg.UI.EnableTUI && cfg.UI.ConfirmCommit {
		confirmed, err := DisplayCommitMessage(formattedMessage)
		if err != nil {
			return "", fmt.Errorf("error getting confirmation: %w", err)
		}

		if !confirmed {
			return "", fmt.Errorf("commit message rejected by user")
		}
	}

	return formattedMessage, nil
}

// generateDefaultBody creates a basic commit body when the AI doesn't provide one
func generateDefaultBody(cfg *config.Config, files []string, changes string) string {
	// Default basic description
	defaultBody := "Update code with necessary changes"

	// Try to generate a more meaningful body based on the changes
	if len(files) == 1 {
		// If only one file was changed, mention it
		fileExt := strings.TrimPrefix(filepath.Ext(files[0]), ".")
		fileName := filepath.Base(files[0])

		if fileExt != "" {
			switch fileExt {
			case "go":
				return fmt.Sprintf("Update %s with improved Go code implementation", fileName)
			case "js", "jsx", "ts", "tsx":
				return fmt.Sprintf("Enhance %s with better JavaScript/TypeScript functionality", fileName)
			case "py":
				return fmt.Sprintf("Update Python implementation in %s", fileName)
			case "md", "markdown":
				return fmt.Sprintf("Improve documentation in %s", fileName)
			case "css", "scss", "sass":
				return fmt.Sprintf("Update styles in %s", fileName)
			case "html":
				return fmt.Sprintf("Update HTML template in %s", fileName)
			case "json", "yaml", "yml":
				return fmt.Sprintf("Update configuration in %s", fileName)
			default:
				return fmt.Sprintf("Update %s file", fileName)
			}
		} else {
			return fmt.Sprintf("Update %s", fileName)
		}
	} else if len(files) > 1 {
		// If multiple files were changed, provide a count
		return fmt.Sprintf("Update %d files with necessary changes", len(files))
	}

	return defaultBody
}

// buildPrompt creates a prompt for the AI based on the configuration using JSON templates
func buildPrompt(cfg *config.Config, files []string, changes string) string {
	// Debug which template is being used
	if cfg.AI.Debug {
		templateType := "Basic template"
		switch cfg.Commit.Convention {
		case config.ConventionalCommits:
			templateType = "Conventional commits template"
		case config.CustomConvention:
			templateType = "Custom template: " + cfg.Commit.CustomTemplate
		}
		debugPrint(cfg, "TEMPLATE TYPE", templateType)
	}

	// Serialize files list to JSON
	filesJSON, _ := json.Marshal(files)

	// Extract the most important changes from the diff if it's in our enhanced format
	if strings.Contains(changes, "# Summary of changes") || strings.Contains(changes, "diff --git") {
		// Prioritize the actual diff content and remove unnecessary headers
		enhancedChanges := extractKeyDiffContent(changes)
		if enhancedChanges != "" {
			changes = enhancedChanges
			if cfg.AI.Debug {
				debugPrint(cfg, "USING ENHANCED DIFF", "Using extracted key diff content")
			}
		}
	}

	// Truncate changes if they're too long
	originalLength := len(changes)
	if len(changes) > cfg.Context.MaxContextLength {
		changes = changes[:cfg.Context.MaxContextLength] + "...[truncated]"
		if cfg.AI.Debug {
			debugPrint(cfg, "TRUNCATED CHANGES", fmt.Sprintf("Original: %d chars, Truncated: %d chars", originalLength, cfg.Context.MaxContextLength))
		}
	}

	// Escape changes for JSON
	changesJSON, _ := json.Marshal(changes)

	// Determine if we want subject only based on config
	subjectOnly := !cfg.Commit.IncludeBody

	// Select template based on commit convention
	var template string
	switch cfg.Commit.Convention {
	case config.ConventionalCommits:
		template = fmt.Sprintf(
			ConventionalCommitsJSON,
			cfg.Commit.MaxLength,
			cfg.Commit.MaxBodyLength,
			cfg.Commit.IncludeBody,
			cfg.Commit.IncludeBody, // Pass include_body value to body_required field
			string(filesJSON),
			string(changesJSON),
			subjectOnly,
		)
	case config.CustomConvention:
		template = fmt.Sprintf(
			CustomCommitJSON,
			cfg.Commit.CustomTemplate,
			cfg.Commit.MaxLength,
			cfg.Commit.MaxBodyLength,
			cfg.Commit.IncludeBody,
			string(filesJSON),
			string(changesJSON),
			subjectOnly,
		)
	default:
		template = fmt.Sprintf(
			BaseTemplateJSON,
			cfg.Commit.MaxLength,
			cfg.Commit.MaxBodyLength,
			cfg.Commit.IncludeBody,
			string(filesJSON),
			string(changesJSON),
			subjectOnly,
		)
	}

	// Check if we have a custom system prompt
	hasCustomPrompt := cfg.AI.SystemPrompt != ""

	// Only add specific formatting instructions if no custom system prompt
	if !hasCustomPrompt {
		// Add explicit instructions to return ONLY valid JSON
		bodyInstructions := ""
		if cfg.Commit.IncludeBody {
			bodyInstructions = "YOU MUST INCLUDE A BODY. The body must be concise, direct, and technical - focusing only on actual changes made. DO NOT include line statistics, file lists, or formatting details like '+X/-Y lines'. DO NOT include raw metadata from the diff. NO marketing language or fluffy descriptions. "
		} else {
			bodyInstructions = "DO NOT include a body. "
		}

		conventionalRulesInstructions := ""
		if cfg.Commit.Convention == config.ConventionalCommits {
			conventionalRulesInstructions = "You MUST follow these conventional commit rules:\n" + ConventionalCommitRules + "\n"
			conventionalRulesInstructions += fmt.Sprintf("\nCRITICAL: The TOTAL length of 'type(scope): subject' MUST be under %d characters.\nExamples of good length: 'fix: update validation logic', 'feat(auth): add login timeout'\n", cfg.Commit.MaxLength)
			conventionalRulesInstructions += "\nALWAYS start your response with a valid type. NEVER start with just a colon.\n"
			conventionalRulesInstructions += "CORRECT: 'feat: add feature'\nINCORRECT: ': add feature'\n"
			conventionalRulesInstructions += "\nSTRICT REQUIREMENTS:\n"
			conventionalRulesInstructions += "1. Type MUST be one of: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert\n"
			conventionalRulesInstructions += "2. Type MUST be lowercase\n"
			conventionalRulesInstructions += "3. Subject MUST be lowercase and not end with a period\n"
			conventionalRulesInstructions += "4. Scope (if used) MUST be lowercase and not contain spaces or special characters\n"
			conventionalRulesInstructions += "5. Body MUST be separated from subject by a blank line\n"
			conventionalRulesInstructions += "6. Body MUST be meaningful and explain what changes were made and why\n"
		}

		return "Your task is to create a commit message based on the specifications below. " +
			"EXTREMELY IMPORTANT: Return ONLY a valid JSON object with no explanatory text. " +
			bodyInstructions +
			conventionalRulesInstructions +
			"DO NOT include any natural language explanation, introduction, or conclusion. " +
			"Return JUST the JSON object and nothing else. " +
			"IMPORTANT: Focus on the actual code changes in the diff and what they accomplish. " +
			fmt.Sprintf("CRITICAL: Ensure total commit subject length is UNDER %d characters.\n", cfg.Commit.MaxLength) +
			"Format:\n\n" +
			"For conventional commits, use this exact structure:\n" +
			"{\n" +
			"  \"type\": \"feat\", // One of: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert\n" +
			"  \"scope\": \"optional scope\", // Optional, must be lowercase\n" +
			"  \"subject\": \"concise subject line\", // Must be lowercase, no period\n" +
			"  \"body\": \"" + bodyExample(cfg.Commit.IncludeBody) + "\"\n" +
			"}\n\n" +
			"Here are the specifications:\n\n" + template
	} else {
		// With custom system prompt, just provide the template data
		return "Generate a commit message based on this specification:\n\n" + template
	}
}

// extractKeyDiffContent focuses on the most important parts of the diff, removing headers and metadata
func extractKeyDiffContent(diff string) string {
	// Split the diff into lines
	lines := strings.Split(diff, "\n")
	var result []string
	inActualDiff := false

	for _, line := range lines {
		// Skip summary and header sections
		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "Summary of changes") {
			continue
		}

		// Detect start of actual diff content
		if strings.HasPrefix(line, "diff --git") {
			inActualDiff = true
		}

		if inActualDiff {
			result = append(result, line)
		}
	}

	// If we didn't find any actual diff content, return the original
	if len(result) == 0 {
		return diff
	}

	return strings.Join(result, "\n")
}

// bodyExample returns the appropriate body example text based on whether body is included
func bodyExample(includeBody bool) string {
	if includeBody {
		return "This commit adds critical validation for commit messages to ensure they follow the conventional commit format. The changes include improved error handling, automatic truncation of long messages, and proper formatting of the commit type and subject."
	}
	return "leave empty"
}

// generateWithOpenAI uses OpenAI to generate a commit message
func generateWithOpenAI(cfg *config.Config, prompt string) (string, error) {
	type Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type Request struct {
		Model       string    `json:"model"`
		Messages    []Message `json:"messages"`
		MaxTokens   int       `json:"max_tokens,omitempty"`
		Temperature float64   `json:"temperature,omitempty"`
	}

	type Response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	// Get or create system prompt
	systemPrompt := getSystemPrompt(cfg)

	// Add a prefix emphasizing length requirements regardless of custom prompts
	lengthPrefix := fmt.Sprintf("MOST IMPORTANT INSTRUCTION: Your commit message subject MUST be under %d characters total. ", cfg.Commit.MaxLength)
	if cfg.Commit.Convention == config.ConventionalCommits {
		lengthPrefix += fmt.Sprintf("For conventional commits, this means the ENTIRE string 'type(scope): subject' must be under %d characters. Be extremely brief.", cfg.Commit.MaxLength)
		lengthPrefix += "\n\nYOU MUST START YOUR RESPONSE WITH A CONVENTIONAL COMMIT TYPE. DO NOT START WITH JUST A COLON."
		lengthPrefix += "\nCORRECT FORMAT: 'feat: add new feature'"
		lengthPrefix += "\nINCORRECT FORMAT: ': add new feature'"
		lengthPrefix += "\nValid types are: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert"

		if cfg.Commit.IncludeBody {
			lengthPrefix += "\n\nYOU MUST INCLUDE A COMMIT BODY AFTER THE SUBJECT. The body must be separated from the subject by a blank line."
			lengthPrefix += "\nThe body MUST NOT be empty and should explain what changes were made and why."
		}
	}

	// Prepend the length requirement to any system prompt
	systemPrompt = lengthPrefix + "\n\n" + systemPrompt

	// Create request
	reqBody := Request{
		Model: cfg.AI.Model,
		Messages: []Message{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   cfg.AI.MaxTokens,
		Temperature: cfg.AI.Temperature,
	}

	// Debug: Show the request being sent to OpenAI
	debugPrint(cfg, "OPENAI REQUEST", reqBody)

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Make API request
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.AI.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read response
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Debug: Show the raw API response
	debugPrint(cfg, "OPENAI RAW RESPONSE", string(respData))

	var response Response
	err = json.Unmarshal(respData, &response)
	if err != nil {
		return "", err
	}

	// Check for API error
	if response.Error.Message != "" {
		return "", fmt.Errorf("OpenAI API error: %s", response.Error.Message)
	}

	// Check if we got results
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI API")
	}

	content := strings.TrimSpace(response.Choices[0].Message.Content)

	// For conventional commits, validate the response starts with a valid type
	if cfg.Commit.Convention == config.ConventionalCommits {
		// Fix if the response starts with a colon instead of a type
		if strings.HasPrefix(content, ": ") {
			content = "chore" + content
			debugPrint(cfg, "FIXED RESPONSE FORMAT", content)
		}
	}

	// Return the generated commit message
	return content, nil
}

// generateWithGemini uses Google's Gemini to generate a commit message
func generateWithGemini(cfg *config.Config, prompt string) (string, error) {
	// Add a length requirement prefix to the prompt
	lengthPrefix := fmt.Sprintf("CRITICAL INSTRUCTION: Your commit message subject MUST be under %d characters total. ", cfg.Commit.MaxLength)
	if cfg.Commit.Convention == config.ConventionalCommits {
		lengthPrefix += fmt.Sprintf("For conventional commits, this means the ENTIRE string 'type(scope): subject' must be under %d characters.", cfg.Commit.MaxLength)
		lengthPrefix += "\n\nYOU MUST START YOUR RESPONSE WITH A CONVENTIONAL COMMIT TYPE. DO NOT START WITH JUST A COLON."
		lengthPrefix += "\nCORRECT: 'feat: add new feature'"
		lengthPrefix += "\nINCORRECT: ': add new feature'"
		lengthPrefix += "\nValid types are: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert"

		if cfg.Commit.IncludeBody {
			lengthPrefix += "\n\nYOU MUST INCLUDE A COMMIT BODY AFTER THE SUBJECT. The body must be separated from the subject by a blank line."
			lengthPrefix += "\nThe body MUST NOT be empty and should explain what changes were made and why."
		}
	}

	// Prepend the length requirement to the prompt
	enhancedPrompt := lengthPrefix + "\n\n" + prompt

	type Request struct {
		Contents []struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"contents"`
	}

	type Response struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	// Create request
	reqBody := Request{
		Contents: []struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		}{
			{
				Parts: []struct {
					Text string `json:"text"`
				}{
					{
						Text: enhancedPrompt,
					},
				},
			},
		},
	}

	// Debug: Show the request being sent to Gemini
	debugPrint(cfg, "GEMINI REQUEST", reqBody)

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Make API request
	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", cfg.AI.Model, cfg.AI.APIKey)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(reqData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read response
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Debug: Show the raw API response
	debugPrint(cfg, "GEMINI RAW RESPONSE", string(respData))

	var response Response
	err = json.Unmarshal(respData, &response)
	if err != nil {
		return "", err
	}

	// Check for API error
	if response.Error.Message != "" {
		return "", fmt.Errorf("Gemini API error: %s", response.Error.Message)
	}

	// Check if we got results
	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from Gemini API")
	}

	content := strings.TrimSpace(response.Candidates[0].Content.Parts[0].Text)

	// For conventional commits, validate the response starts with a valid type
	if cfg.Commit.Convention == config.ConventionalCommits {
		// Fix if the response starts with a colon instead of a type
		if strings.HasPrefix(content, ": ") {
			content = "chore" + content
			debugPrint(cfg, "FIXED RESPONSE FORMAT", content)
		}
	}

	// Return the generated commit message
	return content, nil
}

// generateWithOllama uses Ollama (local) to generate a commit message
func generateWithOllama(cfg *config.Config, prompt string) (string, error) {
	// Add a length requirement prefix to the prompt
	lengthPrefix := fmt.Sprintf("CRITICAL INSTRUCTION: Your commit message subject MUST be under %d characters total. ", cfg.Commit.MaxLength)
	if cfg.Commit.Convention == config.ConventionalCommits {
		lengthPrefix += fmt.Sprintf("For conventional commits, this means the ENTIRE string 'type(scope): subject' must be under %d characters.", cfg.Commit.MaxLength)
		lengthPrefix += "\n\nYOU MUST START YOUR RESPONSE WITH A CONVENTIONAL COMMIT TYPE. DO NOT START WITH JUST A COLON."
		lengthPrefix += "\nCORRECT: 'feat: add new feature'"
		lengthPrefix += "\nINCORRECT: ': add new feature'"
		lengthPrefix += "\nValid types are: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert"

		if cfg.Commit.IncludeBody {
			lengthPrefix += "\n\nYOU MUST INCLUDE A COMMIT BODY AFTER THE SUBJECT. The body must be separated from the subject by a blank line."
			lengthPrefix += "\nThe body MUST NOT be empty and should explain what changes were made and why."
		}
	}

	// Prepend the length requirement to the prompt
	enhancedPrompt := lengthPrefix + "\n\n" + prompt

	type Request struct {
		Model       string  `json:"model"`
		Prompt      string  `json:"prompt"`
		Stream      bool    `json:"stream"`
		Temperature float64 `json:"temperature,omitempty"`
		MaxTokens   int     `json:"max_tokens,omitempty"`
	}

	type Response struct {
		Model    string `json:"model"`
		Response string `json:"response"`
	}

	// This is for non-streaming responses
	type ResponseComplete struct {
		Model     string `json:"model"`
		Response  string `json:"response"`
		CreatedAt string `json:"created_at"`
		Done      bool   `json:"done"`
	}

	// Set default host if not specified
	ollamaHost := cfg.AI.OllamaHost
	if ollamaHost == "" {
		ollamaHost = "http://localhost:11434"
	}

	// Create request for the /api/generate endpoint
	reqBody := Request{
		Model:       cfg.AI.Model,
		Prompt:      enhancedPrompt, // Use the enhanced prompt
		Stream:      false,
		Temperature: cfg.AI.Temperature,
		MaxTokens:   cfg.AI.MaxTokens,
	}

	// Debug: Show the request being sent to Ollama
	debugPrint(cfg, "OLLAMA REQUEST", reqBody)

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Debug: Show the Ollama host being used
	debugPrint(cfg, "OLLAMA HOST", ollamaHost)

	// Make API request - use the completion endpoint instead of generate
	req, err := http.NewRequest("POST", ollamaHost+"/api/generate", bytes.NewBuffer(reqData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// For non-streaming response, we can read the entire body
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Debug: Show the raw API response
	debugPrint(cfg, "OLLAMA RAW RESPONSE", string(respData))

	var response Response
	err = json.Unmarshal(respData, &response)
	if err != nil {
		return "", fmt.Errorf("error parsing Ollama response: %w (response was: %s)", err, string(respData))
	}

	content := strings.TrimSpace(response.Response)

	// For conventional commits, validate the response starts with a valid type
	if cfg.Commit.Convention == config.ConventionalCommits {
		// Fix if the response starts with a colon instead of a type
		if strings.HasPrefix(content, ": ") {
			content = "chore" + content
			debugPrint(cfg, "FIXED RESPONSE FORMAT", content)
		}
	}

	// Return the generated commit message
	return content, nil
}

// generateWithClaude uses Anthropic's Claude to generate a commit message
func generateWithClaude(cfg *config.Config, prompt string) (string, error) {
	// Add a length requirement prefix to the prompt
	lengthPrefix := fmt.Sprintf("CRITICAL INSTRUCTION: Your commit message subject MUST be under %d characters total. ", cfg.Commit.MaxLength)
	if cfg.Commit.Convention == config.ConventionalCommits {
		lengthPrefix += fmt.Sprintf("For conventional commits, this means the ENTIRE string 'type(scope): subject' must be under %d characters.", cfg.Commit.MaxLength)
		lengthPrefix += "\n\nYOU MUST START YOUR RESPONSE WITH A CONVENTIONAL COMMIT TYPE. DO NOT START WITH JUST A COLON."
		lengthPrefix += "\nCORRECT: 'feat: add new feature'"
		lengthPrefix += "\nINCORRECT: ': add new feature'"
		lengthPrefix += "\nValid types are: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert"

		if cfg.Commit.IncludeBody {
			lengthPrefix += "\n\nYOU MUST INCLUDE A COMMIT BODY AFTER THE SUBJECT. The body must be separated from the subject by a blank line."
			lengthPrefix += "\nThe body MUST NOT be empty and should explain what changes were made and why."
		}
	}

	// Prepend the length requirement to the prompt
	enhancedPrompt := lengthPrefix + "\n\n" + prompt

	type Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type Request struct {
		Model     string    `json:"model"`
		Messages  []Message `json:"messages"`
		MaxTokens int       `json:"max_tokens"`
	}

	type Response struct {
		Content struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	// Create request
	reqBody := Request{
		Model: cfg.AI.Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: enhancedPrompt, // Use the enhanced prompt
			},
		},
		MaxTokens: cfg.AI.MaxTokens,
	}

	// Debug: Show the request being sent to Claude
	debugPrint(cfg, "CLAUDE REQUEST", reqBody)

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Make API request
	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", cfg.AI.APIKey)
	req.Header.Set("Anthropic-Version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read response
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Debug: Show the raw API response
	debugPrint(cfg, "CLAUDE RAW RESPONSE", string(respData))

	var response Response
	err = json.Unmarshal(respData, &response)
	if err != nil {
		return "", fmt.Errorf("error parsing Claude response: %w (response: %s)", err, string(respData))
	}

	// Check for API error
	if response.Error.Message != "" {
		return "", fmt.Errorf("Claude API error: %s", response.Error.Message)
	}

	content := strings.TrimSpace(response.Content.Text)

	// For conventional commits, validate the response starts with a valid type
	if cfg.Commit.Convention == config.ConventionalCommits {
		// Fix if the response starts with a colon instead of a type
		if strings.HasPrefix(content, ": ") {
			content = "chore" + content
			debugPrint(cfg, "FIXED RESPONSE FORMAT", content)
		}
	}

	// Return the generated commit message
	return content, nil
}

// generateWithCustomAPI uses a custom web API endpoint to generate a commit message
// This supports various local GPT instances like Open WebUI, local API servers, etc.
func generateWithCustomAPI(cfg *config.Config, prompt string) (string, error) {
	// Validate that API endpoint is provided
	if cfg.AI.APIEndpoint == "" {
		return "", fmt.Errorf("custom API provider requires api_endpoint to be configured")
	}

	// Add a length requirement prefix to the prompt
	lengthPrefix := fmt.Sprintf("CRITICAL INSTRUCTION: Your commit message subject MUST be under %d characters total. ", cfg.Commit.MaxLength)
	if cfg.Commit.Convention == config.ConventionalCommits {
		lengthPrefix += fmt.Sprintf("For conventional commits, this means the ENTIRE string 'type(scope): subject' must be under %d characters.", cfg.Commit.MaxLength)
		lengthPrefix += "\n\nYOU MUST START YOUR RESPONSE WITH A CONVENTIONAL COMMIT TYPE. DO NOT START WITH JUST A COLON."
		lengthPrefix += "\nCORRECT: 'feat: add new feature'"
		lengthPrefix += "\nINCORRECT: ': add new feature'"
		lengthPrefix += "\nValid types are: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert"

		if cfg.Commit.IncludeBody {
			lengthPrefix += "\n\nYOU MUST INCLUDE A COMMIT BODY AFTER THE SUBJECT. The body must be separated from the subject by a blank line."
			lengthPrefix += "\nThe body MUST NOT be empty and should explain what changes were made and why."
		}
	}

	// Prepend the length requirement to the prompt
	enhancedPrompt := lengthPrefix + "\n\n" + prompt

	// Define request structure for OpenAI-compatible API
	type Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type Request struct {
		Model       string    `json:"model"`
		Messages    []Message `json:"messages"`
		Temperature float64   `json:"temperature,omitempty"`
		MaxTokens   int       `json:"max_tokens,omitempty"`
		Stream      bool      `json:"stream,omitempty"`
	}

	type Choice struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}

	type Response struct {
		Choices []Choice `json:"choices"`
		Error   struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	// Create request
	reqBody := Request{
		Model: cfg.AI.Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: enhancedPrompt,
			},
		},
		Temperature: cfg.AI.Temperature,
		MaxTokens:   cfg.AI.MaxTokens,
		Stream:      false,
	}

	// Debug: Show the request being sent to custom API
	debugPrint(cfg, "CUSTOM API REQUEST", reqBody)

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Debug: Show the API endpoint being used
	debugPrint(cfg, "CUSTOM API ENDPOINT", cfg.AI.APIEndpoint)

	// Make API request
	req, err := http.NewRequest("POST", cfg.AI.APIEndpoint, bytes.NewBuffer(reqData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	// Add API key if provided
	if cfg.AI.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.AI.APIKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("custom API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Read response
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Debug: Show the raw API response
	debugPrint(cfg, "CUSTOM API RAW RESPONSE", string(respData))

	var response Response
	err = json.Unmarshal(respData, &response)
	if err != nil {
		return "", fmt.Errorf("error parsing custom API response: %w (response: %s)", err, string(respData))
	}

	// Check for API error
	if response.Error.Message != "" {
		return "", fmt.Errorf("custom API error: %s", response.Error.Message)
	}

	// Check if we have choices
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices from custom API")
	}

	content := strings.TrimSpace(response.Choices[0].Message.Content)

	// For conventional commits, validate the response starts with a valid type
	if cfg.Commit.Convention == config.ConventionalCommits {
		// Fix if the response starts with a colon instead of a type
		if strings.HasPrefix(content, ": ") {
			content = "chore" + content
			debugPrint(cfg, "FIXED RESPONSE FORMAT", content)
		}
	}

	// Return the generated commit message
	return content, nil
}

// Helper function to get system prompt
func getSystemPrompt(cfg *config.Config) string {
	// If custom system prompt is provided, use it
	if cfg.AI.SystemPrompt != "" {
		return cfg.AI.SystemPrompt
	}

	// For conventional commits, use a more specific prompt that matches text prompt style
	if cfg.Commit.Convention == config.ConventionalCommits {
		promptParts := []string{
			"Generate a concise git commit message written in present tense for the following code changes.",
			"YOUR RESPONSE MUST START WITH A CONVENTIONAL COMMIT TYPE FOLLOWED BY A COLON. Valid types are: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert.",
			"INCORRECT: ': description of changes' - This lacks a commit type",
			"CORRECT: 'feat: add new feature' - This has a proper commit type",
			fmt.Sprintf("CRITICAL REQUIREMENT: Commit message subject MUST NOT exceed %d characters total. YOU MUST COUNT THE CHARACTERS YOURSELF AND ENSURE THE TOTAL IS UNDER %d. This is a HARD REQUIREMENT.", cfg.Commit.MaxLength, cfg.Commit.MaxLength),
			fmt.Sprintf("CRITICAL: The TOTAL combined length of 'type(scope): subject' must be strictly under %d characters. Adjust the subject accordingly.", cfg.Commit.MaxLength),
			fmt.Sprintf("If using 'feat(scope): subject' format, the ENTIRE string including 'feat(scope): ' counts toward the %d character limit.", cfg.Commit.MaxLength),
		}

		// Add conventional commit rules
		promptParts = append(promptParts, "You MUST follow these conventional commit rules:")
		promptParts = append(promptParts, ConventionalCommitRules)

		// Add body instructions
		if cfg.Commit.IncludeBody {
			promptParts = append(promptParts, fmt.Sprintf("STRICT REQUIREMENT: Body is REQUIRED and MUST NOT be empty. Body MUST be EXTREMELY BRIEF and MUST NOT exceed %d characters. Use a terse, minimal style focused only on essential technical changes. NEVER include statistics, file lists, or metadata. PRIORITIZE BREVITY ABOVE ALL ELSE.", cfg.Commit.MaxBodyLength))
		} else {
			promptParts = append(promptParts, "Do not include a commit body, only provide the subject line.")
		}

		// Add type descriptions for conventional commits
		promptParts = append(promptParts, `Choose an appropriate type from these options:
- feat: A new feature
- fix: A bug fix
- docs: Documentation only changes
- style: Changes that do not affect the meaning of the code (whitespace, formatting, etc)
- refactor: A code change that neither fixes a bug nor adds a feature
- perf: A code change that improves performance
- test: Adding missing tests or correcting existing tests
- build: Changes that affect the build system or external dependencies
- ci: Changes to CI configuration files and scripts
- chore: Other changes that don't modify source or test files
- revert: Reverts a previous commit`)

		// Add examples of good length subjects
		promptParts = append(promptParts, fmt.Sprintf("Examples of good length subjects that meet the %d character limit:\n- fix: update validation logic (%d chars)\n- feat(auth): add login timeout (%d chars)",
			cfg.Commit.MaxLength,
			len("fix: update validation logic"),
			len("feat(auth): add login timeout")))

		return strings.Join(promptParts, "\n")
	}

	// Otherwise use default system prompt
	return "You are an expert developer who writes clear, concise, and descriptive git commit messages that do not exceed the specified character limits."
}

// debugPrint prints debug information if debug mode is enabled
func debugPrint(cfg *config.Config, message string, data interface{}) {
	if !cfg.AI.Debug {
		return
	}

	// Create a debug marker for visibility
	debugMarker := "\n==== COMMITRON DEBUG ====\n"

	// Format the data based on its type
	var formattedData string
	switch v := data.(type) {
	case string:
		formattedData = v
	case []byte:
		formattedData = string(v)
	default:
		if data != nil {
			jsonData, err := json.MarshalIndent(data, "", "  ")
			if err == nil {
				formattedData = string(jsonData)
			} else {
				formattedData = fmt.Sprintf("%+v", data)
			}
		}
	}

	// Print the debug information
	fmt.Printf("%s%s:\n%s\n%s\n",
		debugMarker,
		message,
		formattedData,
		strings.Repeat("=", len(debugMarker)-1))
}

// GatherEnhancedFileInfo collects detailed information about the changed files
func GatherEnhancedFileInfo(cfg *config.Config, files []string) ([]EnhancedFileInfo, error) {
	var fileInfos []EnhancedFileInfo

	for _, file := range files {
		info := EnhancedFileInfo{
			Path: file,
		}

		// Get file extension for file type
		info.FileType = strings.TrimPrefix(filepath.Ext(file), ".")
		if info.FileType == "" {
			// Try to determine file type from the path or name
			if strings.Contains(file, "Dockerfile") {
				info.FileType = "dockerfile"
			} else if strings.Contains(file, "Makefile") {
				info.FileType = "makefile"
			} else if strings.HasPrefix(filepath.Base(file), ".") {
				// Config files that start with dot
				info.FileType = "config"
			} else {
				info.FileType = "unknown"
			}
		}

		// Get stats about line changes if enabled
		if cfg.Context.IncludeFileStats {
			// Use git diff --numstat to get line changes
			cmd := exec.Command("git", "diff", "--staged", "--numstat", "--", file)
			output, err := cmd.Output()
			if err == nil {
				// Parse the numstat output (format: <added> <removed> <file>)
				parts := strings.Fields(string(output))
				if len(parts) >= 2 {
					// Extract added/removed counts, ignoring binary files (shown as "-")
					if parts[0] != "-" {
						fmt.Sscanf(parts[0], "%d", &info.AddedLines)
					}
					if parts[1] != "-" {
						fmt.Sscanf(parts[1], "%d", &info.RemovedLines)
					}

					// Calculate percentage of file changed
					if info.AddedLines > 0 || info.RemovedLines > 0 {
						// Get total lines in file
						cmd = exec.Command("wc", "-l", file)
						wcOutput, err := cmd.Output()
						if err == nil {
							var totalLines int
							fmt.Sscanf(string(wcOutput), "%d", &totalLines)
							if totalLines > 0 {
								changePercentage := float64(info.AddedLines+info.RemovedLines) / float64(totalLines) * 100
								info.PercentageChange = fmt.Sprintf("%.1f%%", changePercentage)
							}
						}
					}
				}
			}
		}

		// Get file summary if enabled
		if cfg.Context.IncludeFileSummaries {
			// Read the first few lines to generate a summary
			cmd := exec.Command("head", "-n", "10", file)
			output, err := cmd.Output()
			if err == nil {
				lines := strings.Split(string(output), "\n")
				// Try to find a comment that might describe the file
				for _, line := range lines {
					line = strings.TrimSpace(line)
					// Look for comments that might be descriptive
					if (strings.HasPrefix(line, "//") ||
						strings.HasPrefix(line, "#") ||
						strings.HasPrefix(line, "/*") ||
						strings.HasPrefix(line, "<!--")) &&
						len(line) > 5 {
						// Found a likely descriptive comment
						info.Summary = strings.TrimSpace(strings.Trim(strings.Trim(strings.TrimSpace(line), "//"), "#*/<!- "))
						break
					}
				}

				// If we didn't find a descriptive comment, summarize based on file type
				if info.Summary == "" {
					switch info.FileType {
					case "go":
						// Try to extract package and maybe a struct/function name
						for _, line := range lines {
							if strings.HasPrefix(line, "package ") {
								packageName := strings.TrimSpace(strings.TrimPrefix(line, "package "))
								info.Summary = fmt.Sprintf("Go package %s", packageName)
								break
							}
						}
					case "js", "ts", "jsx", "tsx":
						// Look for imports, exports or component definitions
						if strings.Contains(string(output), "import ") && strings.Contains(string(output), "export ") {
							info.Summary = "JavaScript/TypeScript module with imports and exports"
						} else if strings.Contains(string(output), "function ") || strings.Contains(string(output), "class ") {
							info.Summary = "JavaScript/TypeScript file with functions or classes"
						}
					case "md", "markdown":
						// Extract first heading
						for _, line := range lines {
							if strings.HasPrefix(line, "# ") {
								info.Summary = fmt.Sprintf("Documentation: %s", strings.TrimSpace(strings.TrimPrefix(line, "# ")))
								break
							}
						}
						if info.Summary == "" {
							info.Summary = "Documentation file"
						}
					case "yaml", "yml":
						info.Summary = "YAML configuration file"
					case "json":
						info.Summary = "JSON data or configuration file"
					case "sh", "bash":
						info.Summary = "Shell script"
					case "dockerfile":
						info.Summary = "Docker container definition"
					case "makefile":
						info.Summary = "Make build configuration"
					}
				}

				// If still no summary, provide a generic one based on extension
				if info.Summary == "" {
					if info.FileType != "unknown" {
						info.Summary = fmt.Sprintf("%s file", strings.ToUpper(info.FileType))
					} else {
						info.Summary = "File with unknown type"
					}
				}
			}
		}

		// Get first N lines if enabled
		if cfg.Context.ShowFirstLinesOfFile > 0 {
			cmd := exec.Command("head", "-n", fmt.Sprintf("%d", cfg.Context.ShowFirstLinesOfFile), file)
			output, err := cmd.Output()
			if err == nil {
				info.FirstLines = string(output)
			}
		}

		fileInfos = append(fileInfos, info)
	}

	return fileInfos, nil
}

// GetRepoStructure returns a high-level overview of the repository structure
func GetRepoStructure(cfg *config.Config) (string, error) {
	if !cfg.Context.IncludeRepoStructure {
		return "", nil
	}

	// Use find with limited depth to get directory structure
	cmd := exec.Command("find", ".", "-type", "d", "-not", "-path", "*/\\.*", "-maxdepth", "2")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Process the output to create a structured overview
	var result strings.Builder
	result.WriteString("Repository structure:\n")

	dirs := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, dir := range dirs {
		if dir == "." {
			continue // Skip root
		}

		// Count files in directory (using separate commands since pipes aren't directly supported)
		findCmd := exec.Command("find", dir, "-type", "f", "-not", "-path", "*/\\.*", "-maxdepth", "1")
		findOutput, err := findCmd.Output()
		fileCount := "?"
		if err == nil {
			// Count the lines in the output
			lines := strings.Split(strings.TrimSpace(string(findOutput)), "\n")
			if len(lines) == 1 && lines[0] == "" {
				fileCount = "0"
			} else {
				fileCount = fmt.Sprintf("%d", len(lines))
			}
		}

		// Indent based on directory depth
		indentation := strings.Count(dir, "/")
		prefix := strings.Repeat("  ", indentation)
		dirName := filepath.Base(dir)

		result.WriteString(fmt.Sprintf("%s- %s/ (%s files)\n", prefix, dirName, fileCount))
	}

	return result.String(), nil
}

// validateConventionalCommit checks if a commit message follows conventional commit rules
func validateConventionalCommit(msg CommitMessage, cfg *config.Config) error {
	// Check if type is one of the allowed types
	allowedTypes := map[string]bool{
		"feat":     true,
		"fix":      true,
		"docs":     true,
		"style":    true,
		"refactor": true,
		"perf":     true,
		"test":     true,
		"build":    true,
		"ci":       true,
		"chore":    true,
		"revert":   true,
	}

	// Type is required and must be one of the allowed types
	if msg.Type == "" {
		return fmt.Errorf("commit type is required for conventional commits")
	}

	// Validate type is lowercase
	if msg.Type != strings.ToLower(msg.Type) {
		return fmt.Errorf("commit type must be lowercase: %s", msg.Type)
	}

	// Check if type is allowed
	if !allowedTypes[msg.Type] {
		return fmt.Errorf("commit type '%s' is not allowed for conventional commits; must be one of: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert", msg.Type)
	}

	// Subject is required
	if msg.Subject == "" {
		return fmt.Errorf("commit subject is required for conventional commits")
	}

	// Subject should not end with a period
	if strings.HasSuffix(msg.Subject, ".") {
		return fmt.Errorf("commit subject should not end with a period")
	}

	// Subject first letter should not be capitalized (conventional)
	if len(msg.Subject) > 0 && unicode.IsUpper([]rune(msg.Subject)[0]) {
		return fmt.Errorf("commit subject should not start with a capital letter")
	}

	// Subject should not contain newlines
	if strings.Contains(msg.Subject, "\n") {
		return fmt.Errorf("commit subject should not contain newlines")
	}

	// Subject should not be too generic
	genericSubjects := map[string]bool{
		"update": true,
		"fix":    true,
		"change": true,
		"modify": true,
		"add":    true,
		"remove": true,
		"delete": true,
	}

	if genericSubjects[strings.ToLower(msg.Subject)] {
		return fmt.Errorf("commit subject is too generic, please be more specific about what was changed")
	}

	// Body is required if configured
	if cfg.Commit.IncludeBody {
		trimmedBody := strings.TrimSpace(msg.Body)
		if trimmedBody == "" {
			return fmt.Errorf("commit body is required for conventional commits when include_body is true")
		}

		// Check if body is just placeholder text
		if strings.Contains(strings.ToLower(trimmedBody), "<descriptive body") ||
			strings.Contains(strings.ToLower(trimmedBody), "<optional body>") ||
			strings.Contains(strings.ToLower(trimmedBody), "explanat") ||
			strings.Contains(strings.ToLower(trimmedBody), "<commit message>") {
			return fmt.Errorf("commit body contains placeholder text and needs to be replaced with actual content")
		}

		// Ensure body has reasonable length
		if len(trimmedBody) < 10 {
			return fmt.Errorf("commit body is too short (must be at least 10 characters)")
		}

		// Ensure body is separated from subject by a blank line
		if !strings.Contains(msg.Body, "\n\n") {
			return fmt.Errorf("commit body must be separated from subject by a blank line")
		}

		// Check for common issues in body
		if strings.Contains(strings.ToLower(trimmedBody), "this code") ||
			strings.Contains(strings.ToLower(trimmedBody), "the changes") ||
			strings.Contains(strings.ToLower(trimmedBody), "this commit") {
			return fmt.Errorf("commit body should not start with phrases like 'this code', 'the changes', or 'this commit'")
		}

		// Ensure body is not just a list of files
		if strings.Contains(trimmedBody, "file:") || strings.Contains(trimmedBody, "files:") {
			return fmt.Errorf("commit body should not be a list of files, focus on what changed and why")
		}
	}

	// Validate scope format if present
	if msg.Scope != "" {
		// Scope should be lowercase
		if msg.Scope != strings.ToLower(msg.Scope) {
			return fmt.Errorf("commit scope must be lowercase: %s", msg.Scope)
		}

		// Scope should not contain spaces
		if strings.Contains(msg.Scope, " ") {
			return fmt.Errorf("commit scope should not contain spaces")
		}

		// Scope should not contain special characters
		if strings.ContainsAny(msg.Scope, "!@#$%^&*()_+={}[]|\\:;\"'<>,.?/~`") {
			return fmt.Errorf("commit scope should not contain special characters")
		}

		// Scope should not be too generic
		if genericSubjects[strings.ToLower(msg.Scope)] {
			return fmt.Errorf("commit scope is too generic, please be more specific")
		}
	}

	return nil
}

// fixConventionalCommitIssues attempts to fix common issues in conventional commits
func fixConventionalCommitIssues(msg CommitMessage) CommitMessage {
	// Fix type case
	msg.Type = strings.ToLower(msg.Type)

	// Fix common type misspellings
	typeCorrections := map[string]string{
		"feature":       "feat",
		"bugfix":        "fix",
		"document":      "docs",
		"documentation": "docs",
		"styling":       "style",
		"refactoring":   "refactor",
		"performance":   "perf",
		"testing":       "test",
		"tests":         "test",
		"building":      "build",
		"maintenance":   "chore",
	}

	if correctedType, ok := typeCorrections[msg.Type]; ok {
		msg.Type = correctedType
	}

	// Remove trailing period from subject
	if strings.HasSuffix(msg.Subject, ".") {
		msg.Subject = msg.Subject[:len(msg.Subject)-1]
	}

	// Convert first letter of subject to lowercase
	if len(msg.Subject) > 0 && unicode.IsUpper([]rune(msg.Subject)[0]) {
		r := []rune(msg.Subject)
		r[0] = unicode.ToLower(r[0])
		msg.Subject = string(r)
	}

	// Fix generic subjects
	genericSubjects := map[string]string{
		"update": "improve",
		"change": "modify",
		"modify": "enhance",
		"add":    "implement",
		"remove": "delete",
		"delete": "remove",
		"fix":    "resolve",
	}

	if replacement, ok := genericSubjects[strings.ToLower(msg.Subject)]; ok {
		msg.Subject = replacement
	}

	// Clean up body if present
	if msg.Body != "" {
		// Remove common problematic phrases from start of body
		bodyLines := strings.Split(msg.Body, "\n")
		if len(bodyLines) > 0 {
			firstLine := strings.ToLower(bodyLines[0])
			removePhrases := []string{
				"this code",
				"the changes",
				"this commit",
				"the code",
				"the file",
				"the files",
				"the changes made",
				"the changes include",
				"the changes made to",
			}

			for _, phrase := range removePhrases {
				if strings.HasPrefix(firstLine, phrase) {
					bodyLines[0] = strings.TrimSpace(strings.TrimPrefix(bodyLines[0], phrase))
					break
				}
			}
		}

		// Remove file lists
		var cleanedLines []string
		for _, line := range bodyLines {
			if !strings.Contains(strings.ToLower(line), "file:") &&
				!strings.Contains(strings.ToLower(line), "files:") &&
				!strings.Contains(strings.ToLower(line), "changed files:") {
				cleanedLines = append(cleanedLines, line)
			}
		}

		msg.Body = strings.Join(cleanedLines, "\n")
		msg.Body = strings.TrimSpace(msg.Body)

		// Ensure proper separation from subject
		if !strings.Contains(msg.Body, "\n\n") {
			msg.Body = "\n\n" + msg.Body
		}
	}

	// Fix scope if present
	if msg.Scope != "" {
		msg.Scope = strings.ToLower(msg.Scope)

		// Fix generic scopes
		if replacement, ok := genericSubjects[msg.Scope]; ok {
			msg.Scope = replacement
		}
	}

	return msg
}

// isValidCommitType checks if a string is a valid conventional commit type
func isValidCommitType(t string) bool {
	validTypes := map[string]bool{
		"feat":     true,
		"fix":      true,
		"docs":     true,
		"style":    true,
		"refactor": true,
		"perf":     true,
		"test":     true,
		"build":    true,
		"ci":       true,
		"chore":    true,
		"revert":   true,
	}
	return validTypes[t]
}
