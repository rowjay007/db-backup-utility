package storage

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3 struct {
	Client *minio.Client
	Bucket string
}

func NewS3(endpoint, region, bucket, accessKey, secretKey, sessionToken string, useSSL, forcePathStyle, insecure bool) (*S3, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:     credentials.NewStaticV4(accessKey, secretKey, sessionToken),
		Secure:    useSSL,
		Region:    region,
		Transport: transport,
		BucketLookup: func() minio.BucketLookupType {
			if forcePathStyle {
				return minio.BucketLookupPath
			}
			return minio.BucketLookupDNS
		}(),
	})
	if err != nil {
		return nil, err
	}
	return &S3{Client: client, Bucket: bucket}, nil
}

func (s *S3) Put(ctx context.Context, key string, reader io.Reader, size int64, metadata map[string]string) error {
	opts := minio.PutObjectOptions{UserMetadata: metadata}
	_, err := s.Client.PutObject(ctx, s.Bucket, key, reader, size, opts)
	return err
}

func (s *S3) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.Client.GetObject(ctx, s.Bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (s *S3) Stat(ctx context.Context, key string) (ObjectInfo, error) {
	stat, err := s.Client.StatObject(ctx, s.Bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return ObjectInfo{}, err
	}
	return ObjectInfo{Key: key, Size: stat.Size, Modified: stat.LastModified, ETag: stat.ETag, Metadata: stat.UserMetadata, IsManifest: strings.HasSuffix(key, ManifestSuffix)}, nil
}

func (s *S3) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	ch := s.Client.ListObjects(ctx, s.Bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true})
	infos := []ObjectInfo{}
	for obj := range ch {
		if obj.Err != nil {
			return nil, obj.Err
		}
		infos = append(infos, ObjectInfo{Key: obj.Key, Size: obj.Size, Modified: obj.LastModified, ETag: obj.ETag, IsManifest: strings.HasSuffix(obj.Key, ManifestSuffix)})
	}
	return infos, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	return s.Client.RemoveObject(ctx, s.Bucket, key, minio.RemoveObjectOptions{})
}

func (s *S3) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.Client.StatObject(ctx, s.Bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
