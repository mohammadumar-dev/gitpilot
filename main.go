package main

import (
	"fmt"
)

func printWelcome() {
	fmt.Println("Welcome to Git Pilot! Your AI-powered Git assistant.")
	fmt.Println("Commands:")
	fmt.Println("init, status, commit, push, pull, help")
}

func readCommand() string {
	var cmd string
	fmt.Print("Enter command: ")
	fmt.Scanln(&cmd)
	return cmd
}

func executeCommand(cmd string) {
	switch cmd {

	case "init":
		executeInit()

	case "status":
		executeStatus()

	case "commit":
		executeCommit()

	case "push":
		executePush()

	case "pull":
		executePull()

	case "help":
		printHelp()

	default:
		fmt.Println("❌ Invalid command. Type 'help'")
	}
}

func executeInit() {
	fmt.Println("Initializing repository...")
}

func executeStatus() {
	fmt.Println("Checking repository status...")
}

func executeCommit() {
	fmt.Println("Generating commit message...")
}

func executePush() {
	fmt.Println("Pushing to remote...")
}

func executePull() {
	fmt.Println("Pulling latest changes...")
}

func printHelp() {
	printWelcome()
}

func main() {
	printWelcome()

	cmd := readCommand()
	executeCommand(cmd)
}