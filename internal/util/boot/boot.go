package bootutil

import (
	"net"
	"net/url"
	"path"
	"strconv"

	"github.com/wetware/casm/pkg/boot/crawl"
	logutil "github.com/wetware/ww/internal/util/log"

	"github.com/urfave/cli/v2"
)

func NewCrawler(c *cli.Context) (crawl.Crawler, error) {
	u, err := url.Parse(c.String("discover"))
	if err != nil {
		return crawl.Crawler{}, err
	}

	port, err := strconv.Atoi(u.Port())
	if err != nil {
		return crawl.Crawler{}, err
	}

	cidr := path.Join(u.Hostname(), u.Path) // e.g. '10.0.1.0/24'
	log := logutil.New(c).
		WithField("net", u.Scheme).
		WithField("port", port).
		WithField("cidr", cidr)

	return crawl.Crawler{
		Logger: log,
		Dialer: new(net.Dialer),
		Strategy: &crawl.ScanSubnet{
			Logger: log,
			Net:    u.Scheme,
			Port:   port,
			CIDR:   cidr, // e.g. '10.0.1.0/24'
		},
	}, nil
}
