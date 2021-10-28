package filefetcher

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/franela/goblin"
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

func fetchGetterFuncFactoryWithGetterCallback(reader io.Reader, responseStatusCode int, err error, onGetterCall func()) HttpGetFunc {
	return func(url string) (*http.Response, error) {
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

func testDataFetchFuncFactoryWithGetterCallback(responseStatusCode int, err error, onGetterCall func()) (HttpGetFunc, []byte) {
	data := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
	reader := bytes.NewReader(data)

	get := fetchGetterFuncFactoryWithGetterCallback(reader, responseStatusCode, err, onGetterCall)

	return get, data
}

func testDataFetchFuncFactory(responseStatusCode int, err error) (HttpGetFunc, []byte) {
	return testDataFetchFuncFactoryWithGetterCallback(responseStatusCode, err, func() {})
}

func TestDataHubFetcher(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("DataHubFetcher", func() {
		g.It("Should fetch all of data correctly", func() {
			fileName := "http://google.com/image.jpg"
			streamName := fmt.Sprintf(DATA_HUB_FETCHER_STREAM_NAMES_TEMPLATE, fileName)
			getter, testData := testDataFetchFuncFactory(200, nil)

			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockDataStreamOutput := mock_hub.NewMockTestingDataStreamOutput(g, [][]byte{testData}, io.EOF, nil)
			mockDataStreamInput := mock_hub.NewMockTestingDataStreamInput(g, [][]byte{testData}, nil, nil)
			mockDataHub := mock_hub.NewMockDataHub(mockCtrl)
			mockDataHub.EXPECT().GetOrCreateStream(streamName).Return(
				mockDataStreamOutput, &mockDataStreamInput, nil,
			)

			fetcher := DataHubFetcher{mockDataHub, getter}
			fetcher.Fetch(fileName)

			mockDataStreamInput.Wait()
			g.Assert(mockDataStreamInput.SafelyGetDataSegment(0)).Equal(testData)
		})

		g.It("Should return error returned by http.Get method", func() {
			fileName := "http://google.com/image.jpg"
			streamName := fmt.Sprintf(DATA_HUB_FETCHER_STREAM_NAMES_TEMPLATE, fileName)
			getter, _ := testDataFetchFuncFactory(200, io.ErrUnexpectedEOF)

			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockDataStreamOutput := mock_hub.NewMockTestingDataStreamOutput(g, [][]byte{}, io.EOF, nil)
			mockDataStreamInput := mock_hub.NewMockTestingDataStreamInput(g, [][]byte{}, nil, nil)
			mockDataHub := mock_hub.NewMockDataHub(mockCtrl)
			mockDataHub.EXPECT().GetOrCreateStream(streamName).Return(
				mockDataStreamOutput, &mockDataStreamInput, nil,
			)

			fetcher := DataHubFetcher{mockDataHub, getter}
			_, err := fetcher.Fetch(fileName)

			g.Assert(err).Equal(io.ErrUnexpectedEOF)
		})

		g.It("Should return error when get returns non-200 status code", func() {
			fileName := "http://google.com/image.jpg"
			streamName := fmt.Sprintf(DATA_HUB_FETCHER_STREAM_NAMES_TEMPLATE, fileName)
			getter, _ := testDataFetchFuncFactory(404, nil)

			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockDataStreamOutput := mock_hub.NewMockTestingDataStreamOutput(g, [][]byte{}, io.EOF, nil)
			mockDataStreamInput := mock_hub.NewMockTestingDataStreamInput(g, [][]byte{}, nil, nil)
			mockDataHub := mock_hub.NewMockDataHub(mockCtrl)
			mockDataHub.EXPECT().GetOrCreateStream(streamName).Return(
				mockDataStreamOutput, &mockDataStreamInput, nil,
			)

			fetcher := DataHubFetcher{mockDataHub, getter}
			_, err := fetcher.Fetch(fileName)

			g.Assert(err).Equal(ErrResponseStatusNotOK)
		})

		g.It("Should not download file again if file stream already exists", func() {
			fileName := "http://google.com/image.jpg"
			streamName := fmt.Sprintf(DATA_HUB_FETCHER_STREAM_NAMES_TEMPLATE, fileName)
			getter, _ := testDataFetchFuncFactoryWithGetterCallback(200, nil, func() {
				g.Errorf("Started to download the file but should not")
			})

			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockDataStreamOutput := mock_hub.NewMockTestingDataStreamOutput(g, [][]byte{}, io.EOF, nil)
			mockDataHub := mock_hub.NewMockDataHub(mockCtrl)
			mockDataHub.EXPECT().GetOrCreateStream(streamName).Return(
				mockDataStreamOutput, nil, nil,
			)

			fetcher := DataHubFetcher{mockDataHub, getter}
			output, _ := fetcher.Fetch(fileName)

			g.Assert(output).Equal(mockDataStreamOutput)
		})

		g.It("Should forward error returned by http.Get response Body to input stream", func() {
			fileName := "http://google.com/image.jpg"
			streamName := fmt.Sprintf(DATA_HUB_FETCHER_STREAM_NAMES_TEMPLATE, fileName)

			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockReader := mock_globals.NewMockReader(mockCtrl)
			mockReader.EXPECT().Read(gomock.Any()).Return(0, io.ErrUnexpectedEOF)

			getter := fetchGetterFuncFactoryWithGetterCallback(mockReader, 200, nil, func() {})

			mockDataStreamOutput := mock_hub.NewMockTestingDataStreamOutput(g, [][]byte{}, io.EOF, nil)
			mockDataStreamInput := mock_hub.NewMockTestingDataStreamInput(g, [][]byte{}, nil, io.ErrUnexpectedEOF)
			mockDataHub := mock_hub.NewMockDataHub(mockCtrl)
			mockDataHub.EXPECT().GetOrCreateStream(streamName).Return(
				mockDataStreamOutput, &mockDataStreamInput, nil,
			)

			fetcher := DataHubFetcher{mockDataHub, getter}
			fetcher.Fetch(fileName)

			mockDataStreamInput.Wait()
		})

		g.It("Should return error returned by DataHub.GetOrCreateStream", func() {
			fileName := "http://google.com/image.jpg"
			streamName := fmt.Sprintf(DATA_HUB_FETCHER_STREAM_NAMES_TEMPLATE, fileName)
			testErr := errors.New("test error")
			getter, _ := testDataFetchFuncFactory(200, nil)

			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockDataHub := mock_hub.NewMockDataHub(mockCtrl)
			mockDataHub.EXPECT().GetOrCreateStream(streamName).Return(
				nil, nil, testErr,
			)

			fetcher := DataHubFetcher{mockDataHub, getter}
			_, err := fetcher.Fetch(fileName)

			g.Assert(err).Equal(testErr)
		})
	})
}
