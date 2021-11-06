package cacherepositories

import (
	"context"
	"errors"

	dbconnections "github.com/thebartekbanach/imcaxy/pkg/cache/repositories/connections"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type CachedImageModel struct {
	RawRequest       string `json:"rawRequest" bson:"rawRequest"`
	RequestSignature string `json:"requestSignature" bson:"requestSignature"`

	ProcessorType     string `json:"processorType" bson:"processorType"`
	ProcessorEndpoint string `json:"processorEndpoint" bson:"processorEndpoint"`

	MimeType         string              `json:"mimeType" bson:"mimeType"`
	SourceImageURL   string              `json:"sourceImageURL" bson:"sourceImageURL"`
	ProcessingParams map[string][]string `json:"processingParams" bson:"processingParams"`
}

type cachedImagesRepository struct {
	conn dbconnections.CacheDBConnection
}

var _ CachedImagesRepository = (*cachedImagesRepository)(nil)

func NewCachedImagesRepository(conn dbconnections.CacheDBConnection) CachedImagesRepository {
	return &cachedImagesRepository{conn}
}

func (repo *cachedImagesRepository) CreateCachedImageInfo(ctx context.Context, info CachedImageModel) error {
	collection := repo.conn.Collection("cachedImages")

	result := collection.FindOne(ctx, bson.M{"requestSignature": info.RequestSignature, "processorType": info.ProcessorType})
	if result.Err() != mongo.ErrNoDocuments {
		return ErrCachedImageAlreadyExists
	}

	_, err := collection.InsertOne(ctx, info)
	return err
}

func (repo *cachedImagesRepository) DeleteCachedImageInfo(ctx context.Context, requestSignature, processorType string) error {
	collection := repo.conn.Collection("cachedImages")

	result, err := collection.DeleteOne(ctx, bson.M{"requestSignature": requestSignature, "processorType": processorType})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return ErrCachedImageNotFound
	}

	return nil
}

func (repo *cachedImagesRepository) GetCachedImageInfo(ctx context.Context, requestSignature, processorType string) (CachedImageModel, error) {
	collection := repo.conn.Collection("cachedImages")

	var info CachedImageModel
	filter := bson.M{"requestSignature": requestSignature, "processorType": processorType}
	if err := collection.FindOne(ctx, filter).Decode(&info); err != nil {
		if err == mongo.ErrNoDocuments {
			return info, ErrCachedImageNotFound
		}

		return CachedImageModel{}, err
	}

	return info, nil
}

var (
	ErrCachedImageNotFound      = errors.New("cached image not found")
	ErrCachedImageAlreadyExists = errors.New("cached image already exists")
)
