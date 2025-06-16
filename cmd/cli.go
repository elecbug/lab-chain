package main

import (
	"bufio"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/elecbug/lab-chain/internal/blockchain"
	"github.com/elecbug/lab-chain/internal/logger"
	"github.com/elecbug/lab-chain/internal/user"
	"github.com/elecbug/lab-chain/internal/wallet"
)

// CLICommand defines the command-line interface for blockchain operations
func CLICommand(user *user.User) {
	log := logger.AppLogger
	scanner := bufio.NewScanner(os.Stdin)

	log.Infof("cli started. Type 'help' to see available commands.")
	fmt.Printf("cli started. Type 'help' to see available commands\n")

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
		case "genesis":
			genesisFunc(user, args)
		case "chain":
			chainFunc(user, args)

		default:
			fmt.Printf("Unknown command. Type 'help' for options\n")
		}
	}
}

func chainFunc(user *user.User, args []string) {
	log := logger.AppLogger

	if len(args) < 3 {
		fmt.Printf("Usage: chain <command> <file>\n")
		return
	}

	command := args[1]
	file := args[2]

	switch command {
	case "save":
		if user.Blockchain == nil {
			fmt.Printf("Blockchain not initialized\n")
			return
		}

		if err := user.Blockchain.Save(args[2]); err != nil {
			log.Errorf("failed to save blockchain: %v", err)
			fmt.Printf("failed to save blockchain: %v\n", err)

		} else {
			log.Infof("blockchain saved successfully")
			fmt.Printf("blockchain saved successfully\n")
		}
	case "load":
		if user.Blockchain != nil {
			fmt.Printf("Blockchain already loaded. Please reset first\n")
			return
		}

		chain, err := blockchain.Load(file)

		if err != nil {
			log.Errorf("failed to load blockchain: %v", err)
			fmt.Printf("failed to load blockchain: %v\n", err)

			return
		} else {
			log.Infof("blockchain loaded successfully from %s", file)
			fmt.Printf("blockchain loaded successfully from %s\n", file)
		}

		user.Blockchain = chain

		txSub, err := user.TxTopic.Subscribe()

		if err != nil {
			log.Errorf("failed to subscribe to transaction topic: %v", err)
			fmt.Printf("failed to subscribe to transaction topic: %v\n", err)

			return
		} else {
			log.Infof("subscribed to transaction topic successfully")
			fmt.Printf("subscribed to transaction topic successfully\n")
		}

		blockchain.RunSubscribeAndCollectTx(user.Context, txSub, user.MemPool, user.Blockchain)

		blkSub, err := user.BlockTopic.Subscribe()

		if err != nil {
			log.Errorf("failed to subscribe to block topic: %v", err)
			fmt.Printf("failed to subscribe to block topic: %v\n", err)

			return
		} else {
			log.Infof("subscribed to block topic successfully")
			fmt.Printf("subscribed to block topic successfully\n")
		}

		blockchain.RunSubscribeAndCollectBlock(user.Context, blkSub, user.MemPool, user.Blockchain)
	default:
		fmt.Printf("Unknown chain command. Use 'status' or 'save'\n")
	}
}

func mkeyFunc(user *user.User, args []string) {
	log := logger.AppLogger

	if len(args) < 3 {
		fmt.Printf("Usage: master-key <command> <file>\n")
		return
	}

	command := args[1]
	file := args[2]

	if command == "gen" {
		masterKey, err := wallet.GenerateMasterKey()
		if err != nil {
			log.Errorf("failed to generate master key: %v", err)
			fmt.Printf("failed to generate master key: %v\n", err)

			return
		} else {
			log.Infof("master key generated successfully: %s", masterKey.String())
			fmt.Printf("master key generated successfully: %s\n", masterKey.String())
		}

		user.MasterKey = masterKey
	} else if command == "save" {
		if user.MasterKey == nil {
			fmt.Printf("No master key generated. Please generate it first\n")
			return
		}

		if err := wallet.SaveMasterKey(file, user.MasterKey); err != nil {
			log.Errorf("failed to save master key: %v", err)
			fmt.Printf("failed to save master key: %v\n", err)

		} else {
			log.Infof("master key saved to file successfully: %s", file)
			fmt.Printf("master key saved to file successfully: %s\n", file)
		}
	} else if command == "load" {
		if user.MasterKey != nil {
			fmt.Printf("Master key already loaded. Please reset first\n")
			return
		}

		masterKey, err := wallet.LoadMasterKey(file)

		if err != nil {
			log.Errorf("failed to load master key: %v", err)
			fmt.Printf("failed to load master key: %v\n", err)

			return
		} else {
			log.Infof("master key loaded successfully: %s", masterKey.String())
			fmt.Printf("master key loaded successfully: %s\n", masterKey.String())
		}

		user.MasterKey = masterKey
	} else {
		fmt.Printf("Unknown master key command. Use 'gen', 'save', or 'load'\n")
	}
}

