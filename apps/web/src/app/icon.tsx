import { ImageResponse } from "next/og";

// App icons rendered at build time: a signal-orange molded tile with an ink "L/01".
// Inner content stays inside the maskable safe zone (central 80%).

export function generateImageMetadata() {
  return [
    { id: "192", size: { width: 192, height: 192 }, contentType: "image/png" },
    { id: "512", size: { width: 512, height: 512 }, contentType: "image/png" },
  ];
}

export default async function Icon({ id }: { id: Promise<string> }) {
  const size = Number(await id);
  return new ImageResponse(
    <div
      style={{
        width: "100%",
        height: "100%",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        background: "#ff5f1f",
        color: "#1c1e21",
        fontSize: size * 0.26,
        fontWeight: 700,
        letterSpacing: "-0.02em",
      }}
    >
      L/01
    </div>,
    { width: size, height: size },
  );
}
