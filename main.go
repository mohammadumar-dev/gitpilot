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
	"strconv"
	"strings"
	"time"
)

const (
	defaultGroqModel = "llama-3.3-70b-versatile"
	groqAPIURL       = "https://api.groq.com/openai/v1/chat/completions"
	groqAPIKeyConfig = "gitpilot.groq-api-key"
	groqModelConfig  = "gitpilot.groq-model"
	initConfigKey    = "gitpilot.initialized"
	colorReset       = "\033[0m"
	colorDim         = "\033[38;5;245m"
	colorBorder      = "\033[38;5;240m"
	colorAccent      = "\033[38;5;111m"
	colorInfo        = "\033[38;5;117m"
	colorSuccess     = "\033[38;5;114m"
	colorWarn        = "\033[38;5;221m"
	colorError       = "\033[38;5;203m"
	colorStrong      = "\033[1m"
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
	printPanel([]string{
		colorStrong + "Git Pilot" + colorReset,
		colorDim + "AI-assisted Git workflow for structured commits and push approvals" + colorReset,
		"",
		colorDim + "Commands: init, status, diff, commit, push, pull, config, help, exit" + colorReset,
	})
}

func printSection(title string) {
	fmt.Printf("\n%s──%s %s%s%s\n", colorBorder, colorReset, colorStrong, title, colorReset)
}

func printSuccess(message string) {
	fmt.Printf("%s●%s %s%s%s\n", colorSuccess, colorReset, colorSuccess, message, colorReset)
}

func printWarning(message string) {
	fmt.Printf("%s●%s %s%s%s\n", colorWarn, colorReset, colorWarn, message, colorReset)
}

func printError(message string) {
	fmt.Printf("%s●%s %s%s%s\n", colorError, colorReset, colorError, message, colorReset)
}

func printInfo(message string) {
	fmt.Printf("%s●%s %s%s%s\n", colorInfo, colorReset, colorInfo, message, colorReset)
}

func printPanel(lines []string) {
	expanded := make([]string, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, "\n")
		expanded = append(expanded, parts...)
	}

	width := 0
	for _, line := range expanded {
		if len(stripANSI(line)) > width {
			width = len(stripANSI(line))
		}
	}
	if width < 24 {
		width = 24
	}

	fmt.Printf("%s╭%s╮%s\n", colorBorder, strings.Repeat("─", width+2), colorReset)
	for _, line := range expanded {
		padding := width - len(stripANSI(line))
		fmt.Printf("%s│%s %s%s %s│%s\n", colorBorder, colorReset, line, strings.Repeat(" ", padding), colorBorder, colorReset)
	}
	fmt.Printf("%s╰%s╯%s\n", colorBorder, strings.Repeat("─", width+2), colorReset)
}

func stripANSI(text string) string {
	var builder strings.Builder
	inEscape := false

	for _, r := range text {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		builder.WriteRune(r)
	}

	return builder.String()
}

func styleStatus(status string) string {
	switch {
	case strings.Contains(status, "?"):
		return colorInfo + "new     " + colorReset
	case strings.Contains(status, "A"):
		return colorSuccess + "added   " + colorReset
	case strings.Contains(status, "M"):
		return colorWarn + "modified" + colorReset
	case strings.Contains(status, "D"):
		return colorError + "deleted " + colorReset
	case strings.Contains(status, "R"):
		return colorAccent + "renamed " + colorReset
	default:
		return colorDim + status + colorReset
	}
}

func renderChoice(index int, title, description string) string {
	return fmt.Sprintf("%s[%d]%s %s%s%s\n    %s%s%s", colorAccent, index, colorReset, colorStrong, title, colorReset, colorDim, description, colorReset)
}

