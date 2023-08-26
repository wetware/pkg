package cluster

import (
	"os"

	"github.com/wetware/pkg/cluster/pulse"
	"github.com/wetware/pkg/cluster/routing"
)

func (conf Config) preparer() (metaFieldSlice, error) {
	// deduplicate fields
	fields := make(map[string]routing.MetaField, len(conf.Meta))
	for _, tag := range conf.Meta {
		f, err := routing.ParseField(tag)
		if err != nil {
			return nil, err
		}

		fields[f.Key] = f
	}

	// return as slice
	meta := make([]routing.MetaField, 0, len(fields))
	for _, f := range fields {
		meta = append(meta, f)
	}

	return meta, nil
}

type metaFieldSlice []routing.MetaField

func (m metaFieldSlice) Prepare(h pulse.Heartbeat) error {
	if err := h.SetMeta(m); err != nil {
		return err
	}

	// hostname may change over time
	host, err := os.Hostname()
	if err != nil {
		return err
	}

	return h.SetHost(host)
}
