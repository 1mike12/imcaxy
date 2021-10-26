package hub

import (
	"context"
	"testing"

	. "github.com/franela/goblin"
	"github.com/golang/mock/gomock"
	mock_hub "github.com/thebartekbanach/imcaxy/pkg/hub/mocks"
	datahubstorage "github.com/thebartekbanach/imcaxy/pkg/hub/storage"
)

func newRunningDataHubAndStorage(g *G) (DataHub, *mock_hub.MockStorageAdapter, context.CancelFunc) {
	mockCtrl := gomock.NewController(g)
	ctx, cancel := context.WithCancel(context.Background())

	finishAndCancel := func() {
		cancel()
		mockCtrl.Finish()
	}

	mockStorage := mock_hub.NewMockStorageAdapter(mockCtrl)
	mockStorage.EXPECT().StartMonitors(gomock.Any())

	hub := NewDataHub(mockStorage)
	hub.StartMonitors(ctx)

	return hub, mockStorage, finishAndCancel
}

func TestDataHub(t *testing.T) {
	g := Goblin(t)

	g.Describe("DataHub", func() {
		g.It("Should correctly create stream", func() {
			hub, mockStorage, finish := newRunningDataHubAndStorage(g)
			defer finish()
			mockStorage.EXPECT().Create("test").Times(1)

			hub.CreateStream("test")
		})

		g.It("Should forward stream creation error", func() {
			hub, mockStorage, finish := newRunningDataHubAndStorage(g)
			defer finish()
			mockStorage.EXPECT().Create("test").Return(datahubstorage.ErrStreamAlreadyExists)

			_, err := hub.CreateStream("test")

			g.Assert(err).Equal(datahubstorage.ErrStreamAlreadyExists)
		})

		g.It("Should return stream output for given stream", func() {
			hub, mockStorage, finish := newRunningDataHubAndStorage(g)
			defer finish()
			mockStorage.EXPECT().GetStreamReader("test").Return(nil, nil).Times(1)

			hub.GetStreamOutput("test")
		})

		g.It("Should return error when trying to get unknown stream output", func() {
			hub, mockStorage, finish := newRunningDataHubAndStorage(g)
			defer finish()
			mockStorage.EXPECT().GetStreamReader("test").Return(nil, datahubstorage.ErrUnknownStream)

			_, err := hub.GetStreamOutput("test")

			g.Assert(err).Equal(datahubstorage.ErrUnknownStream)
		})
	})
}
