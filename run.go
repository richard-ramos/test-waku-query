package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/muxer/mplex"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/multiformats/go-multiaddr"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

// Default options used in the libp2p node
var DefaultLibP2POptions = []libp2p.Option{
	libp2p.ChainOptions(
		libp2p.Transport(tcp.NewTCPTransport),
	),
	libp2p.ChainOptions(
		libp2p.Muxer("/yamux/1.0.0", yamux.DefaultTransport),
		libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport),
	),
	libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
}

func ContentTopic(t []byte) string {
	enc := hexutil.Encode(t)
	return "/waku/1/" + enc + "/rfc26"
}

// ToTopic converts a string to a whisper topic.
func ToTopic(s string) []byte {
	return crypto.Keccak256([]byte(s))[:types.TopicLength]
}

var nodeList = []string{
	"/dns4/node-01.ac-cn-hongkong-c.status.prod.statusim.net/tcp/30303/p2p/16Uiu2HAkvEZgh3KLwhLwXg95e5ojM8XykJ4Kxi2T7hk22rnA7pJC",
	"/dns4/node-01.do-ams3.status.prod.statusim.net/tcp/30303/p2p/16Uiu2HAm6HZZr7aToTvEBPpiys4UxajCTU97zj5v7RNR2gbniy1D",
	"/dns4/node-01.gc-us-central1-a.status.prod.statusim.net/tcp/30303/p2p/16Uiu2HAkwBp8T6G77kQXSNMnxgaMky1JeyML5yqoTHRM8dbeCBNb",
	"/dns4/node-02.ac-cn-hongkong-c.status.prod.statusim.net/tcp/30303/p2p/16Uiu2HAmFy8BrJhCEmCYrUfBdSNkrPw6VHExtv4rRp1DSBnCPgx8",
	"/dns4/node-02.do-ams3.status.prod.statusim.net/tcp/30303/p2p/16Uiu2HAmSve7tR5YZugpskMv2dmJAsMUKmfWYEKRXNUxRaTCnsXV",
	"/dns4/node-02.gc-us-central1-a.status.prod.statusim.net/tcp/30303/p2p/16Uiu2HAmDQugwDHM3YeUp86iGjrUvbdw3JPRgikC7YoGBsT2ymMg",
}

func main() {

	topic := "0x0324f1c85601c3c8faf6758dc9b79c3565525759e1ff66260019b2033919eb0bf0c2c1efca-c309-4655-b4c4-fb3cb0e32594"
	topicBytes := ToTopic(topic)
	contentTopic := ContentTopic(topicBytes)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host1, err := libp2p.New(DefaultLibP2POptions...)
	if err != nil {
		panic(err)
	}

	s1 := store.NewWakuStore(host1, nil, nil, utils.Logger())
	s1.Start(ctx)
	defer s1.Stop()

	for i, n := range nodeList {
		queryNode(ctx, n, host1, contentTopic, s1, "first", i)
		queryNode(ctx, n, host1, contentTopic, s1, "second", i)
		queryNode(ctx, n, host1, contentTopic, s1, "third", i)

	}
}

func queryNode(ctx context.Context, node string, host1 host.Host, contentTopic string, s1 *store.WakuStore, attempt string, i int) {
	p, err := multiaddr.NewMultiaddr(node)
	if err != nil {
		panic(err)
	}

	info, err := peer.AddrInfoFromP2pAddr(p)
	if err != nil {
		panic(err)
	}

	err = host1.Connect(ctx, *info)
	if err != nil {
		fmt.Printf("Could not connect to %s: %s", info.ID, err.Error())
		return
	}

	cnt := 0
	cursorIterations := 0

	result, err := s1.Query(ctx, store.Query{
		Topic:         "/waku/2/default-waku/proto",
		ContentTopics: []string{contentTopic},
		StartTime:     int64(1668613188 * time.Second),
		EndTime:       int64(1668696055 * time.Second),
	}, store.WithPeer(info.ID), store.WithPaging(false, 100), store.WithRequestId([]byte{1, 2, 3, 4, 5, 6, 7, 8, byte(i)}))
	if err != nil {
		fmt.Printf("Could not query %s: %s", info.ID, err.Error())
		return
	}

	for {
		cnt += len(result.Messages)
		cursorIterations += 1

		if result.IsComplete() {
			break
		}

		result, err = s1.Next(ctx, result)
		if err != nil {
			fmt.Printf("Could not retrieve more results from %s: %s", info.ID, err.Error())
		}
	}

	fmt.Printf(">>>> %d messages found in %s in the %s attempt (Used cursor %d times)\n", cnt, info.ID, attempt, cursorIterations)
}
