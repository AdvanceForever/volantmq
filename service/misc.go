// Copyright (c) 2014 The SurgeMQ Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"encoding/binary"
	"io"
	"net"

	"errors"
	"github.com/troian/surgemq"
	"github.com/troian/surgemq/message"
)

// GetConnectMessage get CONNECT message
func GetConnectMessage(conn io.Closer) (*message.ConnectMessage, error) {
	buf, err := GetMessageBuffer(conn)
	if err != nil {
		appLog.Errorf("Receive error: %v", err)
		return nil, err
	}

	msg := message.NewConnectMessage()

	_, err = msg.Decode(buf)
	return msg, err
}

// GetConnAckMessage get CONACK message
func GetConnAckMessage(conn io.Closer) (*message.ConnAckMessage, error) {
	buf, err := GetMessageBuffer(conn)
	if err != nil {
		appLog.Debugf("Receive error: %v", err)
		return nil, err
	}

	msg := message.NewConnAckMessage()

	_, err = msg.Decode(buf)
	appLog.Debugf("Received: %s", msg)
	return msg, err
}

// WriteMessage into connection
func WriteMessage(conn io.Closer, msg message.Provider) error {
	buf := make([]byte, msg.Len())
	_, err := msg.Encode(buf)
	if err != nil {
		appLog.Debugf("Write error: %v", err)
		return err
	}
	appLog.Debugf("Writing: %s", msg)

	return WriteMessageBuffer(conn, buf)
}

// GetMessageBuffer from connection
func GetMessageBuffer(c io.Closer) ([]byte, error) {
	if c == nil {
		return nil, surgemq.ErrInvalidConnectionType
	}

	conn, ok := c.(net.Conn)
	if !ok {
		return nil, surgemq.ErrInvalidConnectionType
	}

	var buf []byte
	// tmp buffer to read a single byte
	var b = make([]byte, 1)
	// total bytes read
	var l int

	// Let's read enough bytes to get the message header (msg type, remaining length)
	for {
		// If we have read 5 bytes and still not done, then there's a problem.
		if l > 5 {
			return nil, errors.New("connect/getMessage: 4th byte of remaining length has continuation bit set")
		}

		n, err := conn.Read(b[0:])
		if err != nil {
			//if err == io.EOF {
			//	continue
			//} else {
			appLog.Errorf("Read error: %v", err)
			return nil, err
			//}
		}

		// Technically i don't think we will ever get here
		if n == 0 {
			continue
		}

		buf = append(buf, b...)
		l += n

		// Check the remLen byte (1+) to see if the continuation bit is set. If so,
		// increment cnt and continue reading. Otherwise break.
		if l > 1 && b[0] < 0x80 {
			break
		}
	}

	// Get the remaining length of the message
	remLen, _ := binary.Uvarint(buf[1:])
	buf = append(buf, make([]byte, remLen)...)

	for l < len(buf) {
		n, err := conn.Read(buf[l:])
		if err != nil {
			return nil, err
		}
		l += n
	}

	return buf, nil
}

// WriteMessageBuffer into connection
func WriteMessageBuffer(c io.Closer, b []byte) error {
	if c == nil {
		return surgemq.ErrInvalidConnectionType
	}

	conn, ok := c.(net.Conn)
	if !ok {
		return surgemq.ErrInvalidConnectionType
	}

	_, err := conn.Write(b)
	return err
}

// Copied from http://golang.org/src/pkg/net/timeout_test.go
//func isTimeout(err error) bool {
//	e, ok := err.(net.Error)
//	return ok && e.Timeout()
//}
