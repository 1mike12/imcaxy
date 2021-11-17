package cacherepositories

import (
	"context"
	"errors"
	"io"
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
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			writer.Close(ErrImageNotFound)
			return ErrImageNotFound
		}

		writer.Close(err)
		return err
	}
	defer reader.Close()

	_, err = writer.ReadFrom(reader)
	if err != nil && err != io.EOF {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			writer.Close(ErrImageNotFound)
			return ErrImageNotFound
		}

		writer.Close(err)
		return err
	}

	writer.Close(nil)
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

func (s *cachedImagesStorage) makeResourceID(requestSignature, processorType string) string {
	return url.PathEscape(requestSignature) + "::" + url.PathEscape(processorType)
}

var (
	ErrImageAlreadyExists = errors.New("image already exists")
	ErrImageNotFound      = errors.New("image not found")
)
