package mpclient

import "fmt"

type MockMultiplayClient struct {
}

func (m MockMultiplayClient) Allocate(fleet, region string, profile int64, uuid string) (*AllocateResponse, error) {
	fmt.Printf("Allocated: %s, %s, %d, %s\n", fleet, region, profile, uuid)
	return &AllocateResponse{
		ProfileID: 0,
		UUID:      "",
		RegionID:  "",
		Created:   "",
		Error:     "",
	}, nil
}

func (m MockMultiplayClient) Allocations(fleet, region string, profile int64, uuid string) ([]AllocationResponse, error) {
	fmt.Printf("allocations: %s, %s, %d, %s\n", fleet, region, profile, uuid)
	return []AllocationResponse{
		{
			ProfileID: 0,
			UUID:      "123-123-123",
			Regions:   "",
			Created:   "",
			Requested: "",
			Fulfilled: "",
			ServerID:  0,
			FleetID:   "",
			RegionID:  "",
			MachineID: 0,
			IP:        "35.205.254.215",
			GamePort:  9200,
			Error:     "",
		},
	}, nil
}

func (m MockMultiplayClient) Deallocate(fleet, uuid string) error {
	fmt.Printf("deallocate: %s, %s\n", fleet, uuid)
	return nil
}
