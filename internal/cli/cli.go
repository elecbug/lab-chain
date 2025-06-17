package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/elecbug/lab-chain/internal/user"
)

// CliCommand defines the command-line interface for blockchain operations
func CliCommand(user *user.User) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Printf("Cli started. Type 'help' to see available commands.\n")

	for {
		fmt.Print("$ ")
		if !scanner.Scan() {
			break
		}
		input := scanner.Text()
		args := strings.Fields(input)
		if len(args) == 0 {
			continue
		}

		switch args[0] {
		case "help":
		case "exit":
			return
		case "mkey":
			mkeyFunc(user, args)
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
