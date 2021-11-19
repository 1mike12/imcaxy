package cacherepositories

import (
	"context"
	"errors"

	dbconnections "github.com/thebartekbanach/imcaxy/pkg/cache/repositories/connections"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type invalidationRepository struct {
	conn dbconnections.CacheDBConnection
}

var _ InvalidationsRepository = (*invalidationRepository)(nil)

func NewInvalidationsRepository(conn dbconnections.CacheDBConnection) InvalidationsRepository {
	return &invalidationRepository{conn}
}

func (r *invalidationRepository) CreateInvalidation(ctx context.Context, invalidation InvalidationModel) error {
	if invalidation.ProjectName == "" {
		return ErrProjectNameNotAllowed
	}

	if invalidation.CommitHash == "" {
		return ErrCommitHashNotAllowed
	}

	coll := r.conn.Collection("invalidations")
	_, err := coll.InsertOne(ctx, invalidation)
	return err
}

func (r *invalidationRepository) GetLatestInvalidation(ctx context.Context, projectName string) (InvalidationModel, error) {
	if projectName == "" {
		return InvalidationModel{}, ErrProjectNameNotAllowed
	}

	coll := r.conn.Collection("invalidations")
	opts := options.FindOne().SetSort(bson.D{{Key: "invalidationDate", Value: -1}})
	result := coll.FindOne(ctx, bson.D{{Key: "projectName", Value: projectName}}, opts)

	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return InvalidationModel{}, ErrProjectNotFound
		}

		return InvalidationModel{}, result.Err()
	}

	var invalidation InvalidationModel
	err := result.Decode(&invalidation)
	return invalidation, err
}

var (
	ErrCommitHashNotAllowed  = errors.New("this commit hash is not allowed")
	ErrProjectNameNotAllowed = errors.New("this project name is not allowed")
	ErrProjectNotFound       = errors.New("project not found")
)
