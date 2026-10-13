package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/2025STELLU/MagnetarWiki/internal/pgbundle"
	"github.com/2025STELLU/MagnetarWiki/internal/version"
)

func printVersion() error {
	fmt.Printf("magnetar %s\n", version.String())
	if commit := version.Commit(); commit != "" {
		fmt.Printf("commit      %s\n", commit)
	}
	if at := version.BuiltFrom(); !at.IsZero() {
		fmt.Printf("committed   %s\n", at.UTC().Format(time.RFC3339))
	}
	fmt.Printf("platform    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	if pgbundle.Embedded() {
		fmt.Printf("postgresql  %s, bundled\n", pgbundle.Version)
	} else {
		fmt.Println("postgresql  not bundled")
	}
	fmt.Printf("go          %s\n", runtime.Version())
	return nil
}
