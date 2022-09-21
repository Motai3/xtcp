package xtcp_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
	"xtcp"
	"xtcp/container/xlist"
)

var portList = xlist.New()

func init() {
	for i := 9000; i < 10000; i++ {
		portList.PushBack(i)
	}

}

func Test_Package_Basic(t *testing.T) {
	p := portList.PopFront().(int)

	server := xtcp.NewServer(fmt.Sprintf(`:%d`, p), func(conn *xtcp.Conn) {
		defer conn.Close()
		for {
			data, err := conn.RecvPkg()
			if err != nil {
				break
			}
			conn.SendPkg(data)
		}
	})
	go server.Run()
	defer server.Close()
	time.Sleep(100 * time.Millisecond)

	t.Run("BigPackage", func(t *testing.T) {
		conn, err := xtcp.NewConn(fmt.Sprintf("127.0.0.1:%d", p))
		assert.NoError(t, err)
		defer conn.Close()
		data := make([]byte, 65536)
		err = conn.SendPkg(data)
		assert.Error(t, err, "消息太大")
	})

	t.Run("SendRecvPkg", func(t *testing.T) {
		conn, err := xtcp.NewConn(fmt.Sprintf("127.0.0.1:%d", p))
		assert.NoError(t, err, "连接错误")
		defer conn.Close()
		for i := 100; i < 200; i++ {
			data := []byte(strconv.Itoa(i))
			result, err := conn.SendRecvPkg(data)
			assert.NoError(t, err)
			assert.Equal(t, result, data)
		}
	})

	t.Run("sendRecePkgBig", func(t *testing.T) {
		conn, err := xtcp.NewConn(fmt.Sprintf("127.0.0.1:%d", p))
		assert.NoError(t, err, "连接错误")
		defer conn.Close()
		data := make([]byte, 65536)
		result, err := conn.SendRecvPkg(data)
		assert.Error(t, err)
		assert.Empty(t, result)
	})
}

func Test_Package_Timeout(t *testing.T) {
	p := portList.PopFront().(int)

	server := xtcp.NewServer(fmt.Sprintf(`:%d`, p), func(conn *xtcp.Conn) {
		defer conn.Close()
		for {
			data, err := conn.RecvPkg()
			if err != nil {
				break
			}
			time.Sleep(time.Second)
			conn.SendPkg(data)
		}
	})
	go server.Run()
	defer server.Close()
	time.Sleep(100 * time.Millisecond)

	t.Run("SendRecvTimeout", func(t *testing.T) {
		conn, err := xtcp.NewConn(fmt.Sprintf("127.0.0.1:%d", p))
		assert.NoError(t, err)
		defer conn.Close()
		data := []byte("10000")
		result, err := conn.SendRecvPkgWithTimeout(data, time.Millisecond)
		assert.Error(t, err)
		assert.Empty(t, result)
	})

	t.Run("SendRecvNotTimeout", func(t *testing.T) {
		conn, err := xtcp.NewConn(fmt.Sprintf("127.0.0.1:%d", p))
		assert.NoError(t, err)
		defer conn.Close()
		data := []byte("10000")
		result, err := conn.SendRecvPkgWithTimeout(data, time.Millisecond*2000)
		assert.NoError(t, err)
		assert.Equal(t, result, data)
	})
}

func Test_Package_Option(t *testing.T) {
	p := portList.PopFront().(int)

	server := xtcp.NewServer(fmt.Sprintf(`:%d`, p), func(conn *xtcp.Conn) {
		defer conn.Close()
		for {
			data, err := conn.RecvPkg()
			if err != nil {
				break
			}
			err = conn.SendPkg(data)
			assert.NoError(t, err)
		}
	})
	go server.Run()
	defer server.Close()
	time.Sleep(100 * time.Millisecond)

	t.Run("PackOptionOver", func(t *testing.T) {
		conn, err := xtcp.NewConn(fmt.Sprintf("127.0.0.1:%d", p))
		assert.NoError(t, err, "连接错误")
		defer conn.Close()
		data := make([]byte, 0xFF+1)
		result, err := conn.SendRecvPkg(data, xtcp.PkgOption{HeaderSize: 1})
		assert.Error(t, err)
		assert.Empty(t, result)
	})

	//t.Run("PackOption", func(t *testing.T) {
	//	conn, err := xtcp.NewConn(fmt.Sprintf("127.0.0.1:%d", p))
	//	assert.NoError(t, err, "连接错误")
	//	defer conn.Close()
	//	data := make([]byte, 0xFF)
	//	data[100] = byte(65)
	//	data[200] = byte(85)
	//	result, err := conn.SendRecvPkg(data, xtcp.PkgOption{HeaderSize: 1})
	//	assert.NoError(t, err)
	//	assert.Equal(t, result, data)
	//})
}
