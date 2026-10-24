// Package assets embeds the runtime data recovered from the supported disk.
package assets

import "embed"

// Original contains the checked title, map, graphics and custom audio driver.
//
//go:embed original/*
var Original embed.FS
