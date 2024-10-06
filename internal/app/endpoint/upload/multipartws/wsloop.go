package multipartws

import (
	"demo-storage/internal/app/structs"
	"encoding/json"
	"errors"
	"fmt"
	logdoc "github.com/SandQuattro/logdoc-go-appender/logrus"
	"github.com/gorilla/websocket"
	"sync"
)

func (e *Endpoint) processingLoop(ws *websocket.Conn, mu sync.Locker) {
	logger := logdoc.GetLogger()

	err := e.sendStatus(ws, 200, "READY")
	if err != nil {
		panic(err)
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
		return
	}
	if mt != websocket.TextMessage {
		err = e.sendStatus(ws, 400, "Invalid message received, expecting file name and length")
		logger.Error("Error receiving websocket message:", err)
		if err != nil {
			logger.Error("Error receiving websocket message:", err)
			return
		}
		return
	}
	if err = json.Unmarshal(message, header); err != nil {
		err = e.sendStatus(ws, 400, "Error receiving file name and length: "+err.Error())
		if err != nil {
			logger.Error("Error sending status:", err)
			return
		}
		logger.Errorf("Error receiving file name and length: %v", err)
		return
	}

	if len(header.Filename) == 0 {
		err = e.sendStatus(ws, 400, "Filename cannot be empty")
		if err != nil {
			logger.Error("Error sending status:", err)
			return
		}
		return
	}

	if header.Size == 0 {
		err = e.sendStatus(ws, 400, "Upload file is empty")
		if err != nil {
			logger.Error("Error sending status:", err)
			return
		}
		return
	}

	// MAIN DECISION POINT
	// multipart upload requires at least 5MB
	// EACH PART SHOULD BE AT LEAST 5MB !!!
	var bytesRead int
	if header.Size < 5<<20 {
		bytesRead, err = e.singlePartUpload(ws, mu, header)
		if err != nil {
			logger.Errorf(">> singlePartUpload error : %v", err)
			return
		}
	} else {
		bytesRead, err = e.multipartUpload(ws, mu, header)
		if err != nil {
			logger.Errorf(">> multipartUpload error : %v", err)
			return
		}
	}

	err = e.sendStatus(ws, 200, fmt.Sprintf("File upload successful: %s (%d bytes)", header.Filename, bytesRead))
	if err != nil {
		logger.Error("Error sending status:", err)
		return
	}
	err = e.sendCompleted(ws)
	if err != nil {
		logger.Error("Error sending status:", err)
		return
	}
}

func (e *Endpoint) requestNextBlock(ws *websocket.Conn) error {
	return ws.WriteMessage(websocket.TextMessage, []byte("NEXT"))
}

func (e *Endpoint) sendUploadCompleted(ws *websocket.Conn) error {
	return ws.WriteMessage(websocket.TextMessage, []byte("UPLOAD_COMPLETED"))
}

func (e *Endpoint) sendCompleted(ws *websocket.Conn) error {
	return ws.WriteMessage(websocket.TextMessage, []byte("COMPLETED"))
}

func (e *Endpoint) sendStatus(ws *websocket.Conn, code int, status string) error {
	msg, err := json.Marshal(UploadStatus{Code: code, Status: status})
	if err == nil {
		return ws.WriteMessage(websocket.TextMessage, msg)
	}
	return nil
}

func (e *Endpoint) sendPct(ws *websocket.Conn, mu sync.Locker, pct int64) error {
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
