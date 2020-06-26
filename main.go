package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/caarlos0/env"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/lwaddicor/hackweekmatchmaker/mpclient"
)

const (
	matchSize = 2
)

type PlayerInfo struct {
	PlayerUUID string
	ip         string
}

type MatchInfo struct {
	MatchedPlayers bool
	AllocationUUID string       `json:",omitempty"`
	Players        []PlayerInfo `json:",omitempty"`
	IP             string       `json:",omitempty"`
	Port           int          `json:",omitempty"`
	Aborted        bool         `json:",omitempty"`
}

type endMatchRequest struct {
	AllocationUUID string
}

// Config contains settings for starting the matchmaker
type Config struct {
	FleetID   string `env:"MP_FLEET_ID"`
	RegionID  string `env:"MP_REGION_ID"`
	ProfileID int64  `env:"MP_PROFILE_ID"`
}

type SimpleMatchmaker struct {
	mpClient mpclient.MultiplayClient

	matches    map[string]MatchInfo
	matchesMtx sync.Mutex

	unmatchedPlayers    []PlayerInfo
	unmatchedPlayersMtx sync.Mutex

	playerAlloc     map[string]string
	playerAllocsMtx sync.Mutex

	cfg Config
}

func NewSimpleMatchmaker(cfg Config, client mpclient.MultiplayClient) *SimpleMatchmaker {
	return &SimpleMatchmaker{
		mpClient:    client,
		matches:     make(map[string]MatchInfo),
		playerAlloc: make(map[string]string),
		cfg:         cfg,
	}
}

func (m *SimpleMatchmaker) handlePlayer(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handle player")
	var pl PlayerInfo
	if err := json.NewDecoder(r.Body).Decode(&pl); err != nil {
		fmt.Println("failed decoding request: " + err.Error())
		http.Error(w, "decode request", http.StatusBadRequest)
		return
	}
	pl.ip = r.RemoteAddr

	if pl.PlayerUUID == "" {
		fmt.Println("missing player uuid")
		http.Error(w, "missing player uuid", http.StatusBadRequest)
		return
	}

	var mi MatchInfo

	// We know this player
	alloc, ok := m.playerAlloc[pl.PlayerUUID]
	if ok {
		if alloc == "" {
			fmt.Println("no match found yet")
			// No match found for this player yet
			json.NewEncoder(w).Encode(mi)
			return
		}

		mi, ok = m.matches[alloc]
		if !ok {
			http.Error(w, "match missing", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(mi)
		return
	}

	// Mark players as known
	m.playerAllocsMtx.Lock()
	m.playerAlloc[pl.PlayerUUID] = ""
	m.playerAllocsMtx.Unlock()

	m.unmatchedPlayersMtx.Lock()
	m.unmatchedPlayers = append(m.unmatchedPlayers, pl)
	m.unmatchedPlayersMtx.Unlock()

	json.NewEncoder(w).Encode(mi)

	// Trigger the matchmaker to do its thing
	go m.checkMatch()
}

func (m *SimpleMatchmaker) checkMatch() {
	m.unmatchedPlayersMtx.Lock()
	fmt.Printf("match size: %d queued players: %d\n", matchSize, len(m.unmatchedPlayers))
	enoughPlayers := matchSize <= len(m.unmatchedPlayers)
	m.unmatchedPlayersMtx.Unlock()
	if !enoughPlayers {
		return
	}
	m.unmatchedPlayersMtx.Lock()
	matchPlayers := m.unmatchedPlayers[:matchSize]
	m.unmatchedPlayers = m.unmatchedPlayers[matchSize:]
	m.unmatchedPlayersMtx.Unlock()

	mi := MatchInfo{
		MatchedPlayers: true,
		Players:        matchPlayers,
		AllocationUUID: uuid.New().String(),
	}

	m.playerAllocsMtx.Lock()
	for _, p := range matchPlayers {
		m.playerAlloc[p.PlayerUUID] = mi.AllocationUUID
	}
	m.playerAllocsMtx.Unlock()

	m.matchesMtx.Lock()
	m.matches[mi.AllocationUUID] = mi
	m.matchesMtx.Unlock()

	if _, err := m.mpClient.Allocate(m.cfg.FleetID, m.cfg.RegionID, m.cfg.ProfileID, mi.AllocationUUID); err != nil {
		fmt.Errorf("failed to allocate %s", mi.AllocationUUID)
		//TODO(lw): Recover gracefully
	}

	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		allocs, err := m.mpClient.Allocations(m.cfg.FleetID, m.cfg.RegionID, m.cfg.ProfileID, mi.AllocationUUID)
		if err != nil {
			// TODO(lw): Catch non retryable cases, like 404
			continue
		}

		if len(allocs) == 0 {
			// TODO(lw): Catch non retryable cases, in this case it dissapeared
			break
		}

		alloc := allocs[0]
		if alloc.IP != "" {
			fmt.Printf("Got allocation: %s:%d\n", alloc.IP, alloc.GamePort)

			m.matchesMtx.Lock()
			v := m.matches[mi.AllocationUUID]
			v.IP = alloc.IP
			v.Port = alloc.GamePort
			m.matches[mi.AllocationUUID] = v
			m.matchesMtx.Unlock()
			break
		}
		fmt.Printf("Waiting for allocation: %s\n", mi.AllocationUUID)
	}
}

func (m *SimpleMatchmaker) handleEndMatch(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handle end match")
	m.matchesMtx.Lock()
	defer m.matchesMtx.Unlock()

	m.playerAllocsMtx.Lock()
	defer m.playerAllocsMtx.Unlock()

	var mer endMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&mer); err != nil {
		fmt.Println("failed end match decoding request: " + err.Error())
		http.Error(w, "decode request", http.StatusBadRequest)
		return
	}

	_, ok := m.matches[mer.AllocationUUID]
	if !ok {
		fmt.Printf("unknown match: %s\n", mer.AllocationUUID)
		http.Error(w, "unknown match", http.StatusBadRequest)
		return
	}

	if err := m.mpClient.Deallocate(m.cfg.FleetID, mer.AllocationUUID); err != nil {
		fmt.Println("deallocate error: %s", err.Error())
		http.Error(w, "failed to deallocate", http.StatusInternalServerError)
		return
	}

	delete(m.matches, mer.AllocationUUID)
	delete(m.playerAlloc, mer.AllocationUUID)

	json.NewEncoder(w).Encode(m.matches)
}

func main() {

	fmt.Println("Hello")
	mpClient, err := mpclient.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	//mpClient := mpclient.MockMultiplayClient{}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatal(err)
	}

	mm := NewSimpleMatchmaker(cfg, mpClient)

	r := mux.NewRouter()
	r.HandleFunc("/player", mm.handlePlayer)
	r.HandleFunc("/end-match", mm.handleEndMatch)

	if err := http.ListenAndServe(":10855", r); err != nil {
		log.Println(err)
	}
}
