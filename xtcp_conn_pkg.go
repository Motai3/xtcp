package xtcp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

const (
	pkgHeaderSizeDefault = 2 // 默认头长度
	pkgHeaderSizeMax     = 4 // 最大头长度
)

type PkgOption struct {
	HeaderSize  int
	MaxDataSize int
	Retry       Retry
}

func (c *Conn) SendPkg(data []byte, option ...PkgOption) error {
	pkgOption, err := getPkgOption(option...)
	if err != nil {
		return nil
	}
	length := len(data)
	if length > pkgOption.MaxDataSize {
		return fmt.Errorf(`data too long, data size %d`, length)
	}
	offset := pkgHeaderSizeMax - pkgOption.HeaderSize
	buffer := make([]byte, pkgHeaderSizeMax+len(data))
	binary.BigEndian.PutUint32(buffer[0:], uint32(length))
	copy(buffer[pkgHeaderSizeMax:], data)
	if pkgOption.Retry.Count > 0 {
		return c.Send(buffer[offset:], pkgOption.Retry)
	}
	return c.Send(buffer[offset:])
}

func (c *Conn) SendPkgWithTimeout(data []byte, timeout time.Duration, option ...PkgOption) error {
	if err := c.SetSendDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	defer c.SetSendDeadline(time.Time{})
	err := c.SendPkg(data, option...)
	return err
}

func (c *Conn) SendRecvPkg(data []byte, option ...PkgOption) ([]byte, error) {
	if err := c.SendPkg(data, option...); err == nil {
		return c.RecvPkg(option...)
	} else {
		return nil, err
	}
}

// SendRecvPkgWithTimeout writes data to connection and reads response with timeout using simple package protocol.
func (c *Conn) SendRecvPkgWithTimeout(data []byte, timeout time.Duration, option ...PkgOption) ([]byte, error) {
	if err := c.SendPkg(data, option...); err == nil {
		return c.RecvPkgWithTimeout(timeout, option...)
	} else {
		return nil, err
	}
}

func (c *Conn) RecvPkg(option ...PkgOption) (result []byte, err error) {
	var buffer []byte
	var length int
	pkgOption, err := getPkgOption(option...)
	if err != nil {
		return nil, err
	}
	buffer, err = c.Recv(pkgOption.HeaderSize, pkgOption.Retry)
	if err != nil {
		return nil, err
	}
	switch pkgOption.HeaderSize {
	case 1:
		length = int(binary.BigEndian.Uint32([]byte{0, 0, 0, buffer[0]}))
	case 2:
		length = int(binary.BigEndian.Uint32([]byte{0, 0, buffer[0], buffer[1]}))
	case 3:
		length = int(binary.BigEndian.Uint32([]byte{0, buffer[0], buffer[1], buffer[2]}))
	case 4:
		length = int(binary.BigEndian.Uint32([]byte{buffer[0], buffer[1], buffer[2], buffer[3]}))
	}
	if length < 0 || length > pkgOption.MaxDataSize {
		return nil, fmt.Errorf(`data too long, data size is %d`, length)
	}
	if length == 0 {
		return nil, nil
	}
	return c.Recv(length, pkgOption.Retry)
}

func (c *Conn) RecvPkgWithTimeout(timeout time.Duration, option ...PkgOption) (data []byte, err error) {
	if err := c.SetreceiveDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}
	defer c.SetreceiveDeadline(time.Time{})
	data, err = c.RecvPkg(option...)
	return
}

func getPkgOption(option ...PkgOption) (*PkgOption, error) {
	pkgOption := PkgOption{}
	if len(option) > 0 {
		pkgOption = option[0]
	}
	if pkgOption.HeaderSize == 0 {
		pkgOption.HeaderSize = pkgHeaderSizeDefault
	}
	if pkgOption.HeaderSize > pkgHeaderSizeMax {
		return nil, errors.New("pkgHeaderSize is too big")
	}
	if pkgOption.MaxDataSize == 0 {
		switch pkgOption.HeaderSize {
		case 1:
			pkgOption.MaxDataSize = 0xFF
		case 2:
			pkgOption.MaxDataSize = 0xFFFF
		case 3:
			pkgOption.MaxDataSize = 0xFFFFFF
		case 4:
			pkgOption.MaxDataSize = 0x7FFFFFFF
		}
	}
	if pkgOption.MaxDataSize > 0x7FFFFFFF {
		return nil, errors.New("DataSize is too big")
	}
	return &pkgOption, nil
}