func walletFunc(user *user.User, args []string) {
	log := logger.AppLogger

	if len(args) < 2 {
		fmt.Printf("Usage: wallet <command> [args]\n")
		return
	}

	command := args[1]

	switch command {
	case "set":
		if user.MasterKey == nil {
			fmt.Printf("No master key loaded. Please load it first\n")
			return
		}

		idxString := args[2]
		idx, err := strconv.Atoi(idxString)

		if err != nil {
			log.Errorf("invalid index: %v", err)
			fmt.Printf("invalid index: %v\n", err)

			return
		}

		priv, addr, err := wallet.GenerateAddress(user.MasterKey, idx)

		if err != nil {
			log.Errorf("failed to generate address: %v", err)
			fmt.Printf("failed to generate address: %v\n", err)

			return
		} else {
			user.CurrentPrivKey = priv
			user.CurrentAddress = addr

			log.Infof("address generated successfully: index %d, address %s", idx, addr.Hex())
			fmt.Printf("address generated successfully: index %d, address %s\n", idx, addr.Hex())
		}
	case "balance":
		if user.MasterKey == nil {
			fmt.Printf("No master key loaded. Please load it first.")
			return
		}

		balance := user.Blockchain.GetBalance(user.CurrentAddress.Hex())

		log.Infof("Current balance: %s", balance.String())
		fmt.Printf("Current balance: %s\n", balance.String())
	default:
		fmt.Printf("Unknown wallet command. Use 'balance'\n")
	}
}

func txFunc(user *user.User, args []string) {
	log := logger.AppLogger

	if len(args) < 4 {
		fmt.Printf("Usage: tx <to> <amount> <price>\n")
		return
	}

	to := args[1]
	amount, err := strconv.ParseInt(args[2], 10, 64)

	if err != nil {
		log.Errorf("invalid amount: %v", err)
		fmt.Printf("invalid amount: %v\n", err)

		return
	}

	price, err := strconv.ParseInt(args[3], 10, 64)

	if err != nil {
		log.Errorf("invalid price: %v", err)
		fmt.Printf("invalid price: %v\n", err)

		return
	}

	if user.MasterKey == nil {
		fmt.Printf("No master key loaded. Please load it first\n")
		return
	}
	if user.CurrentAddress == nil {
		fmt.Printf("No current address set. Please set it first\n")
		return
	}
	if user.Blockchain == nil {
		fmt.Printf("Blockchain not initialized. Please create genesis block first\n")
		return
	}

	tx, err := blockchain.CreateTx(user.CurrentPrivKey, to, big.NewInt(amount), big.NewInt(price), user.Blockchain)

	if err != nil {
		log.Errorf("failed to create transaction: %v", err)
		fmt.Printf("failed to create transaction: %v\n", err)

		return
	} else {
		log.Infof("transaction created successfully: %s -> %s, amount: %s, price: %s",
			tx.From, tx.To, tx.Amount.String(), tx.Price.String())
		fmt.Printf("transaction created successfully: %s -> %s, amount: %s, price: %s\n",
			tx.From, tx.To, tx.Amount.String(), tx.Price.String())
	}

	if err := blockchain.PublishTx(user.Context, user.TxTopic, tx); err != nil {
		log.Errorf("failed to publish transaction: %v", err)
		fmt.Printf("failed to publish transaction: %v\n", err)

	} else {
		log.Infof("transaction published successfully")
		fmt.Printf("transaction published successfully\n")
	}
}

