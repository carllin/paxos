package cli

import "testing"

func TestNewCli(t *testing.T) {
	NewCli(":9999", ":8888")
}
