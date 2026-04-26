package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	defaultGroqModel = "llama-3.3-70b-versatile"
	groqAPIURL       = "https://api.groq.com/openai/v1/chat/completions"
)

type FileChange struct {
	FileName string
	Status   string
	Diff     string
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Temperature float64       `json:"temperature"`
	Messages    []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type CommitPlan struct {
	Label   string
	Changes []FileChange
}

type commitGroupProposal struct {
	Label string   `json:"label"`
	Files []string `json:"files"`
}

type commitRunSummary struct {
	Mode      string
	Committed []string
	Skipped   []string
	Failed    []string
}

func printWelcome() {
	fmt.Println("Welcome to Git Pilot! Your AI-powered Git assistant.")
	fmt.Println("Commands:")
	fmt.Println("init, status, diff, commit, push, pull, config, help")
}

func readCommand() []string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("gitpilot> ")

	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		fmt.Println("❌ Failed to read command:", err)
		return nil
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	return strings.Fields(line)
}

func runInteractive() {
	for {
		args := readCommand()
		if len(args) == 0 {
			continue
		}

		if args[0] == "exit" {
			fmt.Println("Goodbye!")
			break
		}

		executeCommand(args)
	}
}

func executeCommand(args []string) {
	switch args[0] {
	case "init":
		executeInit()
	case "status":
		executeStatus()
	case "diff":
		executeDiff()
	case "commit":
		executeCommit(args[1:])
	case "push":
		if err := executePush(); err != nil {
			fmt.Println("❌ Push failed:", err)
		}
	case "pull":
		executePull()
	case "config":
		executeConfig(args[1:])
	case "help":
		printHelp()
	default:
		fmt.Println("❌ Invalid command. Type 'help'")
	}
}

func runGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}

	return string(output), nil
}

func getChangedFiles() ([]FileChange, error) {
	output, err := runGitCommand("status", "--porcelain")
	if err != nil {
		return nil, err
	}

	// Preserve leading spaces because porcelain status uses them as data,
	// e.g. " M main.go" for unstaged modifications.
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}

	var changes []FileChange
	seen := make(map[string]struct{})

	for _, line := range lines {
		if len(line) < 4 {
			continue
		}

		status := strings.TrimSpace(line[:2])
		path := strings.TrimSpace(line[3:])
		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			path = strings.TrimSpace(parts[len(parts)-1])
		}

		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}

		diff, diffErr := getDiffForFile(path, status)
		if diffErr != nil {
			fmt.Println("⚠️ Skipping file:", path)
			continue
		}

		changes = append(changes, FileChange{
			FileName: path,
			Status:   status,
			Diff:     diff,
		})
	}

	return changes, nil
}

func getDiffForFile(fileName, status string) (string, error) {
	if strings.Contains(status, "?") {
		return runDiffAllowExitCodeOne("diff", "--no-index", "--", "/dev/null", fileName)
	}

	var sections []string

	if stagedDiff, err := runDiffAllowExitCodeOne("diff", "--cached", "--", fileName); err == nil && strings.TrimSpace(stagedDiff) != "" {
		sections = append(sections, stagedDiff)
	}

	if workingDiff, err := runDiffAllowExitCodeOne("diff", "--", fileName); err == nil && strings.TrimSpace(workingDiff) != "" {
		sections = append(sections, workingDiff)
	}

	if len(sections) == 0 {
		return "", errors.New("no diff available")
	}

	return strings.Join(sections, "\n"), nil
}

func runDiffAllowExitCodeOne(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 && len(output) > 0 {
			return string(output), nil
		}
		return "", err
	}

	return string(output), nil
}

func filterChanges(changes []FileChange, targets []string) ([]FileChange, error) {
	if len(targets) == 0 {
		return changes, nil
	}

	lookup := make(map[string]FileChange, len(changes))
	for _, change := range changes {
		lookup[change.FileName] = change
	}

	var filtered []FileChange
	for _, target := range targets {
		change, ok := lookup[target]
		if !ok {
			return nil, fmt.Errorf("file %q has no detected changes", target)
		}
		filtered = append(filtered, change)
	}

	return filtered, nil
}

