package p2p

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/node/requests"
	nats_lib "github.com/nats-io/go-nats"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

// brokerConnector connects to broker.
type brokerConnector interface {
	Connect(serverURIs ...string) (nats.Connection, error)
}

const pingMaxPorts = 10

func NewManager(broker brokerConnector, address string, signer identity.SignerFactory) *Manager {
	return &Manager{
		broker:        broker,
		brokerAddress: address,
		log: func(msg ...interface{}) {
			fmt.Println("CONSUMER", msg)
		},
		pendingConfigs: map[PublicKey]*p2pConnectConfig{},
		ipResolver:     ip.NewResolver(requests.NewHTTPClient("0.0.0.0", requests.DefaultTimeout), "0.0.0.0", "https://api.ipify.org/?format=json"),
		signer:         signer,
		verifier:       identity.NewVerifierSigned(),
		portPool:       port.NewPool(),
		pinger:         traversal.NewPinger(traversal.DefaultPingConfig(), eventbus.New()),
	}
}

// Manager knows how to exchange p2p keys and encrypted configuration and creates ready to use p2p channels.
type Manager struct {
	portPool      *port.Pool
	broker        brokerConnector
	pinger        traversal.NATPeerPinger
	signer        identity.SignerFactory
	verifier      identity.Verifier
	ipResolver    ip.Resolver
	brokerAddress string
	log           func(msg ...interface{}) // TODO: Temp for debugging. Later add proper debug messages if needed.

	// Keys holds mapping for peer public key and current private key
	// which are used during exchange ack message handling.
	// TODO: This is only needed for provider side. Maybe split this manager into smaller units later.
	pendingConfigs   map[PublicKey]*p2pConnectConfig
	pendingConfigsMu sync.Mutex
}

type p2pConnectConfig struct {
	peerPublicIP string
	peerPorts    []int
	localPorts   []int
	privateKey   PrivateKey
	peerPubKey   PublicKey
}

func (m *Manager) CreateChannel(consumerID, providerID identity.Identity, timeout time.Duration) (*Channel, error) {
	config, err := m.configExchangeInitiator(consumerID, providerID, timeout)
	if err != nil {
		return nil, fmt.Errorf("could not exchange config: %w", err)
	}

	var remotePort, localPort int
	if len(config.peerPorts) == 1 {
		localPort = config.localPorts[0]
		remotePort = config.peerPorts[0]
		m.log("will not ping")
	} else {
		m.log("ping", config.peerPublicIP, config.localPorts, config.peerPorts)
		conns, err := m.pinger.PingPeer(config.peerPublicIP, config.localPorts, config.peerPorts, 128, 1)
		if err != nil {
			return nil, fmt.Errorf("could not ping peer: %w", err)
		}
		localPort = conns[0].LocalAddr().(*net.UDPAddr).Port
		remotePort = conns[0].RemoteAddr().(*net.UDPAddr).Port
		conns[0].Close()
		// conns[1].Close()
	}
	// TODO: Need to close pinger conns so open ports can be used.

	peer := Peer{
		Addr:      &net.UDPAddr{IP: net.ParseIP(config.peerPublicIP), Port: remotePort},
		PublicKey: config.peerPubKey,
	}
	m.log("local port", localPort, "peer port", remotePort)
	channel, err := NewChannel(localPort, config.privateKey, &peer)
	if err != nil {
		return nil, fmt.Errorf("could not create p2p channel: %w", err)
	}
	m.log("channel created")
	return channel, nil
}

