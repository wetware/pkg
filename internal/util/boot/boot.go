package bootutil

import (
	"net"
	"net/url"
	"path"
	"strconv"

	"github.com/lthibault/log"

	"github.com/urfave/cli/v2"
	"github.com/wetware/casm/pkg/boot"
)

func NewCrawler(c *cli.Context, log log.Logger) (boot.Crawler, error) {
	u, err := url.Parse(c.String("discover"))
	if err != nil {
		return boot.Crawler{}, err
	}

	port, err := strconv.Atoi(u.Port())
	if err != nil {
		return boot.Crawler{}, err
	}

	return boot.Crawler{
		Logger: log.WithField("scan", u.String()),
		Dialer: new(net.Dialer),
		Strategy: &boot.ScanSubnet{
			Logger: log.WithField("scan", u.String()),
			Net:    u.Scheme,
			Port:   port,
			CIDR:   path.Join(u.Hostname(), u.Path), // e.g. '10.0.1.0/24'
		},
	}, nil
}
