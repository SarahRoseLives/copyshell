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

    fmt.Println("📋 Copyshell started. Every command output (except 'copytree') is copied to clipboard.")

    for {
       // 1. Display Prompt with Current Directory
       // Fetching the current working directory (pwd)
       cwd, _ := os.Getwd()

       // Replacing the home path with ~ for a cleaner look (optional)
       displayDir := strings.Replace(cwd, currentUser.HomeDir, "~", 1)

       fmt.Printf("\033[32m%s@%s\033[0m:\033[34m%s\033[0m$ ", currentUser.Username, hostname, displayDir)

       // 2. Read Input
       input, err := reader.ReadString('\n')
       if err != nil {
          fmt.Println("Error reading input:", err)
          continue
       }

       input = strings.TrimSpace(input)

       if input == "" {
          continue
       }
       if input == "exit" {
          break
       }

       // 3. Handle 'cd' command
       parts := strings.Fields(input)
       commandName := parts[0]

       if commandName == "cd" {
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
       cmd := exec.Command("sh", "-c", input)

       // 5. Capture Output
       var outputBuf bytes.Buffer
       multiOut := io.MultiWriter(os.Stdout, &outputBuf)
       multiErr := io.MultiWriter(os.Stderr, &outputBuf)

       cmd.Stdout = multiOut
       cmd.Stderr = multiErr

       _ = cmd.Run() // Run the command

       // 6. Copy to Clipboard (Modified Logic)
       // We only copy if the command is NOT 'copytree'
       if commandName != "copytree" {
          capturedOutput := outputBuf.String()
          if len(capturedOutput) > 0 {
             err := clipboard.WriteAll(capturedOutput)
             if err != nil {
                fmt.Printf("⚠️  Failed to copy to clipboard: %v\n", err)
             }
          }
       } else {
          fmt.Println("ℹ️  'copytree' output excluded from clipboard.")
       }
    }
}