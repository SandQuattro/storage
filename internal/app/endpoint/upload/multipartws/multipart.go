package multipartws

import (
	"demo-storage/internal/app/structs"
	"fmt"
	logdoc "github.com/SandQuattro/logdoc-go-appender/logrus"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/websocket"
	"sort"
	"sync"
	"sync/atomic"
)

func (e *Endpoint) multipartUpload(ws *websocket.Conn, mu sync.Locker, header *structs.UploadHeader) (int, error) {
	bytesRead := 0

	logger := logdoc.GetLogger()

	logger.Debug(fmt.Sprintf("Multipart upload started. File name:%s, Size:%d bytes", header.Filename, header.Size))
	logger.Debug("Ready for receiving file chunks...")

	var ch = make(chan structs.PartUploadResult)
	var wg = sync.WaitGroup{}
	var completedParts []*s3.CompletedPart
	var partNum = 1
	var cnt int64

	// Инициируем S3 Multipart Upload сессию
	s3connection, uploadSession, er := e.s.CreateMultipartSession(header.Filename)
	if er != nil {
		er = e.sendStatus(ws, 400, "Error initiating multipart upload: "+er.Error())
		if er != nil {
			logger.Errorf("Error sending status: %v", er)
			return bytesRead, er
		}
		return bytesRead, er
	}

	for {
		mt, message, err := ws.ReadMessage()
		if err != nil {
			err = e.sendStatus(ws, 400, "Error receiving file block: "+err.Error())
			if err != nil {
				logger.Error("Error sending status:", err)
				return bytesRead, err
			}

			return bytesRead, err
		}

		if mt != websocket.BinaryMessage {
			if mt == websocket.TextMessage {
				if string(message) == "CANCEL" {
					err = e.s.AbortMultipartUpload(s3connection, uploadSession)
					if err != nil {
						logger.Error("Abort multipart upload failed: " + err.Error())
						return bytesRead, err
					}
					err = e.sendStatus(ws, 400, "Upload canceled")
					if err != nil {
						logger.Error("Error sending status:", err)
						return bytesRead, err
					}
					return bytesRead, err
				}
			}

			logger.Debug("Invalid file block received, expecting binary chunk, closing")
			err = e.sendStatus(ws, 400, "Invalid file block received, expecting binary chunk, closing")
			if err != nil {
				logger.Error("Error sending status:", err)
				return bytesRead, err
			}

			return bytesRead, err
		}

		wg.Add(1)
		go func(s3connection *s3.S3, uploadSession *s3.CreateMultipartUploadOutput, message []byte, partNum int, wg *sync.WaitGroup) {
			uploadPartResult := e.s.UploadPartToS3(s3connection, uploadSession, message, partNum)
			e.sendPct(ws, mu, atomic.AddInt64(&cnt, 1))
			ch <- uploadPartResult
			wg.Done()
		}(s3connection, uploadSession, message, partNum, &wg)

		bytesRead += len(message)
		logger.Debug(fmt.Sprintf(">> Websocket multipart receiver > binary chunk received, size:%d, total bytes received:%d of total size:%d", len(message), bytesRead, header.Size))

		if bytesRead == header.Size {
			// Ждем, пока отработают все горутины, и закроем канал
			err = e.sendUploadCompleted(ws)
			if err != nil {
				logger.Error("Error sending status:", err)
				return bytesRead, err
			}
			go func() {
				wg.Wait()
				close(ch)
			}()

			for result := range ch {
				if result.Err != nil {
					err = e.s.AbortMultipartUpload(s3connection, uploadSession)
					if err != nil {
						logger.Error("Error sending status:", err)
						return bytesRead, err
					}
				}
				completedParts = append(completedParts, result.CompletedPart)
			}

			// сортируем куски по PartNumber тк
			// каждая часть может грузиться в произвольном порядке
			sort.Slice(completedParts, func(i, j int) bool {
				return *completedParts[i].PartNumber < *completedParts[j].PartNumber
			})

			// Сигналим AWS S3 хранилищу, что наша multiPart загрузка завершена,
			// AWS начинает сборку кусков в единый файл на своей стороне
			err = e.s.CompleteMultipartUpload(s3connection, uploadSession, completedParts)
			if err != nil {
				logger.Error("Error sending status:", err)
				return bytesRead, err
			}

			logger.Debug("Multipart upload completed")
			break
		}

		partNum++
		err = e.requestNextBlock(ws)
		if err != nil {
			logger.Error("Error receiving next block:", err)
			return bytesRead, err
		}
	}

	return bytesRead, nil
}
