package main

import (
	"github.com/2025STELLU/MagnetarWiki/internal/renderer"
	"github.com/2025STELLU/MagnetarWiki/internal/renderer/sidecar"
)

func newSidecarEngine(path string) (renderer.Renderer, func(), error) {
	r, err := sidecar.New(path)
	if err != nil {
		return nil, nil, err
	}
	return r, func() { r.Close() }, nil
}
