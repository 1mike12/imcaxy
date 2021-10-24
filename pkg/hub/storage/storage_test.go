package datahubstorage_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	. "github.com/franela/goblin"
	datahubstorage "github.com/thebartekbanach/imcaxy/pkg/hub/storage"
)

type readerExecutionFeedbackResult struct {
	data           []byte
	forwardedError error
}

type readerExecutionFeedback struct {
	streamID string
	channel  <-chan readerExecutionFeedbackResult
}

type readerExecutionResult struct {
	readerExecutionFeedbackResult
	streamID string
}

func readDataAndSendFeedback(streamID string, reader datahubstorage.StreamReader, result chan readerExecutionFeedbackResult) {
	defer reader.Close()

	var off int64 = 0
	data := make([]byte, 0)

	for {
		buffer := make([]byte, 6)
		n, err := reader.ReadAt(buffer, off)
		off += int64(n)

		if err != nil && err != io.EOF {
			result <- readerExecutionFeedbackResult{
				data,
				err,
			}
			return
		}

		data = append(data, buffer[:n]...)

		if err == io.EOF {
			result <- readerExecutionFeedbackResult{
				data,
				nil,
			}
			return
		}
	}
}

func createReadersExecuteTestAndGetResults(g *G, ctx context.Context, storage *datahubstorage.Storage, streams []string, testExecutor func(streams []string)) []readerExecutionResult {
	feedbacks := make([]readerExecutionFeedback, 0)
	for _, stream := range streams {
		feedbackChannel := make(chan readerExecutionFeedbackResult, 1)
		feedback := readerExecutionFeedback{stream, feedbackChannel}
		feedbacks = append(feedbacks, feedback)

		reader, err := storage.GetStreamReader(stream)
		if err != nil {
			g.Errorf("error ocurred while getting stream reader: %v", err)
		}
		g.Assert(reader).IsNotNil()

		go readDataAndSendFeedback(stream, reader, feedbackChannel)
	}

	testExecutor(streams)

	results := make([]readerExecutionResult, 0)
	for _, feedback := range feedbacks {
		select {
		case <-ctx.Done():
			return results

		case <-time.After(time.Second):
			noDataResult := readerExecutionFeedbackResult{
				data:           []byte{},
				forwardedError: context.DeadlineExceeded,
			}
			result := readerExecutionResult{noDataResult, feedback.streamID}
			results = append(results, result)

		case feedbackResult := <-feedback.channel:
			result := readerExecutionResult{feedbackResult, feedback.streamID}
			results = append(results, result)
		}
	}

	return results
}

func newRunningStorage() (context.Context, context.CancelFunc, *datahubstorage.Storage) {
	ctx, cancel := context.WithCancel(context.Background())
	storage := datahubstorage.NewStorage()
	go storage.StartMonitors(ctx)

	return ctx, cancel, &storage
}

func TestStorage(t *testing.T) {
	g := Goblin(t)

	g.Describe("Storage", func() {
		g.It("Should create stream without error", func() {
			_, cancel, storage := newRunningStorage()
			defer cancel()

			err := storage.Create("test")

			g.Assert(err).IsNil()
		})

		g.It("Should return error when creating and stream already exists", func() {
			_, cancel, storage := newRunningStorage()
			defer cancel()

			storage.Create("test")
			err := storage.Create("test")

			g.Assert(err).Equal(datahubstorage.ErrStreamAlreadyExists)
		})

		g.It("Should correctly write data to readers", func() {
			ctx, cancel, storage := newRunningStorage()
			defer cancel()
			testData1 := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
			testData2 := []byte{0x6, 0x5, 0x4, 0x3, 0x2, 0x1}

			storage.Create("test1")
			storage.Create("test2")
			results := createReadersExecuteTestAndGetResults(g, ctx, storage, []string{"test1", "test1", "test2"}, func(_ []string) {
				storage.Write("test1", testData1[:3])
				storage.Write("test2", testData2[:3])

				storage.Write("test1", testData1[3:])
				storage.Write("test2", testData2[3:])

				storage.Close("test1", nil)
				storage.Close("test2", nil)
			})

			for _, result := range results {
				g.Assert(result.forwardedError).IsNil()

				if result.streamID == "test1" {
					g.Assert(result.data).Equal(testData1, "stream test1 data does not contain expected result data")
					continue
				}

				g.Assert(result.data).Equal(testData2, "stream test2 data does not contain expected result data")
			}
		})

		g.It("Should return error if trying to write data to unknown stream", func() {
			_, cancel, storage := newRunningStorage()
			defer cancel()

			_, err := storage.Write("unknown", []byte{0x0})

			g.Assert(err).Equal(datahubstorage.ErrUnknownStream)
		})

		g.Xit("Should return error if trying to write closed but not disposed stream", func() {
			_, cancel, storage := newRunningStorage()
			defer cancel()

			storage.Create("test")
			storage.Close("test", nil)
			_, err := storage.Write("test", []byte{0x0})

			g.Assert(err).Equal(datahubstorage.ErrStreamAlreadyClosed)
		})

		g.It("Should close stream and forward given error", func() {
			ctx, cancel, storage := newRunningStorage()
			defer cancel()
			testData1 := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
			testData2 := []byte{0x6, 0x5, 0x4, 0x3, 0x2, 0x1}
			testError := errors.New("test error")

			storage.Create("test1")
			storage.Create("test2")
			results := createReadersExecuteTestAndGetResults(g, ctx, storage, []string{"test1", "test1", "test2", "test2"}, func(_ []string) {
				storage.Write("test1", testData1[:3])
				storage.Write("test2", testData2[:3])

				storage.Write("test1", testData1[3:])
				storage.Write("test2", testData2[3:])

				storage.Close("test1", testError)
				storage.Close("test2", nil)
			})

			for _, result := range results {
				if result.streamID == "test1" {
					g.Assert(result.forwardedError).Equal(testError, "stream test1 was closed with wrong error value")
					continue
				}

				g.Assert(result.forwardedError).IsNil("stream test2 was closed with non-nil error value")
			}
		})

		g.It("Should return close error if stream to close is unknown", func() {
			_, cancel, storage := newRunningStorage()
			defer cancel()

			err := storage.Close("unknown", nil)

			g.Assert(err).Equal(datahubstorage.ErrUnknownStream)
		})

		g.It("Should dispose resource when resource was closed and all readers also was closed", func() {
			ctx, cancel, storage := newRunningStorage()
			defer cancel()
			testData1 := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
			testData2 := []byte{0x6, 0x5, 0x4, 0x3, 0x2, 0x1}

			storage.Create("test1")
			storage.Create("test2")
			createReadersExecuteTestAndGetResults(g, ctx, storage, []string{"test1", "test1", "test2", "test2"}, func(_ []string) {
				storage.Write("test1", testData1[:3])
				storage.Write("test2", testData2[:3])

				storage.Write("test1", testData1[3:])
				storage.Write("test2", testData2[3:])

				storage.Close("test1", nil)
				storage.Close("test2", nil)
			})

			if _, err := storage.GetStreamReader("test1"); err != datahubstorage.ErrUnknownStream {
				g.Errorf("stream test1 was not disposed")
			}

			if _, err := storage.GetStreamReader("test2"); err != datahubstorage.ErrUnknownStream {
				g.Errorf("stream test2 was not disposed")
			}
		})
	})
}