func executeInit() {
	fmt.Println("Initializing repository...")
}

func executeStatus() {
	fmt.Println("Checking repository status...")

	output, err := runGitCommand("status")
	if err != nil {
		fmt.Println("⚠️ Git error:")
	}

	fmt.Println(output)
}

func executeDiff() {
	changes, err := getChangedFiles()
	if err != nil {
		fmt.Println("❌ Error:", err)
		return
	}

	if len(changes) == 0 {
		fmt.Println("No changes detected.")
		return
	}

	fmt.Println("Changed files:")
	for _, change := range changes {
		fmt.Printf("- %s [%s]\n", change.FileName, change.Status)
	}
}

func executeCommit(args []string) {
	changes, err := getChangedFiles()
	if err != nil {
		fmt.Println("❌ Error:", err)
		return
	}

	if len(changes) == 0 {
		fmt.Println("No changes detected.")
		return
	}

	apiKey, err := getGroqAPIKey()
	if err != nil {
		fmt.Println("❌", err)
		fmt.Println("Set GROQ_API_KEY or run: config groq-key <your-key>")
		return
	}

	model := getGroqModel()
	mode := ""
	targets := []string(nil)

	if len(args) > 0 {
		mode = args[0]
		targets = args[1:]
	} else {
		mode = promptCommitMode()
		if mode == "" {
			fmt.Println("Commit cancelled.")
			return
		}
	}

	plans, err := buildCommitPlans(apiKey, model, mode, changes, targets)
	if err != nil {
		fmt.Println("❌", err)
		return
	}

	summary := commitRunSummary{Mode: mode}

	for _, plan := range plans {
		fmt.Printf("\nGenerating commit message for %s...\n", plan.Label)

		message, err := generateCommitMessage(apiKey, model, plan.Changes)
		if err != nil {
			fmt.Println("❌ AI generation failed:", err)
			summary.Failed = append(summary.Failed, plan.Label)
			continue
		}

		printCommitPreview(plan, message)
		if !approveCommit(plan, message) {
			fmt.Println("Skipped.")
			summary.Skipped = append(summary.Skipped, plan.Label)
			continue
		}

		if err := performCommit(mode, plan.Changes, message); err != nil {
			fmt.Println("❌ Commit failed:", err)
			summary.Failed = append(summary.Failed, plan.Label)
			continue
		}

		fmt.Println("Committed.")
		summary.Committed = append(summary.Committed, fmt.Sprintf("%s -> %s", plan.Label, message))
	}

	printCommitSummary(summary)
	if len(summary.Committed) > 0 {
		promptPushAfterCommits()
	}
}

func promptCommitMode() string {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\nHow do you want to commit these changes?")
		fmt.Println("1. File wise")
		fmt.Println("2. Group wise (AI categories)")
		fmt.Print("Choose 1 or 2: ")

		answer, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			fmt.Println("❌ Failed to read choice:", err)
			return ""
		}

		switch strings.TrimSpace(answer) {
		case "1":
			return "file"
		case "2":
			return "group"
		case "":
			if errors.Is(err, io.EOF) {
				return ""
			}
		default:
			fmt.Println("Please choose 1 or 2.")
		}
	}
}

