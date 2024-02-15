package interfaces

import (
	"database/sql"
	"demo-storage/internal/app/structs"
)

type UserRepository interface {
	FindFileByName(name string) *structs.File
	CreateFile(name string, filePath string) sql.Result
	UpdateFileStatus(name string, status string) sql.Result
}
