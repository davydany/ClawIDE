package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tempNotifStore(t *testing.T, max int) *NotificationStore {
	t.Helper()
	dir := t.TempDir()
	fp := filepath.Join(dir, "notifications.json")
	s, err := NewNotificationStore(fp, max)
	require.NoError(t, err)
	return s
}

func makeNotif(id, title string) model.Notification {
	return model.Notification{
		ID:        id,
		Title:     title,
		Source:    "test",
		Level:     "info",
		CreatedAt: time.Now(),
	}
}

func TestNotificationStore_AddAndGet(t *testing.T) {
	s := tempNotifStore(t, 200)

	n := makeNotif("n1", "Test Notification")
	require.NoError(t, s.Add(n))

	got, ok := s.Get("n1")
	assert.True(t, ok)
	assert.Equal(t, "Test Notification", got.Title)
	assert.False(t, got.Read)
}

func TestNotificationStore_GetAll(t *testing.T) {
	s := tempNotifStore(t, 200)

	require.NoError(t, s.Add(makeNotif("n1", "First")))
	require.NoError(t, s.Add(makeNotif("n2", "Second")))

	all := s.GetAll()
	assert.Len(t, all, 2)
	// Newest first (prepend)
	assert.Equal(t, "n2", all[0].ID)
	assert.Equal(t, "n1", all[1].ID)
}

func TestNotificationStore_GetUnread(t *testing.T) {
	s := tempNotifStore(t, 200)

	require.NoError(t, s.Add(makeNotif("n1", "First")))
	require.NoError(t, s.Add(makeNotif("n2", "Second")))
	require.NoError(t, s.MarkRead("n1"))

	unread := s.GetUnread()
	assert.Len(t, unread, 1)
	assert.Equal(t, "n2", unread[0].ID)
}

func TestNotificationStore_UnreadCount(t *testing.T) {
	s := tempNotifStore(t, 200)

	require.NoError(t, s.Add(makeNotif("n1", "First")))
	require.NoError(t, s.Add(makeNotif("n2", "Second")))
	assert.Equal(t, 2, s.UnreadCount())

	require.NoError(t, s.MarkRead("n1"))
	assert.Equal(t, 1, s.UnreadCount())
}

func TestNotificationStore_MarkAllRead(t *testing.T) {
	s := tempNotifStore(t, 200)

	require.NoError(t, s.Add(makeNotif("n1", "First")))
	require.NoError(t, s.Add(makeNotif("n2", "Second")))
	require.NoError(t, s.MarkAllRead())

	assert.Equal(t, 0, s.UnreadCount())
	all := s.GetAll()
	for _, n := range all {
		assert.True(t, n.Read)
	}
}

func TestNotificationStore_Delete(t *testing.T) {
	s := tempNotifStore(t, 200)

	require.NoError(t, s.Add(makeNotif("n1", "First")))
	require.NoError(t, s.Add(makeNotif("n2", "Second")))
	require.NoError(t, s.Delete("n1"))

	all := s.GetAll()
	assert.Len(t, all, 1)
	assert.Equal(t, "n2", all[0].ID)
}

func TestNotificationStore_DeleteNotFound(t *testing.T) {
	s := tempNotifStore(t, 200)
	err := s.Delete("nonexistent")
	assert.Error(t, err)
}

func TestNotificationStore_AutoPrune(t *testing.T) {
	s := tempNotifStore(t, 5)

	for i := 0; i < 8; i++ {
		require.NoError(t, s.Add(makeNotif(
			"n"+string(rune('0'+i)),
			"Notification",
		)))
	}

	all := s.GetAll()
	assert.Len(t, all, 5, "should be pruned to max 5")
}

func TestNotificationStore_IdempotencyDedup(t *testing.T) {
	s := tempNotifStore(t, 200)

	n1 := makeNotif("n1", "First")
	n1.IdempotencyKey = "key-abc"
	require.NoError(t, s.Add(n1))

	n2 := makeNotif("n2", "Duplicate")
	n2.IdempotencyKey = "key-abc"
	require.NoError(t, s.Add(n2))

	all := s.GetAll()
	assert.Len(t, all, 1, "duplicate idempotency key should be skipped")
	assert.Equal(t, "n1", all[0].ID)
}

func TestNotificationStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "notifications.json")

	// Write some data
	s1, err := NewNotificationStore(fp, 200)
	require.NoError(t, err)
	require.NoError(t, s1.Add(makeNotif("n1", "Persisted")))

	// Reload from disk
	s2, err := NewNotificationStore(fp, 200)
	require.NoError(t, err)

	all := s2.GetAll()
	assert.Len(t, all, 1)
	assert.Equal(t, "Persisted", all[0].Title)
}

func TestNotificationStore_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "notifications.json")
	// File doesn't exist - should initialize empty
	s, err := NewNotificationStore(fp, 200)
	require.NoError(t, err)
	assert.Len(t, s.GetAll(), 0)
}

func TestNotificationStore_MarkReadNotFound(t *testing.T) {
	s := tempNotifStore(t, 200)
	err := s.MarkRead("nonexistent")
	assert.Error(t, err)
}

func TestNotificationStore_GetNotFound(t *testing.T) {
	s := tempNotifStore(t, 200)
	_, ok := s.Get("nonexistent")
	assert.False(t, ok)
}

func TestNotificationStore_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "notifications.json")
	// Write corrupt data
	os.WriteFile(fp, []byte("not json"), 0644)
	_, err := NewNotificationStore(fp, 200)
	assert.Error(t, err)
}
