package minio

import (
	"bytes"
	"demo-storage/internal/app/repository"
	"demo-storage/internal/app/structs"
	"fmt"
	logdoc "github.com/LogDoc-org/logdoc-go-appender/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gurkankaymak/hocon"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/gommon/log"
	"mime/multipart"
	"time"
)

type MinioService struct {
	config         *hocon.Config
	access         string
	secret         string
	bucket         string
	fileRepository *repository.FileRepository
	RETRIES        int
}

type partUploadResult struct {
	completedPart *s3.CompletedPart
	err           error
}

func New(config *hocon.Config, access string, secret string, db *sqlx.DB) *MinioService {
	repo := repository.New(db)
	return &MinioService{
		config:         config,
		bucket:         config.GetString("minio.bucket"),
		access:         access,
		secret:         secret,
		fileRepository: repo,
		RETRIES:        config.GetInt("minio.retries"),
	}
}

func (s *MinioService) CreateMultipartSession(name string) (*s3.S3, *s3.CreateMultipartUploadOutput, error) {
	s3session := InitS3(s.secret, s.access, s.config)
	expiryDate := time.Now().AddDate(0, 0, 1)

	createdResp, err := s3session.CreateMultipartUpload(&s3.CreateMultipartUploadInput{
		Bucket:  aws.String(s.bucket),
		Key:     aws.String(name),
		Expires: &expiryDate,
	})
	if err != nil {
		return nil, nil, err
	}

	return s3session, createdResp, nil
}

func (s *MinioService) UploadPartToS3(s3connection *s3.S3, multipartSession *s3.CreateMultipartUploadOutput, fileBytes []byte, partNum int) structs.PartUploadResult {
	logger := logdoc.GetLogger()
	var try int
	logger.Debug(fmt.Sprintf(">> UploadPartToS3 > Uploading chunk:%v, part number:%d to S3", len(fileBytes), partNum))
	for try <= s.RETRIES {
		uploadRes, err := s3connection.UploadPart(&s3.UploadPartInput{
			Body:          bytes.NewReader(fileBytes),
			Bucket:        multipartSession.Bucket,
			Key:           multipartSession.Key,
			PartNumber:    aws.Int64(int64(partNum)),
			UploadId:      multipartSession.UploadId,
			ContentLength: aws.Int64(int64(len(fileBytes))),
		})
		if err != nil {
			logger.Error(">> UploadPartToS3 > err: ", err)
			if try == s.RETRIES {
				return structs.PartUploadResult{Err: err}
			}
			try++
			time.Sleep(time.Second * 15)
		} else {
			logger.Debug(fmt.Sprintf(">> Successfully Uploaded part with size:%d, part number:%d to S3", len(fileBytes), partNum))
			return structs.PartUploadResult{
				&s3.CompletedPart{
					ETag:       uploadRes.ETag,
					PartNumber: aws.Int64(int64(partNum)),
				}, nil,
			}
		}
	}
	return structs.PartUploadResult{}
}

func (s *MinioService) CompleteMultipartUpload(s3connection *s3.S3, uploadSession *s3.CreateMultipartUploadOutput, completedParts []*s3.CompletedPart) error {
	logger := logdoc.GetLogger()

	completed, err := s3connection.CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
		Bucket:   uploadSession.Bucket,
		Key:      uploadSession.Key,
		UploadId: uploadSession.UploadId,
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	if err != nil {
		logger.Error("Complete multipart upload failed: " + err.Error())
		return err
	}

	logger.Debug("Multipart completed successfully: " + completed.String())
	return nil
}

func (s *MinioService) AbortMultipartUpload(s3connection *s3.S3, uploadSession *s3.CreateMultipartUploadOutput) error {
	logger := logdoc.GetLogger()

	_, err := s3connection.AbortMultipartUpload(&s3.AbortMultipartUploadInput{
		Bucket:   uploadSession.Bucket,
		Key:      uploadSession.Key,
		UploadId: uploadSession.UploadId,
	})
	if err != nil {
		logger.Error("Abort multipart upload failed: " + err.Error())
		return err
	}
	return nil
}

