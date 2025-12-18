package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/atotto/clipboard"
	"github.com/chzyer/readline"
)

// --- 1. Robust "Split & Match" Completer ---
type FileCompleter struct{}

func (c *FileCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	// 1. Get the input up to the cursor
	inputSoFar := string(line[:pos])

	// 2. Isolate the current word being typed
	// We split by space to get the last token (e.g., "cd ./tar" -> "./tar")
	lastSeparatorPos := strings.LastIndexAny(inputSoFar, " \t")
	var word string
	if lastSeparatorPos == -1 {
		word = inputSoFar
	} else {
		word = inputSoFar[lastSeparatorPos+1:]
	}

	// 3. Split the word into Directory and File parts
	// e.g. "./tar" -> dir="./", file="tar"
	// e.g. "dv"     -> dir="",    file="dv"
	dir, filePrefix := filepath.Split(word)

	// If dir is empty, we are looking in the current directory
	searchDir := dir
	if searchDir == "" {
		searchDir = "."
	}

	// 4. Read the directory
	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return nil, 0
	}

	// 5. Filter matches and calculate Suffix
	for _, entry := range entries {
		name := entry.Name()

		// Check if this file matches the filePrefix
		if strings.HasPrefix(name, filePrefix) {
			// Calculate the missing part (suffix)
			// e.g. name="target", prefix="tar" -> suffix="get"
			suffix := strings.TrimPrefix(name, filePrefix)

			// If it's a directory, add a slash to the suffix
			if entry.IsDir() {
				suffix += "/"
			}

			// Append the SUFFIX to the suggestions
			newLine = append(newLine, []rune(suffix))
		}
	}

	// 6. Return length 0 to tell readline "just append this suffix"
	return newLine, 0
}

func main() {
	currentUser, _ := user.Current()
	hostname, _ := os.Hostname()

	// 2. Setup Readline
	historyFile := filepath.Join(currentUser.HomeDir, ".copyshell_history")

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "> ",
		HistoryFile:     historyFile,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		AutoComplete:    &FileCompleter{}, // Inject fixed completer
	})
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	// 3. Signal Handler
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		for range sigChan {
			// Swallow Ctrl+C
		}
	}()


	for {
		cwd, _ := os.Getwd()
		displayDir := strings.Replace(cwd, currentUser.HomeDir, "~", 1)

		prompt := fmt.Sprintf("\033[32m%s@%s\033[0m:\033[34m%s\033[0m$ ", currentUser.Username, hostname, displayDir)
		rl.SetPrompt(prompt)

		// 4. Read Line
		input, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				if len(input) == 0 {
					continue
				}
			} else if err == io.EOF {
				break
			} else {
				fmt.Println("Error reading input:", err)
				continue
			}
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if input == "exit" {
			break
		}

		// 5. Handle 'cd'
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

		// 6. Execute Command
		cmd := exec.Command("sh", "-c", input)

		var outputBuf bytes.Buffer
		multiOut := io.MultiWriter(os.Stdout, &outputBuf)
		multiErr := io.MultiWriter(os.Stderr, &outputBuf)

		cmd.Stdout = multiOut
		cmd.Stderr = multiErr
		// cmd.Stdin = os.Stdin

		_ = cmd.Run()

		// 7. Copy to Clipboard
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