func readCommand() []string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s%sgitpilot%s %s›%s ", colorAccent, colorStrong, colorReset, colorBorder, colorReset)

	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		printError("Failed to read command: " + err.Error())
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
		printError("Invalid command. Type 'help'")
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
			printWarning("Skipping file: " + path)
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
	printSection("Init")

	if _, err := runGitCommand("rev-parse", "--is-inside-work-tree"); err != nil {
		printError("Current directory is not a Git repository.")
		return
	}

	branch, _ := runGitCommand("branch", "--show-current")
	remoteName := getDefaultRemoteName()
	model := strings.TrimSpace(getGroqModel())

	if _, err := runGitCommand("config", "--get", groqModelConfig); err != nil {
		if err := setGitConfig(groqModelConfig, defaultGroqModel); err != nil {
			printError("Failed to save default Groq model: " + err.Error())
			return
		}
		model = defaultGroqModel
	}

	if _, err := runGitCommand("config", "--get", initConfigKey); err != nil {
		if err := setGitConfig(initConfigKey, "true"); err != nil {
			printError("Failed to save Git Pilot initialization state: " + err.Error())
			return
		}
	}

	printPanel([]string{
		colorStrong + "Repository ready for Git Pilot" + colorReset,
		colorDim + "Branch" + colorReset + "  " + strings.TrimSpace(branch),
		colorDim + "Remote" + colorReset + "  " + remoteName,
		colorDim + "Model" + colorReset + "   " + model,
	})

	if _, err := getGroqAPIKey(); err != nil {
		printWarning("Groq API key is not configured yet.")
		fmt.Println("Set GROQ_API_KEY or run: config groq-key <your-key>")
		return
	}

	printSuccess("Groq API key detected.")
}

func executeStatus() {
	printSection("Repository Status")

	output, err := runGitCommand("status")
	if err != nil {
		printWarning("Git returned an error.")
	}

	fmt.Println(output)
}

func executeDiff() {
	printSection("Changed Files")

	changes, err := getChangedFiles()
	if err != nil {
		printError(err.Error())
		return
	}

	if len(changes) == 0 {
		fmt.Println("No changes detected.")
		return
	}

	lines := []string{
		fmt.Sprintf("%s%d file(s) changed%s", colorStrong, len(changes), colorReset),
	}
	for index, change := range changes {
		lines = append(lines, fmt.Sprintf("%s%2d%s  %s  %s", colorDim, index+1, colorReset, styleStatus(change.Status), change.FileName))
	}
	printPanel(lines)
}

func executeCommit(args []string) {
	printSection("Commit Session")

	changes, err := getChangedFiles()
	if err != nil {
		printError(err.Error())
		return
	}

	if len(changes) == 0 {
		fmt.Println("No changes detected.")
		return
	}

	apiKey, err := getGroqAPIKey()
	if err != nil {
		printError(err.Error())
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
			printWarning("Commit cancelled.")
			return
		}
	}

	plans, err := buildCommitPlans(apiKey, model, mode, changes, targets)
	if err != nil {
		printError(err.Error())
		return
	}

	summary := commitRunSummary{Mode: mode}
	printInfo("Planned commits: " + strconv.Itoa(len(plans)))

	for index, plan := range plans {
		printSection(fmt.Sprintf("Commit %d of %d", index+1, len(plans)))
		printPanel([]string{
			colorStrong + plan.Label + colorReset,
			colorDim + strconv.Itoa(len(plan.Changes)) + " file(s) in this commit" + colorReset,
			"",
			formatFilesForDisplay(plan.Changes),
		})
		printInfo("Generating AI commit message...")

		message, err := generateCommitMessage(apiKey, model, plan.Changes)
		if err != nil {
			printError("AI generation failed: " + err.Error())
			summary.Failed = append(summary.Failed, plan.Label)
			continue
		}

		printCommitPreview(plan, message)
		if !approveCommit(plan, message) {
			printWarning("Commit skipped.")
			summary.Skipped = append(summary.Skipped, plan.Label)
			continue
		}

		if err := performCommit(mode, plan.Changes, message); err != nil {
			printError("Commit failed: " + err.Error())
			summary.Failed = append(summary.Failed, plan.Label)
			continue
		}

		printSuccess("Commit created.")
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
		printSection("Commit Mode")
		fmt.Println("How do you want to split these commits?")
		fmt.Println(renderChoice(1, "File wise", "One commit per changed file."))
		fmt.Println(renderChoice(2, "Group wise", "AI groups related files into logical commit categories."))
		fmt.Printf("%sSelect mode%s %s(1/2)%s: ", colorStrong, colorReset, colorDim, colorReset)

		answer, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			printError("Failed to read choice: " + err.Error())
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
			printWarning("Please choose 1 or 2.")
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
	printPanel([]string{
		colorStrong + "Commit Preview" + colorReset,
		colorDim + "Target" + colorReset + "  " + plan.Label,
		colorDim + "Files" + colorReset + "   " + formatFileList(plan.Changes),
		colorDim + "Message" + colorReset + " " + colorAccent + message + colorReset,
	})
}

