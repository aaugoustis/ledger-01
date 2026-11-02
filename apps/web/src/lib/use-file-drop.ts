"use client";

import { useEffect, useRef, useState } from "react";

const hasFiles = (e: DragEvent) => Array.from(e.dataTransfer?.types ?? []).includes("Files");

/**
 * Lets a file be dropped anywhere on the page: returns whether one is being dragged over it, and
 * calls onDrop with the first dropped file. While enabled, the browser never opens a dropped file
 * in place of the app. Dragging text or links is left alone.
 */
export function useFileDrop(enabled: boolean, onDrop: (file: File) => void): boolean {
  const [dragging, setDragging] = useState(false);
  const depth = useRef(0); // dragenter/dragleave fire for every child crossed
  const handler = useRef(onDrop);
  handler.current = onDrop;

  useEffect(() => {
    if (!enabled) {
      setDragging(false);
      return;
    }
    const enter = (e: DragEvent) => {
      if (!hasFiles(e)) return;
      e.preventDefault();
      depth.current++;
      setDragging(true);
    };
    const over = (e: DragEvent) => {
      if (!hasFiles(e)) return;
      e.preventDefault();
      if (e.dataTransfer) e.dataTransfer.dropEffect = "copy";
    };
    const leave = (e: DragEvent) => {
      if (!hasFiles(e)) return;
      depth.current = Math.max(0, depth.current - 1);
      if (depth.current === 0) setDragging(false);
    };
    const drop = (e: DragEvent) => {
      if (!hasFiles(e)) return;
      e.preventDefault();
      depth.current = 0;
      setDragging(false);
      const file = e.dataTransfer?.files[0];
      if (file) handler.current(file);
    };
    window.addEventListener("dragenter", enter);
    window.addEventListener("dragover", over);
    window.addEventListener("dragleave", leave);
    window.addEventListener("drop", drop);
    return () => {
      window.removeEventListener("dragenter", enter);
      window.removeEventListener("dragover", over);
      window.removeEventListener("dragleave", leave);
      window.removeEventListener("drop", drop);
      depth.current = 0;
    };
  }, [enabled]);

  return dragging;
}
