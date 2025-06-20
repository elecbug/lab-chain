package cli

import (
	"fmt"

	"github.com/elecbug/lab-chain/internal/user"
	"github.com/elecbug/lab-chain/internal/wallet"
)

func masterKeyFunc(user *user.User, args []string) {
	if len(args) != 3 {
		fmt.Printf("Usage: master-key <command> <file>\n")
		return
	}

	command := args[1]
	file := args[2]

	switch command {
	case "gen":
		masterKey, err := wallet.GenerateMasterKey()
		if err != nil {
			fmt.Printf("Failed to generate master key: %v.\n", err)

			return
		} else {
			fmt.Printf("Master key generated successfully.\n")
		}

		user.MasterKey = masterKey

		if err := wallet.SaveMasterKey(file, user.MasterKey); err != nil {
			fmt.Printf("Failed to save master key: %v.\n", err)

		} else {
			fmt.Printf("Master key saved to file successfully: %s.\n", file)
		}

	case "save":
		if user.MasterKey == nil {
			fmt.Printf("No master key generated. Please generate it first.\n")
			return
		}

		if err := wallet.SaveMasterKey(file, user.MasterKey); err != nil {
			fmt.Printf("Failed to save master key: %v.\n", err)

		} else {
			fmt.Printf("Master key saved to file successfully: %s.\n", file)
		}
	case "load":
		if user.MasterKey != nil {
			fmt.Printf("Master key already loaded. Please reset first.\n")
			return
		}

		masterKey, err := wallet.LoadMasterKey(file)

		if err != nil {
			fmt.Printf("Failed to load master key: %v.\n", err)

			return
		} else {
			fmt.Printf("Master key loaded successfully.\n")
		}

		user.MasterKey = masterKey
	default:
		fmt.Printf("Usage: master-key <command> <file>\n")
		return
	}
}
