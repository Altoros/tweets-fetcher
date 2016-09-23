package geocoder

import (
	"golang.org/x/net/context"
	"googlemaps.github.io/maps"
)

func NewGoogle(googleMapsClient *maps.Client) Geocoder {
	return &googleMapsGeocoder{
		googleMapsClient: googleMapsClient,
	}
}

type googleMapsGeocoder struct {
	googleMapsClient *maps.Client
}

func (g *googleMapsGeocoder) Country(lat, lng float64) (string, error) {
	geocodeReq := maps.GeocodingRequest{
		LatLng: &maps.LatLng{
			Lat: lat,
			Lng: lng,
		},
	}

	var country string

	geocodeRes, err := g.googleMapsClient.Geocode(context.Background(), &geocodeReq)
	if err != nil {
		return "", err
	} else {
		for _, addressComponent := range geocodeRes[0].AddressComponents {
			if isIn(addressComponent.Types, "country") {
				country = addressComponent.LongName
			}
		}
	}

	return country, nil
}

func isIn(a []string, item string) bool {
	for _, v := range a {
		if v == item {
			return true
		}
	}
	return false
}
