package pty

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRingBuffer(t *testing.T) {
	rb := NewRingBuffer(1024)
	assert.NotNil(t, rb)
	assert.Equal(t, 1024, rb.size)
	assert.Equal(t, 0, rb.pos)
	assert.False(t, rb.full)
}

func TestRingBufferWriteAndBytes(t *testing.T) {
	t.Run("write less than size", func(t *testing.T) {
		rb := NewRingBuffer(64)
		n, err := rb.Write([]byte("hello"))
		require.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte("hello"), rb.Bytes())
	})

	t.Run("write exact size", func(t *testing.T) {
		rb := NewRingBuffer(5)
		n, err := rb.Write([]byte("abcde"))
		require.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte("abcde"), rb.Bytes())
	})

	t.Run("overflow wrap-around", func(t *testing.T) {
		rb := NewRingBuffer(5)
		rb.Write([]byte("abcde"))
		rb.Write([]byte("fg"))

		// After wrapping: buffer is [f, g, c, d, e] with pos=2
		// Bytes() should return from pos to end + start to pos: [c, d, e, f, g]
		got := rb.Bytes()
		assert.Equal(t, []byte("cdefg"), got)
	})

	t.Run("multiple writes", func(t *testing.T) {
		rb := NewRingBuffer(10)
		rb.Write([]byte("abc"))
		rb.Write([]byte("def"))
		assert.Equal(t, []byte("abcdef"), rb.Bytes())
	})

	t.Run("empty buffer", func(t *testing.T) {
		rb := NewRingBuffer(10)
		assert.Equal(t, []byte{}, rb.Bytes())
	})
}

func TestNewSession(t *testing.T) {
	sess := NewSession("pane-1", "/home/user", "tmux", []string{"-A", "-s", "test"}, 4096, nil)

	assert.Equal(t, "pane-1", sess.ID)
	assert.Equal(t, "/home/user", sess.WorkDir)
	assert.Equal(t, "tmux", sess.Command)
	assert.Equal(t, []string{"-A", "-s", "test"}, sess.Args)
	assert.NotNil(t, sess.clients)
	assert.NotNil(t, sess.scrollback)
	assert.NotNil(t, sess.done)
	assert.False(t, sess.closed)
}

func TestSubscribeAndUnsubscribe(t *testing.T) {
	sess := NewSession("pane-1", "/home/user", "tmux", []string{}, 4096, nil)

	// Write some data to scrollback first
	sess.scrollback.Write([]byte("initial data"))

	// Subscribe
	ch, history := sess.Subscribe("client-1")
	assert.NotNil(t, ch)
	assert.Equal(t, []byte("initial data"), history)

	// Verify client is registered
	sess.mu.RLock()
	_, exists := sess.clients["client-1"]
	sess.mu.RUnlock()
	assert.True(t, exists)

	// Unsubscribe
	sess.Unsubscribe("client-1")

	sess.mu.RLock()
	_, exists = sess.clients["client-1"]
	sess.mu.RUnlock()
	assert.False(t, exists)
}

func TestSubscribeMultipleClients(t *testing.T) {
	sess := NewSession("pane-1", "/home/user", "tmux", []string{}, 4096, nil)

	ch1, _ := sess.Subscribe("client-1")
	ch2, _ := sess.Subscribe("client-2")

	assert.NotNil(t, ch1)
	assert.NotNil(t, ch2)

	sess.mu.RLock()
	assert.Len(t, sess.clients, 2)
	sess.mu.RUnlock()

	sess.Unsubscribe("client-1")

	sess.mu.RLock()
	assert.Len(t, sess.clients, 1)
	sess.mu.RUnlock()
}
