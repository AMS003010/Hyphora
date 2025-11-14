package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

	node, err := raftnode.NewNode(dataDir, bindAddr, raftID, httpPort)
	if err != nil {
		log.Fatalf("failed to start node: %v", err)
	}

	go startAutoCompaction(node, dataDir)

	http.HandleFunc("/put", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		if node.Raft.State() != raft.Leader {
			http.Error(w, "Only leader accepts /put", http.StatusForbidden)
			return
		}
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

	http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		val, err := node.Get(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Write(val)
	})

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

	http.HandleFunc("/addpeer", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		addr := r.URL.Query().Get("addr")
		if id == "" || addr == "" {
			http.Error(w, "id and addr required", http.StatusBadRequest)
			return
		}
		if node == nil || node.Raft == nil {
			http.Error(w, "raft not initialized", http.StatusInternalServerError)
			return
		}
		if node.Raft.State() != raft.Leader {
			http.Error(w, "not leader; send addpeer to leader", http.StatusBadRequest)
			return
		}
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

	http.HandleFunc("/replicate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if req.Path == "" {
			http.Error(w, "'path' required", http.StatusBadRequest)
			return
		}

		data, err := os.ReadFile(req.Path)
		if err != nil {
			http.Error(w, fmt.Sprintf("Cannot read file: %v", err), http.StatusBadRequest)
			return
		}

		filename := filepath.Base(req.Path)

		// === LEADER ===
		if node.Raft.State() == raft.Leader {
			if err := node.Apply("PUT", filename, data); err != nil {
				http.Error(w, "Raft apply failed: "+err.Error(), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"status": "replicated",
				"key":    filename,
				"size":   len(data),
				"from":   "leader",
			})
			return
		}

		// === FOLLOWER: forward to leader ===
		leaderAddr := node.Raft.Leader()
		if leaderAddr == "" {
			http.Error(w, "No leader", http.StatusServiceUnavailable)
			return
		}

		leaderIP := strings.Split(string(leaderAddr), ":")[0]
		leaderRaftPort := strings.Split(string(leaderAddr), ":")[1]
		port, _ := strconv.Atoi(leaderRaftPort)
		leaderHTTPPort := strconv.Itoa(port - 920)
		leaderURL := "http://" + leaderIP + ":" + leaderHTTPPort + "/put"

		fmt.Printf("===============> url: %s", leaderURL)

		payload := map[string]any{
			"key":   filename,
			"value": data,
		}
		body, _ := json.Marshal(payload)

		resp, err := http.Post(leaderURL, "application/json", bytes.NewBuffer(body))
		if err != nil {
			http.Error(w, "Failed to reach leader: "+err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})

	http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "Missing 'key' query param", http.StatusBadRequest)
			return
		}

		data, err := node.Get(key)
		if err != nil {
			http.Error(w, "File not found: "+err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", key))
		w.Write(data)
	})

	log.Printf("Hyphora node started at %s with ID %s", bindAddr, raftID)
	log.Fatal(http.ListenAndServe(":"+httpPort, nil))
}

func shouldCompact(dataDir string) (bool, error) {
	files, err := filepath.Glob(filepath.Join(dataDir, "bitcask", "data-*.db"))
	if err != nil {
		return false, fmt.Errorf("failed to list data files: %w", err)
	}

	const maxFiles = 3
	return len(files) > maxFiles, nil
}

func startAutoCompaction(node *raftnode.Node, dataDir string) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if node == nil || node.Raft == nil {
			log.Println("Auto-compaction: node or Raft not initialized")
			continue
		}
		if node.Raft.State() != raft.Leader {
			continue
		}

		needCompaction, err := shouldCompact(dataDir)
		if err != nil {
			log.Printf("Auto-compaction: failed to check compaction need: %v", err)
			continue
		}
		if !needCompaction {
			continue
		}

		log.Println("Auto-compaction: starting")
		if err := node.Store.InitiateCompaction(); err != nil {
			log.Printf("Auto-compaction: failed: %v", err)
			continue
		}
		fut := node.Raft.Barrier(5 * time.Second)
		if err := fut.Error(); err != nil {
			log.Printf("Auto-compaction: failed to ensure Raft consistency: %v", err)
			continue
		}
		log.Println("Auto-compaction: completed successfully")
	}
}