func buildCommitPlans(apiKey, model, mode string, changes []FileChange, targets []string) ([]CommitPlan, error) {
	switch mode {
	case "group":
		if len(targets) > 0 {
			groupChanges, err := filterChanges(changes, targets)
			if err != nil {
				return nil, err
			}
			return []CommitPlan{{
				Label:   "group: " + formatFileList(groupChanges),
				Changes: groupChanges,
			}}, nil
		}

		return buildGroupedCommitPlans(apiKey, model, changes)
	case "categories":
		return buildGroupedCommitPlans(apiKey, model, changes)
	case "all":
		groupChanges, err := filterChanges(changes, targets)
		if err != nil {
			return nil, err
		}
		return []CommitPlan{{
			Label:   "group: " + formatFileList(groupChanges),
			Changes: groupChanges,
		}}, nil
	case "file":
		fileChanges := changes
		var err error
		if len(targets) > 0 {
			fileChanges, err = filterChanges(changes, targets)
		}
		if err != nil {
			return nil, err
		}
		plans := make([]CommitPlan, 0, len(fileChanges))
		for _, change := range fileChanges {
			plans = append(plans, CommitPlan{
				Label:   "file: " + change.FileName,
				Changes: []FileChange{change},
			})
		}
		return plans, nil
	default:
		return nil, errors.New("unknown commit mode. Use: commit, commit file <file...>, or commit group <file...>")
	}
}

func printCommitPreview(plan CommitPlan, message string) {
	fmt.Printf("Target: %s\n", plan.Label)
	fmt.Printf("Files: %s\n", formatFileList(plan.Changes))
	fmt.Printf("Message: %s\n", message)
}

