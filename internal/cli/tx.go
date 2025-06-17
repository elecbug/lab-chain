package cli

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/elecbug/lab-chain/internal/chain"
	"github.com/elecbug/lab-chain/internal/user"
)

func TxFunc(user *user.User, args []string) {
	if len(args) != 4 {
		fmt.Printf("Usage: tx <to> <amount> <price>\n")
		return
	}

	to := args[1]
	amount, err := strconv.ParseInt(args[2], 10, 64)

	if err != nil {
		fmt.Printf("Invalid amount: %v.\n", err)

		return
	}

	price, err := strconv.ParseInt(args[3], 10, 64)

	if err != nil {
		fmt.Printf("Invalid price: %v.\n", err)

		return
	}

	if user.MasterKey == nil {
		fmt.Printf("No master key loaded. Please load it first.\n")
		return
	}
	if user.CurrentAddress == nil {
		fmt.Printf("No current address set. Please set it first.\n")
		return
	}
	if user.Chain == nil {
		fmt.Printf("Blockchain not initialized. Please create genesis block first.\n")
		return
	}

	tx, err := chain.CreateTx(user.CurrentPrivKey, to, big.NewInt(amount), big.NewInt(price), user.Chain)

	if err != nil {
		fmt.Printf("Failed to create transaction: %v.\n", err)

		return
	} else {
		fmt.Printf("Transaction created successfully: %s -> %s, amount: %s, price: %s.\n",
			tx.From, tx.To, tx.Amount.String(), tx.Price.String())
	}

	if err := tx.PublishTx(user.Context, user.TxTopic); err != nil {
		fmt.Printf("Failed to publish transaction: %v.\n", err)

	} else {
		fmt.Printf("Transaction published successfully.\n")
	}
}
