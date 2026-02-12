package sse

import (
	"sync"
	"testing"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestHub_SubscribeAndBroadcast(t *testing.T) {
	hub := NewHub()

	ch := hub.Subscribe("client-1")

	n := &model.Notification{
		ID:    "n1",
		Title: "Test",
	}

	hub.Broadcast(n)

	select {
	case got := <-ch:
		assert.Equal(t, "n1", got.ID)
		assert.Equal(t, "Test", got.Title)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast")
	}
}

func TestHub_Unsubscribe(t *testing.T) {
	hub := NewHub()

	ch := hub.Subscribe("client-1")
	hub.Unsubscribe("client-1")

	// Channel should be closed
	_, ok := <-ch
	assert.False(t, ok, "channel should be closed after unsubscribe")
}

func TestHub_MultipleClients(t *testing.T) {
	hub := NewHub()

	ch1 := hub.Subscribe("client-1")
	ch2 := hub.Subscribe("client-2")

	n := &model.Notification{ID: "n1", Title: "Multi"}
	hub.Broadcast(n)

	got1 := <-ch1
	got2 := <-ch2
	assert.Equal(t, "n1", got1.ID)
	assert.Equal(t, "n1", got2.ID)
}

func TestHub_SlowConsumerDropsMessage(t *testing.T) {
	hub := NewHub()

	ch := hub.Subscribe("slow-client")

	// Fill the buffer (capacity 50)
	for i := 0; i < 60; i++ {
		hub.Broadcast(&model.Notification{ID: "n"})
	}

	// Should have received up to buffer capacity
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	assert.Equal(t, 50, count, "should receive exactly buffer capacity messages")
}

func TestHub_ConcurrentSubscribeUnsubscribe(t *testing.T) {
	hub := NewHub()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			clientID := "client-" + string(rune('A'+id%26))
			hub.Subscribe(clientID)
			hub.Broadcast(&model.Notification{ID: "n"})
			hub.Unsubscribe(clientID)
		}(i)
	}
	wg.Wait()
}

func TestHub_UnsubscribeNonexistent(t *testing.T) {
	hub := NewHub()
	// Should not panic
	hub.Unsubscribe("nonexistent")
}

func TestHub_BroadcastNoClients(t *testing.T) {
	hub := NewHub()
	// Should not panic
	hub.Broadcast(&model.Notification{ID: "n1"})
}
