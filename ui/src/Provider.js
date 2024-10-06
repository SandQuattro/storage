import {createContext, useRef, useState} from "react";

export const Context = createContext();

export const ContextProvider = ({ children }) => {
  const [progress, setProgress] = useState(0);
  const [status, setStatus] = useState("");
  const [backStatus, setBackStatus] = useState("");
  const [files, setFiles] = useState([]);
  const [isCancel, setCancel] = useState(false);

  const ws = useRef(null);
  const file = useRef(null);
  const blob = useRef(null);
  const reader = useRef(null);
  const curentFile = useRef(0);

  // useEffect(() => {

  //   return () => ws.current.close();
  // }, []);

  // useEffect(() => {
  //   if (files.length >= 1) {
  //   }
  // }, [files]);

  const readBlob = () => {
    let first = file.current.filePos;
    let last = first + file.current.blockSize;
    if (last > file.current.size) {
      last = file.current.size;
    }
    blob.current = file.current.slice(first, last);
    reader.current.readAsArrayBuffer(blob.current);
  };

  const openConnection = () => {
    ws.current = new WebSocket("ws://127.0.0.1:9003/ws/upload");
    ws.current.onopen = () => setBackStatus("Соединение открыто");
    ws.current.onclose = () => {
      setBackStatus("Соединение закрыто");
      setStatus("");
    };
    ws.current.onmessage = (e) => {
      if (typeof e.data === "string") {
        if (e.data === "NEXT") {
          setStatus("Загрузка");
          readBlob();
        } else {
          setBackStatus(e.data);
        }
        if (e.data === "COMPLETED") {
          file.current.status = "Загрузка завершена";
        }
        if (e.data === "UPLOAD_COMPLETED") {
          file.current.status = "Загружен, обработка...";
        }
        if (e.data === "COMPLETED" && curentFile.current >= files.length - 1) {
          ws.current.close();
        }
        if (e.data === "COMPLETED" && curentFile.current < files.length - 1) {
          curentFile.current += 1;
          handleUpload();
        }
      }
    };
  };
  const handleUpload = () => {
    if (ws?.current?.readyState !== WebSocket.OPEN) {
      openConnection();
    }
    let checkConnection = setInterval(() => {
      if (ws?.current?.readyState === WebSocket.OPEN) {
        file.current = files[curentFile.current];

        ws.current.send(
          JSON.stringify({
            filename: file.current.name,
            size: file.current.size,
          })
        );

        file.current.filePos = 0;
        reader.current = new FileReader();
        setCancel(false);

        file.current.status = "Загружается";

        if (!files.length) {
          alert("Please select a file!");
          return;
        }
        file.current.blockSize = 5 * 1024 * 1024//file.current.size / 10;
        readBlob();

        reader.current.onloadend = (evt) => {
          if (evt.target.readyState == FileReader.DONE) {
            ws.current.send(blob.current);
            file.current.filePos = file.current.filePos + blob.current.size;
            setProgress((file.current.filePos / file.current.size) * 100.0);
          }
        };
        clearInterval(checkConnection);
      }
    }, 50);
  };

  const handleCancel = () => {
    setCancel(true);

    ws.current.send("CANCEL");
  };

  const handleFileChange = (files) => {
    setFiles([...files].slice(0,10));
  };

  return (
    <Context.Provider
      value={{
        handleFileChange,
        handleCancel,
        handleUpload,
        files,
        status,
        progress,
        isCancel,
        backStatus,
      }}
    >
      {children}
    </Context.Provider>
  );
};
