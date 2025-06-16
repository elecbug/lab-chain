package main

import (
	"bufio"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/elecbug/lab-chain/internal/block"
	"github.com/elecbug/lab-chain/internal/logger"
	"github.com/elecbug/lab-chain/internal/transaction"
	"github.com/elecbug/lab-chain/internal/user"
	"github.com/elecbug/lab-chain/internal/wallet"
)

// CLICommand defines the command-line interface for blockchain operations
func CLICommand(user *user.User) {
	scanner := bufio.NewScanner(os.Stdin)
	log := logger.AppLogger

	txSub, err := user.TxTopic.Subscribe()

	if err != nil {
		log.Errorf("failed to subscribe to transaction topic: %v", err)
		return
	} else {
		log.Infof("subscribed to transaction topic successfully")
	}

	transaction.RunSubscribeAndCollectTx(user.Context, txSub, user.MemPool)

	blkSub, err := user.BlockTopic.Subscribe()

	if err != nil {
		log.Errorf("failed to subscribe to block topic: %v", err)
		return
	} else {
		log.Infof("subscribed to block topic successfully")
	}

	block.RunSubscribeAndCollectBlock(user.Context, blkSub, user.Blockchain)

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
			mkeyFunc(user, args)
		case "wallet":
			walletFunc(user, args)
		case "tx":
			txFunc(user, args)
		case "mine":
			mineFunc(user, args)

		default:
			fmt.Println("Unknown command. Type 'help' for options.")
		}
	}
}

func mkeyFunc(user *user.User, args []string) {
	log := logger.AppLogger

	if len(args) < 3 {
		fmt.Println("Usage: master-key <command> <file>")
		return
	}

	command := args[1]
	file := args[2]

	if command == "gen" {
		masterKey, err := wallet.GenerateMasterKey()
		if err != nil {
			log.Errorf("failed to generate master key: %v", err)
			return
		} else {
			log.Infof("master key generated successfully: %s", masterKey.String())
		}

		user.MasterKey = masterKey
	} else if command == "save" {
		if user.MasterKey == nil {
			fmt.Println("No master key generated. Please generate it first.")
			return
		}

		if err := wallet.SaveMasterKey(file, user.MasterKey); err != nil {
			log.Errorf("failed to save master key: %v", err)
		} else {
			log.Infof("master key saved to file successfully: %s", file)
		}
	} else if command == "load" {
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

func walletFunc(user *user.User, args []string) {
	log := logger.AppLogger

	if len(args) < 2 {
		fmt.Println("Usage: wallet <command> [args]")
		return
	}

	command := args[1]

	switch command {
	case "set":
		if user.MasterKey == nil {
			fmt.Println("No master key loaded. Please load it first.")
			return
		}

		idxString := args[2]
		idx, err := strconv.Atoi(idxString)

		if err != nil {
			log.Warnf("invalid index: %v", err)
			return
		}

		priv, addr, err := wallet.GenerateAddress(user.MasterKey, idx)

		if err != nil {
			log.Warnf("failed to generate address: %v", err)
			return
		} else {
			user.CurrentPrivKey = priv
			user.CurrentAddress = addr

			log.Infof("address generated successfully: index %d, address %s", idx, addr.Hex())
		}
	case "balance":
		if user.MasterKey == nil {
			fmt.Println("No master key loaded. Please load it first.")
			return
		}

		balance := user.Blockchain.GetBalance(user.CurrentAddress.Hex())

		log.Infof("Current balance: %s", balance.String())
	default:
		fmt.Println("Unknown wallet command. Use 'balance'.")
	}
}

func txFunc(user *user.User, args []string) {
	log := logger.AppLogger

	if len(args) < 4 {
		fmt.Println("Usage: tx <to> <amount> <price>")
		return
	}

	to := args[1]
	amount, err := strconv.ParseInt(args[2], 10, 64)

	if err != nil {
		log.Warnf("invalid amount: %v", err)
		return
	}

	price, err := strconv.ParseInt(args[3], 10, 64)

	if err != nil {
		log.Warnf("invalid price: %v", err)
		return
	}

	if user.MasterKey == nil {
		fmt.Println("No master key loaded. Please load it first.")
		return
	}

	tx, err := transaction.CreateTx(user.CurrentPrivKey, to, big.NewInt(amount), big.NewInt(price), 0)

	if err != nil {
		log.Errorf("failed to create transaction: %v", err)
		return
	} else {
		log.Infof("transaction created successfully: %s -> %s, amount: %s, price: %s",
			tx.From, tx.To, tx.Amount.String(), tx.Price.String())
	}

	if err := transaction.PublishTx(user.Context, user.TxTopic, tx); err != nil {
		log.Errorf("failed to publish transaction: %v", err)
	} else {
		log.Infof("transaction published successfully")
	}
}

func mineFunc(user *user.User, args []string) {
	log := logger.AppLogger

	if user.MasterKey == nil {
		fmt.Println("No master key loaded. Please load it first.")
		return
	}
	if user.CurrentAddress == nil {
		fmt.Println("No current address set. Please set it first.")
		return
	}

	last := user.Blockchain.Blocks[len(user.Blockchain.Blocks)-1]

	b := user.Blockchain.MineBlock(last.Hash, last.Index+1, user.MemPool.PickTopTxs(20), user.CurrentAddress.Hex())

	err := block.PublishBlock(user.Context, user.BlockTopic, b)

	if err != nil {
		log.Errorf("failed to publish block: %v", err)
	} else {
		log.Infof("block mined and published successfully: index %d, miner %s, nonce %d, hash %x",
			b.Index, b.Miner, b.Nonce, b.Hash)
	}

	fmt.Printf("Block mined successfully: index %d, miner %s, nonce %d, hash %x\n",
		b.Index, b.Miner, b.Nonce, b.Hash)
}
