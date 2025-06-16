package user

import "github.com/tyler-smith/go-bip32"

type User struct {
	MasterKey *bip32.Key // BIP-44 master key

}
