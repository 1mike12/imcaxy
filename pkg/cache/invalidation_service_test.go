package cache_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/thebartekbanach/imcaxy/pkg/cache"
	mock_cache "github.com/thebartekbanach/imcaxy/pkg/cache/mocks"
	cacherepositories "github.com/thebartekbanach/imcaxy/pkg/cache/repositories"
	mock_cacherepositories "github.com/thebartekbanach/imcaxy/pkg/cache/repositories/mocks"
	"golang.org/x/net/context"
)

type invalidationEqMatcher struct {
	expected cacherepositories.InvalidationModel
}

func invalidationEq(expected cacherepositories.InvalidationModel) gomock.Matcher {
	return &invalidationEqMatcher{expected}
}

func getSecureString(s *string) string {
	if s == nil {
		return "nil"
	}

	return *s
}

func (m *invalidationEqMatcher) Matches(x interface{}) bool {
	invalidation, ok := x.(cacherepositories.InvalidationModel)
	if !ok {
		return false
	}

	return m.expected.ProjectName == invalidation.ProjectName &&
		m.expected.CommitHash == invalidation.CommitHash &&
		reflect.DeepEqual(m.expected.RequestedInvalidations, invalidation.RequestedInvalidations) &&
		reflect.DeepEqual(m.expected.DoneInvalidations, invalidation.DoneInvalidations) &&
		reflect.DeepEqual(m.expected.InvalidatedImages, invalidation.InvalidatedImages) &&
		getSecureString(m.expected.InvalidationError) == getSecureString(invalidation.InvalidationError)
}

func (m *invalidationEqMatcher) String() string {
	return fmt.Sprintf("expected invalidation to be: %v", m.expected)
}

func findDifferenceBetweenInvalidations(a cacherepositories.InvalidationModel, b cacherepositories.InvalidationModel) string {
	if a.ProjectName != b.ProjectName {
		return fmt.Sprintf("ProjectName: %s != %s", a.ProjectName, b.ProjectName)
	}

	if a.CommitHash != b.CommitHash {
		return fmt.Sprintf("CommitHash: %s != %s", a.CommitHash, b.CommitHash)
	}

	if !reflect.DeepEqual(a.RequestedInvalidations, b.RequestedInvalidations) {
		return fmt.Sprintf("RequestedInvalidations: %v != %v", a.RequestedInvalidations, b.RequestedInvalidations)
	}

	if !reflect.DeepEqual(a.DoneInvalidations, b.DoneInvalidations) {
		return fmt.Sprintf("DoneInvalidations: %v != %v", a.DoneInvalidations, b.DoneInvalidations)
	}

	if !reflect.DeepEqual(a.InvalidatedImages, b.InvalidatedImages) {
		return fmt.Sprintf("InvalidatedImages: %v != %v", a.InvalidatedImages, b.InvalidatedImages)
	}

	if a.InvalidationError != b.InvalidationError {
		return fmt.Sprintf("InvalidationError: %v != %v", a.InvalidationError, b.InvalidationError)
	}

	return ""
}

