package events

import "sync"

type Event struct {
	Type  string
	Topic string
	Data  any
}

type Bus struct {
	mu   sync.RWMutex
	subs []chan Event
}

func NewBus() *Bus {
	return &Bus{}
}

func (b *Bus) Subscribe() <-chan Event {
	ch := make(chan Event, 256)
	b.mu.Lock()
	b.subs = append(b.subs, ch)
	b.mu.Unlock()
	return ch
}

func (b *Bus) Unsubscribe(ch <-chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, s := range b.subs {
		if s == ch {
			b.subs = append(b.subs[:i], b.subs[i+1:]...)
			close(s)
			return
		}
	}
}

func (b *Bus) Publish(e Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs {
		select {
		case ch <- e:
		default:
			// drop on full — slow consumer
		}
	}
}
