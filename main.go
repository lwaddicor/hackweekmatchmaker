package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	matchSize = 2
)

var (
	matches    = make(map[string]MatchInfo)
	matchesMtx sync.Mutex

	unmatchedPlayers    []PlayerInfo
	unmatchedPlayersMtx sync.Mutex

	playerAlloc     = make(map[string]string)
	playerAllocsMtx sync.Mutex
)

type PlayerInfo struct {
	PlayerUUID string
	ip         string
}

type MatchInfo struct {
	MatchedPlayers bool
	AllocationUUID string       `json:",omitempty"`
	Players        []PlayerInfo `json:",omitempty"`
	AllocationIP   string       `json:",omitempty"`
	Aborted        bool         `json:",omitempty"`
}

type endMatchRequest struct {
	AllocationUUID string
}

func handlePlayer(w http.ResponseWriter, r *http.Request) {
	var pl PlayerInfo
	json.NewDecoder(r.Body).Decode(&pl)
	pl.ip = r.RemoteAddr

	if pl.PlayerUUID == "" {
		http.Error(w, "missing player uuid", http.StatusBadRequest)
		return
	}

	var mi MatchInfo

	// We know this player
	alloc, ok := playerAlloc[pl.PlayerUUID]
	if ok {
		if alloc == "" {
			fmt.Println("no match found yet")
			// No match found for this player yet
			json.NewEncoder(w).Encode(mi)
			return
		}

		mi, ok = matches[alloc]
		if !ok {
			http.Error(w, "match missing", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(mi)
		return
	}

	// Mark players as known
	playerAllocsMtx.Lock()
	playerAlloc[pl.PlayerUUID] = ""
	playerAllocsMtx.Unlock()

	unmatchedPlayersMtx.Lock()
	unmatchedPlayers = append(unmatchedPlayers, pl)
	unmatchedPlayersMtx.Unlock()

	json.NewEncoder(w).Encode(mi)

	// Trigger the matchmaker to do its thing
	go checkMatch()
}

func checkMatch() {
	if matchSize > len(playerAlloc) {
		// Not enough players yet
		return
	}

	matchPlayers := unmatchedPlayers[:matchSize]
	unmatchedPlayers = unmatchedPlayers[matchSize:]

	mi := MatchInfo{
		MatchedPlayers: true,
		Players:        matchPlayers,
		AllocationUUID: uuid.New().String(),
	}

	for _, p := range matchPlayers {
		playerAlloc[p.PlayerUUID] = mi.AllocationUUID
	}

	matches[mi.AllocationUUID] = mi

	allocate(mi)
}

func handleEndMatch(w http.ResponseWriter, r *http.Request) {
	matchesMtx.Lock()
	defer matchesMtx.Unlock()

	playerAllocsMtx.Lock()
	defer playerAllocsMtx.Unlock()

	var mer endMatchRequest
	json.NewDecoder(r.Body).Decode(&mer)

	mi, ok := matches[mer.AllocationUUID]
	if !ok {
		http.Error(w, "unknown match", http.StatusBadRequest)
		return
	}

	deallocate(mi)

	delete(matches, mer.AllocationUUID)
	delete(playerAlloc, mer.AllocationUUID)

	json.NewEncoder(w).Encode(matches)
}

func allocate(mi MatchInfo) {
	// TODO(lw): Go and allocate this server
	fmt.Printf("allocate: match: %s", mi.AllocationUUID)
}

func deallocate(mi MatchInfo) {
	// TODO(lw) Go and deallocate this server
	fmt.Printf("deallocate: match: %s", mi.AllocationUUID)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/player", handlePlayer)
	r.HandleFunc("/end-match", handleEndMatch)

	err := http.ListenAndServe(":8080", r)
	if err != nil {
		log.Println(err)
	}
}
