import { useThemeMode } from "../lib/useThemeMode";
import ThemeSwitcher from "../components/ThemeSwitcher";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Button, Input } from "../lib/vercel-ui";
import { loginWithPassword, registerWithProtobuf } from "../lib/api";
import { isWhitelistUid, parseUid } from "../lib/uid";
import { setLastUid, setToken, setUser } from "../lib/storage";

function FieldLabel({ children, htmlFor }) {
  return (
    <label htmlFor={htmlFor} className="text-xs font-semibold tracking-wide text-slate-300">
      {children}
    </label>
  );
}

export default function Login() {
  const navigate = useNavigate();
  const [theme, setTheme] = useThemeMode();
  const [uidInput, setUidInput] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const [registerOpen, setRegisterOpen] = useState(false);
  const [registerUsername, setRegisterUsername] = useState("");
  const [registerEmail, setRegisterEmail] = useState("");
  const [registerPassword, setRegisterPassword] = useState("");
  const [registerError, setRegisterError] = useState("");
  const [registerLoading, setRegisterLoading] = useState(false);

  const handleWhitelistEnter = () => {
    const uid = parseUid(uidInput);
    if (!uid) {
      setError("请输入有效的 uid");
      return;
    }
    setLastUid(uid);
    navigate(`/chat?uid=${uid}`);
  };

  const handleLogin = async (event) => {
    event.preventDefault();
    setError("");
    setLoading(true);
    try {
      const data = await loginWithPassword(email, password);
      const uid = data?.user?.id;
      if (!uid) {
        throw new Error("登录返回缺少 uid");
      }
      setLastUid(uid);
      setToken(uid, data.token);
      setUser(uid, data.user);
      navigate(`/chat?uid=${uid}`);
    } catch (err) {
      setError(err.message || "登录失败");
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (event) => {
    event.preventDefault();
    setRegisterError("");
    setRegisterLoading(true);
    try {
      const data = await registerWithProtobuf(
        registerUsername,
        registerEmail,
        registerPassword,
      );
      const uid = data?.user?.id;
      if (!uid) {
        throw new Error("注册返回缺少 uid");
      }
      setLastUid(uid);
      setToken(uid, data.token);
      setUser(uid, { ...data.user, id: uid });
      setRegisterOpen(false);
      navigate(`/chat?uid=${uid}`);
    } catch (err) {
      setRegisterError(err.message || "注册失败");
    } finally {
      setRegisterLoading(false);
    }
  };

  return (
    <div className="min-h-screen theme-page-bg px-4 py-8 text-slate-100 md:px-8">
      <div className="mx-auto mb-4 flex max-w-6xl justify-end">
        <ThemeSwitcher theme={theme} onChange={setTheme} />
      </div>
      <div className="mx-auto grid max-w-6xl gap-6 lg:grid-cols-[1.15fr_1fr]">
        <section className="discord-surface rounded-2xl p-8 shadow-glow">
          <div className="inline-flex items-center rounded-full border border-[#404249] bg-[#232428] px-3 py-1 text-xs font-semibold text-[#a5b0ff]">
            SOCIAL APP / REALTIME
          </div>
          <h1 className="mt-6 font-display text-4xl font-extrabold leading-tight text-white md:text-5xl">
            登录入口
          </h1>
          <p className="mt-4 max-w-xl text-sm leading-7 text-slate-300">
            Discord 风格的实时消息应用。使用 URL 中的 uid 绑定身份；支持普通账号登录与白名单快速进入。
          </p>
          <div className="mt-10 grid gap-3 sm:grid-cols-3">
            {[
              ["实时消息", "WebSocket + Protobuf"],
              ["多账号", "URL uid 切换"],
              ["关系链路", "关注/粉丝/好友"],
            ].map(([title, text]) => (
              <div key={title} className="rounded-xl border border-[#3c3f45] bg-[#23252b] p-4">
                <p className="font-semibold text-white">{title}</p>
                <p className="mt-1 text-xs text-slate-400">{text}</p>
              </div>
            ))}
          </div>
        </section>

        <section className="discord-surface rounded-2xl p-6 shadow-glow md:p-8">
          <div className="space-y-6">
            <div className="rounded-xl border border-[#3c3f45] bg-[#23252b] p-4">
              <p className="text-sm font-semibold text-white">白名单直达</p>
              <p className="mt-1 text-xs text-slate-400">
                白名单区间：10000000 - 20000000。当前输入
                {isWhitelistUid(parseUid(uidInput)) ? "属于" : "不属于"}白名单。
              </p>
              <div className="mt-4 space-y-3">
                <Input
                  value={uidInput}
                  onChange={(event) => setUidInput(event.target.value)}
                  placeholder="输入 uid，例如 20000001"
                  className="!h-11 !rounded-lg !border-[#3f4248] !bg-[#1e1f22] !text-slate-100 !placeholder:text-slate-500 focus:!border-[#5865f2]"
                />
                <div className="grid grid-cols-2 gap-2">
                  <Button
                    variant="black"
                    className="!h-10 !rounded-lg !border-[#5865f2] !bg-[#5865f2] !text-white hover:!bg-[#4752c4]"
                    onClick={handleWhitelistEnter}
                  >
                    进入聊天
                  </Button>
                  <Button
                    variant="secondary"
                    className="!h-10 !rounded-lg !border-[#4d5159] !bg-[#2b2d31] !text-slate-100 hover:!bg-[#313338]"
                    onClick={() => setUidInput("20000001")}
                  >
                    填入示例 uid
                  </Button>
                </div>
              </div>
            </div>

            <form onSubmit={handleLogin} className="space-y-3">
              <h2 className="font-display text-2xl font-bold text-white">普通账号登录</h2>
              <div className="space-y-2">
                <FieldLabel htmlFor="login-email">登录邮箱</FieldLabel>
                <Input
                  id="login-email"
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                  placeholder="you@example.com"
                  className="!h-11 !rounded-lg !border-[#3f4248] !bg-[#1e1f22] !text-slate-100 !placeholder:text-slate-500 focus:!border-[#5865f2]"
                />
              </div>
              <div className="space-y-2">
                <FieldLabel htmlFor="login-password">登录密码</FieldLabel>
                <Input
                  id="login-password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  type="password"
                  placeholder="••••••"
                  className="!h-11 !rounded-lg !border-[#3f4248] !bg-[#1e1f22] !text-slate-100 !placeholder:text-slate-500 focus:!border-[#5865f2]"
                />
              </div>
              {error ? (
                <div className="rounded-lg border border-[#5f2a33] bg-[#3b1f24] px-3 py-2 text-sm text-[#ffb4bf]">
                  {error}
                </div>
              ) : null}
              <div className="grid grid-cols-2 gap-2">
                <Button
                  type="submit"
                  loading={loading}
                  variant="black"
                  className="!h-10 !rounded-lg !border-[#5865f2] !bg-[#5865f2] !text-white hover:!bg-[#4752c4]"
                >
                  登录
                </Button>
                <Button
                  type="button"
                  variant="secondary"
                  className="!h-10 !rounded-lg !border-[#4d5159] !bg-[#2b2d31] !text-slate-100 hover:!bg-[#313338]"
                  onClick={() => setRegisterOpen(true)}
                >
                  注册账号
                </Button>
              </div>
            </form>
          </div>
        </section>
      </div>

      {registerOpen ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/65 p-4">
          <div className="discord-surface w-full max-w-xl rounded-2xl p-6 shadow-glow">
            <h2 className="font-display text-2xl font-bold text-white">创建新账号</h2>
            <p className="mt-1 text-sm text-slate-400">注册成功后会自动登录并跳转聊天页。</p>
            <form onSubmit={handleRegister} className="mt-5 space-y-3">
              <div className="space-y-2">
                <FieldLabel htmlFor="register-username">用户名</FieldLabel>
                <Input
                  id="register-username"
                  value={registerUsername}
                  onChange={(event) => setRegisterUsername(event.target.value)}
                  className="!h-11 !rounded-lg !border-[#3f4248] !bg-[#1e1f22] !text-slate-100 !placeholder:text-slate-500 focus:!border-[#5865f2]"
                />
              </div>
              <div className="space-y-2">
                <FieldLabel htmlFor="register-email">邮箱</FieldLabel>
                <Input
                  id="register-email"
                  value={registerEmail}
                  onChange={(event) => setRegisterEmail(event.target.value)}
                  className="!h-11 !rounded-lg !border-[#3f4248] !bg-[#1e1f22] !text-slate-100 !placeholder:text-slate-500 focus:!border-[#5865f2]"
                />
              </div>
              <div className="space-y-2">
                <FieldLabel htmlFor="register-password">密码</FieldLabel>
                <Input
                  id="register-password"
                  value={registerPassword}
                  onChange={(event) => setRegisterPassword(event.target.value)}
                  type="password"
                  className="!h-11 !rounded-lg !border-[#3f4248] !bg-[#1e1f22] !text-slate-100 !placeholder:text-slate-500 focus:!border-[#5865f2]"
                />
              </div>
              {registerError ? (
                <div className="rounded-lg border border-[#5f2a33] bg-[#3b1f24] px-3 py-2 text-sm text-[#ffb4bf]">
                  {registerError}
                </div>
              ) : null}
              <div className="grid grid-cols-2 gap-2">
                <Button
                  type="submit"
                  loading={registerLoading}
                  variant="black"
                  className="!h-10 !rounded-lg !border-[#5865f2] !bg-[#5865f2] !text-white hover:!bg-[#4752c4]"
                >
                  创建账号
                </Button>
                <Button
                  type="button"
                  variant="secondary"
                  className="!h-10 !rounded-lg !border-[#4d5159] !bg-[#2b2d31] !text-slate-100 hover:!bg-[#313338]"
                  onClick={() => setRegisterOpen(false)}
                >
                  取消
                </Button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </div>
  );
}
