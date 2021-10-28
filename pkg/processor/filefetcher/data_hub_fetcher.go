package filefetcher

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/thebartekbanach/imcaxy/pkg/hub"
)

const DATA_HUB_FETCHER_STREAM_NAMES_TEMPLATE = "%s::FILE_FETCHER"

type httpGetFunc func(url string) (resp *http.Response, err error)

type DataHubFetcher struct {
	dataHub hub.DataHub
	getter  httpGetFunc
}

var _ Fetcher = (*DataHubFetcher)(nil)

func NewDataHubFetcher(dataHub hub.DataHub) DataHubFetcher {
	return DataHubFetcher{dataHub, http.Get}
}

func (fetcher *DataHubFetcher) Fetch(url string) (hub.DataStreamOutput, error) {
	fileStreamName := fmt.Sprintf(DATA_HUB_FETCHER_STREAM_NAMES_TEMPLATE, url)

	output, input, err := fetcher.dataHub.GetOrCreateStream(fileStreamName)
	if err != nil {
		return nil, err
	}

	if input != nil {
		if err := fetcher.fetch(url, input); err != nil {
			return nil, err
		}
	}

	return output, nil
}

func (fetcher *DataHubFetcher) fetch(url string, input hub.DataStreamInput) error {
	response, err := fetcher.getter(url)
	if err != nil {
		input.Close(err)
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		input.Close(ErrResponseStatusNotOK)
		return ErrResponseStatusNotOK
	}

	go func() {
		if _, err := input.ReadFrom(response.Body); err != nil {
			if err == io.EOF {
				input.Close(nil)
				return
			}

			input.Close(err)
		}

		input.Close(nil)
	}()

	return nil
}

var ErrResponseStatusNotOK = errors.New("response returned non-200 status code")
