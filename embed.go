package main

import "embed"

//go:embed scripts/*
var scriptFS embed.FS
