package multipartws

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"demo-storage/internal/app/interfaces"
	"demo-storage/internal/app/structs"

	logdoc "github.com/LogDoc-org/logdoc-go-appender/logrus"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/websocket"
	"github.com/gurkankaymak/hocon"
	"github.com/labstack/echo/v4"
)

type Endpoint struct {
	config *hocon.Config
	s      interfaces.MinioService
}

func New(s interfaces.MinioService, config *hocon.Config) *Endpoint {
	// Создаем endpoint и возвращаем
	return &Endpoint{s: s, config: config}
}

type UploadStatus struct {
	Code   int    `json:"code,omitempty"`
	Status string `json:"status,omitempty"`
	Pct    *int64 `json:"part,omitempty"` // File processing AFTER uploading is done.
	pct    int64
}

const (
	HandshakeTimeoutSecs = 10
)

func (e *Endpoint) WebSocketUploadHandler(ctx echo.Context) error { // Source
	var err error
	var ws *websocket.Conn

	logger := logdoc.GetLogger()
	mu := sync.Mutex{}

	logger.Debug("WebSocketUploadHandler > Starting...")

	// Open websocket connection.
	upgrader := websocket.Upgrader{HandshakeTimeout: time.Second * HandshakeTimeoutSecs}

	// TODO: Проверяем, откуда идет запрос!!
	// A CheckOrigin function should carefully validate the request origin to
	// prevent cross-site request forgery.
	upgrader.CheckOrigin = func(r *http.Request) bool {
		// logger.Debug("WebSocketUploadHandler > CheckOrigin > Origin:", r.Header.Get("Origin"))
		allowedOrigins := []string{"localhost:3000", e.config.GetString("server.address")}
		return slices.Contains(allowedOrigins, r.Header.Get("Origin"))
		// return true
	}

	ws, err = upgrader.Upgrade(ctx.Response(), ctx.Request(), nil)
	if err != nil {
		logger.Debug("Error on open of websocket connection:", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Error on open of websocket connection")
	}
	defer ws.Close()

START:
	err = e.sendStatus(ws, 200, "READY")
	if err != nil {
		return nil
	}
	logger.Debug(">> WebSocketUploadHandler > Client connected, ready for interaction...")

	// waiting for file name and length from client.
	header := new(structs.UploadHeader)
	mt, message, err := ws.ReadMessage()
	if err != nil {
		var closeErr *websocket.CloseError
		if errors.As(err, &closeErr) {
			// Если err является ошибкой закрытия *websocket.CloseError, проверяем её код:
			if closeErr.Code == websocket.CloseGoingAway {
				logger.Warn("Client closed websocket connection")
			} else {
				logger.Error(fmt.Sprintf("Ошибка закрытия соединения с кодом: %v\n", closeErr.Code))
			}
		}
		return nil
	}
	if mt != websocket.TextMessage {
		err := e.sendStatus(ws, 400, "Invalid message received, expecting file name and length")
		logger.Error("Error receiving websocket message:", err)
		if err != nil {
			return nil
		}
		return nil
	}
	if err := json.Unmarshal(message, header); err != nil {
		err := e.sendStatus(ws, 400, "Error receiving file name and length: "+err.Error())
		if err != nil {
			logger.Error("Error sending status:", err)
			return nil
		}
		logger.Error("Error receiving file name and length: " + err.Error())
		return nil
	}

	if len(header.Filename) == 0 {
		err := e.sendStatus(ws, 400, "Filename cannot be empty")
		if err != nil {
			return nil
		}
		return nil
	}

	if header.Size == 0 {
		err := e.sendStatus(ws, 400, "Upload file is empty")
		if err != nil {
			return nil
		}
		return nil
	}

	// Read file blocks until all bytes are received.
	bytesRead := 0

	// MAIN DECISION POINT
	// multipart upload requires at least 5MB
	// EACH PART SHOULD BE AT LEAST 5MB !!!
	//  (1 MB = 2^20 bytes = 1 << 20 bytes).
	if header.Size < 5<<20 {
		buf := make([]byte, 0, header.Size)
		// грузим файл в s3 через обычный метод
		for {
			mt, message, err := ws.ReadMessage()
			if err != nil {
				err := e.sendStatus(ws, 400, "Error receiving file block: "+err.Error())
				if err != nil {
					return nil
				}
				return nil
			}

			if mt != websocket.BinaryMessage {
				if mt == websocket.TextMessage {
					if string(message) == "CANCEL" {
						err = e.sendStatus(ws, 400, "Upload canceled")
						if err != nil {
							return nil
						}
						return nil
					}
				}
				logger.Debug("Invalid file block received, expecting binary chunk, closing")
				err := e.sendStatus(ws, 400, "Invalid file block received")
				if err != nil {
					return nil
				}
				return nil
			}

			buf = append(buf, message...)
			bytesRead += len(message)
			logger.Debug(fmt.Sprintf(">> Websocket receiver > binary chunk received, size:%d. total bytes received:%d", len(message), bytesRead))

			if bytesRead == header.Size {
				e.s.UploadFileAsBytes(header, buf)
				err := e.sendPct(ws, &mu, 100)
				if err != nil {
					return nil
				}
				break
			}

			err = e.requestNextBlock(ws)
			if err != nil {
				return nil
			}
		}
	} else {
		logger.Debug(fmt.Sprintf("Multipart upload started. File name:%s, Size:%d bytes", header.Filename, header.Size))
		logger.Debug("Ready for receiving file chunks...")

		ch := make(chan structs.PartUploadResult)
		wg := sync.WaitGroup{}
		var completedParts []*s3.CompletedPart
		partNum := 1
		var cnt int64 = 0

		// Инициируем S3 Multipart Upload сессию
		s3connection, uploadSession, err := e.s.CreateMultipartSession(header.Filename)
		if err != nil {
			err := e.sendStatus(ws, 400, "Error initiating multipart upload: "+err.Error())
			if err != nil {
				return nil
			}
			return nil
		}

		for {
			mt, message, err := ws.ReadMessage()
			if err != nil {
				err := e.sendStatus(ws, 400, "Error receiving file block: "+err.Error())
				if err != nil {
					return nil
				}
				return nil
			}
			if mt != websocket.BinaryMessage {
				if mt == websocket.TextMessage {
					if string(message) == "CANCEL" {
						err := e.s.AbortMultipartUpload(s3connection, uploadSession)
						if err != nil {
							logger.Error("Abort multipart upload failed: " + err.Error())
							return nil
						}
						err = e.sendStatus(ws, 400, "Upload canceled")
						if err != nil {
							return nil
						}
						return nil
					}
				}
				logger.Debug("Invalid file block received, expecting binary chunk, closing")
				err := e.sendStatus(ws, 400, "Invalid file block received, expecting binary chunk, closing")
				if err != nil {
					return nil
				}
				return nil
			}

			wg.Add(1)
			go func(s3connection *s3.S3, uploadSession *s3.CreateMultipartUploadOutput, message []byte, partNum int, wg *sync.WaitGroup) {
				uploadPartResult := e.s.UploadPartToS3(s3connection, uploadSession, message, partNum)
				e.sendPct(ws, &mu, atomic.AddInt64(&cnt, 1))
				ch <- uploadPartResult
				wg.Done()
			}(s3connection, uploadSession, message, partNum, &wg)

			bytesRead += len(message)
			logger.Debug(fmt.Sprintf(">> Websocket multipart receiver > binary chunk received, size:%d, total bytes received:%d of total size:%d", len(message), bytesRead, header.Size))

			if bytesRead == header.Size {
				// Ждем, пока отработают все горутины, и закроем канал
				go func() {
					wg.Wait()
					close(ch)
				}()

				for result := range ch {
					if result.Err != nil {
						err := e.s.AbortMultipartUpload(s3connection, uploadSession)
						if err != nil {
							return nil
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
				err := e.s.CompleteMultipartUpload(s3connection, uploadSession, completedParts)
				if err != nil {
					return nil
				}

				logger.Debug("Multipart upload completed")
				break
			}

			partNum++
			err = e.requestNextBlock(ws)
			if err != nil {
				return nil
			}
		}
	}
	err = e.sendStatus(ws, 200, fmt.Sprintf("File upload successful: %s (%d bytes)", header.Filename, bytesRead))
	if err != nil {
		return nil
	}
	err = e.sendComplete(ws)
	if err != nil {
		return nil
	}

	// идем на начало и ждем следующий файл
	goto START
}

func (e *Endpoint) requestNextBlock(ws *websocket.Conn) error {
	return ws.WriteMessage(websocket.TextMessage, []byte("NEXT_CHUNK"))
}

func (e *Endpoint) sendComplete(ws *websocket.Conn) error {
	return ws.WriteMessage(websocket.TextMessage, []byte("COMPLETED"))
}

func (e *Endpoint) sendStatus(ws *websocket.Conn, code int, status string) error {
	msg, err := json.Marshal(UploadStatus{Code: code, Status: status})
	if err == nil {
		return ws.WriteMessage(websocket.TextMessage, msg)
	}
	return nil
}

func (e *Endpoint) sendPct(ws *websocket.Conn, mu *sync.Mutex, pct int64) error {
	// Писать в websocket может только 1 горутина в один момент времени
	stat := UploadStatus{pct: pct, Status: "part upload completed"}
	stat.Pct = &stat.pct
	msg, err := json.Marshal(stat)
	if err == nil {
		mu.Lock()
		err := ws.WriteMessage(websocket.TextMessage, msg)
		mu.Unlock()
		if err != nil {
			return nil
		}
	}
	return nil
}