func (m *Manager) SubscribeChannel(providerID identity.Identity, channelHandler func(ch *Channel)) error {
	brokerConn, err := m.broker.Connect(m.brokerAddress)
	if err != nil {
		return err
	}
	// TODO: Expose func to close broker conn.

	_, err = brokerConn.Subscribe(fmt.Sprintf("%s.p2p-config-exchange", providerID.Address), func(msg *nats_lib.Msg) {
		if err := m.initialExchangeHandler(brokerConn, providerID, msg); err != nil {
			log.Err(err).Msg("Could not handle initial exchange")
			return
		}
	})

	_, err = brokerConn.Subscribe(fmt.Sprintf("%s.p2p-config-exchange-ack", providerID.Address), func(msg *nats_lib.Msg) {
		config, err := m.ackHandler(msg)
		if err != nil {
			log.Err(err).Msg("Could not handle exchange ack")
			return
		}

		// Send ack so consumer can start pinging.
		if err := brokerConn.Publish(msg.Reply, []byte("OK")); err != nil {
			log.Err(err).Msg("Could not publish exchange ack")
		}

		var remotePort, localPort int
		if len(config.peerPorts) == 1 {
			localPort = config.localPorts[0]
			remotePort = config.peerPorts[0]
		} else {
			m.log("ping", config.peerPublicIP, config.localPorts, config.peerPorts)
			conns, err := m.pinger.PingPeer(config.peerPublicIP, config.localPorts, config.peerPorts, 2, 1)
			if err != nil {
				log.Err(err).Msg("Could not ping peer")
				return
			}
			localPort = conns[0].LocalAddr().(*net.UDPAddr).Port
			remotePort = conns[0].RemoteAddr().(*net.UDPAddr).Port
			conns[0].Close()
			// conns[1].Close()
		}
		// TODO: Close open ping conns.

		peer := Peer{
			Addr:      &net.UDPAddr{IP: net.ParseIP(config.peerPublicIP), Port: remotePort},
			PublicKey: config.peerPubKey,
		}
		m.log("local port", localPort, "peer port", remotePort)
		channel, err := NewChannel(localPort, config.privateKey, &peer)
		if err != nil {
			log.Err(err).Msg("Could not create channel")
			return
		}

		channelHandler(channel)
	})
	return err
}

func (m *Manager) initialExchangeHandler(brokerConn nats.Connection, signerID identity.Identity, msg *nats_lib.Msg) error {
	pubKey, privateKey, err := GenerateKey()
	if err != nil {
		return fmt.Errorf("could not generate provider p2p keys: %w", err)
	}

	// Get initial peer exchange with it's public key.
	m.log("1")
	signedMsg, err := m.unpackSignedMsg(msg.Data)
	if err != nil {
		return err
	}
	var peerExchangeMsg pb.P2PConfigExchangeMsg
	if err := proto.Unmarshal(signedMsg.Data, &peerExchangeMsg); err != nil {
		return err
	}
	peerPubKey, err := DecodePublicKey(peerExchangeMsg.PublicKey)
	if err != nil {
		return err
	}
	// Send reply with encrypted exchange config.
	m.log("2")
	publicIP, err := m.ipResolver.GetPublicIP()
	if err != nil {
		return err
	}
	localPorts, err := m.acquireLocalPorts()
	if err != nil {
		return err
	}
	config := pb.P2PConnectConfig{
		PublicIP: publicIP,
		Ports:    intToInt32Slice(localPorts),
	}
	configCiphertext, err := encryptConnConfigMsg(&config, privateKey, peerPubKey)
	if err != nil {
		return err
	}
	exchangeMsg := pb.P2PConfigExchangeMsg{
		PublicKey:        pubKey.Hex(),
		ConfigCiphertext: configCiphertext,
	}
	packedMsg, err := m.packSignedMsg(signerID, &exchangeMsg)
	if err != nil {
		return err
	}
	err = brokerConn.Publish(msg.Reply, packedMsg)
	if err != nil {
		return err
	}

	m.pendingConfigsMu.Lock()
	m.pendingConfigs[peerPubKey] = &p2pConnectConfig{
		localPorts: localPorts,
		privateKey: privateKey,
		peerPubKey: peerPubKey,
	}
	m.pendingConfigsMu.Unlock()
	return nil
}

