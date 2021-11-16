package filefetcher

import (
	"context"

	"github.com/thebartekbanach/imcaxy/pkg/hub"
)

type Fetcher interface {
	Fetch(ctx context.Context, url string, input hub.DataStreamInput) error
}
