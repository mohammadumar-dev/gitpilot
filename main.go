package main

import (
	"fmt"
	"os"
)

func printWelcome() {
	fmt.Println("Welcome to Git Pilot! Your AI-powered Git assistant.")
	fmt.Println("Commands:")
	fmt.Println("init, status, commit, push, pull, help")
}

func readCommand() string {
	var cmd string
	fmt.Print("git-pilot> ")
	fmt.Scanln(&cmd)
	return cmd
}

func runInteractive() {
	for {
		cmd := readCommand()

		if cmd == "exit" {
			fmt.Println("Goodbye!")
			break
		}

		executeCommand(cmd)
	}
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

	if (len(os.Args) > 1) {
		cmd := os.Args[1]
		executeCommand(cmd)
	} else {
		runInteractive()
	}
}