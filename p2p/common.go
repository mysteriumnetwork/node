/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package p2p

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/pb"

	"google.golang.org/protobuf/proto"
)

const (
	pingMaxPorts       = 20
	requiredConnCount  = 2
	consumerInitialTTL = 128
)

type brokerConnector interface {
	Connect(serverURIs ...string) (nats.Connection, error)
}

type natConsumerPinger interface {
	PingProviderPeer(ctx context.Context, ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error)
}

type natProviderPinger interface {
	PingConsumerPeer(ctx context.Context, ip string, localPorts, remotePorts []int, initialTTL int, n int) (conns []*net.UDPConn, err error)
}

func configExchangeSubject(providerID identity.Identity, serviceType string) string {
	return fmt.Sprintf("%s.%s.p2p-config-exchange", providerID.Address, serviceType)
}

func configExchangeACKSubject(providerID identity.Identity, serviceType string) string {
	return fmt.Sprintf("%s.%s.p2p-config-exchange-ack", providerID.Address, serviceType)
}

func channelHandlersReadySubject(providerID identity.Identity, serviceType string) string {
	return fmt.Sprintf("%s.%s.p2p-channel-handlers-ready", providerID.Address, serviceType)
}

func acquireLocalPorts(portPool port.ServicePortSupplier, n int) ([]int, error) {
	ports, err := portPool.AcquireMultiple(n)
	if err != nil {
		return nil, err
	}
	var res []int
	for _, p := range ports {
		res = append(res, p.Num())
	}
	return res, nil
}

// packSignedMsg marshals, signs and returns ready to send bytes.
func packSignedMsg(signer identity.SignerFactory, signerID identity.Identity, msg *pb.P2PConfigExchangeMsg) ([]byte, error) {
	protoBytes, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	signature, err := signer(signerID).Sign(protoBytes)
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

// unpackSignedMsg verifies and unmarshal bytes to signed message.
func unpackSignedMsg(verifier identity.Verifier, b []byte) (*pb.P2PSignedMsg, error) {
	var signedMsg pb.P2PSignedMsg
	if err := proto.Unmarshal(b, &signedMsg); err != nil {
		return nil, err
	}
	if ok := verifier.Verify(signedMsg.Data, identity.SignatureBytes(signedMsg.Signature)); !ok {
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

// decryptConnConfigMsg decrypts bytes to connect config.
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
