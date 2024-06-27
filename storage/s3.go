package storage

import (
	"bytes"
	"fmt"
	"path"

	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3Api struct {
	config *config.StorageConfig
}

func (a *S3Api) joinPath(id string) string {
	return path.Join(a.config.BaseDir, id)
}

func (a *S3Api) newSession() (*session.Session, error) {
	config := a.config
	session, err := session.NewSession(&aws.Config{
		Region:      aws.String(config.Region),
		Credentials: credentials.NewStaticCredentials(config.KeyId, config.Secret, ""),
		Endpoint:    aws.String(config.Endpoint),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create sessionion: %w", err)
	}
	return session, nil
}

func (a *S3Api) SaveLog(log *LogFile) error {
	session, err := a.newSession()
	if err != nil {
		return err
	}
	svc := s3.New(session)

	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(a.config.Bucket),
		Key:    aws.String(a.joinPath(log.FileId)),
		Body:   bytes.NewReader(log.UpdateFile),
		ACL:    aws.String("private"),
	})

	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}
	return nil
}

func (a *S3Api) GetLog(fileId string) (*LogFile, error) {
	session, err := a.newSession()
	if err != nil {
		return nil, err
	}
	svc := s3.New(session)

	result, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(a.config.Bucket),
		Key:    aws.String(a.joinPath(fileId)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	return &LogFile{
		FileId:    fileId,
		FileSteam: result.Body,
	}, nil
}

func (a *S3Api) RemoveLog(fileId string) error {
	session, err := a.newSession()
	if err != nil {
		return err
	}
	svc := s3.New(session)

	_, err = svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(a.config.Bucket),
		Key:    aws.String(a.joinPath(fileId)),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	err = svc.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(a.config.Bucket),
		Key:    aws.String(a.joinPath(fileId)),
	})

	if err != nil {
		return fmt.Errorf("error occurred while waiting for object to be deleted: %w", err)
	}

	return nil
}
