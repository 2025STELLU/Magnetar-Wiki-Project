//go:build cgo && !nocgo

package main

import (
	"github.com/2025STELLU/MagnetarWiki/internal/renderer"
	ftml "github.com/2025STELLU/MagnetarWiki/internal/renderer/cgo"
)

func newRenderer(sidecarPath string) (renderer.Renderer, func(), error) {
	if sidecarPath != "" {
		return newSidecarEngine(sidecarPath)
	}
	return ftml.New(), func() {}, nil
}
