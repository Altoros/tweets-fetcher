package fetcher

import "fmt"

type Tweet struct {
	Id          string
	Text        string
	User        string
	Coordinates Coordinates
}

type Coordinates struct {
	Lat  float64
	Long float64
}

func (c Coordinates) String() string {
	return fmt.Sprintf("[%d,%d]", c.Long, c.Lat)
}
