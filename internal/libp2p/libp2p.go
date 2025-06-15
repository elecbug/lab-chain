package libp2p

import (
	"context"
	"fmt"

	"github.com/elecbug/lab-chain/internal/cfg"
	"github.com/elecbug/lab-chain/internal/logger"
	"github.com/elecbug/lab-chain/internal/logging"
	"github.com/libp2p/go-libp2p"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
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

// SetLibp2pHost creates a new libp2p host with the provided configuration
func SetLibp2pHost(cfg cfg.Config, priv crypto.PrivKey) (host.Host, error) {
	// Create a new libp2p host with the provided configuration
	rm, err := getResourceManager(cfg)

	if err != nil {
		return nil, fmt.Errorf("failed to create resource manager: %v", err)
	}

	h, err := libp2p.New(
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/%s/tcp/%d", cfg.Network.IPAddress, 12000)),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Muxer(yamux.ID, yamux.DefaultTransport),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.ResourceManager(rm),
		libp2p.Identity(priv),
	)

	identify.NewIDService(h)

	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %v", err)
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
		return nil, fmt.Errorf("failed to create resource manager: %v", err)
	}

	return rm, nil
}

// SetKadDHT initializes the Kademlia DHT for peer discovery and routing
func SetKadDHT(ctx context.Context, h host.Host, cfg cfg.Config) (*kaddht.IpfsDHT, error) {
	log := logger.AppLogger

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
			log.Infof("invalid multiaddr: %s", p)
			continue
		}

		peerInfo, err := peer.AddrInfoFromP2pAddr(peerAddr)

		if err != nil {
			log.Infof("failed to parse peer info from multiaddr: %s", p)
			continue
		}

		err = h.Connect(ctx, *peerInfo)

		if err != nil {
			log.Infof("failed to connect to peer: %s", p)
		}
	}

	if len(cfg.DHT.BootstrapPeers) > 0 {
		if err := dht.Bootstrap(ctx); err != nil {
			return nil, fmt.Errorf("failed to bootstrap DHT: %v", err)
		}
	}

	return dht, nil
}

// getKadMode determines the Kademlia DHT mode based on the configuration
func getKadMode(cfg cfg.Config) kaddht.ModeOpt {
	log := logger.AppLogger
	switch cfg.DHT.Mode {
	case "server":
		return kaddht.ModeServer
	case "client":
		return kaddht.ModeClient
	default:
		log.Infof("unknown DHT mode %s, defaulting to server mode", cfg.DHT.Mode)
		return kaddht.ModeServer
	}
}

// SetGossipSub initializes the GossipSub pubsub topics for block and transaction propagation
func SetGossipSub(ctx context.Context, h host.Host) (*pubsub.Topic, *pubsub.Topic, error) {
	ps, err := pubsub.NewGossipSub(ctx, h,
		pubsub.WithEventTracer(&logging.GossipsubTracer{}),
		pubsub.WithMessageSigning(true),
	)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GossipSub: %v", err)
	}

	blockTopic, err := ps.Join("lab-chain-blocks")

	if err != nil {
		return nil, nil, fmt.Errorf("failed to join block topic: %v", err)
	}

	txTopic, err := ps.Join("lab-chain-transactions")

	if err != nil {
		return nil, nil, fmt.Errorf("failed to join transaction topic: %v", err)
	}

	return blockTopic, txTopic, nil
}
