package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"nhooyr.io/websocket"

	"github.com/nathanmauro/agent-observatory/internal/events"
	"github.com/nathanmauro/agent-observatory/internal/models"
)

const (
	replaySize    = 1000
	heartbeatFreq = 30 * time.Second
	writeTimeout  = 5 * time.Second
)

type Broker struct {
	bus  *events.Bus
	seq  atomic.Uint64
	mu   sync.RWMutex
	ring []models.WSMessage
	head int

	clientsMu sync.Mutex
	clients   map[*client]struct{}

	done chan struct{}
}

type client struct {
	conn   *websocket.Conn
	send   chan []byte
	topics map[string]bool
	mu     sync.Mutex
}

func NewBroker(bus *events.Bus) *Broker {
	return &Broker{
		bus:     bus,
		ring:    make([]models.WSMessage, replaySize),
		clients: make(map[*client]struct{}),
		done:    make(chan struct{}),
	}
}

func (b *Broker) Run() {
	ch := b.bus.Subscribe()
	heartbeat := time.NewTicker(heartbeatFreq)
	defer heartbeat.Stop()

	for {
		select {
		case <-b.done:
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			b.broadcast(ev)
		case <-heartbeat.C:
			b.broadcastHeartbeat()
		}
	}
}

func (b *Broker) Stop() {
	close(b.done)
	b.clientsMu.Lock()
	for c := range b.clients {
		close(c.send)
	}
	b.clientsMu.Unlock()
}

func (b *Broker) broadcast(ev events.Event) {
	seq := b.seq.Add(1)
	msg := models.WSMessage{
		Seq:           seq,
		SchemaVersion: 1,
		Type:          ev.Type,
		Topic:         ev.Topic,
		SentAt:        time.Now().UTC().Format(time.RFC3339Nano),
		Data:          ev.Data,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("ws marshal: %v", err)
		return
	}

	b.mu.Lock()
	b.ring[b.head%replaySize] = msg
	b.head++
	b.mu.Unlock()

	b.clientsMu.Lock()
	for c := range b.clients {
		c.mu.Lock()
		if c.topics == nil || c.topics[ev.Topic] {
			select {
			case c.send <- data:
			default:
			}
		}
		c.mu.Unlock()
	}
	b.clientsMu.Unlock()
}

func (b *Broker) broadcastHeartbeat() {
	seq := b.seq.Add(1)
	msg := models.WSMessage{
		Seq:           seq,
		SchemaVersion: 1,
		Type:          "heartbeat",
		Topic:         "",
		SentAt:        time.Now().UTC().Format(time.RFC3339Nano),
	}
	data, _ := json.Marshal(msg)

	b.clientsMu.Lock()
	for c := range b.clients {
		select {
		case c.send <- data:
		default:
		}
	}
	b.clientsMu.Unlock()
}

func (b *Broker) replay(lastSeq uint64) [][]byte {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var msgs [][]byte
	start := b.head - replaySize
	if start < 0 {
		start = 0
	}
	for i := start; i < b.head; i++ {
		m := b.ring[i%replaySize]
		if m.Seq > lastSeq {
			data, err := json.Marshal(m)
			if err == nil {
				msgs = append(msgs, data)
			}
		}
	}
	return msgs
}

func (b *Broker) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}

	c := &client{
		conn: conn,
		send: make(chan []byte, 128),
	}

	b.clientsMu.Lock()
	b.clients[c] = struct{}{}
	b.clientsMu.Unlock()

	defer func() {
		b.clientsMu.Lock()
		delete(b.clients, c)
		b.clientsMu.Unlock()
		conn.Close(websocket.StatusNormalClosure, "")
	}()

	ctx := r.Context()

	go b.writePump(ctx, c)
	b.readPump(ctx, c)
}

func (b *Broker) writePump(ctx context.Context, c *client) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			wctx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := c.conn.Write(wctx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func (b *Broker) readPump(ctx context.Context, c *client) {
	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			return
		}

		var msg models.WSClientMessage
		if json.Unmarshal(data, &msg) != nil {
			continue
		}

		switch msg.Type {
		case "hello":
			helloMsg := models.WSMessage{
				Seq:           b.seq.Load(),
				SchemaVersion: 1,
				Type:          "hello",
				SentAt:        time.Now().UTC().Format(time.RFC3339Nano),
			}
			helloData, _ := json.Marshal(helloMsg)
			select {
			case c.send <- helloData:
			default:
			}

			if msg.LastSeq > 0 {
				for _, m := range b.replay(msg.LastSeq) {
					select {
					case c.send <- m:
					default:
					}
				}
			}

		case "subscribe":
			c.mu.Lock()
			if c.topics == nil {
				c.topics = make(map[string]bool)
			}
			for _, t := range msg.Topics {
				c.topics[t] = true
			}
			c.mu.Unlock()

		case "unsubscribe":
			c.mu.Lock()
			for _, t := range msg.Topics {
				delete(c.topics, t)
			}
			c.mu.Unlock()
		}
	}
}
