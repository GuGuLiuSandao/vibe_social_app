import { useThemeMode } from "../lib/useThemeMode";
import ThemeSwitcher from "../components/ThemeSwitcher";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { loginWithPassword, loginWithUid, registerWithProtobuf } from "../lib/api";
import { setLastUid, setToken, setUser } from "../lib/storage";
import { isWhitelistUid, parseUid } from "../lib/uid";

const STAR_LAYOUT = [
  { left: "4%", top: "16%", size: 2, delay: "-0.8s", duration: "3.8s", opacity: 0.72 },
  { left: "10%", top: "34%", size: 3, delay: "-2.1s", duration: "5.6s", opacity: 0.86 },
  { left: "17%", top: "66%", size: 2, delay: "-1.2s", duration: "4.4s", opacity: 0.68 },
  { left: "22%", top: "21%", size: 2, delay: "-3.7s", duration: "6.2s", opacity: 0.75 },
  { left: "28%", top: "48%", size: 3, delay: "-2.6s", duration: "5.1s", opacity: 0.8 },
  { left: "33%", top: "82%", size: 2, delay: "-0.4s", duration: "4.8s", opacity: 0.7 },
  { left: "39%", top: "12%", size: 3, delay: "-1.8s", duration: "6.5s", opacity: 0.9 },
  { left: "45%", top: "38%", size: 2, delay: "-2.9s", duration: "4.2s", opacity: 0.66 },
  { left: "51%", top: "57%", size: 2, delay: "-1.4s", duration: "5.7s", opacity: 0.77 },
  { left: "56%", top: "74%", size: 3, delay: "-3.3s", duration: "6.1s", opacity: 0.84 },
  { left: "62%", top: "26%", size: 2, delay: "-0.9s", duration: "4.5s", opacity: 0.7 },
  { left: "68%", top: "44%", size: 3, delay: "-2.2s", duration: "5.2s", opacity: 0.87 },
  { left: "73%", top: "68%", size: 2, delay: "-3.6s", duration: "6.4s", opacity: 0.73 },
  { left: "79%", top: "14%", size: 2, delay: "-1.6s", duration: "4.1s", opacity: 0.65 },
  { left: "84%", top: "36%", size: 3, delay: "-0.5s", duration: "5.9s", opacity: 0.88 },
  { left: "90%", top: "58%", size: 2, delay: "-2.7s", duration: "4.9s", opacity: 0.74 },
  { left: "95%", top: "80%", size: 2, delay: "-1.1s", duration: "5.5s", opacity: 0.76 },
  { left: "13%", top: "8%", size: 2, delay: "-2.5s", duration: "6.6s", opacity: 0.82 },
  { left: "27%", top: "90%", size: 2, delay: "-0.7s", duration: "4.6s", opacity: 0.69 },
  { left: "58%", top: "6%", size: 3, delay: "-3.1s", duration: "5.4s", opacity: 0.86 },
  { left: "76%", top: "86%", size: 2, delay: "-1.9s", duration: "6.3s", opacity: 0.78 },
  { left: "88%", top: "8%", size: 2, delay: "-2.8s", duration: "4.7s", opacity: 0.71 },
];

