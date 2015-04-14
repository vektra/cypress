package geoip

import (
	"fmt"
	"net"

	maxmind "github.com/oschwald/geoip2-golang"
	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
)

type GeoIP struct {
	Path   string `short:"p" long:"path" description:"Path to maxmind GeoIP database"`
	Field  string `short:"f" long:"field" description:"Field containing an ip to calculate geo information from"`
	Strict bool   `short:"s" long:"strict" description:"Error out rather than ignore problems calculate the geoip info"`

	All bool `short:"a" long:"all" description:"Include all geo information"`

	r *maxmind.Reader
}

func NewGeoIP() (*GeoIP, error) {
	g := &GeoIP{}

	return g, nil
}

func ImplicitPath() string {
	var cfg struct{ Path string }

	err := cypress.GlobalConfig().Load("geoip", &cfg)
	if err == nil && cfg.Path != "" {
		return cfg.Path
	}

	path, ok := cypress.UserFile("geoip.db")
	if ok {
		return path
	}

	path, ok = cypress.GlobalFile("geoip.db")
	if ok {
		return path
	}

	return ""
}

func (g *GeoIP) Open() error {
	path := g.Path
	if path == "" {
		path = ImplicitPath()
		if path == "" {
			return fmt.Errorf("Unable to find a geoip database file to use")
		}
	}

	r, err := maxmind.Open(path)
	if err != nil {
		return err
	}

	g.r = r

	return nil
}

func (g *GeoIP) Filterer() (cypress.Filterer, error) {
	err := g.Open()
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (g *GeoIP) Filter(m *cypress.Message) (*cypress.Message, error) {
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

func (g *GeoIP) Execute(args []string) error {
	err := g.Open()
	if err != nil {
		return err
	}

	return cypress.StandardStreamFilter(g)
}

func init() {
	commands.Add("geoip", "calculate geoip information based on a field", "", &GeoIP{})
	cypress.AddPlugin("geoip", func() cypress.Plugin { return &GeoIP{} })
}
