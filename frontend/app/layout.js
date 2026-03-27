import "./globals.css";

export const metadata = {
  title: "Reel Downloader",
  description: "Download Instagram reels",
};

export default function RootLayout({ children }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
