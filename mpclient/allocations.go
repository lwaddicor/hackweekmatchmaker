package mpclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type AllocationResponse struct {
	ProfileID int64
	UUID      string
	Regions   string
	Created   string
	Requested string
	Fulfilled string
	ServerID  int64
	FleetID   string
	RegionID  string
	MachineID int64
	IP        string
	GamePort  int `json:"game_port"`
	Error     string
}

type allocationsResponseWrapper struct {
	Success     bool
	Allocations []AllocationResponse
}

func (m *multiplayClient) Allocations(fleet, region string, profile int64, uuid string) ([]AllocationResponse, error) {
	urlStr := fmt.Sprintf("%s/cfp/v1/server/allocations", m.baseURL)
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("parse url %s", urlStr)
	}

	params := url.Values{}
	params.Add("uuid", uuid)
	params.Add("fleetid", fleet)
	u.RawQuery = params.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("allocations new request")
	}

	if _, err = m.signer.Sign(req, nil, authService, authRegion, time.Now().UTC()); err != nil {
		return nil, fmt.Errorf("sign allocations request: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send allocations request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("allocations call failed: %d: %s", res.StatusCode, getBody(res.Body))
	}

	var ar allocationsResponseWrapper
	if err := json.NewDecoder(res.Body).Decode(&ar); err != nil {
		return nil, fmt.Errorf("decode allocations response: %w", err)
	}

	if !ar.Success {
		return nil, fmt.Errorf("allocations request failed")
	}

	return ar.Allocations, nil
}
