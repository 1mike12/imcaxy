package imaginaryprocessor

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/franela/goblin"
	"github.com/golang/mock/gomock"
	mock_hub "github.com/thebartekbanach/imcaxy/pkg/hub/mocks"
	"github.com/thebartekbanach/imcaxy/pkg/processor"
)

type httpResponseBody struct {
	io.Reader
	readError error
}

func (body *httpResponseBody) Read(p []byte) (n int, err error) {
	if body.readError != nil {
		return 0, body.readError
	}

	return body.Reader.Read(p)
}

func (body *httpResponseBody) Close() error {
	return nil
}

func testReqFunc(statusCode int, response []byte, callError, responseBodyError error, requestAssert func(req *http.Request)) httpRequestFunc {
	return func(req *http.Request) (*http.Response, error) {
		requestAssert(req)

		if callError != nil {
			return nil, callError
		}

		reader := bytes.NewReader(response)
		body := httpResponseBody{reader, responseBodyError}

		return &http.Response{
			StatusCode: statusCode,
			Body:       &body,
			Header: http.Header{
				"Content-Type": []string{"image/png"},
			},
		}, nil
	}
}

func noAssertions(req *http.Request) {}

func TestImaginaryProcessor(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Processor", func() {
		g.Describe("ParseRequest", func() {
			g.It("Should correctly destruct given request path into request information", func() {
				config := Config{}

				processor := NewProcessor(config)
				result, _ := processor.ParseRequest("/crop?abc=1&def=2&url=http://google.com/image.jpg")

				g.Assert(result.ProcessorEndpoint).Equal("/crop")
				g.Assert(result.SourceImageURL).Equal("http://google.com/image.jpg")
				g.Assert(result.ProcessingParams).Equal(map[string][]string{
					"abc": {"1"},
					"def": {"2"},
					"url": {"http://google.com/image.jpg"},
				})
			})

			g.It("Should generate correct unique checksum of request", func() {
				config := Config{}

				processor := NewProcessor(config)
				firstResult, _ := processor.ParseRequest("/crop?abc=1&def=2&url=http://google.com/image.jpg")
				secondResult, _ := processor.ParseRequest("/crop?abc=1&url=http://google.com/image.jpg&def=2")

				g.Assert(firstResult.Signature).Equal(secondResult.Signature)
			})

			g.It("Should return error if sourceImageURL not found in request", func() {
				config := Config{}

				processor := NewProcessor(config)
				_, err := processor.ParseRequest("/crop?abc=1&def=2")

				g.Assert(err).IsNotNil()
			})

			g.It("Should return error if processorEndpoint is not correct", func() {
				config := Config{}

				processor := NewProcessor(config)
				_, err := processor.ParseRequest("/unknown?abc=1&def=2&url=http://google.com/image.jpg")

				g.Assert(err).IsNotNil()
			})
		})

		g.Describe("ProcessImage", func() {
			g.It("Should correctly construct and send request to imaginary service", func() {
				mockCtrl := gomock.NewController(g)
				defer mockCtrl.Finish()

				config := Config{ImaginaryServiceURL: "http://localhost:3000"}
				testData := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
				inputStream := mock_hub.NewMockTestingDataStreamInput(g, [][]byte{testData}, nil, nil)
				parsedRequest := processor.ParsedRequest{
					Signature:         "abc",
					SourceImageURL:    "http://google.com/image.jpg",
					ProcessorEndpoint: "/crop",
					ProcessingParams: map[string][]string{
						"width":  {"500"},
						"height": {"500"},
					},
				}
				requestMaker := testReqFunc(200, testData, nil, nil, func(req *http.Request) {
					g.Assert(req.Method).Equal(http.MethodGet)
					g.Assert(req.URL.Host).Equal("http://localhost:3000")
					g.Assert(req.URL.Path).Equal("/crop")
				})

				proc := Processor{config, requestMaker}
				contentType, _ := proc.ProcessImage(parsedRequest, &inputStream)

				g.Assert(contentType).Equal("image/png")
			})

			g.It("Should write all contents of imaginary service response into data stream input", func() {
				config := Config{ImaginaryServiceURL: "http://localhost:3000"}
				testData := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
				inputStream := mock_hub.NewMockTestingDataStreamInput(g, [][]byte{testData}, nil, nil)
				parsedRequest := processor.ParsedRequest{
					Signature:         "abc",
					SourceImageURL:    "http://google.com/image.jpg",
					ProcessorEndpoint: "/crop",
					ProcessingParams: map[string][]string{
						"width":  {"500"},
						"height": {"500"},
					},
				}
				requestMaker := testReqFunc(200, testData, nil, nil, noAssertions)

				proc := Processor{config, requestMaker}
				proc.ProcessImage(parsedRequest, &inputStream)

				inputStream.Wait()
				g.Assert(inputStream.SafelyGetDataSegment(0)).Equal(testData)
			})

			g.It("Should return error when http request returns error", func() {
				config := Config{ImaginaryServiceURL: "http://localhost:3000"}
				testData := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
				inputStream := mock_hub.NewMockTestingDataStreamInput(g, [][]byte{testData}, nil, nil)
				parsedRequest := processor.ParsedRequest{
					Signature:         "abc",
					SourceImageURL:    "http://google.com/image.jpg",
					ProcessorEndpoint: "/crop",
					ProcessingParams: map[string][]string{
						"width":  {"500"},
						"height": {"500"},
					},
				}
				requestMaker := testReqFunc(200, nil, io.ErrUnexpectedEOF, nil, noAssertions)

				proc := Processor{config, requestMaker}
				_, err := proc.ProcessImage(parsedRequest, &inputStream)

				g.Assert(err).Equal(io.ErrUnexpectedEOF)
			})

			g.It("Should return error if imaginary service responds with not-200 error code", func() {
				config := Config{ImaginaryServiceURL: "http://localhost:3000"}
				testData := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
				inputStream := mock_hub.NewMockTestingDataStreamInput(g, [][]byte{testData}, nil, nil)
				parsedRequest := processor.ParsedRequest{
					Signature:         "abc",
					SourceImageURL:    "http://google.com/image.jpg",
					ProcessorEndpoint: "/crop",
					ProcessingParams: map[string][]string{
						"width":  {"500"},
						"height": {"500"},
					},
				}
				requestMaker := testReqFunc(500, testData, nil, nil, noAssertions)

				proc := Processor{config, requestMaker}
				_, err := proc.ProcessImage(parsedRequest, &inputStream)

				g.Assert(err).Equal(ErrResponseStatusNotOK)
			})

			g.It("Should close input data stream with error that ocurred while fetching given image", func() {
				config := Config{ImaginaryServiceURL: "http://localhost:3000"}
				testData := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
				inputStream := mock_hub.NewMockTestingDataStreamInput(g, nil, nil, nil)
				parsedRequest := processor.ParsedRequest{
					Signature:         "abc",
					SourceImageURL:    "http://google.com/image.jpg",
					ProcessorEndpoint: "/crop",
					ProcessingParams: map[string][]string{
						"width":  {"500"},
						"height": {"500"},
					},
				}
				requestMaker := testReqFunc(200, testData, nil, io.ErrUnexpectedEOF, noAssertions)

				proc := Processor{config, requestMaker}
				proc.ProcessImage(parsedRequest, &inputStream)

				inputStream.Wait()
				g.Assert(inputStream.ForwardedError).Equal(io.ErrUnexpectedEOF)
			})
		})
	})
}
