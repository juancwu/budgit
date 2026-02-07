package assets

import "embed"

//go:embed js/* css/* fonts/* favicon/*
var AssetsFS embed.FS
