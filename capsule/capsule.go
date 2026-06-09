// Package capsule is the public integration surface of the mykeep memory component.
//
// Standalone, capsule ships as its own binary (cmd/mykeep) that prompts for a
// password and derives its own key. To compose capsule into the mykeep *suite* (one
// binary, one unlock for several components), the suite aggregator lives in a separate
// Go module and therefore cannot import capsule's internal/ packages. This package is
// the thin, stable bridge: it lives inside the mykeep.ai module (so it may reach
// internal/), and exposes only stdlib + []byte across the boundary.
//
// The contract is duck-typed — the aggregator declares the interface it needs and a
// *Component satisfies it structurally; nothing here imports the aggregator.
package capsule

import (
	"context"
	"net/http"

	"mykeep.ai/internal/app"
	"mykeep.ai/internal/paths"
	"mykeep.ai/internal/server"
)

// ID is capsule's stable identifier within the suite (route namespace, logs, GUI tab).
const ID = "capsule"

// Options is everything the host (standalone cmd or the suite aggregator) supplies.
type Options struct {
	DataDir  string // resolved data directory; capsule's files live directly inside it
	Portable bool   // from paths.Layout.Portable (surfaced in /v1/health)
	Version  string // host binary version
	Token    string // optional bearer for /v1/* ("" = no token required)
}

// Component is the capsule capability. Construct it LOCKED with New; Unlock activates
// it with an injected key; Mount attaches its routes; Lock tears it down.
type Component struct {
	opts   Options
	layout paths.Layout
	rt     *app.Runtime
	srv    *server.Server
}

// New builds a locked component bound to a data dir. Cheap: no crypto, no secret I/O.
func New(opts Options) (*Component, error) {
	return &Component{
		opts:   opts,
		layout: paths.Layout{DataDir: opts.DataDir, Portable: opts.Portable},
	}, nil
}

// ID returns the stable component identifier.
func (c *Component) ID() string { return ID }

// FirstLaunch reports whether capsule has never been initialized in this data dir.
func (c *Component) FirstLaunch() bool { return c.layout.IsFirstLaunch() }

// Unlock activates capsule with an externally supplied 32-byte DEK. capsule uses the
// DEK directly to open (or, on first launch, create) its encrypted store; it does NOT
// run its own argon2id. The dek slice is adopted by the component's key store and
// wiped on Lock — the caller must not reuse or wipe it afterward.
func (c *Component) Unlock(ctx context.Context, dek []byte) error {
	rt, err := app.OpenWithDEK(ctx, c.layout, dek, c.layout.IsFirstLaunch(), c.opts.Version)
	if err != nil {
		return err
	}
	c.rt = rt
	c.srv = server.New(rt.Config, rt.Store, rt.Ingest, rt.Recall,
		c.opts.Version, rt.EmbedderName(), c.layout.Portable, c.opts.Token)
	return nil
}

// Mount attaches the memory API onto a shared mux. capsule owns the /v1/ namespace;
// components with a more specific prefix (e.g. the vault's /v1/vault/) win by Go's
// longest-prefix matching, so they coexist on one mux without collision.
func (c *Component) Mount(mux *http.ServeMux) {
	if c.srv == nil {
		return
	}
	mux.Handle("/v1/", c.srv.Handler())
}

// Lock flushes + reseals the store and zeroizes the key. Idempotent.
func (c *Component) Lock() error {
	if c.rt == nil {
		return nil
	}
	err := c.rt.Close()
	c.rt, c.srv = nil, nil
	return err
}
