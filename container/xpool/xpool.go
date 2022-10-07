package xpool

import (
	"errors"
	"github.com/motai3/xtcp/container/xlist"
	"sync/atomic"
	"time"

	"github.com/motai3/xcron"
)

type Pool struct {
	list       *xlist.List
	closed     int32
	TTL        time.Duration
	NewFunc    func() (interface{}, error)
	ExpireFunc func(interface{}) //销毁用的函数
}

type poolItem struct {
	value    interface{}
	expireAt int64
}

type NewFunc func() (interface{}, error)

type ExpireFunc func(interface{})

func New(ttl time.Duration, newFunc NewFunc, expireFunc ...ExpireFunc) *Pool {
	r := &Pool{
		list:    xlist.New(),
		TTL:     ttl,
		NewFunc: newFunc,
	}
	if len(expireFunc) > 0 {
		r.ExpireFunc = expireFunc[0]
	}
	cron := xcron.New()
	cron.AddFunc("0/1 * * * * *", r.checkExpireItems)
	go cron.Run()
	return r
}

func (p *Pool) Put(value interface{}) error {
	if p.closed == 1 {
		return errors.New("pool is closed")
	}
	item := &poolItem{
		value: value,
	}
	if p.TTL == 0 {
		item.expireAt = 0
	} else {
		item.expireAt = time.Now().UnixNano()/1e6 + p.TTL.Nanoseconds()/1000000
	}
	p.list.PushBack(item)
	return nil
}

func (p *Pool) Clear() {
	if p.ExpireFunc != nil {
		for {
			if r := p.list.PopFront(); r != nil {
				p.ExpireFunc(r.(*poolItem).value)
			} else {
				break
			}
		}
	} else {
		p.list.RemoveAll()
	}
}

func (p *Pool) Get() (interface{}, error) {
	for p.closed == 0 {
		if r := p.list.PopFront(); r != nil {
			f := r.(*poolItem)
			if f.expireAt == 0 || f.expireAt > time.Now().UnixNano()/1e6 {
				return f.value, nil
			} else if p.ExpireFunc != nil {
				p.ExpireFunc(f.value)
			}
		} else {
			break
		}
	}
	if p.NewFunc != nil {
		return p.NewFunc()
	}
	return nil, errors.New("pool and NewFunc is empty")
}

func (p *Pool) Size() int {
	return p.list.Len()
}

func (p *Pool) Close() {
	atomic.SwapInt32(&p.closed, 1)
}

func (p *Pool) checkExpireItems() {
	if p.closed == 1 {
		if p.ExpireFunc != nil {
			for {
				if r := p.list.PopFront(); r != nil {
					p.ExpireFunc(r.(*poolItem).value)
				} else {
					break
				}
			}
		}
		xcron.FuncExit()
	}

	if p.TTL == 0 {
		return
	}
	var latestExpire int64 = -1
	var timestampMilli = time.Now().UnixNano() / 1e6
	for {
		if latestExpire > timestampMilli {
			break
		}
		if r := p.list.PopFront(); r != nil {
			item := r.(*poolItem)
			latestExpire = item.expireAt
			if item.expireAt > timestampMilli {
				p.list.PushFront(item)
				break
			}
			if p.ExpireFunc != nil {
				p.ExpireFunc(item.value)
			}
		} else {
			break
		}

	}
}
