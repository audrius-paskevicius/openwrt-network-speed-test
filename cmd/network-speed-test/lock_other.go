//go:build !linux

package main

import "errors"

func lockFile(string) (func(), error) { return nil, errors.New("Linux is required") }
