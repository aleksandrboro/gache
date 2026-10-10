package pubsub

import (
	"sync"
)

type Hub struct {
	channels map[string]map[*Subscriber]struct{}
	mu       sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		channels: make(map[string]map[*Subscriber]struct{}),
	}
}

func (h *Hub) Subscribe(sub *Subscriber, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.channels[channel]; !ok {
		h.channels[channel] = make(map[*Subscriber]struct{})
	}

	h.channels[channel][sub] = struct{}{}
}

func (h *Hub) Unsubscribe(sub *Subscriber, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.channels[channel]; !ok {
		return
	}

	delete(h.channels[channel], sub)

	if len(h.channels[channel]) == 0 {
		delete(h.channels, channel)
	}
}

func (h *Hub) Publish(channel, message string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	ch, ok := h.channels[channel]
	if !ok {
		return 0
	}

	count := 0

	for sub := range ch {
		go func(s *Subscriber) {
			select {
			case s.MsgChan <- Message{
				Type:    "message",
				Channel: channel,
				Data:    message,
			}:
			case <-s.Done:
			}

		}(sub)
		count++
	}

	return count
}

func (h *Hub) UnsubscribeAll(sub *Subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for ch, subs := range h.channels {
		delete(subs, sub)

		if len(subs) == 0 {
			delete(h.channels, ch)
		}
	}
}
