package assets

import "embed"

//go:embed js/* css/* fonts/*
var AssetsFS embed.FS
