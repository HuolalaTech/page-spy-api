package storage

import (
	"bytes"
	"fmt"
	"io"
	"path"

	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/aws/aws-sdk-go/aws"
	awsErr "github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type RemoteApi struct {
	config *config.StorageConfig
}

func (a *RemoteApi) joinPath(id string) string {
	return path.Join(a.config.BaseDir, a.config.GetLogDir(), id)
}

func (a *RemoteApi) newSession() (*session.Session, error) {
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

func (a *RemoteApi) Save(path string, data io.ReadSeeker) error {
	session, err := a.newSession()
	if err != nil {
		return err
	}
	svc := s3.New(session)

	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(a.config.Bucket),
		Key:    aws.String(path),
		Body:   data,
		ACL:    aws.String("private"),
	})

	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}
	return nil
}

func (a *RemoteApi) SaveLog(log *LogFile) error {
	err := a.Save(a.joinPath(log.FileId), bytes.NewReader(log.UpdateFile))

	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}
	return nil
}

func (a *RemoteApi) Exist(path string) (bool, error) {
	session, err := a.newSession()
	if err != nil {
		return false, err
	}
	svc := s3.New(session)

	_, err = svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(a.config.Bucket),
		Key:    aws.String(path),
	})

	if err != nil {
		s3Error, ok := err.(awsErr.Error)
		if ok && (s3Error.Code() == s3.ErrCodeNoSuchKey || s3Error.Code() == "NotFound") {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (a *RemoteApi) Get(path string) (io.ReadCloser, int64, error) {
	session, err := a.newSession()
	if err != nil {
		return nil, 0, err
	}
	svc := s3.New(session)

	result, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(a.config.Bucket),
		Key:    aws.String(path),
	})

	if err != nil {
		return nil, 0, err
	}

	return result.Body, *result.ContentLength, nil
}

func (a *RemoteApi) ExistLog(fileId string) (bool, error) {
	return a.Exist(a.joinPath(fileId))
}

func (a *RemoteApi) GetLog(fileId string) (*LogFile, error) {
	body, size, err := a.Get(a.joinPath(fileId))
	if err != nil {
		return nil, err
	}

	return &LogFile{
		FileId:    fileId,
		Size:      size,
		FileSteam: body,
	}, nil
}

func (a *RemoteApi) RemoveLog(fileId string) error {
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
