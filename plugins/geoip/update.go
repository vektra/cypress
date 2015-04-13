package geoip

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/cheggaaa/pb"
	"github.com/vektra/cypress/cli/commands"
)

type Update struct {
	Path string `short:"p" long:"path" description:"Where to store the database"`
}

const URL = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-City.mmdb.gz"
const MD5URL = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-City.md5"

func (u *Update) Execute(args []string) error {
	tmp := u.Path + ".tmp"

	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	defer os.Remove(tmp)
	defer f.Close()

	resp, err := http.Get(URL)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Unexpected HTTP status: %s", resp.Status)
	}

	mod := resp.Header.Get("Last-Modified")
	if mod == "" {
		mod = "unknown"
	}

	defer resp.Body.Close()

	fmt.Printf("Downloading GeoLite2 DB, updated: %s\n", mod)

	bar := pb.New64(resp.ContentLength)
	bar.ShowTimeLeft = true
	bar.ShowSpeed = true
	bar.SetUnits(pb.U_BYTES)

	h := md5.New()

	zw, err := gzip.NewReader(bar.NewProxyReader(resp.Body))
	if err != nil {
		return err
	}

	defer zw.Close()

	bar.Start()

	defer bar.Finish()

	_, err = io.Copy(io.MultiWriter(f, h), zw)
	if err != nil {
		return err
	}

	hashresp, err := http.Get(MD5URL)
	if err != nil {
		return err
	}

	defer hashresp.Body.Close()

	if hashresp.StatusCode == 200 {
		data, err := ioutil.ReadAll(hashresp.Body)
		if err != nil {
			return err
		}

		hexpect, err := hex.DecodeString(string(bytes.TrimSpace(data)))
		if err != nil {
			return err
		}

		hactual := h.Sum(nil)

		if !bytes.Equal(hexpect, hactual) {
			return fmt.Errorf("MD5 sums did not match: %s != %s",
				hex.EncodeToString(hactual), hex.EncodeToString(hexpect))
		}
	}

	os.Rename(tmp, u.Path)
	return nil
}

func init() {
	commands.Add("geoip:update", "Fetch a new GeoIP database", "", &Update{})
}
