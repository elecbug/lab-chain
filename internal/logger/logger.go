package logger

import (
	ipfslog "github.com/ipfs/go-log/v2"
)

var AppLogger = ipfslog.Logger("app")
var GossipsubLogger = ipfslog.Logger("gossipsub")
var LabChainLogger = ipfslog.Logger("lab-chain")
