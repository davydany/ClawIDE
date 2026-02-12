package handler

import (
	"github.com/davydany/ClawIDE/internal/config"
	ptyPkg "github.com/davydany/ClawIDE/internal/pty"
	"github.com/davydany/ClawIDE/internal/sse"
	"github.com/davydany/ClawIDE/internal/store"
	"github.com/davydany/ClawIDE/internal/tmpl"
)

type Handlers struct {
	cfg               *config.Config
	store             *store.Store
	renderer          *tmpl.Renderer
	ptyManager        *ptyPkg.Manager
	snippetStore      *store.SnippetStore
	notificationStore *store.NotificationStore
	noteStore         *store.NoteStore
	sseHub            *sse.Hub
}

func New(cfg *config.Config, st *store.Store, renderer *tmpl.Renderer, ptyMgr *ptyPkg.Manager, snippetSt *store.SnippetStore, notifSt *store.NotificationStore, noteSt *store.NoteStore, hub *sse.Hub) *Handlers {
	return &Handlers{
		cfg:               cfg,
		store:             st,
		renderer:          renderer,
		ptyManager:        ptyMgr,
		snippetStore:      snippetSt,
		notificationStore: notifSt,
		noteStore:         noteSt,
		sseHub:            hub,
	}
}