func approveCommit(plan CommitPlan, message string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Approve commit for %s? [y/N]: ", plan.Label)

	answer, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		printError("Failed to read approval: " + err.Error())
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
		printError("Failed to read approval: " + err.Error())
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
	printSection("Commit Session Summary")
	lines := []string{
		colorDim + "Mode" + colorReset + "      " + summary.Mode,
		colorSuccess + "Committed" + colorReset + " " + strconv.Itoa(len(summary.Committed)),
		colorWarn + "Skipped" + colorReset + "   " + strconv.Itoa(len(summary.Skipped)),
		colorError + "Failed" + colorReset + "    " + strconv.Itoa(len(summary.Failed)),
	}
	if len(summary.Committed) > 0 {
		lines = append(lines, "", colorStrong+"Created"+colorReset)
		for _, item := range summary.Committed {
			lines = append(lines, "  + "+item)
		}
	}
	if len(summary.Skipped) > 0 {
		lines = append(lines, "", colorStrong+"Skipped"+colorReset)
		for _, item := range summary.Skipped {
			lines = append(lines, "  - "+item)
		}
	}
	if len(summary.Failed) > 0 {
		lines = append(lines, "", colorStrong+"Failed"+colorReset)
		for _, item := range summary.Failed {
			lines = append(lines, "  x "+item)
		}
	}
	printPanel(lines)
}

func promptPushAfterCommits() {
	printSection("Push")
	if !promptYesNo("Push committed changes now? [y/N]: ") {
		printWarning("Push skipped.")
		return
	}

	if err := executePush(); err != nil {
		printError("Push failed: " + err.Error())
	}
}

func formatFileList(changes []FileChange) string {
	names := make([]string, 0, len(changes))
	for _, change := range changes {
		names = append(names, change.FileName)
	}
	return strings.Join(names, ", ")
}

func formatFilesForDisplay(changes []FileChange) string {
	lines := make([]string, 0, len(changes))
	for _, change := range changes {
		lines = append(lines, fmt.Sprintf("  %s %s", styleStatus(change.Status), change.FileName))
	}
	return strings.Join(lines, "\n")
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

	output, err := runGitCommand("config", "--get", groqAPIKeyConfig)
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

	output, err := runGitCommand("config", "--get", groqModelConfig)
	if err == nil {
		if model := strings.TrimSpace(output); model != "" {
			return model
		}
	}

	return defaultGroqModel
}

func executeConfig(args []string) {
	if len(args) == 0 {
		printSection("Config")
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
		if err := setGitConfig(groqAPIKeyConfig, args[1]); err != nil {
			printError("Failed to save Groq API key: " + err.Error())
			return
		}
		printSuccess("Saved Groq API key to local git config.")
	case "groq-model":
		if len(args) < 2 {
			fmt.Println("Usage: config groq-model <model>")
			return
		}
		if err := setGitConfig(groqModelConfig, args[1]); err != nil {
			printError("Failed to save Groq model: " + err.Error())
			return
		}
		printSuccess("Saved Groq model to local git config.")
	case "show":
		printConfig()
	default:
		printError("Unknown config option.")
	}
}

