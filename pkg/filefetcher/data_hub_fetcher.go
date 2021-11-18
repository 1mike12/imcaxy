package filefetcher

import (
	"context"
	"errors"
	"net/http"

	"github.com/thebartekbanach/imcaxy/pkg/hub"
)

type httpGetFunc func(ctx context.Context, url string) (resp *http.Response, err error)

type DataHubFetcher struct {
	getter httpGetFunc
}

var _ Fetcher = (*DataHubFetcher)(nil)

func NewDataHubFetcher() Fetcher {
	getFunc := func(ctx context.Context, url string) (resp *http.Response, err error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}

		return http.DefaultClient.Do(req)
	}

	return &DataHubFetcher{getFunc}
}

func (fetcher *DataHubFetcher) Fetch(ctx context.Context, url string, input hub.DataStreamInput) error {
	response, err := fetcher.getter(ctx, url)
	if err != nil {
		input.Close(err)
		return err
	}
	defer response.Body.Close()

	err = nil
	if response.StatusCode == http.StatusNotFound {
		err = ErrResponseStatus404
	} else if response.StatusCode != http.StatusOK {
		err = ErrResponseStatusNotOK
	}

	if err != nil {
		input.Close(err)
		return err
	}

	go func() {
		_, err := input.ReadFrom(response.Body)
		input.Close(err)
	}()

	return nil
}

var (
	ErrResponseStatusNotOK = errors.New("response returned non-200 status code")
	ErrResponseStatus404   = errors.New("response returned 404 status code")
)
