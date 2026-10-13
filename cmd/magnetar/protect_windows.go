//go:build windows

package main

import (
	"github.com/2025STELLU/MagnetarWiki/internal/paths"
	"github.com/2025STELLU/MagnetarWiki/internal/service"
)

func protectInstance(*paths.Paths, string) string { return "" }

func releaseInstance(string, string, int, int) {}

func settingsFileOpenTo(string) string { return "" }

func prepareInstall(*service.Spec) error { return nil }

func forgetInstance(string) {}

func unpackPostgres([]string) error { return nil }

func prepareStart(string) {}

func exposure(string) string { return "" }

func pinnedMirror(*paths.Paths, string) bool { return false }

func configLinkTrusted(string) error { return nil }
