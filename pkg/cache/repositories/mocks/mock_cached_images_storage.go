package mock_cacherepositories

import (
	"bytes"
	context "context"
	"io"
	"sync"

	cacherepositories "github.com/thebartekbanach/imcaxy/pkg/cache/repositories"
	"github.com/thebartekbanach/imcaxy/pkg/hub"
)

type MockCachedImagesStorage struct {
	images map[string][]byte
	lock   sync.Mutex
	err    error
}

func NewMockCachedImagesStorage() *MockCachedImagesStorage {
	return &MockCachedImagesStorage{
		images: make(map[string][]byte),
		lock:   sync.Mutex{},
	}
}

func (s *MockCachedImagesStorage) InstantSave(requestSignature, processorType string, data []byte) {
	s.lock.Lock()
	defer s.lock.Unlock()

	resourceID := s.generateResourceID(requestSignature, processorType)
	s.images[resourceID] = data
}

func (s *MockCachedImagesStorage) ReturnError(err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.err = err
}

func (s *MockCachedImagesStorage) Save(ctx context.Context, requestSignature, processorType, mimeType string, size int64, reader hub.DataStreamOutput) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.err != nil {
		return s.err
	}

	resourceID := s.generateResourceID(requestSignature, processorType)
	if _, exists := s.images[resourceID]; exists {
		return cacherepositories.ErrImageAlreadyExists
	}

	buff := bytes.NewBuffer([]byte{})
	if _, err := reader.WriteTo(buff); err != io.EOF {
		return err
	}

	s.images[resourceID] = buff.Bytes()
	return nil
}

func (s *MockCachedImagesStorage) Get(ctx context.Context, requestSignature, processorType string, writer hub.DataStreamInput) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.err != nil {
		return s.err
	}

	resourceID := s.generateResourceID(requestSignature, processorType)
	if data, ok := s.images[resourceID]; ok {
		buff := bytes.NewBuffer(data)
		_, err := writer.ReadFrom(buff)
		writer.Close(err)
		return err
	}

	return cacherepositories.ErrImageNotFound
}

func (s *MockCachedImagesStorage) Delete(ctx context.Context, requestSignature, processorType string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.err != nil {
		return s.err
	}

	resourceID := s.generateResourceID(requestSignature, processorType)
	if _, ok := s.images[resourceID]; ok {
		delete(s.images, resourceID)
		return nil
	}

	return cacherepositories.ErrImageNotFound
}

func (s *MockCachedImagesStorage) Exists(requestSignature, processorType string) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	resourceID := s.generateResourceID(requestSignature, processorType)
	_, ok := s.images[resourceID]
	return ok
}

func (s *MockCachedImagesStorage) generateResourceID(requestSignature, processorType string) string {
	return requestSignature + "::" + processorType
}
