package main

import (
	"fmt"
	"net/http"
	"time"
)

func renderLoginPage(errorMsg string) string {
	errorBlock := ""
	if errorMsg != "" {
		errorBlock = `<div class="error">Invalid username or password.</div>`
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Sign in — Miner Control Panel</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&display=swap" rel="stylesheet">
<style>
  :root {
    --bg: #0a0b0f;
    --bg-grad: radial-gradient(circle at 20%% -10%%, rgba(247,147,26,0.16), transparent 40%%),
               radial-gradient(circle at 90%% 15%%, rgba(99,102,241,0.12), transparent 35%%);
    --card: #14161d;
    --border: #23262f;
    --text: #edeef2;
    --muted: #8b92a3;
    --accent: #f7931a;
    --bad: #ef4444;
    --bad-soft: rgba(239,68,68,0.14);
  }
  * { box-sizing: border-box; }
  html { background: var(--bg); }
  body {
    margin: 0; min-height: 100vh;
    display: flex; align-items: center; justify-content: center;
    background-color: var(--bg);
    background-image: var(--bg-grad);
    background-attachment: fixed;
    background-repeat: no-repeat;
    color: var(--text);
    font-family: "Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    -webkit-font-smoothing: antialiased;
  }
  @keyframes fadeUp { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }
  @keyframes glowPulse { 0%%, 100%% { box-shadow: 0 4px 20px rgba(247,147,26,0.35); } 50%% { box-shadow: 0 4px 32px rgba(247,147,26,0.55); } }
  .login-card {
    width: 100%%; max-width: 380px; background: var(--card);
    border: 1px solid var(--border); border-radius: 20px;
    padding: 36px 32px; box-shadow: 0 24px 70px rgba(0,0,0,0.55);
    animation: fadeUp 0.4s ease both;
    position: relative; overflow: hidden;
  }
  .login-card::before {
    content: ""; position: absolute; inset: 0 0 auto 0; height: 3px;
    background: linear-gradient(90deg, var(--accent), #6366f1, var(--accent));
    background-size: 200%% auto; animation: shine 3s linear infinite;
  }
  @keyframes shine { to { background-position: -200%% 0; } }
  .brand { display: flex; align-items: center; gap: 12px; margin-bottom: 28px; }
  .brand-icon {
    width: 44px; height: 44px; border-radius: 13px;
    background: linear-gradient(135deg, var(--accent), #ffb84d);
    display: flex; align-items: center; justify-content: center; font-size: 22px;
    animation: glowPulse 2.5s ease-in-out infinite;
  }
  .brand h1 { font-size: 17px; margin: 0; font-weight: 800; letter-spacing: -0.01em; }
  .brand .sub { font-size: 11px; color: var(--muted); margin-top: 2px; }
  label { display: block; font-size: 12px; font-weight: 600; color: var(--muted); margin-bottom: 6px; margin-top: 18px; }
  input {
    width: 100%%; padding: 12px 14px; border-radius: 10px;
    border: 1px solid var(--border); background: #0d0f14; color: var(--text);
    font-size: 14px; font-family: inherit; outline: none; transition: border-color 0.15s ease, box-shadow 0.15s ease;
  }
  input:focus { border-color: var(--accent); box-shadow: 0 0 0 3px rgba(247,147,26,0.15); }
  button {
    width: 100%%; margin-top: 26px; padding: 13px; border-radius: 10px; border: none;
    background: linear-gradient(135deg, var(--accent), #e8850f); color: #111;
    font-size: 14px; font-weight: 700; font-family: inherit; cursor: pointer;
    transition: filter 0.15s ease, transform 0.15s ease;
  }
  button:hover { filter: brightness(1.1); transform: translateY(-1px); }
  button:active { transform: translateY(0); }
  .error {
    background: var(--bad-soft); color: #fca5a5; font-size: 13px;
    padding: 10px 12px; border-radius: 10px; margin-top: 18px;
    border: 1px solid rgba(239,68,68,0.25);
  }
</style>
</head>
<body>
  <form class="login-card" method="POST" action="/login/submit">
    <div class="brand">
      <div class="brand-icon">⛏️</div>
      <div>
        <h1>Miner Control Panel</h1>
        <div class="sub">Sign in to continue</div>
      </div>
    </div>
    <label for="username">Username</label>
    <input id="username" name="username" type="text" autocomplete="username" autofocus required>
    <label for="password">Password</label>
    <input id="password" name="password" type="password" autocomplete="current-password" required>
    %s
    <button type="submit">Sign in</button>
  </form>
</body>
</html>`, errorBlock)
}

func (d *Dashboard) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	errMsg := r.URL.Query().Get("error")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(renderLoginPage(errMsg)))
}

func (d *Dashboard) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
		return
	}
	user := r.FormValue("username")
	pass := r.FormValue("password")

	if d.auth == nil || !d.auth.Verify(user, pass) {
		time.Sleep(400 * time.Millisecond)
		http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
		return
	}

	token, expiry := d.sessions.Create()
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiry,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (d *Dashboard) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		d.sessions.Delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
