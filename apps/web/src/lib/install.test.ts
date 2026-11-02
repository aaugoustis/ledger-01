import { describe, expect, test } from "bun:test";
import { installMode } from "./install";

const iphone = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15";
const mac = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15";
const android = "Mozilla/5.0 (Linux; Android 15) AppleWebKit/537.36 Chrome/140.0 Mobile";

describe("installMode", () => {
  test("nothing inside the installed app", () => {
    expect(
      installMode({ standalone: true, canPrompt: true, userAgent: android, maxTouchPoints: 5 }),
    ).toBe("none");
  });

  test("a button when the browser can prompt", () => {
    expect(
      installMode({ standalone: false, canPrompt: true, userAgent: android, maxTouchPoints: 5 }),
    ).toBe("prompt");
  });

  test("instructions on iPhone and iPad", () => {
    expect(
      installMode({ standalone: false, canPrompt: false, userAgent: iphone, maxTouchPoints: 5 }),
    ).toBe("ios");
    expect(
      installMode({ standalone: false, canPrompt: false, userAgent: mac, maxTouchPoints: 5 }),
    ).toBe("ios");
  });

  test("nothing on a desktop browser that can't install", () => {
    expect(
      installMode({ standalone: false, canPrompt: false, userAgent: mac, maxTouchPoints: 0 }),
    ).toBe("none");
  });
});
