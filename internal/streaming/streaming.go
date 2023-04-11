package streaming

import "sync"

type Payload struct {
	Event string
	Data  any
}

type muxContextKey string

const MuxContextKey = muxContextKey("")

type Mux struct {
	mu            sync.Mutex
	subscriptions map[*Subscription]chan<- Payload
}

func (m *Mux) Publish(event string, data any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for sub, ch := range m.subscriptions {
		select {
		case ch <- Payload{Event: event, Data: data}:
		default:
			// too slow, unsubscribe
			m.cancel(sub)
		}
	}
	return nil
}

func (m *Mux) Subscribe() *Subscription {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch := make(chan Payload, 1)
	sub := &Subscription{
		mux: m,
		C:   ch,
	}
	if m.subscriptions == nil {
		m.subscriptions = make(map[*Subscription]chan<- Payload)
	}
	m.subscriptions[sub] = ch
	return sub
}

func (m *Mux) cancel(sub *Subscription) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch, ok := m.subscriptions[sub]
	if ok {
		delete(m.subscriptions, sub)
		close(ch)
	}
}

type Subscription struct {
	mux *Mux
	// The channel to which events are received.
	C <-chan Payload
}

func (s *Subscription) Cancel() {
	s.mux.cancel(s)
}
