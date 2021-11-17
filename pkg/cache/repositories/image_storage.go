package cacherepositories

import (
	"context"
	"errors"
	"net/url"

	"github.com/minio/minio-go/v7"
	dbconnections "github.com/thebartekbanach/imcaxy/pkg/cache/repositories/connections"
	"github.com/thebartekbanach/imcaxy/pkg/hub"
)

type cachedImagesStorage struct {
	conn dbconnections.MinioBlockStorageConnection
}

var _ CachedImagesStorage = (*cachedImagesStorage)(nil)

func NewCachedImagesStorage(conn dbconnections.MinioBlockStorageConnection) CachedImagesStorage {
	return &cachedImagesStorage{conn}
}

func (s *cachedImagesStorage) Save(ctx context.Context, requestSignature, processorType, mimeType string, size int64, reader hub.DataStreamOutput) error {
	resourceID := s.makeResourceID(requestSignature, processorType)
	exists, err := s.conn.ObjectExists(ctx, resourceID)
	if err != nil {
		return err
	}
	if exists {
		return ErrImageAlreadyExists
	}

	defer reader.Close()
	return s.conn.PutObject(ctx, resourceID, size, mimeType, reader)
}

func (s *cachedImagesStorage) Get(ctx context.Context, requestSignature, processorType string, writer hub.DataStreamInput) error {
	resourceID := s.makeResourceID(requestSignature, processorType)
	reader, err := s.conn.GetObject(ctx, resourceID)
	if err != nil {
		return s.convertToKnownError(err)
	}

	if _, err := reader.Stat(); err != nil {
		return s.convertToKnownError(err)
	}

	go func() {
		_, err = writer.ReadFrom(reader)
		writer.Close(s.convertToKnownError(err))
		reader.Close()
	}()

	return nil
}

func (s *cachedImagesStorage) Delete(ctx context.Context, requestSignature, processorType string) error {
	resourceID := s.makeResourceID(requestSignature, processorType)
	exists, err := s.conn.ObjectExists(ctx, resourceID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrImageNotFound
	}

	return s.conn.DeleteObject(ctx, resourceID)
}

func (s *cachedImagesStorage) convertToKnownError(err error) error {
	if minio.ToErrorResponse(err).Code == "NoSuchKey" {
		return ErrImageNotFound
	}

	return err
}

func (s *cachedImagesStorage) makeResourceID(requestSignature, processorType string) string {
	return url.PathEscape(requestSignature) + "::" + url.PathEscape(processorType)
}

var (
	ErrImageAlreadyExists = errors.New("image already exists")
	ErrImageNotFound      = errors.New("image not found")
)
