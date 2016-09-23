package geocoder

type Geocoder interface {
	Country(lat, lng float64) (string, error)
}