func (m *Manager) ackHandler(msg *nats_lib.Msg) (*p2pConnectConfig, error) {
	signedMsg, err := m.unpackSignedMsg(msg.Data)
	if err != nil {
		return nil, fmt.Errorf("could not unpack signed msg: %w", err)
	}
	var peerExchangeMsg pb.P2PConfigExchangeMsg
	if err := proto.Unmarshal(signedMsg.Data, &peerExchangeMsg); err != nil {
		return nil, fmt.Errorf("could not unmarshal exchange msg: %w", err)
	}
	peerPubKey, err := DecodePublicKey(peerExchangeMsg.PublicKey)
	if err != nil {
		return nil, err
	}

	m.pendingConfigsMu.Lock()
	config, ok := m.pendingConfigs[peerPubKey]
	m.pendingConfigsMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("pending config not found for key %s", peerPubKey.Hex())
	}

	peerConfig, err := decryptConnConfigMsg(peerExchangeMsg.ConfigCiphertext, config.privateKey, peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt peer conn config: %w", err)
	}

	return &p2pConnectConfig{
		privateKey:   config.privateKey,
		localPorts:   config.localPorts,
		peerPubKey:   config.peerPubKey,
		peerPublicIP: peerConfig.PublicIP,
		peerPorts:    int32ToIntSlice(peerConfig.Ports),
	}, nil
}

func (m *Manager) configExchangeInitiator(consumerID, providerID identity.Identity, timeout time.Duration) (*p2pConnectConfig, error) {
	pubKey, privateKey, err := GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("could not generate consumer p2p keys: %w", err)
	}

	brokerConn, err := m.broker.Connect(m.brokerAddress)
	if err != nil {
		return nil, fmt.Errorf("could not open broker conn: %w", err)
	}
	defer brokerConn.Close()

	// Send initial exchange with signed consumer public key.
	m.log("1")
	beginExchangeMsg := &pb.P2PConfigExchangeMsg{
		PublicKey: pubKey.Hex(),
	}
	exchangeMsgBrokerReply, err := m.sendSignedMsg(brokerConn, fmt.Sprintf("%s.p2p-config-exchange", providerID.Address), consumerID, beginExchangeMsg, timeout)
	if err != nil {
		return nil, fmt.Errorf("could not send signed message: %w", err)
	}

	// Parse provider response with public key and encrypted and signed connection config.
	m.log("2")
	exchangeMsgReplySignedMsg, err := m.unpackSignedMsg(exchangeMsgBrokerReply)
	if err != nil {
		return nil, fmt.Errorf("could not unpack peer siged message: %w", err)
	}
	var exchangeMsgReply pb.P2PConfigExchangeMsg
	if err := proto.Unmarshal(exchangeMsgReplySignedMsg.Data, &exchangeMsgReply); err != nil {
		return nil, fmt.Errorf("could not unmarshal peer signed message payload: %w", err)
	}
	peerPubKey, err := DecodePublicKey(exchangeMsgReply.PublicKey)
	if err != nil {
		return nil, err
	}
	peerConnConfig, err := decryptConnConfigMsg(exchangeMsgReply.ConfigCiphertext, privateKey, peerPubKey)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt peer conn config: %w", err)
	}
	// Finally send consumer encrypted and signed connect config in ack message.
	m.log("3")
	publicIP, err := m.ipResolver.GetPublicIP()
	if err != nil {
		return nil, err
	}
	localPorts, err := m.acquireLocalPorts()
	if err != nil {
		return nil, err
	}
	connConfig := &pb.P2PConnectConfig{
		PublicIP: publicIP,
		Ports:    intToInt32Slice(localPorts),
	}
	connConfigCiphertext, err := encryptConnConfigMsg(connConfig, privateKey, peerPubKey)
	if err != nil {
		return nil, err
	}
	endExchangeMsg := &pb.P2PConfigExchangeMsg{
		PublicKey:        pubKey.Hex(),
		ConfigCiphertext: connConfigCiphertext,
	}
	_, err = m.sendSignedMsg(brokerConn, fmt.Sprintf("%s.p2p-config-exchange-ack", providerID.Address), consumerID, endExchangeMsg, timeout)
	if err != nil {
		return nil, err
	}

	return &p2pConnectConfig{
		privateKey:   privateKey,
		localPorts:   localPorts,
		peerPubKey:   peerPubKey,
		peerPublicIP: peerConnConfig.PublicIP,
		peerPorts:    int32ToIntSlice(peerConnConfig.Ports),
	}, nil
}

