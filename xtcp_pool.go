package xtcp

import (
	"sync"
	"time"
	"xtcp/container/xpool"
)

// PoolConn 是一个具有池特性的连接
type PoolConn struct {
	*Conn
	pool   *xpool.Pool
	status int
}

const (
	defaultPoolExpire = 10 * time.Second
	connStatusUnknown = 0
	connStatusActive  = 1
	connStatusError   = 2 //意味着应该从池中关闭并删除
)

//提供地址到池子的映射
var addressPoolMap sync.Map

func NewPoolConn(addr string, timeout ...time.Duration) (*PoolConn, error) {
	var pool *xpool.Pool
	xpool.New(defaultPoolExpire, func() (interface{}, error) {
		if conn, err := NewConn(addr, timeout...); err == nil {
			return &PoolConn{conn, pool, connStatusActive}, nil
		} else {
			return nil, err
		}
	})
	v, _ := addressPoolMap.LoadOrStore(addr, pool)

	if value, err := v.(*xpool.Pool).Get(); err == nil {
		return value.(*PoolConn), nil
	} else {
		return nil, err
	}
}

func (c *PoolConn) Send(data []byte, retry ...Retry) error {
	err := c.Conn.Send(data, retry...)
	if err != nil && (c.status == connStatusUnknown || c.status == connStatusError) {
		if v, e := c.pool.Get(); e == nil {
			c.Conn = v.(*PoolConn).Conn
			err = c.Send(data, retry...)
		} else {
			err = e
		}
	}
	if err != nil {
		c.status = connStatusError
	} else {
		c.status = connStatusActive
	}
	return err
}

func (c *PoolConn) Recv(length int, retry ...Retry) ([]byte, error) {
	data, err := c.Conn.Recv(length, retry...)
	if err != nil {
		c.status = connStatusError
	} else {
		c.status = connStatusActive
	}
	return data, err
}

func (c *PoolConn) RecvLine(retry ...Retry) ([]byte, error) {
	data, err := c.Conn.RecvLine(retry...)
	if err != nil {
		c.status = connStatusError
	} else {
		c.status = connStatusActive
	}
	return data, err
}

func (c *PoolConn) RecvTil(til []byte, retry ...Retry) ([]byte, error) {
	data, err := c.Conn.RecvTil(til, retry...)
	if err != nil {
		c.status = connStatusError
	} else {
		c.status = connStatusActive
	}
	return data, err
}

func (c *PoolConn) RecvWithTimeout(length int, timeout time.Duration, retry ...Retry) (data []byte, err error) {
	if err := c.SetreceiveDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}
	defer c.SetreceiveDeadline(time.Time{})
	data, err = c.Recv(length, retry...)
	return
}

func (c *PoolConn) SendWithTimeout(data []byte, timeout time.Duration, retry ...Retry) (err error) {
	if err := c.SetSendDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	defer c.SetSendDeadline(time.Time{})
	err = c.Send(data, retry...)
	return
}

func (c *PoolConn) SendRecv(data []byte, receive int, retry ...Retry) ([]byte, error) {
	if err := c.Send(data, retry...); err == nil {
		return c.Recv(receive, retry...)
	} else {
		return nil, err
	}
}

func (c *PoolConn) SendRecvWithTimeout(data []byte, receive int, timeout time.Duration, retry ...Retry) ([]byte, error) {
	if err := c.Send(data, retry...); err == nil {
		return c.RecvWithTimeout(receive, timeout, retry...)
	} else {
		return nil, err
	}
}
