import React from "react";
import {ContextProvider} from "./Provider";
import Home from "./Home";

const App = () => {
  return (
    <ContextProvider>
      <Home />
    </ContextProvider>
  );
};

export default App;
