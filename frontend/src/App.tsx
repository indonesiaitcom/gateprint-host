import { useState, useEffect, useRef } from "react";
import {
  GetPrinters,
  GetSelectedPrinter,
  SetSelectedPrinter,
  PrintTest,
  GetAppKey,
  SetAppKey,
  GetServerPort,
  SetServerPort,
} from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";
import "./App.css";

function App() {
  const [printers, setPrinters] = useState<string[]>([]);
  const [selectedPrinter, setSelectedPrinterState] = useState("");
  const [appKey, setAppKeyState] = useState("");
  const [showAppKey, setShowAppKey] = useState(false);
  const [serverPort, setServerPortState] = useState("");
  const [statusMessage, setStatusMessage] = useState("");
  const [isError, setIsError] = useState(false);
  const [logs, setLogs] = useState<string[]>([]);

  const logEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    loadInitialState();
    EventsOn("server-log", (msg: string) => {
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

      const port = await GetServerPort();
      setServerPortState(port || "9100");

      const p = await GetPrinters();
      setPrinters(p || []);

      const sel = await GetSelectedPrinter();
      if (sel) {
        setSelectedPrinterState(sel);
      } else if (p && p.length > 0) {
        await handleSelectPrinter(p[0]);
      }
    } catch (e: any) {
      showMessage("Failed to load: " + e, true);
    }
  };

  const handleSelectPrinter = async (name: string) => {
    try {
      await SetSelectedPrinter(name);
      setSelectedPrinterState(name);
      showMessage("Printer set to " + name, false);
    } catch (e: any) {
      showMessage("Error: " + e, true);
    }
  };

  const handleSaveAppKey = async () => {
    try {
      await SetAppKey(appKey);
      showMessage("App Key saved!", false);
    } catch (e: any) {
      showMessage("Error saving key: " + e, true);
    }
  };

  const handleSavePort = async () => {
    try {
      if (!serverPort || isNaN(Number(serverPort))) {
        showMessage("Invalid port", true);
        return;
      }
      showMessage("Restarting server...", false);
      await SetServerPort(serverPort);
      showMessage("Port updated!", false);
    } catch (e: any) {
      showMessage("Error: " + e, true);
    }
  };

  const handleTestPrint = async () => {
    if (!selectedPrinter) {
      showMessage("Select printer first", true);
      return;
    }
    try {
      showMessage("Printing test page...", false);
      await PrintTest(selectedPrinter);
      showMessage("Test print sent!", false);
    } catch (e: any) {
      showMessage("Failed: " + e, true);
    }
  };

  const showMessage = (msg: string, error: boolean = false) => {
    setStatusMessage(msg);
    setIsError(error);
    setTimeout(() => setStatusMessage(""), 4000);
  };

  const getLogClass = (log: string) => {
    if (log.includes("[ERROR]")) return "log-error";
    if (log.includes("[SUCCESS]")) return "log-success";
    if (log.includes("[INFO]")) return "log-info";
    return "log-default";
  };

  const extractTimestamp = (log: string) => {
    const match = log.match(/^\[(.*?)\]/);
    return match ? match[1] : "";
  };
  
  const extractMessage = (log: string) => {
    const match = log.match(/^\[.*?\]\s(.*)/);
    return match ? match[1] : log;
  };

  return (
    <div className="app-viewport">
      <div className="main-layout">
        {/* CONFIGURATION PANEL */}
        <div className="panel-config">
          <header className="brand-header">
            <h1>GatePrint</h1>
            <div className="status-badge">
              <div className="status-dot"></div>
              <span>PORT: {serverPort}</span>
            </div>
          </header>

          <div className="controls-grid">
            <div className="input-field">
              <label>API Key (Auth)</label>
              <div className="input-group">
                <input
                  type={showAppKey ? "text" : "password"}
                  className="input-base"
                  value={appKey}
                  placeholder="Enter secret key..."
                  onChange={(e) => setAppKeyState(e.target.value)}
                />
                <button
                  className="btn-icon"
                  onClick={() => setShowAppKey(!showAppKey)}
                  title="Toggle visibility"
                >
                  {showAppKey ? "🙈" : "👁️"}
                </button>
                <button className="btn-action" onClick={handleSaveAppKey}>
                  Save
                </button>
              </div>
            </div>

            <div className="input-field">
              <label>Bridge Port</label>
              <div className="input-group">
                <input
                  type="text"
                  className="input-base"
                  value={serverPort}
                  onChange={(e) => setServerPortState(e.target.value)}
                />
                <button className="btn-action" onClick={handleSavePort}>
                  Set
                </button>
              </div>
            </div>

            <div className="input-field">
              <label>Target Printer</label>
              <select
                className="select-base"
                value={selectedPrinter}
                onChange={(e) => handleSelectPrinter(e.target.value)}
              >
                <option value="" disabled>-- Select Local Printer --</option>
                {printers.map((p, idx) => (
                  <option key={idx} value={p}>{p}</option>
                ))}
              </select>
            </div>

            <button className="btn-primary" onClick={handleTestPrint}>
              Test Print Connection
            </button>
          </div>
        </div>

        {/* LIVE LOGS PANEL */}
        <div className="panel-log">
          <div className="log-header">
            <span>🔴</span> Live Activity Logs
          </div>
          <div className="log-scroll-area">
            {logs.length === 0 ? (
              <div className="log-empty">Waiting for print requests...</div>
            ) : (
              logs.map((log, index) => {
                const ts = extractTimestamp(log);
                const msg = extractMessage(log);
                return (
                  <div key={index} className={`log-line ${getLogClass(log)}`}>
                    {ts && <span className="log-timestamp">[{ts}]</span>}
                    <span>{msg}</span>
                  </div>
                );
              })
            )}
            <div ref={logEndRef} />
          </div>
        </div>
      </div>
      
      {statusMessage && (
        <div className={`status-toast ${isError ? "text-error" : "text-success"}`}>
          {statusMessage}
        </div>
      )}
    </div>
  );
}

export default App;
