import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import Chat from "./Chat";
import Login from "./Login";
import { setLastUid } from "../lib/storage";

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual("../lib/api");
  return {
    ...actual,
    loginWithUid: vi.fn(async () => ({
      token: "wl-token",
      user: { id: 10000000, username: "whitelist_10000000", email: "whitelist@local" },
    })),
  };
});

const LocationDisplay = () => {
  const location = useLocation();
  return <div>{location.pathname}{location.search}</div>;
};

describe("Chat routing", () => {
  beforeEach(() => {
    const store = new Map();
    globalThis.localStorage = {
      getItem: (key) => (store.has(key) ? store.get(key) : null),
      setItem: (key, value) => {
        store.set(key, String(value));
      },
      removeItem: (key) => {
        store.delete(key);
      },
      clear: () => {
        store.clear();
      },
    };
    globalThis.localStorage.clear();
  });

  it("redirects to last uid when missing in url", async () => {
    setLastUid(10000000);
    render(
      <MemoryRouter initialEntries={["/chat"]}>
        <LocationDisplay />
        <Routes>
          <Route path="/chat" element={<Chat />} />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() =>
      expect(screen.getByText("/chat?uid=10000000")).toBeInTheDocument()
    );
  });

  it("redirects to login when non-whitelist uid has no token", async () => {
    render(
      <MemoryRouter initialEntries={["/chat?uid=1"]}>
        <LocationDisplay />
        <Routes>
          <Route path="/chat" element={<Chat />} />
          <Route path="/login" element={<Login />} />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() => expect(screen.getByText("/login")).toBeInTheDocument());
  });
});
