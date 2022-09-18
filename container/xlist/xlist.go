package xlist

import (
	"container/list"
	"sync"
)

type (
	List struct {
		mu   *sync.RWMutex
		list *list.List
	}
	Element = list.Element
)

func New() *List {
	return &List{
		mu:   new(sync.RWMutex),
		list: list.New(),
	}
}

func (l *List) PushFront(v interface{}) (e *Element) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
	}
	e = l.list.PushFront(v)
	return
}

func (l *List) PushBack(v interface{}) (e *Element) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
	}
	e = l.list.PushBack(v)
	return
}

func (l *List) PopBack() (value interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
		return
	}
	if e := l.list.Back(); e != nil {
		value = l.list.Remove(e)
	}
	return
}

func (l *List) PopFront() (value interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
		return
	}
	if e := l.list.Front(); e != nil {
		value = l.list.Remove(e)
	}
	return
}

func (l *List) RemoveAll() {
	l.mu.Lock()
	l.list = list.New()
	l.mu.Unlock()
}

func (l *List) Len() (length int) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.list == nil {
		return
	}
	length = l.list.Len()
	return
}
