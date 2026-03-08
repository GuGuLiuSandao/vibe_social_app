import { fireEvent, render, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import Login from "./Login";

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual("../lib/api");
  return {
    ...actual,
    loginWithUid: vi.fn(async () => ({
      token: "wl-token",
      user: { id: 10000000n, username: "whitelist_10000000", email: "whitelist@local" },
      message: "ok",
    })),
    registerWithProtobuf: vi.fn(async () => ({
      token: "reg-token",
      user: { id: 10000001n, username: "neo", email: "neo@example.com" },
      message: "ok",
    })),
  };
});

describe("Login page", () => {
  it("renders centered auth entry with bottom register entry link", () => {
    const { container } = render(
      <MemoryRouter>
        <Login />
      </MemoryRouter>
    );
    const view = within(container);
    expect(view.getByText("欢迎回来")).toBeInTheDocument();
    expect(view.getAllByRole("button", { name: "登录" }).length).toBeGreaterThan(0);
    expect(view.getByRole("button", { name: "立即注册" })).toBeInTheDocument();
    expect(view.queryByText("白名单直达")).not.toBeInTheDocument();
  });

  it("submits registration from bottom register entry link", async () => {
    const { container } = render(
      <MemoryRouter>
        <Login />
      </MemoryRouter>
    );
    const view = within(container);

    fireEvent.click(view.getByRole("button", { name: "立即注册" }));
    fireEvent.change(view.getByLabelText("用户名"), {
      target: { value: "neo" },
    });
    fireEvent.change(view.getByLabelText("邮箱"), {
      target: { value: "neo@example.com" },
    });
    fireEvent.change(view.getByLabelText("密码"), {
      target: { value: "secret12" },
    });

    fireEvent.click(view.getByRole("button", { name: "创建账号" }));

    const { registerWithProtobuf } = await import("../lib/api");
    await waitFor(() =>
      expect(registerWithProtobuf).toHaveBeenCalledWith(
        "neo",
        "neo@example.com",
        "secret12"
      )
    );
  });

  it("supports hidden whitelist uid login without showing whitelist prompt", async () => {
    const { container } = render(
      <MemoryRouter>
        <Login />
      </MemoryRouter>
    );
    const view = within(container);

    fireEvent.change(view.getByPlaceholderText("you@example.com"), {
      target: { value: "10000000" },
    });
    fireEvent.click(view.getByRole("button", { name: "登录" }));

    const { loginWithUid } = await import("../lib/api");
    await waitFor(() => expect(loginWithUid).toHaveBeenCalledWith("10000000"));
    expect(view.queryByText("白名单直达")).not.toBeInTheDocument();
  });
});