func TestInvalidationService_ShouldCorrectlyGetLastKnownInvalidationForGivenProject(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockInvalidationsRepository := mock_cacherepositories.NewMockInvalidationsRepository(mockCtrl)
	mockCacheService := mock_cache.NewMockCacheService(mockCtrl)

	invalidation := cacherepositories.InvalidationModel{ProjectName: "project", CommitHash: "hash"}
	mockInvalidationsRepository.EXPECT().GetLatestInvalidation(gomock.Any(), "project").Return(invalidation, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	invalidationService := cache.NewInvalidationService(mockInvalidationsRepository, mockCacheService)
	lastInvalidation, _ := invalidationService.GetLastKnownInvalidation(ctx, "project")

	if !reflect.DeepEqual(lastInvalidation, invalidation) {
		t.Errorf("Expected last invalidation %#v, got %#v", invalidation, lastInvalidation)
	}
}

func TestInvalidationService_GetLastKnownInvalidationForGivenProjectShouldReturnErrorReturnedByRepository(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockInvalidationsRepository := mock_cacherepositories.NewMockInvalidationsRepository(mockCtrl)
	mockCacheService := mock_cache.NewMockCacheService(mockCtrl)

	mockInvalidationsRepository.EXPECT().GetLatestInvalidation(gomock.Any(), "project").Return(cacherepositories.InvalidationModel{}, cacherepositories.ErrProjectNameNotAllowed)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	invalidationService := cache.NewInvalidationService(mockInvalidationsRepository, mockCacheService)
	_, err := invalidationService.GetLastKnownInvalidation(ctx, "project")

	if err != cacherepositories.ErrProjectNameNotAllowed {
		t.Errorf("Expected to get ErrProjectNameNotAllowed error, but got: %v", err)
	}
}

func TestInvalidationService_GetLastKnownInvalidationShouldReturnErrProjectNameNotAllowedIfProvidedProjectNameIsEmpty(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockInvalidationsRepository := mock_cacherepositories.NewMockInvalidationsRepository(mockCtrl)
	mockCacheService := mock_cache.NewMockCacheService(mockCtrl)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	invalidationService := cache.NewInvalidationService(mockInvalidationsRepository, mockCacheService)
	_, err := invalidationService.GetLastKnownInvalidation(ctx, "")

	if err != cacherepositories.ErrProjectNameNotAllowed {
		t.Errorf("Expected to get ErrProjectNameNotAllowed error, but got: %v", err)
	}
}

func TestInvalidationService_ShouldCorrectlyInvalidateImagesAndSaveResult(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockInvalidationsRepository := mock_cacherepositories.NewMockInvalidationsRepository(mockCtrl)
	mockCacheService := mock_cache.NewMockCacheService(mockCtrl)

	invalidatedEntries := []cacherepositories.CachedImageModel{
		{RawRequest: "request1", RequestSignature: "signature1"},
		{RawRequest: "request2", RequestSignature: "signature2"},
	}
	mockCacheService.EXPECT().InvalidateAllEntriesForURL(gomock.Any(), "image").Return(invalidatedEntries, nil)

	invalidation := cacherepositories.InvalidationModel{
		ProjectName:            "project",
		CommitHash:             "hash",
		RequestedInvalidations: []string{"image"},
		DoneInvalidations:      []string{"image"},
		InvalidatedImages:      invalidatedEntries,
		InvalidationError:      nil,
	}
	mockInvalidationsRepository.EXPECT().CreateInvalidation(gomock.Any(), invalidationEq(invalidation)).Return(nil)

	invalidationService := cache.NewInvalidationService(mockInvalidationsRepository, mockCacheService)
	result, err := invalidationService.Invalidate(context.Background(), "project", "hash", []string{"image"})

	if err != nil {
		t.Errorf("Expected no invalidation error, but got: %v", err)
	}

	if !invalidationEq(invalidation).Matches(result) {
		t.Errorf("Expected invalidation differs: %s", findDifferenceBetweenInvalidations(invalidation, result))
	}
}

func TestInvalidationService_ShouldCorrectlyInvalidateMultipleImagesAndSaveResult(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockInvalidationsRepository := mock_cacherepositories.NewMockInvalidationsRepository(mockCtrl)
	mockCacheService := mock_cache.NewMockCacheService(mockCtrl)

	invalidatedEntriesForImage1 := []cacherepositories.CachedImageModel{
		{RawRequest: "request1", RequestSignature: "signature1"},
		{RawRequest: "request2", RequestSignature: "signature2"},
	}
	mockCacheService.EXPECT().InvalidateAllEntriesForURL(gomock.Any(), "image1").Return(invalidatedEntriesForImage1, nil)

	invalidatedEntriesForImage2 := []cacherepositories.CachedImageModel{
		{RawRequest: "request3", RequestSignature: "signature3"},
		{RawRequest: "request4", RequestSignature: "signature4"},
	}
	mockCacheService.EXPECT().InvalidateAllEntriesForURL(gomock.Any(), "image2").Return(invalidatedEntriesForImage2, nil)

	invalidation := cacherepositories.InvalidationModel{
		ProjectName:            "project",
		CommitHash:             "hash",
		RequestedInvalidations: []string{"image1", "image2"},
		DoneInvalidations:      []string{"image1", "image2"},
		InvalidatedImages:      append(invalidatedEntriesForImage1, invalidatedEntriesForImage2...),
		InvalidationError:      nil,
	}
	mockInvalidationsRepository.EXPECT().CreateInvalidation(gomock.Any(), invalidationEq(invalidation)).Return(nil)

	invalidationService := cache.NewInvalidationService(mockInvalidationsRepository, mockCacheService)
	invalidationService.Invalidate(context.Background(), "project", "hash", []string{"image1", "image2"})
}

func TestInvalidationService_ShouldProvideCorrectInformationAboutDoneInvalidations(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockInvalidationsRepository := mock_cacherepositories.NewMockInvalidationsRepository(mockCtrl)
	mockCacheService := mock_cache.NewMockCacheService(mockCtrl)

	invalidatedEntriesForImage1 := []cacherepositories.CachedImageModel{
		{RawRequest: "request1", RequestSignature: "signature1"},
		{RawRequest: "request2", RequestSignature: "signature2"},
	}
	mockCacheService.EXPECT().InvalidateAllEntriesForURL(gomock.Any(), "image1").Return(invalidatedEntriesForImage1, nil)

	invalidationErrorText := "invalidation error"
	invalidationError := errors.New(invalidationErrorText)
	invalidatedEntriesForImage2 := []cacherepositories.CachedImageModel{
		{RawRequest: "request3", RequestSignature: "signature3"},
		{RawRequest: "request4", RequestSignature: "signature4"},
	}
	mockCacheService.EXPECT().InvalidateAllEntriesForURL(gomock.Any(), "image2").Return(invalidatedEntriesForImage2, invalidationError)

	invalidation := cacherepositories.InvalidationModel{
		ProjectName:            "project",
		CommitHash:             "hash",
		RequestedInvalidations: []string{"image1", "image2"},
		DoneInvalidations:      []string{"image1"},
		InvalidatedImages:      append(invalidatedEntriesForImage1, invalidatedEntriesForImage2...),
		InvalidationError:      &invalidationErrorText,
	}
	mockInvalidationsRepository.EXPECT().CreateInvalidation(gomock.Any(), invalidationEq(invalidation)).Return(nil)

	invalidationService := cache.NewInvalidationService(mockInvalidationsRepository, mockCacheService)
	invalidationService.Invalidate(context.Background(), "project", "hash", []string{"image1", "image2"})
}

func TestInvalidationService_ShouldSaveInvalidationResultWithErrorReturnedByCacheService(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockInvalidationsRepository := mock_cacherepositories.NewMockInvalidationsRepository(mockCtrl)
	mockCacheService := mock_cache.NewMockCacheService(mockCtrl)

	invalidationErrorText := "some error"
	invalidationError := errors.New(invalidationErrorText)

	invalidatedEntries := []cacherepositories.CachedImageModel{
		{RawRequest: "request1", RequestSignature: "signature1"},
		{RawRequest: "request2", RequestSignature: "signature2"},
	}
	mockCacheService.EXPECT().InvalidateAllEntriesForURL(gomock.Any(), "image").Return(invalidatedEntries, invalidationError)

	invalidation := cacherepositories.InvalidationModel{
		ProjectName:            "project",
		CommitHash:             "hash",
		RequestedInvalidations: []string{"image"},
		DoneInvalidations:      []string{},
		InvalidatedImages:      invalidatedEntries,
		InvalidationError:      &invalidationErrorText,
	}
	mockInvalidationsRepository.EXPECT().CreateInvalidation(gomock.Any(), invalidationEq(invalidation)).Return(nil)

	invalidationService := cache.NewInvalidationService(mockInvalidationsRepository, mockCacheService)
	invalidationService.Invalidate(context.Background(), "project", "hash", []string{"image"})
}

func TestInvalidationService_ShouldReturnErrorReturnedByCacheService(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockInvalidationsRepository := mock_cacherepositories.NewMockInvalidationsRepository(mockCtrl)
	mockCacheService := mock_cache.NewMockCacheService(mockCtrl)

	invalidationErrorText := "some error"
	invalidationError := errors.New(invalidationErrorText)

	invalidatedEntries := []cacherepositories.CachedImageModel{
		{RawRequest: "request1", RequestSignature: "signature1"},
		{RawRequest: "request2", RequestSignature: "signature2"},
	}
	mockCacheService.EXPECT().InvalidateAllEntriesForURL(gomock.Any(), "image").Return(invalidatedEntries, invalidationError)

	invalidation := cacherepositories.InvalidationModel{
		ProjectName: "project", CommitHash: "hash",
		RequestedInvalidations: []string{"image"},
		DoneInvalidations:      []string{},
		InvalidatedImages:      invalidatedEntries,
		InvalidationError:      &invalidationErrorText,
	}
	mockInvalidationsRepository.EXPECT().CreateInvalidation(gomock.Any(), invalidationEq(invalidation)).Return(nil)

	invalidationService := cache.NewInvalidationService(mockInvalidationsRepository, mockCacheService)
	_, err := invalidationService.Invalidate(context.Background(), "project", "hash", []string{"image"})

	if err != invalidationError {
		t.Errorf("Expected to return error returned by InvalidateAllEntriesForURL, got: %v", err)
	}
}

func TestInvalidationService_ShouldReturnErrorReturnedByInvalidationsRepository(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockInvalidationsRepository := mock_cacherepositories.NewMockInvalidationsRepository(mockCtrl)
	mockCacheService := mock_cache.NewMockCacheService(mockCtrl)

	invalidationErrorText := "some error"
	invalidationError := errors.New(invalidationErrorText)

	invalidatedEntries := []cacherepositories.CachedImageModel{
		{RawRequest: "request1", RequestSignature: "signature1"},
		{RawRequest: "request2", RequestSignature: "signature2"},
	}
	mockCacheService.EXPECT().InvalidateAllEntriesForURL(gomock.Any(), "image").Return(invalidatedEntries, invalidationError)

	invalidation := cacherepositories.InvalidationModel{
		ProjectName: "project", CommitHash: "hash",
		RequestedInvalidations: []string{"image"},
		DoneInvalidations:      []string{},
		InvalidatedImages:      invalidatedEntries,
		InvalidationError:      &invalidationErrorText,
	}
	creationError := errors.New("some error")
	mockInvalidationsRepository.EXPECT().CreateInvalidation(gomock.Any(), invalidationEq(invalidation)).Return(creationError)

	invalidationService := cache.NewInvalidationService(mockInvalidationsRepository, mockCacheService)
	_, err := invalidationService.Invalidate(context.Background(), "project", "hash", []string{"image"})

	if err != creationError {
		t.Errorf("Expected to return error returned by CreateInvalidation, got: %v", err)
	}
}

func TestInvalidationService_InvalidateShouldReturnErrProjectNameNotAllowedIfProvidedProjectNameIsEmpty(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockInvalidationsRepository := mock_cacherepositories.NewMockInvalidationsRepository(mockCtrl)
	mockCacheService := mock_cache.NewMockCacheService(mockCtrl)

	invalidationService := cache.NewInvalidationService(mockInvalidationsRepository, mockCacheService)
	_, err := invalidationService.Invalidate(context.Background(), "", "hash", []string{"image"})

	if err != cacherepositories.ErrProjectNameNotAllowed {
		t.Errorf("Expected to get ErrProjectNameNotAllowed error, but got: %v", err)
	}
}

func TestInvalidationService_InvalidateShouldReturnErrCommitHashNotAllowedIfProvidedProjectNameIsEmpty(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockInvalidationsRepository := mock_cacherepositories.NewMockInvalidationsRepository(mockCtrl)
	mockCacheService := mock_cache.NewMockCacheService(mockCtrl)

	invalidationService := cache.NewInvalidationService(mockInvalidationsRepository, mockCacheService)
	_, err := invalidationService.Invalidate(context.Background(), "project", "", []string{"image"})

	if err != cacherepositories.ErrCommitHashNotAllowed {
		t.Errorf("Expected to get ErrCommitHashNotAllowed error, but got: %v", err)
	}
}
