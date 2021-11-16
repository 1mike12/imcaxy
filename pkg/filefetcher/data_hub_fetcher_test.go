package filefetcher

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	mock_hub "github.com/thebartekbanach/imcaxy/pkg/hub/mocks"
	mock_globals "github.com/thebartekbanach/imcaxy/test/mocks"
)

type httpResponseBody struct {
	io.Reader
}

func (body *httpResponseBody) Close() error {
	return nil
}

func fetchGetterFuncFactoryWithGetterCallback(reader io.Reader, responseStatusCode int, err error, onGetterCall func()) httpGetFunc {
	return func(_ context.Context, url string) (*http.Response, error) {
		onGetterCall()

		if err != nil {
			return nil, err
		}

		body := httpResponseBody{reader}
		resp := http.Response{
			Body:       &body,
			StatusCode: responseStatusCode,
		}

		return &resp, nil
	}
}

func testDataFetchFuncFactoryWithGetterCallback(responseStatusCode int, err error, onGetterCall func()) (httpGetFunc, []byte) {
	data := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
	reader := bytes.NewReader(data)

	get := fetchGetterFuncFactoryWithGetterCallback(reader, responseStatusCode, err, onGetterCall)

	return get, data
}

func testDataFetchFuncFactory(responseStatusCode int, err error) (httpGetFunc, []byte) {
	return testDataFetchFuncFactoryWithGetterCallback(responseStatusCode, err, func() {})
}

func fetchGetterFuncFactoryWithErrorReturnedByReader(t *testing.T, err error) httpGetFunc {
	mockCtrl := gomock.NewController(t)
	mockReader := mock_globals.NewMockReader(mockCtrl)
	mockReader.EXPECT().Read(gomock.Any()).Return(0, err)

	get := fetchGetterFuncFactoryWithGetterCallback(mockReader, 200, nil, func() {})

	return get
}

func TestDataHubFetcher_ShouldPassRequestBodyCorrectlyToDataStreamInput(t *testing.T) {
	getter, testData := testDataFetchFuncFactory(200, nil)
	mockStreamInput := mock_hub.NewMockTestingDataStreamInput(t, [][]byte{testData}, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fetcher := DataHubFetcher{getter}
	fetcher.Fetch(ctx, "http://google.com/image.jpg", &mockStreamInput)

	mockStreamInput.Wait()
}

func TestDataHubFetcher_ShouldReturn404ErrorIf404IsReturnedByRequest(t *testing.T) {
	getter, _ := testDataFetchFuncFactory(404, nil)
	mockStreamInput := mock_hub.NewMockTestingDataStreamInput(t, nil, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fetcher := DataHubFetcher{getter}
	err := fetcher.Fetch(ctx, "http://google.com/image.jpg", &mockStreamInput)

	if err != ErrResponseStatus404 {
		t.Errorf("Expected fetch error to be %v, got %v", ErrResponseStatus404, err)
	}

	if mockStreamInput.ForwardedError != ErrResponseStatus404 {
		t.Errorf("Expected stream close frowarded error to be %v, got %v", ErrResponseStatus404, mockStreamInput.ForwardedError)
	}
}

func TestDataHubFetcher_ShouldReturnErrorIfErrorReturnedByRequestIsNot200(t *testing.T) {
	getter, _ := testDataFetchFuncFactory(500, nil)
	mockStreamInput := mock_hub.NewMockTestingDataStreamInput(t, nil, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fetcher := DataHubFetcher{getter}
	err := fetcher.Fetch(ctx, "http://google.com/image.jpg", &mockStreamInput)

	if err != ErrResponseStatusNotOK {
		t.Errorf("Expected fetch error to be %v, got %v", ErrResponseStatusNotOK, err)
	}
}

func TestDataHubFetcher_ShouldCloseInputWithErrorReturnedByStreamRead(t *testing.T) {
	getter := fetchGetterFuncFactoryWithErrorReturnedByReader(t, io.ErrUnexpectedEOF)
	mockStreamInput := mock_hub.NewMockTestingDataStreamInput(t, nil, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fetcher := DataHubFetcher{getter}
	fetcher.Fetch(ctx, "http://google.com/image.jpg", &mockStreamInput)

	mockStreamInput.Wait()

	if mockStreamInput.ForwardedError != io.ErrUnexpectedEOF {
		t.Errorf("Expected stream close frowarded error to be %v, got %v", io.ErrUnexpectedEOF, mockStreamInput.ForwardedError)
	}
}
