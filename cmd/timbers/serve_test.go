package main

import "testing"

// TestNewServeCmd verifies the serve command wires up correctly.
func TestNewServeCmd(t *testing.T) {
	cmd := newServeCmd()

	if cmd.Use != "serve" {
		t.Errorf("Use = %q, want %q", cmd.Use, "serve")
	}
	if cmd.RunE == nil {
		t.Error("RunE is nil")
	}
}
