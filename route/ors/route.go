// Package ors provides access to openrouteservice API.
package ors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/vearutop/gpxt/gpx"
)

// Profile is an enum type.
type Profile string

// Profile values enumeration.
const (
	ProfileDrivingCar      = Profile("driving-car")
	ProfileDrivingHgv      = Profile("driving-hgv")
	ProfileCyclingRegular  = Profile("cycling-regular")
	ProfileCyclingRoad     = Profile("cycling-road")
	ProfileCyclingMountain = Profile("cycling-mountain")
	ProfileCyclingElectric = Profile("cycling-electric")
	ProfileFootWalking     = Profile("foot-walking")
	ProfileFootHiking      = Profile("foot-hiking")
	ProfileWheelchair      = Profile("wheelchair")
	ProfilePublicTransport = Profile("public-transport")
)

// GetRoute uses ORS API to get route information.
func GetRoute(ctx context.Context, profile Profile, points []gpx.Point) (*GeoJSON, error) {
	apiKey := os.Getenv("ORS_KEY")
	if apiKey == "" {
		return nil, errors.New("missing ORS_KEY env var, you can get one at https://openrouteservice.org/dev/")
	}

	type payload struct {
		Coordinates [][2]float64 `json:"coordinates"`
	}

	var pl payload

	for _, p := range points {
		pl.Coordinates = append(pl.Coordinates, [2]float64{p.Longitude, p.Latitude})
	}

	j, err := json.Marshal(pl)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.openrouteservice.org/v2/directions/"+string(profile)+"/geojson", bytes.NewReader(j))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }() //nolint

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d, %q", resp.StatusCode, string(respBody))
	}

	var gj GeoJSON
	if err := json.Unmarshal(respBody, &gj); err != nil {
		return nil, err
	}

	return &gj, nil
}
