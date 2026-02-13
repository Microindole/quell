package main

import "embed"

//go:embed scripts/*
var scriptFS embed.FS

//go:embed all:web
var webFS embed.FS
