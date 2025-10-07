package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/AMS003010/Hyphora/internal/raftnode"
	"github.com/hashicorp/raft"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: hyphora-node <dataDir> <raftAddr> <nodeID> <httpPort>")
		os.Exit(1)
	}

	dataDir := os.Args[1]
	bindAddr := os.Args[2]
	raftID := os.Args[3]
	httpPort := os.Args[4]

	node, err := raftnode.NewNode(dataDir, bindAddr, raftID)
	if err != nil {
		log.Fatalf("failed to start node: %v", err)
	}

	// PUT handler
	http.HandleFunc("/put", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := node.Apply("PUT", req.Key, []byte(req.Value)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// GET handler
	http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		val, err := node.Get(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Write(val)
	})

	// DEL handler
	http.HandleFunc("/del", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Key string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := node.Apply("DEL", req.Key, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// ADDPEER handler
	http.HandleFunc("/addpeer", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		addr := r.URL.Query().Get("addr")
		if id == "" || addr == "" {
			http.Error(w, "id and addr required", http.StatusBadRequest)
			return
		}

		// Ensure we have a raft instance
		if node == nil || node.Raft == nil {
			http.Error(w, "raft not initialized", http.StatusInternalServerError)
			return
		}

		// Optionally ensure this node is the leader before adding a voter
		if node.Raft.State() != raft.Leader {
			http.Error(w, "not leader; send addpeer to leader", http.StatusBadRequest)
			return
		}

		// Add the peer (wait for completion)
		fut := node.Raft.AddVoter(raft.ServerID(id), raft.ServerAddress(addr), 0, time.Second*0)
		if err := fut.Error(); err != nil {
			http.Error(w, "failed to add peer: "+err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Peer %s (%s) added successfully\n", id, addr)
	})

	http.HandleFunc("/compact", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if node == nil || node.Raft == nil {
			http.Error(w, "raft not initialized", http.StatusInternalServerError)
			return
		}
		if node.Raft.State() != raft.Leader {
			http.Error(w, "not leader; send compact to leader", http.StatusBadRequest)
			return
		}
		if err := node.Store.InitiateCompaction(); err != nil {
			http.Error(w, fmt.Sprintf("Compaction failed: %v", err), http.StatusInternalServerError)
			return
		}
		fut := node.Raft.Barrier(5 * time.Second)
		if err := fut.Error(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to ensure Raft consistency post-compaction: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Compaction completed")
	})

	log.Printf("Hyphora node started at %s with ID %s", bindAddr, raftID)
	log.Fatal(http.ListenAndServe(":"+httpPort, nil))
}
