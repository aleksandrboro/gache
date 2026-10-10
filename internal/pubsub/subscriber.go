package pubsub

import "sync"

type Subscriber struct {
	MsgChan  chan Message
	Done     chan struct{}
	Channels map[string]struct{}
	Mu       sync.Mutex
}

type Message struct {
	Type    string
	Channel string
	Data    string
}

func NewSubscriber() *Subscriber {
	return &Subscriber{
		MsgChan:  make(chan Message, 256),
		Done:     make(chan struct{}),
		Channels: make(map[string]struct{}),
	}
}
