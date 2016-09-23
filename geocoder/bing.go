package geocoder

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var (
	bingGeocodeUrl = "http://dev.virtualearth.net/REST/v1/Locations/%f,%f?o=json&key=%s"
)

type bingGeocodeResponse struct {
	ResourceSets []bingResourceSet
}

type bingResourceSet struct {
	Resources []bingResource
}

type bingResource struct {
	Address bingAddress
}

type bingAddress struct {
	CountryRegion string
}

func NewBing(apiKey string) Geocoder {
	return &bingMapsGeocoder{
		apiKey: apiKey,
	}
}

type bingMapsGeocoder struct {
	apiKey string
}

func (g *bingMapsGeocoder) Country(lat, lng float64) (string, error) {
	resp, err := http.Get(fmt.Sprintf(bingGeocodeUrl, lat, lng, g.apiKey))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var bingResp bingGeocodeResponse

	if err = json.NewDecoder(resp.Body).Decode(&bingResp); err != nil {
		return "", err
	}

	return bingResp.ResourceSets[0].Resources[0].Address.CountryRegion, nil
}
