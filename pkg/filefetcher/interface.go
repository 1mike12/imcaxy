package filefetcher

import "github.com/thebartekbanach/imcaxy/pkg/hub"

type Fetcher interface {
	Fetch(url string) (hub.DataStreamOutput, error)
}
