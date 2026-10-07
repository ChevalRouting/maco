//go:build linux

package main

func managesRedis() bool { return false }

func installNetHelper(_, _ string) error { return nil }
