package acceptor

import (
	"github.com/pears-to-peers/utils"
	"testing"
)

func TestNewAcceptor(t *testing.T) {
	cluster := make([]utils.NodeInfo, 0, 5)
	NewAcceptor(cluster)
}
