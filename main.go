package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/atotto/clipboard"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	currentUser, _ := user.Current()
	hostname, _ := os.Hostname()

	fmt.Println("📋 Copyshell started. Every command output is copied to clipboard.")
	fmt.Println("Type 'exit' to quit.")
	fmt.Println("----------------------------------------------------------------")

	for {
		// 1. Display Prompt
		// simple prompt: user@host $
		fmt.Printf("\033[32m%s@%s\033[0m $ ", currentUser.Username, hostname)

		// 2. Read Input
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}

		input = strings.TrimSpace(input)

		// Handle empty input or exit
		if input == "" {
			continue
		}
		if input == "exit" {
			break
		}

		// 3. Handle 'cd' command strictly within the parent process
		// You cannot exec 'cd' because it runs in a child process
		parts := strings.Fields(input)
		if parts[0] == "cd" {
			dir := ""
			if len(parts) > 1 {
				dir = parts[1]
			} else {
				dir = currentUser.HomeDir
			}
			err := os.Chdir(dir)
			if err != nil {
				fmt.Printf("cd: %s\n", err)
			}
			continue
		}

		// 4. Execute Command
		// We use a shell (sh -c) to support pipes, redirects, and environment expansion
		cmd := exec.Command("sh", "-c", input)

		// For Windows support, you might switch the above line to:
		// cmd := exec.Command("cmd", "/C", input)

		// 5. Capture Output
		// We use MultiWriter to write to both stdout (terminal) and a buffer (for clipboard)
		var outputBuf bytes.Buffer
		multiOut := io.MultiWriter(os.Stdout, &outputBuf)
		multiErr := io.MultiWriter(os.Stderr, &outputBuf)

		cmd.Stdout = multiOut
		cmd.Stderr = multiErr

		err = cmd.Run()
		if err != nil {
			// Don't panic, just print the error (e.g. command not found)
			// We still copy the error output if any
			// fmt.Println(err)
		}

		// 6. Copy to Clipboard
		capturedOutput := outputBuf.String()
		if len(capturedOutput) > 0 {
			err := clipboard.WriteAll(capturedOutput)
			if err != nil {
				fmt.Printf("⚠️  Failed to copy to clipboard: %v\n", err)
			}
		}
	}
}