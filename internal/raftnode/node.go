package raftnode

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/AMS003010/Hyphora/internal/bitcask"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

type Node struct {
	Raft  *raft.Raft
	Store *bitcask.Bitcask
}

func NewNode(dataDir string, bindAddr string, raftID string) (*Node, error) {
	// Register Raft command struct
	gob.Register(struct {
		Op  string
		Key string
		Val []byte
	}{})

	// Setup directories
	raftDir := filepath.Join(dataDir, "raft")
	if err := os.MkdirAll(raftDir, 0755); err != nil {
		return nil, err
	}

	// Open Bitcask
	Store, err := bitcask.Open(filepath.Join(dataDir, "bitcask"))
	if err != nil {
		return nil, err
	}

	// Raft config
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(raftID)

	// Raft communication
	addr, err := raft.NewTCPTransport(bindAddr, nil, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, err
	}

	// Stable Store (BoltDB)
	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "stable.db"))
	if err != nil {
		return nil, err
	}

	// Log Store (BoltDB)
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft-log.db"))
	if err != nil {
		return nil, err
	}

	// Snapshot Store
	snapshots, err := raft.NewFileSnapshotStore(raftDir, 1, os.Stderr)
	if err != nil {
		return nil, err
	}

	// FSM
	fsm := NewFSM(Store)

	// Raft instance
	r, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshots, addr)
	if err != nil {
		return nil, err
	}

	node := &Node{
		Raft:  r,
		Store: Store,
	}

	// Check if Raft has any existing configuration
	hasState, err := raft.HasExistingState(logStore, stableStore, snapshots)
	if err != nil {
		return nil, fmt.Errorf("checking existing raft state: %w", err)
	}

	// If no state, bootstrap the cluster with this node
	if !hasState {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      config.LocalID,
					Address: addr.LocalAddr(),
				},
			},
		}
		f := r.BootstrapCluster(configuration)
		if f.Error() != nil {
			return nil, fmt.Errorf("failed to bootstrap cluster: %w", f.Error())
		}
	}

	return node, nil
}

func (n *Node) Apply(op, key string, val []byte) error {
	var buf bytes.Buffer
	cmd := struct {
		Op  string
		Key string
		Val []byte
	}{op, key, val}

	if err := gob.NewEncoder(&buf).Encode(cmd); err != nil {
		return err
	}

	f := n.Raft.Apply(buf.Bytes(), 5*time.Second)
	return f.Error()
}

func (n *Node) Get(key string) ([]byte, error) {
	return n.Store.Get(key)
}
