package ws

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Conn represents a minimal WebSocket client connection.
type Conn struct {
	conn net.Conn
	br   *bufio.Reader
	mu   sync.Mutex
}

// Dial establishes a WebSocket client connection.
func Dial(ctx context.Context, rawURL string) (*Conn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse websocket url: %w", err)
	}

	host := u.Host
	if !strings.Contains(host, ":") {
		if u.Scheme == "wss" {
			host += ":443"
		} else {
			host += ":80"
		}
	}

	var d net.Dialer
	netConn, err := d.DialContext(ctx, "tcp", host)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", host, err)
	}

	var conn net.Conn = netConn

	if u.Scheme == "wss" {
		tlsConn := tls.Client(netConn, &tls.Config{ServerName: strings.Split(u.Host, ":")[0]})
		if deadline, ok := ctx.Deadline(); ok {
			_ = tlsConn.SetDeadline(deadline)
		}
		if err := tlsConn.Handshake(); err != nil {
			netConn.Close()
			return nil, fmt.Errorf("tls handshake: %w", err)
		}
		conn = tlsConn
	}

	key, err := newWebSocketKey()
	if err != nil {
		conn.Close()
		return nil, err
	}

	request := buildHandshakeRequest(u, key)
	if _, err := conn.Write([]byte(request)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("write handshake: %w", err)
	}

	br := bufio.NewReader(conn)

	statusLine, err := br.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("read handshake status: %w", err)
	}
	if !strings.Contains(statusLine, " 101 ") {
		conn.Close()
		return nil, fmt.Errorf("unexpected handshake response: %s", strings.TrimSpace(statusLine))
	}

	for {
		line, err := br.ReadString('\n')
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("read handshake header: %w", err)
		}
		if line == "\r\n" {
			break
		}
	}

	return &Conn{conn: conn, br: br}, nil
}

func buildHandshakeRequest(u *url.URL, key string) string {
	path := u.RequestURI()
	if path == "" {
		path = "/"
	}

	headers := []string{
		fmt.Sprintf("GET %s HTTP/1.1", path),
		fmt.Sprintf("Host: %s", u.Host),
		"Upgrade: websocket",
		"Connection: Upgrade",
		fmt.Sprintf("Sec-WebSocket-Key: %s", key),
		"Sec-WebSocket-Version: 13",
		"User-Agent: trade-bot/1.0",
		"Accept: */*",
		"\r\n",
	}
	return strings.Join(headers, "\r\n")
}

func newWebSocketKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate websocket key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// ReadMessage reads the next text or binary frame payload from the server.
func (c *Conn) ReadMessage() ([]byte, error) {
	for {
		firstByte, err := c.br.ReadByte()
		if err != nil {
			return nil, err
		}

		fin := firstByte&0x80 != 0
		opcode := firstByte & 0x0F

		secondByte, err := c.br.ReadByte()
		if err != nil {
			return nil, err
		}

		masked := secondByte&0x80 != 0
		payloadLen := int64(secondByte & 0x7F)

		switch payloadLen {
		case 126:
			var extended uint16
			if err := binaryRead(c.br, &extended); err != nil {
				return nil, err
			}
			payloadLen = int64(extended)
		case 127:
			var extended uint64
			if err := binaryRead(c.br, &extended); err != nil {
				return nil, err
			}
			payloadLen = int64(extended)
		}

		var maskKey []byte
		if masked {
			maskKey = make([]byte, 4)
			if _, err := io.ReadFull(c.br, maskKey); err != nil {
				return nil, err
			}
		}

		if payloadLen > math.MaxInt32 {
			return nil, errors.New("payload too large")
		}

		payload := make([]byte, payloadLen)
		if _, err := io.ReadFull(c.br, payload); err != nil {
			return nil, err
		}

		if masked {
			for i := range payload {
				payload[i] ^= maskKey[i%4]
			}
		}

		switch opcode {
		case 0x1, 0x2: // text or binary
			if !fin {
				return nil, errors.New("fragmented frames are not supported")
			}
			return payload, nil
		case 0x8: // close
			_ = c.writeControl(0x8, payload)
			return nil, io.EOF
		case 0x9: // ping
			if err := c.writeControl(0xA, payload); err != nil {
				return nil, err
			}
		case 0xA: // pong
			continue
		default:
			continue
		}
	}
}

func binaryRead(r io.Reader, data any) error {
	buf := make([]byte, 8)
	var length int
	switch data.(type) {
	case *uint16:
		length = 2
	case *uint64:
		length = 8
	default:
		return errors.New("unsupported type for binaryRead")
	}

	if _, err := io.ReadFull(r, buf[:length]); err != nil {
		return err
	}

	switch v := data.(type) {
	case *uint16:
		*v = uint16(buf[0])<<8 | uint16(buf[1])
	case *uint64:
		var result uint64
		for i := 0; i < 8; i++ {
			result = (result << 8) | uint64(buf[i])
		}
		*v = result
	}
	return nil
}

func (c *Conn) writeControl(opcode byte, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	header := []byte{0x80 | opcode}
	length := len(payload)

	switch {
	case length < 126:
		header = append(header, byte(0x80|length))
	case length <= math.MaxUint16:
		header = append(header, 0x80|126, byte(length>>8), byte(length))
	default:
		header = append(header, 0x80|127)
		extended := make([]byte, 8)
		for i := 7; i >= 0; i-- {
			extended[i] = byte(length & 0xFF)
			length >>= 8
		}
		header = append(header, extended...)
	}

	maskKey := make([]byte, 4)
	if _, err := rand.Read(maskKey); err != nil {
		return err
	}

	header = append(header, maskKey...)

	maskedPayload := make([]byte, len(payload))
	for i, b := range payload {
		maskedPayload[i] = b ^ maskKey[i%4]
	}

	if _, err := c.conn.Write(header); err != nil {
		return err
	}
	if len(maskedPayload) > 0 {
		if _, err := c.conn.Write(maskedPayload); err != nil {
			return err
		}
	}
	return nil
}

// SetReadDeadline sets the read deadline on the underlying connection.
func (c *Conn) SetReadDeadline(t time.Time) error {
	if deadlineConn, ok := c.conn.(interface{ SetReadDeadline(time.Time) error }); ok {
		return deadlineConn.SetReadDeadline(t)
	}
	return nil
}

// Close closes the underlying connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}
