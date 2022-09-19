package xtcp

import "time"

func (c *PoolConn) SendPkg(data []byte, option ...PkgOption) (err error) {
	if err = c.Conn.SendPkg(data, option...); err != nil && c.status == connStatusUnknown {
		if v, e := c.pool.NewFunc(); e == nil {
			c.Conn = v.(*PoolConn).Conn
			err = c.Conn.SendPkg(data, option...)
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

func (c *PoolConn) RecvPkg(option ...PkgOption) ([]byte, error) {
	data, err := c.Conn.RecvPkg(option...)
	if err != nil {
		c.status = connStatusError
	} else {
		c.status = connStatusActive
	}
	return data, err
}

func (c *PoolConn) RecvPkgWithTimeout(timeout time.Duration, option ...PkgOption) (data []byte, err error) {
	if err := c.SetreceiveDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}
	defer c.SetreceiveDeadline(time.Time{})
	data, err = c.RecvPkg(option...)
	return
}

func (c *PoolConn) SendPkgWithTimeout(data []byte, timeout time.Duration, option ...PkgOption) (err error) {
	if err := c.SetSendDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	defer c.SetSendDeadline(time.Time{})
	err = c.SendPkg(data, option...)
	return
}

func (c *PoolConn) SendRecvPkg(data []byte, option ...PkgOption) ([]byte, error) {
	if err := c.SendPkg(data, option...); err == nil {
		return c.RecvPkg(option...)
	} else {
		return nil, err
	}
}

func (c *PoolConn) SendRecvPkgWithTimeout(data []byte, timeout time.Duration, option ...PkgOption) ([]byte, error) {
	if err := c.SendPkg(data, option...); err == nil {
		return c.RecvPkgWithTimeout(timeout, option...)
	} else {
		return nil, err
	}
}
