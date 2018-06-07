/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package management

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/textproto"
)

type mockMiddleware struct {
	OnStart        func(CommandWriter) error
	OnStop         func(CommandWriter) error
	OnLineReceived func(line string) (bool, error)
}

func (mm *mockMiddleware) Start(cmdWriter CommandWriter) error {
	if mm.OnStart != nil {
		return mm.OnStart(cmdWriter)
	}
	return nil
}

func (mm *mockMiddleware) Stop(cmdWriter CommandWriter) error {
	if mm.OnStop != nil {
		return mm.OnStop(cmdWriter)
	}
	return nil
}

func (mm *mockMiddleware) ConsumeLine(line string) (consumed bool, err error) {
	if mm.OnLineReceived != nil {
		return mm.OnLineReceived(line)
	}
	return true, nil
}

type mockOpenvpnProcess struct {
	conn    net.Conn
	CmdChan chan string
}

func (mop *mockOpenvpnProcess) Send(line string) error {
	_, err := io.WriteString(mop.conn, line)
	return err
}
func (mop *mockOpenvpnProcess) Disconnect() error {
	return mop.conn.Close()
}

func connectTo(addr Addr) (*mockOpenvpnProcess, error) {
	conn, err := net.Dial("tcp", addr.String())
	if err != nil {
		return nil, err
	}

	commandChannel := make(chan string, 100)
	go sendStringsToChannel(conn, commandChannel)
	return &mockOpenvpnProcess{
		conn:    conn,
		CmdChan: commandChannel,
	}, nil
}

func sendStringsToChannel(input io.Reader, ch chan<- string) {
	reader := textproto.NewReader(bufio.NewReader(input))
	for {
		line, err := reader.ReadLine()
		if err != nil {
			fmt.Println("Woops error:", err)
			return
		}
		ch <- line
	}
}
