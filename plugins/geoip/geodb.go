package geoip

import (
	"fmt"
	"net"

	maxmind "github.com/oschwald/geoip2-golang"
	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
)

type GeoDB struct {
	Path   string `short:"p" long:"path" description:"Path to maxmind GeoIP database"`
	Field  string `short:"f" long:"field" description:"Field containing an ip to calculate geo information from"`
	Strict bool   `short:"s" long:"strict" description:"Error out rather than ignore problems calculate the geoip info"`

	All bool `short:"a" long:"all" description:"Include all geo information"`

	r *maxmind.Reader
}

func NewGeoDB() (*GeoDB, error) {
	g := &GeoDB{}

	return g, nil
}

func (g *GeoDB) Open() error {
	if g.Path == "" {
		return fmt.Errorf("no database path given")
	}

	r, err := maxmind.Open(g.Path)
	if err != nil {
		return err
	}

	g.r = r

	return nil
}

func (g *GeoDB) Filterer() (cypress.Filterer, error) {
	err := g.Open()
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (g *GeoDB) Filter(m *cypress.Message) (*cypress.Message, error) {
	ipStr, ok := m.GetString(g.Field)
	if !ok {
		return m, nil
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return m, nil
	}

	city, err := g.r.City(ip)
	if err != nil {
		if g.Strict {
			return nil, err
		}

		return m, nil
	}

	if g.All {
		m.Add("geoip.city_name", city.City.Names["en"])
		m.Add("geoip.continent_code", city.Continent.Code)
		m.Add("geoip.country_code", city.Country.IsoCode)
		m.Add("geoip.postal_code", city.Postal.Code)
		m.Add("geoip.timezone", city.Location.TimeZone)
	}

	m.Add("geoip.latitude", city.Location.Latitude)
	m.Add("geoip.longitude", city.Location.Longitude)
	m.Add("geoip.location",
		fmt.Sprintf("%f,%f", city.Location.Latitude, city.Location.Longitude))

	return m, nil
}

func (g *GeoDB) Execute(args []string) error {
	err := g.Open()
	if err != nil {
		return err
	}

	return cypress.StandardStreamFilter(g)
}

func init() {
	commands.Add("geoip", "calculate geoip information based on a field", "", &GeoDB{})
	cypress.AddPlugin("geoip", func() cypress.Plugin { return &GeoDB{} })
}
