package trash

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/davydany/ClawIDE/internal/git"
	"github.com/davydany/ClawIDE/internal/store"
)

const (
	retentionDays  = 30
	cleanInterval  = 1 * time.Hour
	initialDelay   = 1 * time.Minute
)

// Cleaner periodically removes trashed features older than 30 days.
type Cleaner struct {
	store  *store.Store
	stopCh chan struct{}
	done   chan struct{}
}

func NewCleaner(st *store.Store) *Cleaner {
	return &Cleaner{
		store:  st,
		stopCh: make(chan struct{}),
		done:   make(chan struct{}),
	}
}

func (c *Cleaner) Start() {
	go c.loop()
}

func (c *Cleaner) Stop() {
	close(c.stopCh)
	<-c.done
}

func (c *Cleaner) loop() {
	defer close(c.done)

	// Initial check after a short delay to avoid slowing startup.
	select {
	case <-time.After(initialDelay):
		c.cleanup()
	case <-c.stopCh:
		return
	}

	ticker := time.NewTicker(cleanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCh:
			return
		}
	}
}

func (c *Cleaner) cleanup() {
	cutoff := time.Now().Add(-retentionDays * 24 * time.Hour)

	// Best-effort branch deletion for expired items before removing from store.
	for _, tf := range c.store.GetTrashedFeatures() {
		if tf.TrashedAt.Before(cutoff) {
			if err := git.DeleteBranch(tf.ProjectPath, tf.Feature.BranchName); err != nil {
				log.Printf("[trash] could not delete branch %s in %s: %v", tf.Feature.BranchName, tf.ProjectPath, err)
			}
		}
	}

	count, err := c.store.DeleteExpiredTrashedFeatures(cutoff)
	if err != nil {
		log.Printf("[trash] error cleaning expired features: %v", err)
	}
	if count > 0 {
		log.Printf("[trash] permanently deleted %d expired feature(s)", count)
	}

	// Sweep expired trashed projects. The store returns the removed entries
	// so we can delete their on-disk trash directories.
	expiredProjects, err := c.store.DeleteExpiredTrashedProjects(cutoff)
	if err != nil {
		log.Printf("[trash] error cleaning expired projects: %v", err)
	}
	for _, tp := range expiredProjects {
		wrapper := filepath.Dir(tp.TrashedPath)
		if err := os.RemoveAll(wrapper); err != nil {
			log.Printf("[trash] could not delete trashed project dir %s: %v", wrapper, err)
		}
	}
	if len(expiredProjects) > 0 {
		log.Printf("[trash] permanently deleted %d expired project(s)", len(expiredProjects))
	}
}
