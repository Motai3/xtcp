package xtcp_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
	"xtcp"
)

func Test_Pool_Package_Basic(t *testing.T) {
	p := portList.PopFront()
	s := xtcp.NewServer(fmt.Sprintf(`:%d`, p), func(conn *xtcp.Conn) {
		defer conn.Close()
		for {
			data, err := conn.RecvPkg()
			if err != nil {
				break
			}
			conn.SendPkg(data)
		}
	})
	go s.Run()
	defer s.Close()
	time.Sleep(100 * time.Millisecond)

	t.Run("sendPkg", func(t *testing.T) {
		conn, err := xtcp.NewPoolConn(fmt.Sprintf("127.0.0.1:%d", p))
		assert.NoError(t, err)
		defer conn.Close()
		for i := 0; i < 100; i++ {
			err := conn.SendPkg([]byte(strconv.Itoa(i)))
			assert.NoError(t, err)
		}
		//for i := 0; i < 100; i++ {
		//	err := conn.SendPkgWithTimeout([]byte(strconv.Itoa(i)), time.Second)
		//	assert.NoError(t, err)
		//}
	})

	t.Run("RecvPkg", func(t *testing.T) {
		conn, err := xtcp.NewPoolConn(fmt.Sprintf("127.0.0.1:%d", p))
		assert.NoError(t, err)
		defer conn.Close()

		for i := 0; i < 100; i++ {
			data := []byte(strconv.Itoa(i))
			result, err := conn.RecvPkg()
			assert.NoError(t, err)
			assert.Equal(t, result, data)
		}
	})
}

func Test_Pool_Basic1(t *testing.T) {
	p := portList.PopFront()
	s := xtcp.NewServer(fmt.Sprintf(`:%d`, p), func(conn *xtcp.Conn) {
		defer conn.Close()
		for {
			data, err := conn.RecvPkg()
			if err != nil {
				break
			}
			conn.SendPkg(data)
		}
	})
	go s.Run()
	defer s.Close()
	time.Sleep(100 * time.Millisecond)
	data := []byte("9999")
	t.Run("sendPkgTimeout", func(t *testing.T) {
		conn, err := xtcp.NewPoolConn(fmt.Sprintf("127.0.0.1:%d", p))
		assert.NoError(t, err)
		defer conn.Close()
		err = conn.SendPkg(data)
		assert.NoError(t, err)
		err = conn.SendPkgWithTimeout(data, time.Second)
		assert.NoError(t, err)
	})

	t.Run("recvPreConnData", func(t *testing.T) {
		conn, err := xtcp.NewPoolConn(fmt.Sprintf("127.0.0.1:%d", p))
		assert.NoError(t, err)
		defer conn.Close()
		result, err := conn.RecvPkg()
		assert.NoError(t, err)
		assert.Equal(t, result, data)
	})
}
