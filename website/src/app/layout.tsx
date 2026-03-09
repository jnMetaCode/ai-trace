import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "AI-Trace - Enterprise AI Decision Audit & Tamper-Proof Evidence System",
  description: "Open source tamper-proof audit trail for LLM applications. Works with OpenAI, Claude, and Gemini. Merkle tree certificates, blockchain anchoring, and zero-knowledge proofs.",
  keywords: ["AI audit", "AI compliance", "LLM tracing", "AI governance", "Merkle tree", "blockchain", "OpenAI", "Claude", "Gemini"],
  icons: {
    icon: [
      { url: "/favicon.svg", type: "image/svg+xml" },
    ],
    apple: "/favicon.svg",
  },
  openGraph: {
    title: "AI-Trace - Tamper-Proof AI Audit",
    description: "Make your AI decisions trustworthy and verifiable",
    type: "website",
    siteName: "AI-Trace",
  },
  twitter: {
    card: "summary_large_image",
    title: "AI-Trace - Tamper-Proof AI Audit",
    description: "Make your AI decisions trustworthy and verifiable",
  },
  metadataBase: new URL("https://aitrace.cc"),
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        {children}
      </body>
    </html>
  );
}
