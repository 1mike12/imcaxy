package cache

import (
	"context"
	"errors"

	cacherepositories "github.com/thebartekbanach/imcaxy/pkg/cache/repositories"
	"github.com/thebartekbanach/imcaxy/pkg/hub"
)

type cacheService struct {
	imagesRepository cacherepositories.CachedImagesRepository
	imagesStorage    cacherepositories.CachedImagesStorage
}

func NewCacheService(
	imagesRepository cacherepositories.CachedImagesRepository,
	imagesStorage cacherepositories.CachedImagesStorage,
) CacheService {
	return &cacheService{
		imagesRepository,
		imagesStorage,
	}
}

func (s *cacheService) Get(ctx context.Context, requestSignature, processorType string, w hub.DataStreamInput) error {
	if err := s.imagesStorage.Get(ctx, requestSignature, processorType, w); err != nil {
		if err == cacherepositories.ErrImageNotFound {
			w.Close(ErrEntryNotFound)
			return ErrEntryNotFound
		}

		w.Close(err)
		return err
	}

	w.Close(nil)
	return nil
}

func (s *cacheService) Save(ctx context.Context, imageInfo cacherepositories.CachedImageModel, r hub.DataStreamOutput) error {
	defer r.Close()

	if err := s.imagesRepository.CreateCachedImageInfo(ctx, imageInfo); err != nil {
		if err == cacherepositories.ErrCachedImageAlreadyExists {
			return ErrEntryAlreadyExists
		}

		return err
	}

	if err := s.imagesStorage.Save(ctx, imageInfo.RequestSignature, imageInfo.ProcessorType, imageInfo.MimeType, imageInfo.ImageSize, r); err != nil {
		s.imagesRepository.DeleteCachedImageInfo(ctx, imageInfo.RequestSignature, imageInfo.ProcessorType)
		s.imagesStorage.Delete(ctx, imageInfo.RequestSignature, imageInfo.ProcessorType)
		return err
	}

	return nil
}

func (s *cacheService) InvalidateAllEntriesForURL(ctx context.Context, sourceImageURL string) (removedEntries []cacherepositories.CachedImageModel, err error) {
	entries, err := s.imagesRepository.GetCachedImageInfosOfSource(ctx, sourceImageURL)
	if err != nil {
		return
	}

	for _, entry := range entries {
		err = s.imagesRepository.DeleteCachedImageInfo(ctx, entry.RequestSignature, entry.ProcessorType)
		if err != nil {
			return
		}

		err = s.imagesStorage.Delete(ctx, entry.RequestSignature, entry.ProcessorType)
		if err != nil {
			return
		}

		removedEntries = append(removedEntries, entry)
	}

	return
}

var (
	ErrEntryNotFound      = errors.New("entry not found")
	ErrEntryAlreadyExists = errors.New("entry already exists")
)
