package raftnode

import (
	"bytes"
	"encoding/gob"
	"io"

	"github.com/AMS003010/Hyphora/internal/bitcask"

	"github.com/hashicorp/raft"
)

type FSM struct {
	store *bitcask.Bitcask
}

func NewFSM(store *bitcask.Bitcask) *FSM {
	return &FSM{store: store}
}

func (f *FSM) Apply(log *raft.Log) interface{} {
	var cmd struct {
		Op  string
		Key string
		Val []byte
	}
	if err := gob.NewDecoder(bytes.NewReader(log.Data)).Decode(&cmd); err != nil {
		return err
	}
	return f.store.ApplyCommand(cmd.Op, cmd.Key, cmd.Val)
}

func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	entries, err := f.store.Entries()
	if err != nil {
		return nil, err
	}
	return &snapshot{data: entries}, nil
}

func (f *FSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()
	dec := gob.NewDecoder(rc)
	data := make(map[string][]byte)
	if err := dec.Decode(&data); err != nil {
		return err
	}
	return f.store.RestoreFromSnapshot(data)
}

type snapshot struct {
	data map[string][]byte
}

func (s *snapshot) Persist(sink raft.SnapshotSink) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(s.data); err != nil {
		sink.Cancel()
		return err
	}
	if _, err := sink.Write(buf.Bytes()); err != nil {
		sink.Cancel()
		return err
	}
	return sink.Close()
}

func (s *snapshot) Release() {}
