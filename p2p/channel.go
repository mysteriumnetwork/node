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
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/xtaci/kcp-go/v5"
	"golang.org/x/crypto/nacl/box"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const (
	maxRequestBodySize = 1 << 20 // 1MB.
)

// Channel represents p2p communication channel which can send and received data over encrypted and reliable UDP transport.
type Channel struct {
	mu sync.RWMutex

	srv         *http.Server
	listenPort  int
	handlers    map[string]HandlerFunc
	privateKey  PrivateKey
	blockCrypt  kcp.BlockCrypt
	sendTimeout time.Duration

	peer *Peer
}

// Peer represents p2p peer to which channel can send data.
type Peer struct {
	client *http.Client

	// Addr is public peer's endpoint address.
	Addr *net.UDPAddr
	// PublicKey is peer's public key.
	PublicKey PublicKey
}

// HandlerFunc is channel request handler func signature.
type HandlerFunc func(c Context) error

// NewChannel creates new p2p channel with initialized crypto primitives for data encryption
// and starts listening for connections.
func NewChannel(listenPort int, privateKey PrivateKey, peer *Peer) (*Channel, error) {
	blockCrypt, err := newBlockCrypt(privateKey, peer.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("could not create block crypt: %w", err)
	}

	c := &Channel{
		listenPort:  listenPort,
		privateKey:  privateKey,
		handlers:    make(map[string]HandlerFunc),
		blockCrypt:  blockCrypt,
		sendTimeout: 30 * time.Second,
		peer:        peer,
	}
	c.initPeer()

	ln, err := c.listen()
	if err != nil {
		return nil, err
	}
	go c.serve(ln)

	return c, nil
}

func (c *Channel) listen() (*kcp.Listener, error) {
	addr := fmt.Sprintf(":%d", c.listenPort)
	ln, err := kcp.ListenWithOptions(addr, c.blockCrypt, 10, 3)
	if err != nil {
		return nil, fmt.Errorf("could not create p2p listener: %w", err)
	}
	return ln, nil
}

func (c *Channel) serve(ln *kcp.Listener) error {
	// Configure server to use h2c.
	h2s := &http2.Server{}
	server := &http.Server{
		Handler:      h2c.NewHandler(&requestsHandler{ch: c}, h2s),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return server.Serve(ln)
}

// Close closes channel.
func (c *Channel) Close() error {
	return c.srv.Close()
}

// SetSendTimeout overrides default send timeout.
func (c *Channel) SetSendTimeout(t time.Duration) {
	c.sendTimeout = t
}

// Send sends data to given topic. Peer listening to topic will receive message.
func (c *Channel) Send(topic string, msg *Message) (*Message, error) {
	// Lock just to check if peer is added.
	c.mu.Lock()
	if c.peer == nil {
		c.mu.Unlock()
		return nil, errors.New("peer must be joined before send")
	}
	c.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), c.sendTimeout)
	defer cancel()

	reply, err := c.sendMsg(ctx, topic, msg)
	if err != nil {
		return nil, fmt.Errorf("could not send message: %w", err)
	}
	return reply, nil
}

// Handle registers handler for given topic. Handle should be called before listen similar as with HTTP server.
func (c *Channel) Handle(topic string, handler HandlerFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.handlers[topic] = handler
}

func (c *Channel) initPeer() {
	// TODO: Peer source port could change. We need to detect such possible change and update peer port automatically.
	transport := http2.Transport{
		DialTLS: func(network, addr string, cfg *tls.Config) (conn net.Conn, err error) {
			return kcp.DialWithOptions(addr, c.blockCrypt, 10, 3)
		},
		// Allow to use h2c.
		AllowHTTP: true,
	}
	c.peer.client = &http.Client{
		Transport: &transport,
		Timeout:   60 * time.Second,
	}
	// TODO: Start keep alive pings.
}

// sendMsg sends data bytes via HTTP POST request and optionally reads response body.
func (c *Channel) sendMsg(ctx context.Context, topic string, msg *Message) (*Message, error) {
	// Prepare new HTTP POST request with body payload.
	url := fmt.Sprintf("http://%s:%d/%s", c.peer.Addr.IP, c.peer.Addr.Port, topic)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(msg.Data))
	if err != nil {
		return nil, err
	}

	// Send request.
	res, err := c.peer.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// Check response status and parse possible peer response error.
	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusBadRequest {
			resErr, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("peer error response: %s", string(resErr))
		}
		return nil, fmt.Errorf("send failed: expected status %d, got %d", http.StatusOK, res.StatusCode)
	}

	// Return reply with data bytes.
	resData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return &Message{Data: resData}, nil
}

type requestsHandler struct {
	ch *Channel
}

// ServeHTTP implements http.Handler interface.
func (h *requestsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	topic := strings.TrimPrefix(r.URL.Path, "/")
	h.ch.mu.RLock()
	handler, found := h.ch.handlers[topic]
	h.ch.mu.RUnlock()
	if found {
		h.handle(w, r, handler)
	} else {
		log.Warn().Msgf("Handler for topic %q not found", topic)
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *requestsHandler) handle(w http.ResponseWriter, r *http.Request, handler HandlerFunc) {
	// Limit body size to prevent possible malicious clients sending big payloads.
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var reqData []byte
	reqData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Err(err).Msg("Failed to read request body")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	ctx := defaultContext{req: &Message{Data: reqData}}
	err = handler(&ctx)

	if err != nil {
		log.Err(err).Msg("Internal handler error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if ctx.publicError != nil {
		log.Err(ctx.publicError).Msg("Handled error")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(ctx.publicError.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	if ctx.res != nil {
		_, _ = w.Write(ctx.res.Data)
	}
}

func newBlockCrypt(privateKey PrivateKey, peerPublicKey PublicKey) (kcp.BlockCrypt, error) {
	// Compute shared key. Nonce for each message will be added inside kcp salsa block crypt.
	var sharedKey [32]byte
	box.Precompute(&sharedKey, (*[32]byte)(&peerPublicKey), (*[32]byte)(&privateKey))
	blockCrypt, err := kcp.NewSalsa20BlockCrypt(sharedKey[:])
	if err != nil {
		return nil, fmt.Errorf("could not create Sasla20 block crypt: %w", err)
	}
	return blockCrypt, nil
}
