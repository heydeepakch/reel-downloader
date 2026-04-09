"use client";

import { useState } from "react";

const API_BASE =
  process.env.NODE_ENV === "development"
    ? ""
    : process.env.NEXT_PUBLIC_API_URL || "";

export default function Home() {
  const [urlInput, setUrlInput] = useState("");
  const [status, setStatus] = useState("");
  const [statusClass, setStatusClass] = useState("status");
  const [loading, setLoading] = useState(false);
  const [videoUrl, setVideoUrl] = useState(null);
  const [showPreview, setShowPreview] = useState(false);
  const [downloading, setDownloading] = useState(false);

  async function handleGetVideo() {
    const raw = urlInput.trim();
    if (!raw) {
      setStatus("Please enter a URL");
      setStatusClass("status error");
      return;
    }
    setStatus("Fetching...");
    setStatusClass("status");
    setLoading(true);
    setVideoUrl(null);
    setShowPreview(false);

    try {
      // Unique URL every click so disk cache cannot reuse JSON across sessions/refreshes (ignored by API).
      const apiUrl = `${API_BASE}/api/reel?url=${encodeURIComponent(raw)}&_cb=${Date.now()}`;
      // Instagram CDN URLs expire; avoid any cached API response reusing a dead video_url.
      const res = await fetch(apiUrl, { cache: "no-store" });
      const data = await res.json();

      if (!res.ok) {
        setStatus(data.error || "Failed to get video");
        setStatusClass("status error");
        return;
      }

      setVideoUrl(data.video_url);
      setStatus("Ready! Click Download.");
      setStatusClass("status success");
      setShowPreview(true);
    } catch (e) {
      setStatus("Network error: " + e.message);
      setStatusClass("status error");
    } finally {
      setLoading(false);
    }
  }

  async function handleDownload() {
    if (!videoUrl) return;
    setDownloading(true);
    try {
      const res = await fetch(videoUrl, { cache: "no-store" });
      const blob = await res.blob();
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = "reel.mp4";
      a.click();
      URL.revokeObjectURL(a.href);
    } catch (e) {
      setStatus("Download failed. Try opening: " + videoUrl);
      setStatusClass("status error");
    } finally {
      setDownloading(false);
    }
  }

  return (
    <div className="page-wrap">
      <div className="main-area">
        <div className="card">
          <h1>Instagram Reel Downloader</h1>
          <input
            type="url"
            value={urlInput}
            onChange={(e) => setUrlInput(e.target.value)}
            placeholder="Paste Instagram reel URL"
            autoComplete="off"
            disabled={loading}
          />
          <button onClick={handleGetVideo} disabled={loading}>
            {loading ? "Fetching..." : "Get Video"}
          </button>
          <p className={statusClass}>{status}</p>
          {showPreview && videoUrl && (
            <>
              <video src={videoUrl} controls />
              <button
                className="downloadBtn"
                onClick={handleDownload}
                disabled={downloading}
              >
                {downloading ? "Downloading..." : "Download"}
              </button>
            </>
          )}
        </div>
      </div>
      <footer className="built-by">
        Built by{" "}
        <a
          href="https://github.com/heydeepakch"
          target="_blank"
          rel="noopener noreferrer"
        >
          deepak
        </a>
      </footer>
    </div>
  );
}