export default function Login() {
  const navigate = useNavigate();
  const [theme, setTheme] = useThemeMode();
  const [mode, setMode] = useState("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const [registerUsername, setRegisterUsername] = useState("");
  const [registerEmail, setRegisterEmail] = useState("");
  const [registerPassword, setRegisterPassword] = useState("");
  const [registerError, setRegisterError] = useState("");
  const [registerLoading, setRegisterLoading] = useState(false);
  const isRegister = mode === "register";

  const handleLogin = async (event) => {
    event.preventDefault();
    setError("");
    setLoading(true);
    try {
      const normalizedUid = parseUid(email);
      const shouldUseWhitelistLogin = normalizedUid && isWhitelistUid(normalizedUid);
      const data = shouldUseWhitelistLogin
        ? await loginWithUid(normalizedUid)
        : await loginWithPassword(email, password);
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
      setMode("login");
      navigate(`/chat?uid=${uid}`);
    } catch (err) {
      setRegisterError(err.message || "注册失败");
    } finally {
      setRegisterLoading(false);
    }
  };

  return (
    <div className="auth-shell theme-page-bg text-slate-100">
      <div className="auth-grid-overlay" aria-hidden="true" />
      <div className="auth-ambient-glow" aria-hidden="true" />
      <div className="auth-starfield" aria-hidden="true">
        {STAR_LAYOUT.map((star, index) => (
          <span
            // eslint-disable-next-line react/no-array-index-key
            key={index}
            className="auth-star"
            style={{
              left: star.left,
              top: star.top,
              "--star-size": `${star.size}px`,
              "--star-delay": star.delay,
              "--star-duration": star.duration,
              "--star-opacity": star.opacity,
            }}
          />
        ))}
      </div>
      <div className="absolute right-4 top-4 z-20 md:right-6 md:top-6">
        <ThemeSwitcher theme={theme} onChange={setTheme} compact />
      </div>

      <div className="auth-center px-4 py-8 md:px-8">
        <Card className="auth-card w-full max-w-[500px] rounded-3xl border-border bg-card shadow-glow backdrop-blur-xl">
          <CardHeader className="space-y-3 pb-4 text-center">
            <div className="mx-auto inline-flex items-center rounded-full border border-border bg-muted px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.16em] text-primary">
              Vibe Social
            </div>
            <CardTitle className="font-display text-3xl font-extrabold text-foreground md:text-[34px]">
              {isRegister ? "创建账号" : "欢迎回来"}
            </CardTitle>
            <CardDescription className="text-sm leading-6 text-muted-foreground">
              {isRegister ? "填写信息后即可进入实时社交会话。" : "使用账号继续访问你的实时社交会话。"}
            </CardDescription>
          </CardHeader>

          <CardContent className="space-y-6 px-6 pb-7 pt-0 md:px-8 md:pb-8">
          {isRegister ? (
            <form onSubmit={handleRegister} className="space-y-5">
              <div className="space-y-2.5">
                <Label htmlFor="register-username" className="text-xs font-semibold tracking-wide text-foreground">用户名</Label>
                <Input
                  id="register-username"
                  value={registerUsername}
                  onChange={(event) => setRegisterUsername(event.target.value)}
                  placeholder="请输入用户名"
                  className="auth-input h-11 rounded-xl border-input bg-background text-foreground placeholder:text-muted-foreground"
                />
              </div>
              <div className="space-y-2.5">
                <Label htmlFor="register-email" className="text-xs font-semibold tracking-wide text-foreground">邮箱</Label>
                <Input
                  id="register-email"
                  value={registerEmail}
                  onChange={(event) => setRegisterEmail(event.target.value)}
                  placeholder="you@example.com"
                  className="auth-input h-11 rounded-xl border-input bg-background text-foreground placeholder:text-muted-foreground"
                />
              </div>
              <div className="space-y-2.5">
                <Label htmlFor="register-password" className="text-xs font-semibold tracking-wide text-foreground">密码</Label>
                <Input
                  id="register-password"
                  value={registerPassword}
                  onChange={(event) => setRegisterPassword(event.target.value)}
                  type="password"
                  placeholder="至少 6 位"
                  className="auth-input h-11 rounded-xl border-input bg-background text-foreground placeholder:text-muted-foreground"
                />
              </div>
              {registerError ? (
                <div className="rounded-lg border border-destructive bg-destructive px-3 py-2 text-sm text-destructive-foreground">
                  {registerError}
                </div>
              ) : null}
              <Button
                type="submit"
                loading={registerLoading}
                className="h-11 w-full rounded-xl text-sm font-semibold"
              >
                创建账号
              </Button>
              <p className="text-center text-xs text-muted-foreground">
                已有账号？
                {" "}
                <button
                  type="button"
                  className="font-semibold text-primary transition-colors hover:text-foreground"
                  onClick={() => {
                    setMode("login");
                    setRegisterError("");
                  }}
                >
                  去登录
                </button>
              </p>
            </form>
          ) : (
            <form onSubmit={handleLogin} className="space-y-5">
              <div className="space-y-2.5">
                <Label htmlFor="login-email" className="text-xs font-semibold tracking-wide text-foreground">邮箱</Label>
                <Input
                  id="login-email"
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                  placeholder="you@example.com"
                  className="auth-input h-11 rounded-xl border-input bg-background text-foreground placeholder:text-muted-foreground"
                />
              </div>
              <div className="space-y-2.5">
                <Label htmlFor="login-password" className="text-xs font-semibold tracking-wide text-foreground">密码</Label>
                <Input
                  id="login-password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  type="password"
                  placeholder="••••••"
                  className="auth-input h-11 rounded-xl border-input bg-background text-foreground placeholder:text-muted-foreground"
                />
              </div>
              {error ? (
                <div className="rounded-lg border border-destructive bg-destructive px-3 py-2 text-sm text-destructive-foreground">
                  {error}
                </div>
              ) : null}
              <Button
                type="submit"
                loading={loading}
                className="h-11 w-full rounded-xl text-sm font-semibold"
              >
                登录
              </Button>
              <p className="text-center text-xs text-muted-foreground">
                没有账号？
                {" "}
                <button
                  type="button"
                  className="font-semibold text-primary transition-colors hover:text-foreground"
                  onClick={() => {
                    setMode("register");
                    setError("");
                  }}
                >
                  立即注册
                </button>
              </p>
            </form>
          )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
