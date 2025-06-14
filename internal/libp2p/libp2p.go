package libp2p

import (
	"context"
	"fmt"
	"log"

	"github.com/elecbug/lab-chain/internal/cfg"
	"github.com/libp2p/go-libp2p"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/multiformats/go-multiaddr"
)

// setLibp2pHost creates a new libp2p host with the provided configuration
func SetLibp2pHost(cfg cfg.Config) (host.Host, error) {
	// Create a new libp2p host with the provided configuration
	rm, err := getResourceManager(cfg)

	h, err := libp2p.New(
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/%s/tcp/%d", cfg.Network.IPAddress, 12000)),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Muxer(yamux.ID, yamux.DefaultTransport),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.ResourceManager(rm),
	)

	identify.NewIDService(h)

	if err != nil {
		return nil, err
	}

	return h, nil
}

// getResourceManager configures the resource manager for the libp2p host
func getResourceManager(cfg cfg.Config) (network.ResourceManager, error) {
	// Set up resource limits based on the configuration
	limits := rcmgr.PartialLimitConfig{
		System: rcmgr.ResourceLimits{
			Conns: rcmgr.LimitVal(int(cfg.Network.MaxPeers)),
		},
	}

	// Create a fixed limiter with the configured limits
	limiter := rcmgr.NewFixedLimiter(limits.Build(rcmgr.DefaultLimits.AutoScale()))

	rm, err := rcmgr.NewResourceManager(limiter)

	if err != nil {
		return nil, err
	}

	return rm, err
}

// SetKadDHT initializes the Kademlia DHT for peer discovery and routing
func SetKadDHT(ctx context.Context, h host.Host, cfg cfg.Config) (*kaddht.IpfsDHT, error) {
	// Create a new Kademlia DHT instance with the provided host and configuration
	dht, err := kaddht.New(ctx, h,
		kaddht.Mode(getKadMode(cfg)),
		kaddht.ProtocolPrefix("/lab-chain-dht/1.0.0"),
		kaddht.BucketSize(20),
	)

	if err != nil {
		return nil, err
	}

	// Optionally set bootstrap peers if provided in the configuration
	for _, p := range cfg.DHT.BootstrapPeers {
		peerAddr, err := multiaddr.NewMultiaddr(p)

		if err != nil {
			log.Printf("Invalid multiaddr %s: %v", p, err)
			continue
		}

		peerInfo, err := peer.AddrInfoFromP2pAddr(peerAddr)

		if err != nil {
			log.Printf("Failed to parse peer info from multiaddr %s: %v", p, err)
			continue
		}

		err = h.Connect(ctx, *peerInfo)

		if err != nil {
			log.Printf("Failed to connect to peer %s: %v", p, err)
		}
	}

	if len(cfg.DHT.BootstrapPeers) > 0 {
		if err := dht.Bootstrap(ctx); err != nil {
			return nil, err
		}
	}

	return dht, nil
}

// getKadMode determines the Kademlia DHT mode based on the configuration
func getKadMode(cfg cfg.Config) kaddht.ModeOpt {
	switch cfg.DHT.Mode {
	case "server":
		return kaddht.ModeServer
	case "client":
		return kaddht.ModeClient
	default:
		log.Printf("Unknown DHT mode %s, defaulting to server mode", cfg.DHT.Mode)
		return kaddht.ModeServer
	}
}

// SetGossipSub initializes the GossipSub pubsub topics for block and transaction propagation
func SetGossipSub(ctx context.Context, h host.Host) (*pubsub.Topic, *pubsub.Topic, error) {
	ps, err := pubsub.NewGossipSub(ctx, h)

	blockTopic, err := ps.Join("lab-chain-blocks")

	if err != nil {
		return nil, nil, err
	}

	txTopic, err := ps.Join("lab-chain-transactions")

	if err != nil {
		return nil, nil, err
	}

	return blockTopic, txTopic, nil
}
