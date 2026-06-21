import React from "react";
import ReactDOM from "react-dom/client";
import { App } from "./App";
import { EditorWindowApp } from "./EditorWindowApp";
import { parseEditorWindowParams } from "./lib/workbenchView";
import "./styles.css";

const editorFindingId = parseEditorWindowParams();

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    {editorFindingId ? <EditorWindowApp findingId={editorFindingId} /> : <App />}
  </React.StrictMode>
);