func approveCommit(plan CommitPlan, message string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Approve commit for %s? [y/N]: ", plan.Label)

	answer, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		fmt.Println("❌ Failed to read approval:", err)
		return false
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

func promptYesNo(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)

	answer, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		fmt.Println("❌ Failed to read approval:", err)
		return false
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

func performCommit(mode string, changes []FileChange, message string) error {
	if len(changes) == 0 {
		return errors.New("no changes to commit")
	}

	if len(changes) > 1 && mode == "all" {
		if _, err := runGitCommand("add", "-A"); err != nil {
			return err
		}
		_, err := runGitCommand("commit", "-m", message)
		return err
	}

	paths := make([]string, 0, len(changes))
	for _, change := range changes {
		paths = append(paths, change.FileName)
	}

	addArgs := append([]string{"add", "-A", "--"}, paths...)
	if _, err := runGitCommand(addArgs...); err != nil {
		return err
	}

	commitArgs := append([]string{"commit", "--only", "-m", message, "--"}, paths...)
	_, err := runGitCommand(commitArgs...)
	return err
}

func buildGroupedCommitPlans(apiKey, model string, changes []FileChange) ([]CommitPlan, error) {
	content, err := generateCommitGroups(apiKey, model, changes)
	if err != nil {
		return nil, err
	}

	proposals, err := parseCommitGroups(content)
	if err != nil {
		return nil, err
	}

	lookup := make(map[string]FileChange, len(changes))
	for _, change := range changes {
		lookup[change.FileName] = change
	}

	used := make(map[string]struct{})
	var plans []CommitPlan

	for _, proposal := range proposals {
		if len(proposal.Files) == 0 {
			continue
		}

		var group []FileChange
		for _, file := range proposal.Files {
			change, ok := lookup[file]
			if !ok {
				continue
			}
			if _, seen := used[file]; seen {
				continue
			}

			used[file] = struct{}{}
			group = append(group, change)
		}

		if len(group) == 0 {
			continue
		}

		label := proposal.Label
		if strings.TrimSpace(label) == "" {
			label = formatFileList(group)
		}

		plans = append(plans, CommitPlan{
			Label:   label,
			Changes: group,
		})
	}

	for _, change := range changes {
		if _, seen := used[change.FileName]; seen {
			continue
		}
		plans = append(plans, CommitPlan{
			Label:   "remaining: " + change.FileName,
			Changes: []FileChange{change},
		})
	}

	if len(plans) == 0 {
		return nil, errors.New("AI did not produce any valid commit groups")
	}

	return plans, nil
}

func generateCommitGroups(apiKey, model string, changes []FileChange) (string, error) {
	payload := chatCompletionRequest{
		Model:       model,
		Temperature: 0.1,
		Messages: []chatMessage{
			{
				Role: "system",
				Content: "You group changed files into logical git commits. Return JSON only. " +
					`Use this schema: [{"label":"short category label","files":["path1","path2"]}]. ` +
					"Each file must appear at most once.",
			},
			{
				Role:    "user",
				Content: buildGroupingPrompt(changes),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, groqAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var completion chatCompletionResponse
	if err := json.Unmarshal(responseBody, &completion); err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		if completion.Error != nil && completion.Error.Message != "" {
			return "", errors.New(completion.Error.Message)
		}
		return "", fmt.Errorf("groq request failed with status %s", resp.Status)
	}

	if len(completion.Choices) == 0 {
		return "", errors.New("groq returned no choices")
	}

	return strings.TrimSpace(completion.Choices[0].Message.Content), nil
}

func buildGroupingPrompt(changes []FileChange) string {
	var builder strings.Builder
	builder.WriteString("Group these changed files into logical commits.\n")
	builder.WriteString("Prefer a small number of clear categories. Keep unrelated files separate.\n\n")

	for _, change := range changes {
		builder.WriteString("File: ")
		builder.WriteString(change.FileName)
		builder.WriteString("\nStatus: ")
		builder.WriteString(change.Status)
		builder.WriteString("\nDiff:\n")
		builder.WriteString(change.Diff)
		builder.WriteString("\n\n")
	}

	return builder.String()
}

func parseCommitGroups(content string) ([]commitGroupProposal, error) {
	cleaned := strings.TrimSpace(content)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var proposals []commitGroupProposal
	if err := json.Unmarshal([]byte(cleaned), &proposals); err != nil {
		return nil, fmt.Errorf("failed to parse AI commit groups: %w", err)
	}

	return proposals, nil
}

func printCommitSummary(summary commitRunSummary) {
	fmt.Println("\nCommit session summary")
	fmt.Println("Mode:", summary.Mode)
	fmt.Printf("Committed: %d\n", len(summary.Committed))
	for _, item := range summary.Committed {
		fmt.Println("- " + item)
	}
	fmt.Printf("Skipped: %d\n", len(summary.Skipped))
	for _, item := range summary.Skipped {
		fmt.Println("- " + item)
	}
	fmt.Printf("Failed: %d\n", len(summary.Failed))
	for _, item := range summary.Failed {
		fmt.Println("- " + item)
	}
}

func promptPushAfterCommits() {
	fmt.Println("\nCommit session finished.")
	if !promptYesNo("Push committed changes now? [y/N]: ") {
		fmt.Println("Push skipped.")
		return
	}

	if err := executePush(); err != nil {
		fmt.Println("❌ Push failed:", err)
	}
}

func formatFileList(changes []FileChange) string {
	names := make([]string, 0, len(changes))
	for _, change := range changes {
		names = append(names, change.FileName)
	}
	return strings.Join(names, ", ")
}

func generateCommitMessage(apiKey, model string, changes []FileChange) (string, error) {
	payload := chatCompletionRequest{
		Model:       model,
		Temperature: 0.2,
		Messages: []chatMessage{
			{
				Role:    "system",
				Content: "You write concise git commit messages. Reply with exactly one imperative commit subject line using a conventional prefix such as feat:, fix:, refactor:, docs:, test:, chore:, style:, perf:, build:, or ci:. Choose the most accurate tag from the diff. No quotes, no bullet points, max 72 characters.",
			},
			{
				Role:    "user",
				Content: buildCommitPrompt(changes),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, groqAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var completion chatCompletionResponse
	if err := json.Unmarshal(responseBody, &completion); err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		if completion.Error != nil && completion.Error.Message != "" {
			return "", errors.New(completion.Error.Message)
		}
		return "", fmt.Errorf("groq request failed with status %s", resp.Status)
	}

	if len(completion.Choices) == 0 {
		return "", errors.New("groq returned no choices")
	}

	return strings.TrimSpace(completion.Choices[0].Message.Content), nil
}

func buildCommitPrompt(changes []FileChange) string {
	var builder strings.Builder
	builder.WriteString("Generate a git commit message for these changes.\n")
	builder.WriteString("Summarize the main behavior change, not the implementation trivia.\n\n")

	for _, change := range changes {
		builder.WriteString("File: ")
		builder.WriteString(change.FileName)
		builder.WriteString("\nStatus: ")
		builder.WriteString(change.Status)
		builder.WriteString("\nDiff:\n")
		builder.WriteString(change.Diff)
		builder.WriteString("\n\n")
	}

	return builder.String()
}

func getGroqAPIKey() (string, error) {
	if key := strings.TrimSpace(os.Getenv("GROQ_API_KEY")); key != "" {
		return key, nil
	}

	output, err := runGitCommand("config", "--get", "gitpilot.groqApiKey")
	if err == nil {
		if key := strings.TrimSpace(output); key != "" {
			return key, nil
		}
	}

	return "", errors.New("Groq API key is not configured")
}

func getGroqModel() string {
	if model := strings.TrimSpace(os.Getenv("GROQ_MODEL")); model != "" {
		return model
	}

	output, err := runGitCommand("config", "--get", "gitpilot.groqModel")
	if err == nil {
		if model := strings.TrimSpace(output); model != "" {
			return model
		}
	}

	return defaultGroqModel
}

func executeConfig(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage:")
		fmt.Println("config groq-key <api-key>")
		fmt.Println("config groq-model <model>")
		fmt.Println("config show")
		return
	}

	switch args[0] {
	case "groq-key":
		if len(args) < 2 {
			fmt.Println("Usage: config groq-key <api-key>")
			return
		}
		if err := setGitConfig("gitpilot.groqApiKey", args[1]); err != nil {
			fmt.Println("❌ Failed to save Groq API key:", err)
			return
		}
		fmt.Println("Saved Groq API key to local git config.")
	case "groq-model":
		if len(args) < 2 {
			fmt.Println("Usage: config groq-model <model>")
			return
		}
		if err := setGitConfig("gitpilot.groqModel", args[1]); err != nil {
			fmt.Println("❌ Failed to save Groq model:", err)
			return
		}
		fmt.Println("Saved Groq model to local git config.")
	case "show":
		printConfig()
	default:
		fmt.Println("❌ Unknown config option.")
	}
}

func setGitConfig(key, value string) error {
	_, err := runGitCommand("config", "--local", key, value)
	return err
}

func printConfig() {
	keySource := "missing"
	if strings.TrimSpace(os.Getenv("GROQ_API_KEY")) != "" {
		keySource = "environment"
	} else if key, err := runGitCommand("config", "--get", "gitpilot.groqApiKey"); err == nil && strings.TrimSpace(key) != "" {
		keySource = "git config"
	}

	fmt.Println("Groq key source:", keySource)
	fmt.Println("Groq model:", getGroqModel())
}

func executePush() error {
	fmt.Println("Push preview:")

	statusOutput, statusErr := runGitCommand("status", "--short", "--branch")
	if statusErr == nil && strings.TrimSpace(statusOutput) != "" {
		fmt.Println(statusOutput)
	}

	if !promptYesNo("Approve push? [y/N]: ") {
		fmt.Println("Push cancelled.")
		return nil
	}

	fmt.Println("Pushing to remote...")
	output, err := runGitCommand("push")
	if strings.TrimSpace(output) != "" {
		fmt.Println(output)
	}
	if err != nil {
		return err
	}

	fmt.Println("Push completed.")
	return nil
}

func executePull() {
	fmt.Println("Pulling latest changes...")
}

func printHelp() {
	printWelcome()
	fmt.Println("Examples:")
	fmt.Println("commit                  # asks file wise vs AI group wise")
	fmt.Println("commit file main.go     # direct file-wise mode")
	fmt.Println("commit group            # direct AI group-wise mode")
	fmt.Println("commit group main.go go.mod")
	fmt.Println("config groq-key <api-key>")
	fmt.Printf("config groq-model %s\n", defaultGroqModel)
}

func main() {
	if len(os.Args) > 1 {
		executeCommand(os.Args[1:])
	} else {
		printWelcome()
		runInteractive()
	}
}
