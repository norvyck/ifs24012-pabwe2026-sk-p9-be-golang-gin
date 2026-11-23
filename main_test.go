package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMainCallsRunApp(t *testing.T) {
	orig := runApp
	defer func() { runApp = orig }()

	called := false
	runApp = func() { called = true }

	main()
	assert.True(t, called)
}