func (s *MinioService) UploadFileAsBytes(fileHeader *structs.UploadHeader, data []byte) *s3.PutObjectOutput {
	logger := logdoc.GetLogger()

	f := s.fileRepository.FindFileByName(fileHeader.Filename)
	if f == nil || f.Id == 0 {
		logger.Warn("Файл " + fileHeader.Filename + " не найден в БД, создаем новый")
		s.fileRepository.CreateFile(fileHeader.Filename, "TODO")
	} else {
		s.fileRepository.UpdateFileStatus(fileHeader.Filename, "UPLOADING")
		time.Sleep(5 * time.Second)
	}

	// Устанавливаем параметры загрузки
	params := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fileHeader.Filename),
		Body:   bytes.NewReader(data),
	}

	s3Api := InitS3(s.secret, s.access, s.config)
	// Загружаем файл на Amazon S3
	uploaded, err := s3Api.PutObject(params)
	if err != nil {
		logger.Error("Unable to upload file,", err)
		s.fileRepository.UpdateFileStatus(fileHeader.Filename, "ERROR")
		return nil
	}

	logger.Debug("Successfully uploaded file to" + uploaded.String())
	go func() {
		_ = s.fileRepository.UpdateFileParams(fileHeader.Filename, "COMPLETED", "TODO")
	}()
	return uploaded
}

func (s *MinioService) UploadFile(fileHeader *multipart.FileHeader, filePath string) *s3.PutObjectOutput {
	logger := logdoc.GetLogger()
	src, err := fileHeader.Open()
	if err != nil {
		logger.Error("Ошибка открытия файла ", fileHeader.Filename)
		s.fileRepository.UpdateFileStatus(fileHeader.Filename, "ERROR")
		return nil
	}
	defer src.Close()

	f := s.fileRepository.FindFileByName(fileHeader.Filename)
	if f == nil || f.Id == 0 {
		logger.Warn("Файл " + fileHeader.Filename + " не найден в БД, создаем новый")
		s.fileRepository.CreateFile(fileHeader.Filename, filePath)
	} else {
		s.fileRepository.UpdateFileStatus(fileHeader.Filename, "UPLOADING")
		time.Sleep(5 * time.Second)
	}

	// Открываем файл, который хотим загрузить
	file, err := fileHeader.Open()
	if err != nil {
		logger.Error("Unable to open file, ", err.Error())
		s.fileRepository.UpdateFileStatus(fileHeader.Filename, "ERROR")
		return nil
	}
	defer file.Close()

	// Устанавливаем параметры загрузки
	params := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(fileHeader.Filename),
		Body:   file,
	}

	s3Api := InitS3(s.secret, s.access, s.config)
	// Загружаем файл на Amazon S3
	fupl, err := s3Api.PutObject(params)
	if err != nil {
		logger.Error("Unable to upload file,", err)
		s.fileRepository.UpdateFileStatus(fileHeader.Filename, "ERROR")
		return nil
	}

	logger.Debug("Successfully uploaded file to" + fupl.String())
	go func() {
		_ = s.fileRepository.UpdateFileParams(fileHeader.Filename, "COMPLETED", filePath)
	}()
	return fupl
}

func (s *MinioService) DownloadFile(fileName string) *s3.GetObjectOutput {
	logger := logdoc.GetLogger()

	s3Api := InitS3(s.secret, s.access, s.config)
	bucketName := aws.String(s.bucket)
	result, err := s3Api.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(*bucketName),
		Key:    aws.String(fileName),
	})
	if err != nil {
		logger.Error(err.Error())
		return nil
	}

	return result
}

func (s *MinioService) ListBuckets() []*s3.Bucket {
	logger := logdoc.GetLogger()

	// Показываем список бакетов
	s3Api := InitS3(s.secret, s.access, s.config)
	resp, err := s3Api.ListBuckets(nil)
	if err != nil {
		logger.Error("Unable to list buckets\n" + err.Error())
	}

	return resp.Buckets
}

func (s *MinioService) ListObjects(bucket string) *s3.ListObjectsV2Output {
	logger := logdoc.GetLogger()

	s3Api := InitS3(s.secret, s.access, s.config)
	// Запрашиваем список файлов в бакете
	bucketName := aws.String(bucket)
	result, err := s3Api.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(*bucketName),
	})
	if err != nil {
		logger.Error(fmt.Sprintf("Ошибка чтения файлов из бакета %s", bucket))
		return nil
	}

	for _, object := range result.Contents {
		log.Printf("objects=%s size=%d Bytes last modified=%s", aws.StringValue(object.Key), object.Size, object.LastModified.Format("2006-01-02 15:04:05 Monday"))
	}

	return result
}

func InitS3(secret string, access string, config *hocon.Config) *s3.S3 {
	// Создаем новую сессию AWS
	accessKey := access
	secretKey := secret
	creds := credentials.NewStaticCredentials(accessKey, secretKey, "")
	sess, err := session.NewSession(&aws.Config{
		Credentials:      creds,
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
		Endpoint:         aws.String(config.GetString("minio.address") + ":" + config.GetString("minio.port")),
		Region:           aws.String("us-west-2"),
	})
	if err != nil {
		panic(err)
	}

	// Создаем новый клиент Amazon S3
	return s3.New(sess)
}
