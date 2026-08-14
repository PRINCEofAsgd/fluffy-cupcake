// 页面交互：点击反馈始终可用，只有登录状态才同步并展示数据库记录。
(() => {
  "use strict";

  const loginPanel = document.querySelector("#login-panel");
  const loginButton = document.querySelector("#login-button");
  const closeLoginButton = document.querySelector("#close-login-button");
  const loginForm = document.querySelector("#login-form");
  const loginMessage = document.querySelector("#login-message");
  const authStatus = document.querySelector("#auth-status");
  const logoutButton = document.querySelector("#logout-button");
  const button = document.querySelector("#miss-button");
  const floatLayer = document.querySelector("#float-layer");
  const liveMessage = document.querySelector("#live-message");
  const totalCount = document.querySelector("#total-count");
  const dailyCounts = document.querySelector("#daily-counts");
  const clickMinutes = document.querySelector("#click-minutes");
  const syncStatus = document.querySelector("#sync-status");
  const statsPanel = document.querySelector("#stats-panel");
  const messageText = button instanceof HTMLButtonElement ? button.dataset.message : "";
  let clickAudioGraph = null;
  let sessionClickCount = 0;
  let pendingClicks = 0;
  let flushTimer = 0;
  let retryTimer = 0;
  let requestInFlight = false;
  let authenticated = false;

  if (!(loginForm instanceof HTMLFormElement) || !(button instanceof HTMLButtonElement) ||
      !(floatLayer instanceof HTMLElement) || !messageText) {
    return;
  }

  // setLoginPanel 控制独立登录入口，不影响主按钮是否可用。
  const setLoginPanel = (visible) => {
    if (loginPanel instanceof HTMLElement) {
      loginPanel.hidden = !visible;
    }
    if (loginButton instanceof HTMLButtonElement) {
      loginButton.setAttribute("aria-expanded", String(visible));
    }
    if (visible) {
      document.querySelector("#username")?.focus();
    }
  };

  // setAuthenticated 只切换登录操作和数据库统计，主按钮始终保持可点击。
  const setAuthenticated = (user) => {
    authenticated = Boolean(user);
    if (loginButton instanceof HTMLButtonElement) {
      loginButton.hidden = authenticated;
    }
    if (logoutButton instanceof HTMLButtonElement) {
      logoutButton.hidden = !authenticated;
    }
    if (statsPanel instanceof HTMLElement) {
      statsPanel.hidden = !authenticated;
    }
    if (authStatus instanceof HTMLElement) {
      authStatus.textContent = authenticated
        ? `你好，${user.username}：点击将同步记录`
        : "未登录：本次点击只显示页面效果";
    }
    if (!authenticated) {
      setLoginPanel(false);
    }
  };

  // requestJSON 统一发送同源 JSON 请求；Cookie 由浏览器自动携带。
  const requestJSON = async (url, options = {}) => {
    const response = await fetch(url, {
      credentials: "same-origin",
      ...options,
      headers: options.body ? { "Content-Type": "application/json", ...options.headers } : options.headers,
    });
    const body = await response.json().catch(() => ({}));
    if (!response.ok) {
      const error = new Error(body.message || "请求失败");
      error.status = response.status;
      throw error;
    }
    return body;
  };

  // renderList 用文本节点渲染统计，避免把服务端数据拼接为 HTML。
  const renderList = (element, items, formatter) => {
    if (!(element instanceof HTMLElement)) {
      return;
    }
    element.replaceChildren();
    const visibleItems = items.slice(-8).reverse();
    if (visibleItems.length === 0) {
      const item = document.createElement("li");
      item.textContent = "还没有记录";
      element.append(item);
      return;
    }
    for (const value of visibleItems) {
      const item = document.createElement("li");
      item.textContent = formatter(value);
      element.append(item);
    }
  };

  // loadStats 读取共享总数、UTC 每日数和按分钟合并后的时间列表。
  const loadStats = async () => {
    if (!authenticated) {
      return;
    }
    try {
      const stats = await requestJSON("/api/yanlili/clicks/stats");
      if (totalCount instanceof HTMLElement) {
        totalCount.textContent = String(stats.total_count ?? 0);
      }
      renderList(dailyCounts, stats.daily_counts ?? [], (item) => `${item.date}：${item.count} 次`);
      renderList(clickMinutes, stats.click_minutes ?? [], (item) => {
        const localTime = new Date(item.time).toLocaleString([], { dateStyle: "short", timeStyle: "short" });
        return `${localTime}：${item.count} 次`;
      });
    } catch (error) {
      if (error.status === 401) {
        setAuthenticated(null);
      }
      if (syncStatus instanceof HTMLElement) {
        syncStatus.textContent = "统计读取失败";
      }
    }
  };

  // scheduleFlush 在短窗口内合并快速连续点击，失败时保留 pending 数量重试。
  const scheduleFlush = (delay = 750) => {
    window.clearTimeout(flushTimer);
    flushTimer = window.setTimeout(() => void flushClicks(), delay);
  };

  const flushClicks = async (keepalive = false) => {
    if (!authenticated || requestInFlight || pendingClicks <= 0) {
      return;
    }
    const batch = Math.min(pendingClicks, 100);
    pendingClicks -= batch;
    requestInFlight = true;
    if (syncStatus instanceof HTMLElement) {
      syncStatus.textContent = `正在同步 ${batch} 次…`;
    }
    try {
      await requestJSON("/api/yanlili/clicks", {
        method: "POST",
        body: JSON.stringify({ count: batch }),
        keepalive,
      });
      if (syncStatus instanceof HTMLElement) {
        syncStatus.textContent = pendingClicks > 0 ? `待同步 ${pendingClicks} 次` : "已同步";
      }
      void loadStats();
    } catch (error) {
      pendingClicks += batch;
      if (syncStatus instanceof HTMLElement) {
        syncStatus.textContent = `同步失败，保留 ${pendingClicks} 次待重试`;
      }
      if (error.status === 401) {
        setAuthenticated(null);
      } else {
        window.clearTimeout(retryTimer);
        retryTimer = window.setTimeout(() => {
          retryTimer = 0;
          void flushClicks();
        }, 2000);
      }
    } finally {
      requestInFlight = false;
      if (authenticated && pendingClicks > 0 && !retryTimer) {
        scheduleFlush(0);
      }
    }
  };

  // getClickAudioGraph 延迟创建共享音频上下文和限幅器，避免页面加载时申请音频设备。
  const getClickAudioGraph = () => {
    if (clickAudioGraph) {
      return clickAudioGraph;
    }
    const AudioContextClass = window.AudioContext || window.webkitAudioContext;
    if (typeof AudioContextClass !== "function") {
      return null;
    }
    try {
      const context = new AudioContextClass();
      const limiter = context.createDynamicsCompressor();
      limiter.threshold.value = -10;
      limiter.knee.value = 8;
      limiter.ratio.value = 12;
      limiter.attack.value = 0.003;
      limiter.release.value = 0.12;
      limiter.connect(context.destination);
      clickAudioGraph = { context, limiter };
      return clickAudioGraph;
    } catch {
      return null;
    }
  };

  // playClickSound 为每次点击创建独立短声部；声部可并发，并由共享限幅器防止叠加爆音。
  const playClickSound = () => {
    const graph = getClickAudioGraph();
    if (!graph) {
      return;
    }
    const { context, limiter } = graph;
    if (context.state === "suspended") {
      void context.resume().catch(() => {
        // 浏览器若因设备策略拒绝音频，仍保留文字动画和点击同步功能。
      });
    }

    const startedAt = context.currentTime;
    const endedAt = startedAt + 0.09;
    const oscillator = context.createOscillator();
    const envelope = context.createGain();

    oscillator.type = "triangle";
    oscillator.frequency.setValueAtTime(620, startedAt);
    oscillator.frequency.exponentialRampToValueAtTime(190, endedAt);
    envelope.gain.setValueAtTime(0.0001, startedAt);
    envelope.gain.exponentialRampToValueAtTime(0.14, startedAt + 0.004);
    envelope.gain.exponentialRampToValueAtTime(0.0001, endedAt);
    oscillator.connect(envelope);
    envelope.connect(limiter);
    oscillator.addEventListener("ended", () => {
      oscillator.disconnect();
      envelope.disconnect();
    }, { once: true });
    oscillator.start(startedAt);
    oscillator.stop(endedAt);
  };

  // createFloatingMessage 创建独立动画元素，使连续点击也能逐条显示。
  const createFloatingMessage = () => {
    sessionClickCount += 1;
    const message = document.createElement("span");
    const horizontalOffset = Math.round((Math.random() - 0.5) * 34);
    const tilt = (Math.random() - 0.5) * 5;

    message.className = "floating-message";
    message.textContent = messageText;
    message.style.left = `${50 + horizontalOffset}%`;
    message.style.setProperty("--tilt", `${tilt.toFixed(2)}deg`);
    message.addEventListener("animationend", () => message.remove(), { once: true });
    floatLayer.append(message);

    if (liveMessage instanceof HTMLElement) {
      liveMessage.textContent = `${messageText}，本次访问共 ${sessionClickCount} 次`;
    }

    // 未登录点击仅保留本地反馈，不加入稍后可能发送的 pending 队列。
    if (authenticated) {
      pendingClicks += 1;
      scheduleFlush();
    }
    playClickSound();
  };

  button.addEventListener("click", createFloatingMessage);
  button.addEventListener("pointerdown", () => button.classList.add("is-pressed"));
  for (const eventName of ["pointerup", "pointercancel", "pointerleave"]) {
    button.addEventListener(eventName, () => button.classList.remove("is-pressed"));
  }

  loginButton?.addEventListener("click", () => setLoginPanel(true));
  closeLoginButton?.addEventListener("click", () => setLoginPanel(false));

  loginForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    const data = new FormData(loginForm);
    if (loginMessage instanceof HTMLElement) {
      loginMessage.textContent = "正在登录…";
    }
    try {
      await requestJSON("/api/auth/login", {
        method: "POST",
        body: JSON.stringify({ username: data.get("username"), password: data.get("password") }),
      });
      loginForm.reset();
      const user = await requestJSON("/api/auth/me");
      setAuthenticated(user);
      setLoginPanel(false);
      if (loginMessage instanceof HTMLElement) {
        loginMessage.textContent = "";
      }
      await loadStats();
      if (pendingClicks > 0) {
        scheduleFlush(0);
      }
    } catch (error) {
      if (loginMessage instanceof HTMLElement) {
        loginMessage.textContent = error.message;
      }
    }
  });

  logoutButton?.addEventListener("click", async () => {
    await flushClicks();
    try {
      await requestJSON("/api/auth/logout", { method: "POST" });
    } finally {
      setAuthenticated(null);
    }
  });

  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "hidden") {
      void flushClicks(true);
    }
  });
  window.addEventListener("pagehide", () => void flushClicks(true));

  // 页面载入先用 HttpOnly Cookie 查询当前用户，JavaScript 从不读取 Token 本身。
  void requestJSON("/api/auth/me")
    .then((user) => {
      setAuthenticated(user);
      return loadStats();
    })
    .catch(() => setAuthenticated(null));
})();
