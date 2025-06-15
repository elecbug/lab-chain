package logging

import (
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/elecbug/lab-chain/internal/logger"

	pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Gossipsub event tracer
type GossipsubTracer struct{}

// Gossipsub event tracer method
func (t *GossipsubTracer) Trace(evt *pb.TraceEvent) {
	log := logger.GossipsubLogger

	innerType := evt.Type.String()
	innerType = strings.ReplaceAll(strings.ToLower(innerType), "_", " ")

	switch *evt.Type.Enum() {
	case *pb.TraceEvent_PUBLISH_MESSAGE.Enum():
		evt := evt.PublishMessage

		log.Infow(innerType, "messageID", hex.EncodeToString(evt.MessageID), "topic", *evt.Topic)

	case *pb.TraceEvent_REJECT_MESSAGE.Enum():
		evt := evt.RejectMessage

		peerID, _ := peer.IDFromBytes(evt.ReceivedFrom)

		log.Infow(innerType, "messageID", hex.EncodeToString(evt.MessageID), "receivedFrom", peerID, "reason", *evt.Reason, "topic", *evt.Topic)

	case *pb.TraceEvent_DUPLICATE_MESSAGE.Enum():
		evt := evt.DuplicateMessage

		peerID, _ := peer.IDFromBytes(evt.ReceivedFrom)

		log.Infow(innerType, "messageID", hex.EncodeToString(evt.MessageID), "receivedFrom", peerID, "topic", *evt.Topic)

	case *pb.TraceEvent_DELIVER_MESSAGE.Enum():
		evt := evt.DeliverMessage

		peerID, _ := peer.IDFromBytes(evt.ReceivedFrom)

		log.Infow(innerType, "messageID", hex.EncodeToString(evt.MessageID), "receivedFrom", peerID, "topic", *evt.Topic)

	case *pb.TraceEvent_ADD_PEER.Enum():
		evt := evt.AddPeer

		peerID, _ := peer.IDFromBytes(evt.PeerID)

		log.Infow(innerType, "to", peerID, "proto", *evt.Proto)

	case *pb.TraceEvent_REMOVE_PEER.Enum():
		evt := evt.RemovePeer

		peerID, _ := peer.IDFromBytes(evt.PeerID)

		log.Infow(innerType, "to", peerID)

	case *pb.TraceEvent_JOIN.Enum():
		evt := evt.Join

		log.Infow(innerType, "topic", *evt.Topic)

	case *pb.TraceEvent_LEAVE.Enum():
		evt := evt.Leave

		log.Infow(innerType, "topic", *evt.Topic)

	case *pb.TraceEvent_GRAFT.Enum():
		evt := evt.Graft
		peerID, _ := peer.IDFromBytes(evt.PeerID)

		log.Infow(innerType, "to", peerID, "topic", *evt.Topic)

	case *pb.TraceEvent_PRUNE.Enum():
		evt := evt.Prune
		peerID, _ := peer.IDFromBytes(evt.PeerID)

		log.Infow(innerType, "to", peerID, "topic", *evt.Topic)

	case *pb.TraceEvent_SEND_RPC.Enum():
		evt := evt.SendRPC
		peerID, _ := peer.IDFromBytes(evt.SendTo)

		meta, _ := json.Marshal(evt.Meta)

		log.Debugw(innerType, "to", peerID, "meta", string(meta))

	case *pb.TraceEvent_RECV_RPC.Enum():
		evt := evt.RecvRPC
		peerID, _ := peer.IDFromBytes(evt.ReceivedFrom)

		meta, _ := json.Marshal(evt.Meta)

		log.Debugw(innerType, "from", peerID, "meta", string(meta))

	case *pb.TraceEvent_DROP_RPC.Enum():
		evt := evt.DropRPC
		peerID, _ := peer.IDFromBytes(evt.SendTo)

		meta, _ := json.Marshal(evt.Meta)

		log.Debugw(innerType, "to", peerID, "meta", string(meta))
	}
}
