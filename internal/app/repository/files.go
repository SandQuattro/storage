package repository

import (
	"database/sql"

	"demo-storage/internal/app/structs"
	logdoc "github.com/SandQuattro/logdoc-go-appender/logrus"
	"github.com/jmoiron/sqlx"
)

type FileRepository struct {
	DB *sqlx.DB
}

func New(db *sqlx.DB) *FileRepository {
	return &FileRepository{DB: db}
}

func (r *FileRepository) FindFileByName(name string) *structs.File {
	logger := logdoc.GetLogger()

	params := map[string]interface{}{"name": name}
	rows, err := r.DB.NamedQuery(`SELECT * FROM files where file_name = :name`, params)
	if err != nil {
		logger.Error("FindFileByName prepare query error")
		return nil
	}
	var file structs.File
	for rows.Next() {
		err = rows.StructScan(&file)
	}
	if err != nil {
		logger.Error("FindFileByName query error")
		return nil
	}
	return &file
}

func (r *FileRepository) CreateFile(name string, filePath string) sql.Result {
	logger := logdoc.GetLogger()

	params := map[string]interface{}{"name": name, "status": "UPLOADING", "link": filePath}
	nstmt, err := r.DB.PrepareNamed(`INSERT INTO files(file_name, upload_status, storage_link) values (:name,:status,:link)`)
	if err != nil {
		logger.Error("CreateFile prepare error")
		return nil
	}

	res, err := nstmt.Exec(params)
	if err != nil {
		logger.Error("CreateFile exec error")
		return nil
	}

	return res
}

func (r *FileRepository) UpdateFileStatus(name string, status string) sql.Result {
	logger := logdoc.GetLogger()

	params := map[string]interface{}{"name": name, "status": status}
	nstmt, err := r.DB.PrepareNamed(`update files set upload_status=:status where file_name = :name`)
	if err != nil {
		logger.Error("UpdateFileStatus prepare error")
		return nil
	}

	res, err := nstmt.Exec(params)
	if err != nil {
		logger.Error("UpdateFileStatus exec error")
		return nil
	}

	return res
}

func (r *FileRepository) UpdateFileParams(name string, status string, link string) sql.Result {
	logger := logdoc.GetLogger()

	params := map[string]interface{}{"name": name, "status": status, "link": link}
	nstmt, err := r.DB.PrepareNamed(`update files set storage_link=:link, upload_status=:status where file_name = :name`)
	if err != nil {
		logger.Error("UpdateFileLink prepare error")
		return nil
	}

	res, err := nstmt.Exec(params)
	if err != nil {
		logger.Error("UpdateFileLink exec error")
		return nil
	}

	return res
}
