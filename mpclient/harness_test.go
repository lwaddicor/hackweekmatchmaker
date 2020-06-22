package mpclient

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const (
	fleetID   = "4278baf9-aea4-44c8-8fa7-ddc7eb318621"
	regionID  = "a960646a-f694-11e5-aa0b-0242ac110008"
	profileID = 1075753
)

func TestMultiplayClient_Allocate(t *testing.T) {
	c, err := NewClientFromEnv()
	require.NoError(t, err)

	allocUUID := uuid.New().String()

	fmt.Println("Making allocation")
	fmt.Println("Fleet: ", fleetID)
	fmt.Println("RegionID: ", regionID)
	fmt.Println("Profile: ", profileID)
	fmt.Println("UUID: ", allocUUID)
	_, err = c.Allocate(fleetID, regionID, profileID, allocUUID)
	require.NoError(t, err)

	ticker := time.NewTicker(time.Second)

	fmt.Println("Waiting for allocation")
	for range ticker.C {
		allocs, err := c.Allocations(fleetID, regionID, profileID, allocUUID)
		require.NoError(t, err)

		if len(allocs) > 0 && allocs[0].IP != "" {
			fmt.Printf("Got allocation: %s:%d\n", allocs[0].IP, allocs[0].GamePort)
			break
		}
	}

	fmt.Println("Deallocating")
	require.NoError(t, c.Deallocate(fleetID, allocUUID))
	fmt.Println("Deallocated")

}
