package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/elecbug/lab-chain/internal/logger"
	"github.com/elecbug/lab-chain/internal/user"
	"github.com/elecbug/lab-chain/internal/wallet"
)

// CLICommand defines the command-line interface for blockchain operations
func CLICommand(user *user.User) {
	scanner := bufio.NewScanner(os.Stdin)
	log := logger.AppLogger

	log.Infof("cli started. Type 'help' to see available commands.")

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
			masterKey(user, args)
			case ""
		default:
			fmt.Println("Unknown command. Type 'help' for options.")
		}
	}
}

func masterKey(user *user.User, args []string) {
	log := logger.AppLogger

	if len(args) < 3 {
		fmt.Println("Usage: master-key <order> <file>")
		return
	}

	order := args[1]
	file := args[2]

	if order == "gen" {
		masterKey, err := wallet.GenerateMasterKey()
		if err != nil {
			log.Errorf("failed to generate master key: %v", err)
			return
		} else {
			log.Infof("master key generated successfully: %s", masterKey.String())
		}

		user.MasterKey = masterKey
	} else if order == "save" {
		if user.MasterKey == nil {
			fmt.Println("No master key generated. Please generate it first.")
			return
		}

		if err := wallet.SaveMasterKey(file, user.MasterKey); err != nil {
			log.Errorf("failed to save master key: %v", err)
		} else {
			log.Infof("master key saved to file successfully: %s", file)
		}
	} else if order == "load" {
		if user.MasterKey != nil {
			fmt.Println("Master key already loaded. Please reset first.")
			return
		}

		masterKey, err := wallet.LoadMasterKey(file)

		if err != nil {
			log.Errorf("failed to load master key: %v", err)
			return
		} else {
			log.Infof("master key loaded successfully: %s", masterKey.String())
		}

		user.MasterKey = masterKey
	} else {
		fmt.Println("Unknown master key command. Use 'gen', 'save', or 'load'.")
	}
}
