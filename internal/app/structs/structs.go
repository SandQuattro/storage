package structs

import (
	"database/sql"

	"github.com/aws/aws-sdk-go/service/s3"
)

type IncomingUser struct {
	Login    string `json:"login" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type User struct {
	Id    int            `db:"id" validate:"required"`
	First string         `db:"first_name" validate:"required"`
	Last  sql.NullString `db:"last_name" validate:"required"`
	Email string         `db:"email" validate:"required,email"`
}

type File struct {
	Id           int            `db:"id" validate:"required"`
	Name         string         `db:"file_name" validate:"required"`
	UploadStatus string         `db:"upload_status" validate:"required"`
	StorageLink  sql.NullString `db:"storage_link"`
}

type Token struct {
	Token string `json:"token" xml:"token"`
}

type Response struct {
	FilePath string `json:"filePath" xml:"filePath"`
	Result   string `json:"result" xml:"result"`
}

type UploadHeader struct {
	Filename string
	Size     int
}

type PartUploadResult struct {
	CompletedPart *s3.CompletedPart
	Err           error
}
