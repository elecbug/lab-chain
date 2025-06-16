package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/elecbug/lab-chain/internal/block"
	"github.com/elecbug/lab-chain/internal/logger"
	"github.com/elecbug/lab-chain/internal/transaction"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// CLICommand defines the command-line interface for blockchain operations
func CLICommand(bc *block.Blockchain, mempool *transaction.Mempool, pubBlk *pubsub.Topic, pubTx *pubsub.Topic, walletAddr string) {
	scanner := bufio.NewScanner(os.Stdin)
	log := logger.AppLogger

	log.Infof("CLI started. Type 'help' to see available commands.")

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
			fmt.Println("Available commands:")
			fmt.Println("  save [path]       - Save blockchain to file")
			fmt.Println("  load [path]       - Load blockchain from file")
			fmt.Println("  print             - Print the current blockchain")
			fmt.Println("  tx [to] [amount]  - Create and broadcast a transaction")
			fmt.Println("  mine              - Mine a new block and broadcast it")
			fmt.Println("  exit              - Exit the CLI")
		case "exit":
			return
		case "print":
			bc.Mu.Lock()
			for _, block := range bc.Blocks {
				fmt.Printf("Index: %d, Miner: %s, Hash: %x\n", block.Index, block.Miner, block.Hash)
			}
			bc.Mu.Unlock()
		case "save":
			if len(args) < 2 {
				fmt.Println("Usage: save [path]")
				continue
			}
			if err := bc.Save(args[1]); err != nil {
				fmt.Printf("Failed to save: %v\n", err)
			} else {
				fmt.Println("Blockchain saved.")
			}
		case "load":
			if len(args) < 2 {
				fmt.Println("Usage: load [path]")
				continue
			}
			if err := bc.Load(args[1]); err != nil {
				fmt.Printf("Failed to load: %v\n", err)
			} else {
				fmt.Println("Blockchain loaded.")
			}
		case "tx":

		case "mine":

		default:
			fmt.Println("Unknown command. Type 'help' for options.")
		}
	}
}
