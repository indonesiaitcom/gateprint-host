import { useState, useEffect, useRef } from "react";
import {
  GetPrinters,
  GetSelectedPrinter,
  SetSelectedPrinter,
  PrintTest,
  GetAppKey,
  SetAppKey,
} from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";
import "./App.css";

function App() {
  const [printers, setPrinters] = useState<string[]>([]);
  const [selectedPrinter, setSelectedPrinterState] = useState("");
  const [appKey, setAppKeyState] = useState("");
  const [statusMessage, setStatusMessage] = useState("");
  const [isError, setIsError] = useState(false);
  const [logs, setLogs] = useState<string[]>([]);

  const logEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    loadInitialState();

    // Listen for server logs
    const unsubscribe = EventsOn("server-log", (msg: string) => {
      setLogs((prev) => [...prev, msg]);
    });

    return () => {};
  }, []);

  useEffect(() => {
    logEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [logs]);

  const loadInitialState = async () => {
    try {
      const key = await GetAppKey();
      setAppKeyState(key || "");

      const p = await GetPrinters();
      setPrinters(p || []);
      const sel = await GetSelectedPrinter();
      if (sel) {
        setSelectedPrinterState(sel);
      } else if (p && p.length > 0) {
        await handleSelectPrinter(p[0]);
      }
    } catch (e: any) {
      showMessage("Failed to load initial state: " + e, true);
    }
  };

  const handleSelectPrinter = async (name: string) => {
    try {
      await SetSelectedPrinter(name);
      setSelectedPrinterState(name);
      showMessage("Printer set to " + name, false);
    } catch (e: any) {
      showMessage("Error setting printer: " + e, true);
    }
  };

  const handleSaveAppKey = async () => {
    try {
      await SetAppKey(appKey);
      showMessage("App Key saved successfully!", false);
    } catch (e: any) {
      showMessage("Error saving App Key: " + e, true);
    }
  };

  const handleTestPrint = async () => {
    if (!selectedPrinter) {
      showMessage("Please select a printer first", true);
      return;
    }
    try {
      showMessage("Sending test print...", false);
      await PrintTest(selectedPrinter);
      showMessage("Test print sent successfully!", false);
    } catch (e: any) {
      showMessage("Test print failed: " + e, true);
    }
  };

  const showMessage = (msg: string, error: boolean = false) => {
    setStatusMessage(msg);
    setIsError(error);
    if (!error && msg.startsWith("Printer set")) {
      setTimeout(() => setStatusMessage(""), 3000);
    }
  };

  return (
    <div className="app-container">
      <div className="glass-card">
        <h1>GatePrint</h1>

        <div className="status-badge">
          <div className="status-dot"></div>
          Server Gateprint is running
        </div>

        <div className="form-group">
          <label htmlFor="app-key">API Security (App-Key Header)</label>
          <div style={{ display: "flex", gap: "10px", width: "100%" }}>
            <input
              type="password"
              id="app-key"
              className="input-text"
              placeholder="Leave empty for open access"
              value={appKey}
              onChange={(e) => setAppKeyState(e.target.value)}
            />
            <button className="btn-secondary" onClick={handleSaveAppKey}>
              Save
            </button>
          </div>
        </div>

        <div className="form-group">
          <label htmlFor="printer-select">Select Default Printer</label>
          <select
            id="printer-select"
            className="select-printer"
            value={selectedPrinter}
            onChange={(e) => handleSelectPrinter(e.target.value)}
          >
            <option value="" disabled>
              -- Choose a Printer --
            </option>
            {printers.map((p, idx) => (
              <option key={idx} value={p}>
                {p}
              </option>
            ))}
          </select>
        </div>

        <button className="btn-primary" onClick={handleTestPrint}>
          Test Print
        </button>

        {statusMessage && (
          <div className={isError ? "error-text" : "success-text"}>
            {statusMessage}
          </div>
        )}
      </div>

      <div className="log-container">
        <div className="log-header">Server Logs</div>
        <div className="log-body">
          {logs.length === 0 ? (
            <span style={{ color: "#6b7280" }}>
              No logs yet. Waiting for requests...
            </span>
          ) : (
            logs.map((log, index) => (
              <div
                key={index}
                className={`log-entry ${log.includes("[ERROR]") ? "log-error" : log.includes("[SUCCESS]") ? "log-success" : "log-info"}`}
              >
                {log}
              </div>
            ))
          )}
          <div ref={logEndRef} />
        </div>
      </div>
    </div>
  );
}

export default App;
