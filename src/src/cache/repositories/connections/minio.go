package connections

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Necessary Minio connection information
// provided through environment variables
type MinioEnv struct {
	MinioEndpointURL     string
	MinioAccessKeyID     string
	MinioSecretAccessKey string
}

// MinioConnectionFactory interface
type IMinioConnectionFactory interface {
	CreateMinioConnection() (*minio.Client, error)
}

// Minio connection factory, creates new
// Minio connections using connection info
// provided through environment variables
type MinioConnectionFactory struct {
	endpointURL     string
	accessKeyID     string
	secretAccessKey string
}

// Creates new Minio connection and returns it,
// or error when connection cannot be established
func (factory MinioConnectionFactory) CreateMinioConnection() (*minio.Client, error) {
	return minio.New(factory.endpointURL, &minio.Options{
		Creds: credentials.NewStaticV4(factory.accessKeyID, factory.secretAccessKey, ""),
	})
}

// Creates new MinioConnectionFactory using provided
// MinioEnv env information
func CreateMinioConnectionFactory(env MinioEnv) IMinioConnectionFactory {
	return MinioConnectionFactory{
		endpointURL:     env.MinioEndpointURL,
		accessKeyID:     env.MinioAccessKeyID,
		secretAccessKey: env.MinioSecretAccessKey,
	}
}
