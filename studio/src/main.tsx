import React from "react";
import ReactDOM from "react-dom/client";
import { App } from "./App";
import { installBridge } from "./bridge";
import "./index.css";

installBridge();

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
