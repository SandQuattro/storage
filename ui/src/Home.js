import {useContext, useState} from "react";
import {Context} from "./Provider";
import ProgressBar from "@ramonak/react-progress-bar";

function Home() {
  const [drag, setDrag] = useState(false);
  const {
    handleFileChange,
    handleCancel,
    handleUpload,
    files,
    status,
    backStatus,
    progress,
  } = useContext(Context);

  const convertBytes = (bytes) => {
    if (bytes < 1024) {
      return bytes + " B";
    } else if (bytes < 1024 * 1024) {
      return (bytes / 1024).toFixed(1) + " KB";
    } else {
      return (bytes / (1024 * 1024)).toFixed(1) + " MB";
    }
  };

  const dragStartHandler = (e) => {
    e.preventDefault();
    setDrag(true);
  };

  const dragLeaveHandler = (e) => {
    e.preventDefault();
    setDrag(false);
  };

  const onFileDeleteHandler = () => {};

  const onDropHandler = (e) => {
    e.preventDefault();
    handleFileChange(e.dataTransfer.files);
    setDrag(false);
  };

  return (
    <div className="Home">
      <div className="wrapper">
        <header>Загрузка файлов</header>
        <img src="/transfer.gif" alt={'transfer'}/>
        <input
            className="file-input"
            type="file"
            id="file"
            multiple
            hidden
            onChange={(e) => handleFileChange(e.target.files)}
        />
        <label
            className={`upload ${drag ? "drag" : ""}`}
            htmlFor="file"
            onDragStart={dragStartHandler}
            onDragLeave={dragLeaveHandler}
            onDragOver={dragStartHandler}
            onDrop={onDropHandler}
        >
          <img src="/upload.svg"/>
          <span>
            Выберите необходимые файлы <br/>
            для загрузки
          </span>
        </label>
        <div className="wssStatus">
          <span className="title">Статус: {backStatus ? backStatus : "Нет соединения"}</span>
        </div>
        <div className="info">
          <button
              className={`uploadHandler ${status == "Загрузка" ? "cancel" : ""} ${
                  files.length < 1 ? "disable" : ""
              }`}
              onClick={status == "Загрузка" ? handleCancel : handleUpload}
              disabled={files.length < 1}
          >
            {status == "Загрузка" ? (
                <>
                  <img src="/cancel.svg"/>
                  <span>Отменить</span>
                </>
            ) : (
                <>
                  <img src="/upload-second.svg"/>
                  <span>Загрузить</span>
                </>
            )}
          </button>
        </div>

        <section className="progress-area">
          {files?.map((file, index) => {
            return (
                <li className="row" key={index}>
                  <img src="/File.svg"/>
                  <div className="content">
                    <div className="details">
                      <div>
                        <span className="name">{file.name}</span>
                        <span className="separator"></span>
                        <span className="name status">
                        {file.status ?? "Ожидает загрузки"}
                      </span>
                      </div>
                      {file.status == "Загружается" && (
                          <span className="percent">
                        {Math.round(progress) ?? 0}%
                      </span>
                      )}
                    </div>
                    <div className="details__lower">
                      {file.status == "Загружается" ? (
                          <ProgressBar
                              bgColor={"#6990f2"}
                              completed={progress}
                              maxCompleted={100}
                              isLabelVisible={false}
                              height={"6px"}
                              transitionDuration={0.1}
                          />
                      ) : (
                          <div className="size">{convertBytes(file.size)}</div>
                      )}
                    </div>
                    {file.status == "Загружен" && (
                        <img src="/mark.svg" className="mark"/>
                    )}
                  </div>
                </li>
            );
          })}
        </section>
        <section className="uploaded-area"></section>
      </div>
    </div>
  );
}

export default Home;