func mineFunc(user *user.User, args []string) {
	log := logger.AppLogger

	if user.MasterKey == nil {
		fmt.Printf("No master key loaded. Please load it first\n")
		return
	}
	if user.CurrentAddress == nil {
		fmt.Printf("No current address set. Please set it first\n")
		return
	}
	if user.Blockchain == nil {
		fmt.Printf("Blockchain not initialized. Please create genesis block first\n")
		return
	}

	last := user.Blockchain.Blocks[len(user.Blockchain.Blocks)-1]

	txs := user.MemPool.PickTopTxs(20)

	b := user.Blockchain.MineBlock(last.Hash, last.Index+1, txs, user.CurrentAddress.Hex())
	user.Blockchain.Blocks = append(user.Blockchain.Blocks, b)

	err := blockchain.PublishBlock(user.Context, user.BlockTopic, b)

	if err != nil {
		log.Errorf("failed to publish block: %v", err)
		fmt.Printf("failed to publish block: %v\n", err)

	} else {
		log.Infof("block mined and published successfully: index %d, miner %s, nonce %d, hash %x",
			b.Index, b.Miner, b.Nonce, b.Hash)
		fmt.Printf("block mined and published successfully: index %d, miner %s, nonce %d, hash %x\n",
			b.Index, b.Miner, b.Nonce, b.Hash)
	}

	fmt.Printf("Block mined successfully: index %d, miner %s, nonce %d, hash %x\n",
		b.Index, b.Miner, b.Nonce, b.Hash)
}

func genesisFunc(user *user.User, args []string) {
	log := logger.AppLogger

	if user.MasterKey == nil {
		fmt.Printf("No master key loaded. Please load it first\n")
		return
	}
	if user.CurrentAddress == nil {
		fmt.Printf("No current address set. Please set it first\n")
		return
	}

	user.Blockchain = blockchain.InitBlockchain(user.CurrentAddress.Hex())

	log.Infof("genesis block created successfully: index %d, miner %s, nonce %d, hash %x",
		user.Blockchain.Blocks[0].Index,
		user.Blockchain.Blocks[0].Miner,
		user.Blockchain.Blocks[0].Nonce,
		user.Blockchain.Blocks[0].Hash,
	)
	fmt.Printf("genesis block created successfully: index %d, miner %s, nonce %d, hash %x\n",
		user.Blockchain.Blocks[0].Index,
		user.Blockchain.Blocks[0].Miner,
		user.Blockchain.Blocks[0].Nonce,
		user.Blockchain.Blocks[0].Hash,
	)

	b := user.Blockchain.Blocks[0]
	err := blockchain.PublishBlock(user.Context, user.BlockTopic, b)

	if err != nil {
		log.Errorf("failed to publish block: %v", err)
		fmt.Printf("failed to publish block: %v\n", err)

	} else {
		log.Infof("block mined and published successfully: index %d, miner %s, nonce %d, hash %x",
			b.Index, b.Miner, b.Nonce, b.Hash)
		fmt.Printf("block mined and published successfully: index %d, miner %s, nonce %d, hash %x\n",
			b.Index, b.Miner, b.Nonce, b.Hash)
	}

	txSub, err := user.TxTopic.Subscribe()

	if err != nil {
		log.Errorf("failed to subscribe to transaction topic: %v", err)
		fmt.Printf("failed to subscribe to transaction topic: %v\n", err)

		return
	} else {
		log.Infof("subscribed to transaction topic successfully")
		fmt.Printf("subscribed to transaction topic successfully\n")
	}

	blockchain.RunSubscribeAndCollectTx(user.Context, txSub, user.MemPool, user.Blockchain)

	blkSub, err := user.BlockTopic.Subscribe()

	if err != nil {
		log.Errorf("failed to subscribe to block topic: %v", err)
		fmt.Printf("failed to subscribe to block topic: %v\n", err)

		return
	} else {
		log.Infof("subscribed to block topic successfully")
		fmt.Printf("subscribed to block topic successfully\n")
	}

	blockchain.RunSubscribeAndCollectBlock(user.Context, blkSub, user.MemPool, user.Blockchain)
}
