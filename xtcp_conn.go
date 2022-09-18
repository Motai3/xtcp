package xtcp

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"time"
)

type Conn struct {
	net.Conn
	reader            *bufio.Reader
	receiveDeadline   time.Time
	sendDeadline      time.Time
	receiveBufferWait time.Duration //读取缓冲的间隔时间
}

const receiveAllWaitTimeout = time.Millisecond

func NewConn(addr string, timeout ...time.Duration) (*Conn, error) {
	if conn, err := NewNetConn(addr, timeout...); err == nil {
		return NewConnByNetConn(conn), nil
	} else {
		return nil, err
	}
}

func NewConnTLS(addr string, tlsConfig *tls.Config) (*Conn, error) {
	if conn, err := NewNetConnTLS(addr, tlsConfig); err == nil {
		return NewConnByNetConn(conn), nil
	} else {
		return nil, err
	}
}

func NewConnByKeyCrt(addr, crtFile, keyFile string) (*Conn, error) {
	if conn, err := NewNetConnKeyCrt(addr, crtFile, keyFile); err == nil {
		return NewConnByNetConn(conn), nil
	} else {
		return nil, err
	}
}

func NewConnByNetConn(conn net.Conn) *Conn {
	return &Conn{
		Conn:              conn,
		reader:            bufio.NewReader(conn),
		receiveDeadline:   time.Time{},
		sendDeadline:      time.Time{},
		receiveBufferWait: receiveAllWaitTimeout,
	}
}

func (c *Conn) Send(data []byte, retry ...Retry) error {
	for {
		if _, err := c.Write(data); err != nil {
			if err == io.EOF {
				return err
			}

			if len(retry) == 0 || retry[0].Count == 0 {
				return err
			}
			if len(retry) > 0 {
				retry[0].Count--
				if retry[0].Interval == 0 {
					retry[0].Interval = defaultRetryInternal
				}
				time.Sleep(retry[0].Interval)
			}
		} else {
			return nil
		}
	}
}

func (c *Conn) Recv(length int, retry ...Retry) ([]byte, error) {
	var err error
	var size int
	var index int
	var buffer []byte
	var bufferWait bool

	if length > 0 {
		buffer = make([]byte, length)
	} else {
		buffer = make([]byte, defaultReadBufferSize)
	}

	for {
		if length < 0 && index > 0 {
			bufferWait = true
			if err = c.SetReadDeadline(time.Now().Add(c.receiveBufferWait)); err != nil {
				return nil, err
			}
		}
		size, err = c.reader.Read(buffer[index:])
		if size > 0 {
			index += size
			if length > 0 {
				if index == length {
					break
				}
			} else {
				if index >= defaultReadBufferSize {
					buffer = append(buffer, make([]byte, defaultReadBufferSize)...)
				} else {
					if !bufferWait {
						break
					}
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			if bufferWait && isTimeout(err) {
				if err = c.SetReadDeadline(c.sendDeadline); err != nil {
					return nil, err
				}
				err = nil
				break
			}

			if len(retry) > 0 {
				// It fails even it retried.
				if retry[0].Count == 0 {
					break
				}
				retry[0].Count--
				if retry[0].Interval == 0 {
					retry[0].Interval = defaultRetryInternal
				}
				time.Sleep(retry[0].Interval)
				continue
			}
			break
		}
		if length == 0 {
			break
		}
	}
	return buffer[:index], err
}

func (c *Conn) RecvLine(retry ...Retry) ([]byte, error) {
	var err error
	var buffer []byte
	data := make([]byte, 0)
	for {
		buffer, err = c.Recv(1, retry...)
		if len(buffer) > 0 {
			if buffer[0] == '\n' {
				data = append(data, buffer[:len(buffer)-1]...)
			} else {
				data = append(data, buffer...)
			}
		}
		if err != nil {
			break
		}
	}
	return data, err
}

// RecvTil 自定义分隔符，包含分隔符
func (c *Conn) RecvTil(til []byte, retry ...Retry) ([]byte, error) {
	var err error
	var buffer []byte
	data := make([]byte, 0)
	length := len(til)
	for {
		buffer, err = c.Recv(1, retry...)
		if len(buffer) > 0 {
			if length > 0 &&
				len(data) >= length-1 &&
				buffer[0] == til[length-1] &&
				bytes.EqualFold(data[len(data)-length+1:], til[:length-1]) {
				data = append(data, buffer...)
				break
			} else {
				data = append(data, buffer...)
			}
		}
		if err != nil {
			break
		}
	}
	return data, err
}

func (c *Conn) RecvWithTimeout(length int, timeout time.Duration, retry ...Retry) (data []byte, err error) {
	if err := c.SetreceiveDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}
	defer c.SetreceiveDeadline(time.Time{})
	data, err = c.Recv(length, retry...)
	return
}

func (c *Conn) SendWithTimeout(data []byte, timeout time.Duration, retry ...Retry) (err error) {
	if err := c.SetSendDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	defer c.SetSendDeadline(time.Time{})
	err = c.Send(data, retry...)
	return
}

func (c *Conn) SendRecv(data []byte, length int, retry ...Retry) ([]byte, error) {
	if err := c.Send(data, retry...); err == nil {
		return c.Recv(length, retry...)
	} else {
		return nil, err
	}
}

// SendRecvWithTimeout writes data to the connection and reads response with timeout.
func (c *Conn) SendRecvWithTimeout(data []byte, length int, timeout time.Duration, retry ...Retry) ([]byte, error) {
	if err := c.Send(data, retry...); err == nil {
		return c.RecvWithTimeout(length, timeout, retry...)
	} else {
		return nil, err
	}
}

func (c *Conn) SetDeadline(t time.Time) error {
	err := c.Conn.SetDeadline(t)
	if err == nil {
		c.receiveDeadline = t
		c.sendDeadline = t
	}
	return err
}

func (c *Conn) SetreceiveDeadline(t time.Time) error {
	err := c.SetReadDeadline(t)
	if err == nil {
		c.receiveDeadline = t
	}
	return err
}

func (c *Conn) SetSendDeadline(t time.Time) error {
	err := c.SetWriteDeadline(t)
	if err == nil {
		c.sendDeadline = t
	}
	return err
}

// SetreceiveBufferWait sets the buffer waiting timeout when reading all data from connection.
// The waiting duration cannot be too long which might delay receiving data from remote address.
func (c *Conn) SetreceiveBufferWait(bufferWaitDuration time.Duration) {
	c.receiveBufferWait = bufferWaitDuration
}