func (m *Manager) acquireLocalPorts() ([]int, error) {
	ports, err := m.portPool.AcquireMultiple(pingMaxPorts)
	if err != nil {
		return nil, err
	}
	var res []int
	for _, p := range ports {
		res = append(res, p.Num())
	}
	return res, nil
}

func (m *Manager) sendSignedMsg(brokerConn nats.Connection, subject string, senderID identity.Identity, msg *pb.P2PConfigExchangeMsg, timeout time.Duration) ([]byte, error) {
	packedMsg, err := m.packSignedMsg(senderID, msg)
	if err != nil {
		return nil, err
	}
	reply, err := brokerConn.Request(subject, packedMsg, timeout)
	if err != nil {
		return nil, err
	}
	return reply.Data, nil
}

// packSignedMsg marshals, signs and returns ready to send bytes.
func (m *Manager) packSignedMsg(signerID identity.Identity, msg *pb.P2PConfigExchangeMsg) ([]byte, error) {
	protoBytes, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	signature, err := m.signer(signerID).Sign(protoBytes)
	if err != nil {
		return nil, err
	}
	signedMsg := &pb.P2PSignedMsg{Data: protoBytes, Signature: signature.Bytes()}
	signedMsgProtoBytes, err := proto.Marshal(signedMsg)
	if err != nil {
		return nil, err
	}
	return signedMsgProtoBytes, nil
}

func (m *Manager) unpackSignedMsg(b []byte) (*pb.P2PSignedMsg, error) {
	var signedMsg pb.P2PSignedMsg
	if err := proto.Unmarshal(b, &signedMsg); err != nil {
		return nil, err
	}
	if ok := m.verifier.Verify(signedMsg.Data, identity.SignatureBytes(signedMsg.Signature)); !ok {
		return nil, errors.New("message signature is invalid")
	}
	return &signedMsg, nil
}

// encryptConnConfigMsg encrypts proto message and returns bytes.
func encryptConnConfigMsg(msg *pb.P2PConnectConfig, privateKey PrivateKey, peerPubKey PublicKey) ([]byte, error) {
	protoBytes, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	ciphertext, err := privateKey.Encrypt(peerPubKey, protoBytes)
	if err != nil {
		return nil, err
	}
	return ciphertext, nil
}

func decryptConnConfigMsg(ciphertext []byte, privateKey PrivateKey, peerPubKey PublicKey) (*pb.P2PConnectConfig, error) {
	peerConnectConfigProtoBytes, err := privateKey.Decrypt(peerPubKey, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt config to proto bytes: %w", err)
	}
	var peerProtoConnectConfig pb.P2PConnectConfig
	if err := proto.Unmarshal(peerConnectConfigProtoBytes, &peerProtoConnectConfig); err != nil {
		return nil, fmt.Errorf("could not unmarshal decrypted conn config: %w", err)
	}
	return &peerProtoConnectConfig, nil
}

func int32ToIntSlice(arr []int32) []int {
	var res []int
	for _, v := range arr {
		res = append(res, int(v))
	}
	return res
}

func intToInt32Slice(arr []int) []int32 {
	var res []int32
	for _, v := range arr {
		res = append(res, int32(v))
	}
	return res
}
