package cli

import (
	"fmt"
	"strings"

	"github.com/chzyer/readline"
	"github.com/elecbug/lab-chain/internal/user"
)

// CliCommand defines the command-line interface for blockchain operations
func CliCommand(user *user.User) {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "$ ",
		HistoryFile:     "/tmp/labchain_cli_history.tmp",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		AutoComplete:    &cliCompleter{},
	})

	if err != nil {
		panic(err)
	}
	defer rl.Close()

	fmt.Println("Cli started. Type 'help' to see available commands.")

	for {
		line, err := rl.Readline()

		if err != nil {
			break
		}

		input := strings.TrimSpace(line)

		if input == "" {
			continue
		}

		args := strings.Fields(input)

		switch args[0] {
		case "help":
			fmt.Println("Available commands: help, exit, master-key, wallet, tx, mine, chain")
		case "exit":
			return
		case "master-key":
			masterKeyFunc(user, args)
		case "wallet":
			walletFunc(user, args)
		case "tx":
			txFunc(user, args)
		case "mine":
			mineFunc(user, args)
		case "chain":
			chainFunc(user, args)
		default:
			fmt.Printf("Unknown command. Type 'help' for options.\n")
		}
	}
}

type cliCompleter struct{}

// Completer implements the readline completer interface for command-line autocompletion
func (c *cliCompleter) Do(line []rune, pos int) ([][]rune, int) {
	cmdMap := map[string][]string{
		"master-key": {"gen", "save", "load"},
		"wallet":     {"set", "balance"},
		"tx":         {},
		"mine":       {"genesis"},
		"chain":      {"save", "load", "request"},
		"help":       {},
		"exit":       {},
	}

	var suggestions [][]rune
	input := string(line[:pos])
	tokens := strings.Fields(input)

	if len(tokens) == 0 {
		for cmd := range cmdMap {
			suggestions = append(suggestions, []rune(cmd)[pos:])
		}
		return suggestions, 0
	}

	if len(tokens) == 1 && !strings.HasSuffix(input, " ") {
		for cmd := range cmdMap {
			if strings.HasPrefix(cmd, tokens[0]) {
				suggestions = append(suggestions, []rune(cmd)[len(tokens[0]):])
			}
		}
		return suggestions, 0
	}

	rootCmd := tokens[0]
	subCmds, exists := cmdMap[rootCmd]
	if !exists {
		return nil, 0
	}

	if len(tokens) == 2 && !strings.HasSuffix(input, " ") {
		for _, sub := range subCmds {
			if strings.HasPrefix(sub, tokens[1]) {
				suggestions = append(suggestions, []rune(sub)[len(tokens[1]):])
			}
		}
		return suggestions, 0
	}

	if len(tokens) == 1 && strings.HasSuffix(input, " ") {
		for _, sub := range subCmds {
			suggestions = append(suggestions, []rune(sub))
		}
		return suggestions, 0
	}

	return nil, 0
}
