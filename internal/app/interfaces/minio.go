package interfaces

import (
	"demo-storage/internal/app/structs"
	"github.com/aws/aws-sdk-go/service/s3"
	"mime/multipart"
)

type MinioService interface {
	CreateMultipartSession(name string) (*s3.S3, *s3.CreateMultipartUploadOutput, error)
	UploadPartToS3(s3connection *s3.S3, multipartSession *s3.CreateMultipartUploadOutput, fileBytes []byte, partNum int) structs.PartUploadResult
	CompleteMultipartUpload(s3connection *s3.S3, uploadSession *s3.CreateMultipartUploadOutput, completedParts []*s3.CompletedPart) error
	AbortMultipartUpload(s3connection *s3.S3, uploadSession *s3.CreateMultipartUploadOutput) error
	UploadFileAsBytes(fileHeader *structs.UploadHeader, data []byte) *s3.PutObjectOutput
	UploadFile(fileHeader *multipart.FileHeader, filePath string) *s3.PutObjectOutput
	DownloadFile(fileName string) *s3.GetObjectOutput
	ListBuckets() []*s3.Bucket
	ListObjects(bucket string) *s3.ListObjectsV2Output
}
