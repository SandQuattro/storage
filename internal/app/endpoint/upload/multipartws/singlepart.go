package multipartws

import (
	"demo-storage/internal/app/structs"
	"fmt"
	logdoc "github.com/SandQuattro/logdoc-go-appender/logrus"
	"github.com/gorilla/websocket"
	"sync"
)

func (e *Endpoint) singlePartUpload(ws *websocket.Conn, mu sync.Locker, header *structs.UploadHeader) (int, error) {
	logger := logdoc.GetLogger()
	buf := make([]byte, 0, header.Size)
	bytesRead := 0

	// грузим файл в s3 через обычный метод
	for {
		mt, message, err := ws.ReadMessage()
		if err != nil {
			err = e.sendStatus(ws, 400, fmt.Sprintf("Error receiving file block: %s", err.Error()))
			if err != nil {
				logger.Error("Error sending status:", err)
				return bytesRead, err
			}
			return bytesRead, err
		}

		if mt != websocket.BinaryMessage {
			if mt == websocket.TextMessage {
				if string(message) == "CANCEL" {
					err = e.sendStatus(ws, 400, "Upload canceled")
					if err != nil {
						logger.Error("Error sending status:", err)
						return bytesRead, err
					}
					return bytesRead, err
				}
			}

			logger.Debug("Invalid file block received, expecting binary chunk, closing")
			err = e.sendStatus(ws, 400, "Invalid file block received")
			if err != nil {
				return bytesRead, err
			}

			return bytesRead, err
		}

		buf = append(buf, message...)
		bytesRead += len(message)
		logger.Debug(fmt.Sprintf(">> Websocket receiver > binary chunk received, size:%d. total bytes received:%d", len(message), bytesRead))

		if bytesRead == header.Size {
			e.s.UploadFileAsBytes(header, buf)
			err = e.sendPct(ws, mu, 100)
			if err != nil {
				return bytesRead, err
			}
			break
		}

		err = e.requestNextBlock(ws)
		if err != nil {
			logger.Error("Error receiving next block:", err)
			return bytesRead, err
		}
	}

	return bytesRead, nil
}