func setGitConfig(key, value string) error {
	_, err := runGitCommand("config", "--local", key, value)
	return err
}

func printConfig() {
	printSection("Config")
	keySource := "missing"
	if strings.TrimSpace(os.Getenv("GROQ_API_KEY")) != "" {
		keySource = "environment"
	} else if key, err := runGitCommand("config", "--get", groqAPIKeyConfig); err == nil && strings.TrimSpace(key) != "" {
		keySource = "git config"
	}

	printPanel([]string{
		colorDim + "Groq key source" + colorReset + "  " + keySource,
		colorDim + "Groq model" + colorReset + "       " + getGroqModel(),
	})
}

func getDefaultRemoteName() string {
	output, err := runGitCommand("remote")
	if err != nil {
		return "missing"
	}

	for _, remote := range strings.Split(strings.TrimSpace(output), "\n") {
		remote = strings.TrimSpace(remote)
		if remote != "" {
			return remote
		}
	}

	return "missing"
}

func executePush() error {
	printSection("Push Preview")

	statusOutput, statusErr := runGitCommand("status", "--short", "--branch")
	if statusErr == nil && strings.TrimSpace(statusOutput) != "" {
		printPanel([]string{
			colorStrong + "Repository state" + colorReset,
			colorDim + strings.TrimSpace(statusOutput) + colorReset,
		})
	}

	if !promptYesNo("Approve push? [y/N]: ") {
		printWarning("Push cancelled.")
		return nil
	}

	fmt.Println("Running git push...")
	output, err := runGitCommand("push")
	if strings.TrimSpace(output) != "" {
		fmt.Println(output)
	}
	if err != nil {
		return err
	}

	printSuccess("Push completed.")
	return nil
}

func executePull() {
	printSection("Pull Preview")

	branchOutput, branchErr := runGitCommand("branch", "--show-current")
	branch := strings.TrimSpace(branchOutput)
	if branchErr != nil || branch == "" {
		branch = "(unknown)"
	}

	remote := getDefaultRemoteName()
	statusOutput, statusErr := runGitCommand("status", "--short", "--branch")

	lines := []string{
		colorStrong + "Pull target" + colorReset,
		colorDim + "Remote" + colorReset + "  " + remote,
		colorDim + "Branch" + colorReset + "  " + branch,
	}

	if statusErr == nil && strings.TrimSpace(statusOutput) != "" {
		lines = append(lines, "", colorStrong+"Repository state"+colorReset, colorDim+strings.TrimSpace(statusOutput)+colorReset)
	}

	printPanel(lines)

	if remote == "missing" {
		printError("No Git remote is configured.")
		return
	}

	if !promptYesNo("Approve pull? [y/N]: ") {
		printWarning("Pull cancelled.")
		return
	}

	printInfo("Running git pull --ff-only...")
	output, err := runGitCommand("pull", "--ff-only", remote, branch)
	if strings.TrimSpace(output) != "" {
		fmt.Println(output)
	}
	if err != nil {
		printError("Pull failed: " + err.Error())
		return
	}

	printSuccess("Pull completed.")
}

func printHelp() {
	printWelcome()
	printSection("Examples")
	printPanel([]string{
		"commit",
		colorDim + "Interactive commit session with approval per commit" + colorReset,
		"",
		"commit file main.go",
		colorDim + "Direct file-wise mode for selected files" + colorReset,
		"",
		"commit group",
		colorDim + "Direct AI group-wise mode" + colorReset,
		"",
		"config groq-key <api-key>",
		fmt.Sprintf("%sconfig groq-model %s%s", colorDim, defaultGroqModel, colorReset),
	})
}

func main() {
	if len(os.Args) > 1 {
		executeCommand(os.Args[1:])
	} else {
		printWelcome()
		runInteractive()
	}
}
