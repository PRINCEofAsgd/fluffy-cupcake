// 页面交互：匿名和未绑定用户保留本地反馈，只有双向绑定后才按方向同步想念记录。
(() => {
  "use strict";

  const byID = (id) => document.getElementById(id);
  const loginPanel = byID("login-panel");
  const loginButton = byID("login-button");
  const closeLoginButton = byID("close-login-button");
  const loginForm = byID("login-form");
  const loginMessage = byID("login-message");
  const authStatus = byID("auth-status");
  const logoutButton = byID("logout-button");
  const companionButton = byID("companion-button");
  const companionPanel = byID("companion-panel");
  const closeCompanionButton = byID("close-companion-button");
  const companionForm = byID("companion-form");
  const companionMessage = byID("companion-message");
  const inboxButton = byID("inbox-button");
  const inboxPanel = byID("inbox-panel");
  const closeInboxButton = byID("close-inbox-button");
  const inboxList = byID("inbox-list");
  const inboxMessage = byID("inbox-message");
  const noteButton = byID("note-button");
  const notePanel = byID("note-panel");
  const closeNoteButton = byID("close-note-button");
  const noteForm = byID("note-form");
  const noteMessage = byID("note-message");
  const partnerNote = byID("partner-note");
  const detailsButton = byID("details-button");
  const detailsPanel = byID("details-panel");
  const closeDetailsButton = byID("close-details-button");
  const partnerSelect = byID("partner-select");
  const detailsList = byID("details-list");
  const previousPage = byID("previous-page");
  const nextPage = byID("next-page");
  const pageStatus = byID("page-status");
  const button = byID("miss-button");
  const floatLayer = byID("float-layer");
  const liveMessage = byID("live-message");
  const totalCount = byID("total-count");
  const dailyCounts = byID("daily-counts");
  const clickMinutes = byID("click-minutes");
  const syncStatus = byID("sync-status");
  const statsPanel = byID("stats-panel");
  const statsTitle = byID("stats-title");
  const directionSwitch = byID("direction-switch");
  const qrScanButton = byID("qr-scan-button");
  const qrPickButton = byID("qr-pick-button");
  const qrFileInput = byID("qr-file-input");
  const qrVideo = byID("qr-video");
  const qrCanvas = byID("qr-canvas");

  let clickAudioGraph = null;
  let sessionClickCount = 0;
  let pendingClicks = 0;
  let flushTimer = 0;
  let retryTimer = 0;
  let requestInFlight = false;
  let currentUser = null;
  let refreshInFlight = null;
  let refreshGeneration = 0;
  const loginMarker = "fluffy-cupcake-signed-in";
  let currentBinding = { bound: false };
  let statsDirection = "mine";
  let detailPage = 1;
  let detailTotalPages = 0;
  let qrScanFrame = 0;
  let qrScanFrameType = "";
  let qrScanGeneration = 0;
  let qrLastScanAt = 0;
  let qrStream = null;
  let qrLoginInFlight = false;

  const qrCameraMaxEdge = 1400;
  const qrImageMaxEdge = 1800;
  const qrScanIntervalMs = 120;

  if (!(loginForm instanceof HTMLFormElement) || !(companionForm instanceof HTMLFormElement) ||
      !(noteForm instanceof HTMLFormElement) || !(button instanceof HTMLButtonElement) ||
      !(floatLayer instanceof HTMLElement)) {
    return;
  }

  // fetchJSON 执行一次同源请求并保留状态码，供鉴权流程区分中间件拒绝和业务错误。
  const fetchJSON = async (url, options = {}) => {
    const response = await fetch(url, {
      credentials: "same-origin",
      ...options,
      headers: options.body ? { "Content-Type": "application/json", ...options.headers } : options.headers,
    });
    const body = await response.json().catch(() => ({}));
    return { response, body };
  };

  // authRequired 只识别鉴权中间件在业务 Handler 执行之前发出的拒绝。
  const authRequired = ({ response, body }) => response.status === 401 && body.code === "AUTH_REQUIRED";

  // sessionUnavailable 表示原业务请求还未执行，可以在暂时故障恢复后重试。
  const sessionUnavailable = () => Object.assign(
    new Error("暂时无法校验登录，请检查网络或稍后重试"), { safeToRetry: true },
  );

  // signedInBefore 记录是否曾登录，用于在刷新 Cookie 自然到期后显示重新登录提示。
  const signedInBefore = () => {
    try { return localStorage.getItem(loginMarker) === "1"; } catch { return false; }
  };

  // rememberLogin 只保存界面状态，认证凭证始终由浏览器 Cookie 管理。
  const rememberLogin = (signedIn) => {
    try {
      if (signedIn) localStorage.setItem(loginMarker, "1");
      else localStorage.removeItem(loginMarker);
    } catch { /* 禁用本地存储时仍可正常登录。 */ }
  };

  // hasValidAccess 检查浏览器是否已收到其他标签页签发的新访问 Cookie。
  const hasValidAccess = async () => {
    const result = await fetchJSON("/api/auth/me");
    if (result.response.ok) return true;
    if (authRequired(result)) return false;
    throw sessionUnavailable();
  };

  // recoverSession 协调同源标签页的轮换；浏览器不支持 Web Locks 时由后端识别并发。
  const recoverSession = async () => {
    const recover = async () => {
      if (await hasValidAccess()) return;
      const { response, body } = await fetchJSON("/api/auth/refresh", { method: "POST" });
      if (response.ok) return;
      if (response.status === 401 && (body.code === "SESSION_ENDED" || body.code === "NO_SESSION")) {
        const prompt = body.code === "SESSION_ENDED" || Boolean(currentUser) || signedInBefore();
        setAuthenticated(null);
        if (prompt && authStatus instanceof HTMLElement) authStatus.textContent = "登录已失效，请重新登录";
        throw Object.assign(new Error(prompt ? "登录已失效，请重新登录" : "请先登录"), { sessionEnded: true });
      }
      if (response.status === 409 && body.code === "REFRESH_CONFLICT") {
        // 等待获胜请求写入共享 Cookie，只用只读接口检查，避免再次轮换或重复提交写入。
        for (const delay of [100, 250, 500, 1000, 1500]) {
          await new Promise((resolve) => window.setTimeout(resolve, delay));
          if (await hasValidAccess()) return;
        }
      }
      throw sessionUnavailable();
    };
    try {
      if (navigator.locks?.request) await navigator.locks.request("fluffy-cupcake-auth-refresh", recover);
      else await recover();
      refreshGeneration += 1;
    } catch (error) {
      if (error.sessionEnded) throw error;
      throw sessionUnavailable();
    }
  };

  // requestJSON 合并同页并发刷新，只重放被中间件拦截的原请求一次。
  const requestJSON = async (url, options = {}) => {
    const generation = refreshGeneration;
    let result = await fetchJSON(url, options);
    if (authRequired(result)) {
      if (generation === refreshGeneration) {
        refreshInFlight ??= recoverSession().finally(() => { refreshInFlight = null; });
        await refreshInFlight;
      }
      result = await fetchJSON(url, options);
      if (authRequired(result)) throw sessionUnavailable();
    }
    const { response, body } = result;
    if (!response.ok) {
      const error = new Error(body.message || "请求失败");
      error.status = response.status;
      error.safeToRetry = response.status === 503 && body.code === "AUTH_UNAVAILABLE";
      throw error;
    }
    return body;
  };

  const setPanel = (panel, visible, focusSelector = "") => {
    if (panel instanceof HTMLElement) {
      panel.hidden = !visible;
      if (visible) {
        panel.scrollIntoView({ behavior: "smooth", block: "center" });
      }
    }
    if (visible && focusSelector) {
      document.querySelector(focusSelector)?.focus();
    }
  };

  const setLoginPanel = (visible) => {
    setPanel(loginPanel, visible, "#username");
    loginButton?.setAttribute("aria-expanded", String(visible));
    if (!visible) stopQrScan();
  };

  // setAuthenticated 更新入口显示；绑定状态由独立接口读取，不能由登录状态推断。
  const setAuthenticated = (user) => {
    currentUser = user || null;
    const authenticated = Boolean(currentUser);
    rememberLogin(authenticated);
    if (loginButton instanceof HTMLButtonElement) loginButton.hidden = authenticated;
    if (logoutButton instanceof HTMLButtonElement) logoutButton.hidden = !authenticated;
    if (inboxButton instanceof HTMLButtonElement) inboxButton.hidden = !authenticated;
    if (authStatus instanceof HTMLElement) {
      authStatus.textContent = authenticated ? `你好，${currentUser.username}` : "未登录：点击只显示页面效果";
    }
    if (!authenticated) {
      currentBinding = { bound: false };
      pendingClicks = 0;
      window.clearTimeout(retryTimer);
      retryTimer = 0;
      setLoginPanel(false);
      setPanel(companionPanel, false);
      setPanel(inboxPanel, false);
      setPanel(notePanel, false);
      setPanel(detailsPanel, false);
    }
    renderBindingState();
  };

  // renderBindingState 决定绑定入口、备注入口、统计区和点击宾语。
  const renderBindingState = () => {
    const authenticated = Boolean(currentUser);
    const bound = authenticated && Boolean(currentBinding.bound);
    if (companionButton instanceof HTMLButtonElement) companionButton.hidden = bound;
    if (noteButton instanceof HTMLButtonElement) noteButton.hidden = !bound;
    if (statsPanel instanceof HTMLElement) statsPanel.hidden = !bound;
    if (!bound) setPanel(notePanel, false);
    if (authStatus instanceof HTMLElement && bound) {
      const displayName = currentBinding.partner_note || currentBinding.partner_username;
      authStatus.textContent = `已与 ${displayName} 完成陪伴绑定`;
    }
  };

  const loadCompanionState = async () => {
    if (!currentUser) return;
    currentBinding = await requestJSON("/api/companion/state");
    if (!currentBinding.bound) pendingClicks = 0;
    renderBindingState();
    if (currentBinding.bound) await loadStats();
  };

  const renderList = (element, items, formatter) => {
    if (!(element instanceof HTMLElement)) return;
    element.replaceChildren();
    if (items.length === 0) {
      const item = document.createElement("li");
      item.textContent = "还没有记录";
      element.append(item);
      return;
    }
    for (const value of items) {
      const item = document.createElement("li");
      item.textContent = formatter(value);
      element.append(item);
    }
  };

  // loadStats 的两个列表已经由数据库分别倒序限制为最新 8 条。
  const loadStats = async () => {
    if (!currentUser || !currentBinding.bound) return;
    try {
      const utcOffsetMinutes = -new Date().getTimezoneOffset();
      const stats = await requestJSON(`/api/yanlili/clicks/stats?direction=${statsDirection}&utc_offset_minutes=${utcOffsetMinutes}`);
      if (statsTitle instanceof HTMLElement) statsTitle.textContent = statsDirection === "mine" ? "我想ta" : "ta想我";
      if (totalCount instanceof HTMLElement) totalCount.textContent = String(stats.total_count ?? 0);
      renderList(dailyCounts, stats.daily_counts ?? [], (item) => `${item.date}：${item.count} 次`);
      renderList(clickMinutes, stats.click_minutes ?? [], (item) => {
        const localTime = new Date(item.time).toLocaleString([], { dateStyle: "short", timeStyle: "short" });
        return `${localTime}：${item.count} 次`;
      });
      if (syncStatus instanceof HTMLElement) syncStatus.textContent = "已同步";
    } catch (error) {
      if (error.status === 409) await loadCompanionState().catch(() => {});
      if (!error.sessionEnded && syncStatus instanceof HTMLElement) {
        syncStatus.textContent = error.safeToRetry ? error.message : "统计读取失败";
      }
    }
  };

  const scheduleFlush = (delay = 750) => {
    window.clearTimeout(flushTimer);
    flushTimer = window.setTimeout(() => void flushClicks(), delay);
  };

  const flushClicks = async (keepalive = false) => {
    if (!currentUser || !currentBinding.bound || requestInFlight || pendingClicks <= 0) return;
    const batch = Math.min(pendingClicks, 100);
    pendingClicks -= batch;
    requestInFlight = true;
    if (syncStatus instanceof HTMLElement) syncStatus.textContent = `正在同步 ${batch} 次…`;
    try {
      await requestJSON("/api/yanlili/clicks", { method: "POST", body: JSON.stringify({ count: batch }), keepalive });
      if (syncStatus instanceof HTMLElement) syncStatus.textContent = pendingClicks > 0 ? `待同步 ${pendingClicks} 次` : "已同步";
      void loadStats();
    } catch (error) {
      if (error.sessionEnded) return;
      if (error.status === 409) {
        pendingClicks = 0;
        await loadCompanionState().catch(() => {});
      } else if (error.safeToRetry && currentUser && currentBinding.bound) {
        pendingClicks += batch;
        if (syncStatus instanceof HTMLElement) syncStatus.textContent = `登录校验暂未恢复，保留 ${pendingClicks} 次待重试`;
        window.clearTimeout(retryTimer);
        retryTimer = window.setTimeout(() => { retryTimer = 0; void flushClicks(); }, 2000);
      } else if (syncStatus instanceof HTMLElement) {
        // 写请求发出后连接中断时可能已经写入，自动重发会重复计数。
        syncStatus.textContent = "这批点击的同步结果未知，请查看记录";
      }
    } finally {
      requestInFlight = false;
      if (currentUser && currentBinding.bound && pendingClicks > 0 && !retryTimer) scheduleFlush(0);
    }
  };

  const getClickAudioGraph = () => {
    if (clickAudioGraph) return clickAudioGraph;
    const AudioContextClass = window.AudioContext || window.webkitAudioContext;
    if (typeof AudioContextClass !== "function") return null;
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
    } catch { return null; }
  };

  const playClickSound = () => {
    const graph = getClickAudioGraph();
    if (!graph) return;
    const { context, limiter } = graph;
    if (context.state === "suspended") void context.resume().catch(() => {});
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
    oscillator.addEventListener("ended", () => { oscillator.disconnect(); envelope.disconnect(); }, { once: true });
    oscillator.start(startedAt);
    oscillator.stop(endedAt);
  };

  // getFeedbackText 只有双向绑定时使用用户名或自己的备注，否则固定使用“你”。
  const getFeedbackText = () => {
    if (!currentBinding.bound) return "按按钮，想你+1";
    return `按按钮，想${currentBinding.partner_note || currentBinding.partner_username}+1`;
  };

  const createFloatingMessage = () => {
    sessionClickCount += 1;
    const messageText = getFeedbackText();
    const message = document.createElement("span");
    message.className = "floating-message";
    message.textContent = messageText;
    message.style.left = `${50 + Math.round((Math.random() - 0.5) * 34)}%`;
    message.style.setProperty("--tilt", `${((Math.random() - 0.5) * 5).toFixed(2)}deg`);
    message.addEventListener("animationend", () => message.remove(), { once: true });
    floatLayer.append(message);
    if (liveMessage instanceof HTMLElement) liveMessage.textContent = `${messageText}，本次访问共 ${sessionClickCount} 次`;
    if (currentUser && currentBinding.bound) { pendingClicks += 1; scheduleFlush(); }
    playClickSound();
  };

  const loadInbox = async () => {
    if (!currentUser || !(inboxList instanceof HTMLElement)) return;
    const data = await requestJSON("/api/companion/inbox");
    inboxList.replaceChildren();
    if (!data.items?.length) { const empty = document.createElement("p"); empty.textContent = "还没有绑定信件"; inboxList.append(empty); return; }
    const statusText = { pending: "等待接受", active: "已绑定", ended: "已解绑", superseded: "已失效" };
    for (const letter of data.items) {
      const mineIsInviter = letter.inviter_id === currentUser.id;
      const partnerName = mineIsInviter ? letter.invitee_username : letter.inviter_username;
      const card = document.createElement("article");
      card.className = "inbox-card";
      const heading = document.createElement("h3");
      heading.textContent = `与 ${letter.my_note || partnerName} 的绑定信件`;
      const meta = document.createElement("p");
      meta.textContent = `${statusText[letter.status] || letter.status} · ${new Date(letter.created_at).toLocaleString()}`;
      card.append(heading, meta);

      // 解绑子状态保留最近一次申请及处理结果，双方看到同一事实但使用各自视角的提示。
      const requesterName = letter.unbind_requested_by === currentUser.id ? "你" : partnerName;
      const responderName = letter.unbind_responded_by === currentUser.id ? "你" : partnerName;
      const unbindFeedback = {
        pending: `${requesterName}已发起解绑申请，等待处理`,
        cancelled: `${requesterName}发起的解绑申请已由${responderName}取消`,
        rejected: `${requesterName}发起的解绑申请已由${responderName}拒绝`,
        accepted: `${requesterName}发起的解绑申请已由${responderName}接受，双方已解绑`,
        direct: "因一方连续30天未登录，已直接双向解绑",
      }[letter.unbind_status];
      if (unbindFeedback) {
        const feedback = document.createElement("p");
        feedback.className = "unbind-feedback";
        feedback.textContent = unbindFeedback;
        card.append(feedback);
      }
      const actions = document.createElement("div");
      actions.className = "letter-actions";
      const addAction = (action, label) => {
        const actionButton = document.createElement("button");
        actionButton.type = "button";
        actionButton.dataset.action = action;
        actionButton.dataset.id = String(letter.id);
        actionButton.textContent = label;
        actions.append(actionButton);
      };
      if (letter.status === "pending" && !mineIsInviter) addAction("accept-binding", "接受绑定");
      if (letter.status === "active" && letter.unbind_status !== "pending") addAction("request-unbind", "解绑");
      if (letter.status === "active" && letter.unbind_status === "pending" && letter.unbind_requested_by !== currentUser.id) {
        addAction("accept-unbind", "接受解绑");
        addAction("reject-unbind", "拒绝解绑");
      }
      if (letter.status === "active" && letter.unbind_status === "pending" && letter.unbind_requested_by === currentUser.id) {
        addAction("cancel-unbind", "取消解绑申请");
      }
      if (actions.childNodes.length) card.append(actions);
      inboxList.append(card);
    }
  };

  const refreshCompanionViews = async () => {
    await loadCompanionState();
    await Promise.all([loadInbox(), loadPartners()]);
  };

  const loadPartners = async () => {
    if (!currentUser || !(partnerSelect instanceof HTMLSelectElement)) return;
    const data = await requestJSON("/api/companion/partners");
    const previous = partnerSelect.value;
    partnerSelect.replaceChildren();
    for (const partner of data.items ?? []) {
      const option = document.createElement("option");
      option.value = String(partner.id);
      option.textContent = partner.username;
      partnerSelect.append(option);
    }
    if (previous && [...partnerSelect.options].some((option) => option.value === previous)) partnerSelect.value = previous;
    if (detailsButton instanceof HTMLButtonElement) detailsButton.hidden = !data.items?.length;
  };

  // loadDetails 加载历史分页，临时鉴权故障时保留当前页并给出反馈。
  const loadDetails = async () => {
    if (!(partnerSelect instanceof HTMLSelectElement) || !partnerSelect.value) return;
    try {
      const data = await requestJSON(`/api/yanlili/clicks/details?partner_id=${encodeURIComponent(partnerSelect.value)}&page=${detailPage}`);
      detailTotalPages = data.total_pages ?? 0;
      renderList(detailsList, data.items ?? [], (item) => {
        const direction = item.direction === "mine" ? "我想ta" : "ta想我";
        return `${new Date(item.time).toLocaleString()} · ${direction} · ${item.count} 次`;
      });
      if (pageStatus instanceof HTMLElement) pageStatus.textContent = `第 ${data.page} / ${Math.max(data.total_pages, 1)} 页`;
      if (previousPage instanceof HTMLButtonElement) previousPage.disabled = data.page <= 1;
      if (nextPage instanceof HTMLButtonElement) nextPage.disabled = data.total_pages === 0 || data.page >= data.total_pages;
    } catch (error) {
      if (!error.sessionEnded && pageStatus instanceof HTMLElement) pageStatus.textContent = error.message;
    }
  };

  // confirmUnbindRequest 在发送任何解绑请求前连续取得三次明确确认。
  const confirmUnbindRequest = () => {
    const prompts = [
      "确认要发起解绑吗？",
      "解绑后将停止记录当前陪伴关系，是否继续？",
      "请最后确认：确定要解绑吗？",
    ];
    return prompts.every((message) => window.confirm(message));
  };

  // setQrControlsDisabled 防止图片解码或登录提交期间重复触发扫码流程。
  const setQrControlsDisabled = (disabled) => {
    if (qrScanButton instanceof HTMLButtonElement) qrScanButton.disabled = disabled;
    if (qrPickButton instanceof HTMLButtonElement) qrPickButton.disabled = disabled;
  };

  // stopQrScan 停止摄像头、隐藏预览并使所有迟到的异步回调失效。
  const stopQrScan = () => {
    qrScanGeneration += 1;
    if (qrScanFrame) {
      if (qrScanFrameType === "video" && qrVideo instanceof HTMLVideoElement && typeof qrVideo.cancelVideoFrameCallback === "function") {
        qrVideo.cancelVideoFrameCallback(qrScanFrame);
      } else {
        cancelAnimationFrame(qrScanFrame);
      }
      qrScanFrame = 0;
      qrScanFrameType = "";
    }
    if (qrStream) {
      for (const track of qrStream.getTracks()) track.stop();
      qrStream = null;
    }
    if (qrVideo instanceof HTMLVideoElement) {
      qrVideo.srcObject = null;
      qrVideo.hidden = true;
    }
  };

  // decodeQrCanvas 读取当前画布像素；同时尝试普通和反色二维码。
  const decodeQrCanvas = () => {
    if (typeof window.jsQR !== "function" || !(qrCanvas instanceof HTMLCanvasElement)) return null;
    const context = qrCanvas.getContext("2d", { willReadFrequently: true });
    if (!context) return null;
    const data = context.getImageData(0, 0, qrCanvas.width, qrCanvas.height);
    const result = window.jsQR(data.data, data.width, data.height, { inversionAttempts: "attemptBoth" });
    return result ? result.data : null;
  };

  // drawQrCandidate 将源图的指定区域等比缩放到受控画布，避免超大照片耗尽移动端内存。
  const drawQrCandidate = (source, sourceWidth, sourceHeight, crop, maxEdge) => {
    if (!(qrCanvas instanceof HTMLCanvasElement)) return null;
    const scale = Math.min(1, maxEdge / Math.max(crop.width, crop.height));
    qrCanvas.width = Math.max(1, Math.round(crop.width * scale));
    qrCanvas.height = Math.max(1, Math.round(crop.height * scale));
    const context = qrCanvas.getContext("2d", { willReadFrequently: true });
    if (!context) return null;
    context.drawImage(source, crop.x, crop.y, crop.width, crop.height, 0, 0, qrCanvas.width, qrCanvas.height);
    return decodeQrCanvas();
  };

  // decodeQrSource 先识别中心取景区域，再识别完整画面，兼顾远距离高密度码和非居中相册照片。
  const decodeQrSource = (source, sourceWidth, sourceHeight, maxEdge) => {
    if (sourceWidth <= 0 || sourceHeight <= 0) return null;
    const shortEdge = Math.min(sourceWidth, sourceHeight);
    const centerSide = Math.max(1, Math.round(shortEdge * 0.72));
    const candidates = [
      {
        x: Math.round((sourceWidth - centerSide) / 2),
        y: Math.round((sourceHeight - centerSide) / 2),
        width: centerSide,
        height: centerSide,
      },
      { x: 0, y: 0, width: sourceWidth, height: sourceHeight },
    ];
    for (const crop of candidates) {
      const text = drawQrCandidate(source, sourceWidth, sourceHeight, crop, maxEdge);
      if (text) return text;
    }
    return null;
  };

  // scheduleQrScan 优先跟随真实视频帧调度，不支持时回退到动画帧。
  const scheduleQrScan = (generation) => {
    if (generation !== qrScanGeneration || !qrStream || !(qrVideo instanceof HTMLVideoElement)) return;
    if (typeof qrVideo.requestVideoFrameCallback === "function") {
      qrScanFrameType = "video";
      qrScanFrame = qrVideo.requestVideoFrameCallback(() => scanQrUntilFound(generation));
    } else {
      qrScanFrameType = "animation";
      qrScanFrame = requestAnimationFrame(() => scanQrUntilFound(generation));
    }
  };

  // scanQrUntilFound 持续读取摄像头帧；未就绪或单帧异常都不能终止后续扫描。
  const scanQrUntilFound = (generation) => {
    qrScanFrame = 0;
    qrScanFrameType = "";
    if (generation !== qrScanGeneration || !qrStream || !(qrVideo instanceof HTMLVideoElement)) return;
    const now = performance.now();
    if (now - qrLastScanAt < qrScanIntervalMs) {
      scheduleQrScan(generation);
      return;
    }
    qrLastScanAt = now;
    let text = null;
    try {
      if (qrVideo.readyState >= HTMLMediaElement.HAVE_CURRENT_DATA && qrVideo.videoWidth > 0 && qrVideo.videoHeight > 0) {
        text = decodeQrSource(qrVideo, qrVideo.videoWidth, qrVideo.videoHeight, qrCameraMaxEdge);
      }
    } catch {
      // 某一帧可能在摄像头切换或页面后台化时失效，下一帧继续尝试。
    }
    if (text) {
      stopQrScan();
      void completeQrLogin(text);
      return;
    }
    scheduleQrScan(generation);
  };

  // waitForQrVideo 等待视频真正产生可读尺寸，避免授权完成但首帧尚未到达时误停扫描。
  const waitForQrVideo = (video, generation) => new Promise((resolve, reject) => {
    if (video.readyState >= HTMLMediaElement.HAVE_CURRENT_DATA && video.videoWidth > 0 && video.videoHeight > 0) {
      resolve();
      return;
    }
    const timeout = window.setTimeout(() => finish(new Error("摄像头画面等待超时")), 10000);
    const finish = (error) => {
      window.clearTimeout(timeout);
      video.removeEventListener("loadeddata", onLoaded);
      video.removeEventListener("error", onError);
      if (generation !== qrScanGeneration) {
        reject(new Error("扫码已取消"));
      } else if (error) {
        reject(error);
      } else {
        resolve();
      }
    };
    const onLoaded = () => finish();
    const onError = () => finish(new Error("无法读取摄像头画面"));
    video.addEventListener("loadeddata", onLoaded, { once: true });
    video.addEventListener("error", onError, { once: true });
  });

  // startQrCameraScan 打开后置摄像头并把帧交给识别循环。
  const startQrCameraScan = async () => {
    if (typeof navigator.mediaDevices?.getUserMedia !== "function") {
      if (loginMessage instanceof HTMLElement) loginMessage.textContent = "当前环境不支持调用摄像头，请使用“从相册选择”";
      return;
    }
    const generation = qrScanGeneration;
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        video: {
          facingMode: { ideal: "environment" },
          width: { ideal: 1920 },
          height: { ideal: 1080 },
        },
        audio: false,
      });
      if (generation !== qrScanGeneration) {
        for (const track of stream.getTracks()) track.stop();
        return;
      }
      if (!(qrVideo instanceof HTMLVideoElement)) throw new Error("扫码预览不可用");
      qrStream = stream;
      qrVideo.srcObject = stream;
      qrVideo.hidden = false;
      await qrVideo.play();
      await waitForQrVideo(qrVideo, generation);
      qrLastScanAt = 0;
      if (loginMessage instanceof HTMLElement) loginMessage.textContent = "请将二维码放在画面中央并保持稳定";
      scheduleQrScan(generation);
    } catch (error) {
      if (generation !== qrScanGeneration) return;
      stopQrScan();
      if (loginMessage instanceof HTMLElement) loginMessage.textContent = "无法打开摄像头，请使用“从相册选择”或账号密码登录";
    }
  };

  // loadQrImageElement 为不支持 createImageBitmap 的浏览器提供兼容路径，并及时释放 blob URL。
  const loadQrImageElement = (file) => new Promise((resolve, reject) => {
    const image = new Image();
    const objectURL = URL.createObjectURL(file);
    image.addEventListener("load", () => resolve({ source: image, close: () => URL.revokeObjectURL(objectURL) }), { once: true });
    image.addEventListener("error", () => {
      URL.revokeObjectURL(objectURL);
      reject(new Error("浏览器无法读取这张图片"));
    }, { once: true });
    image.src = objectURL;
  });

  // loadQrImage 优先直接解码文件，自动应用照片方向且不依赖 CSP 中的 blob 图片加载。
  const loadQrImage = async (file) => {
    if (typeof window.createImageBitmap === "function") {
      try {
        const bitmap = await window.createImageBitmap(file, { imageOrientation: "from-image" });
        return { source: bitmap, close: () => bitmap.close() };
      } catch {
        try {
          const bitmap = await window.createImageBitmap(file);
          return { source: bitmap, close: () => bitmap.close() };
        } catch {
          // 继续使用兼容路径，以支持部分旧版 Safari 和特殊图片格式。
        }
      }
    }
    return loadQrImageElement(file);
  };

  // decodeQrFile 校验相册文件并通过受控画布完成多区域识别。
  const decodeQrFile = async (file) => {
    if (file.type && !file.type.startsWith("image/")) throw new Error("请选择图片文件");
    if (file.size > 30 * 1024 * 1024) throw new Error("图片不能超过 30MB");
    const loaded = await loadQrImage(file);
    try {
      const width = loaded.source.naturalWidth || loaded.source.width;
      const height = loaded.source.naturalHeight || loaded.source.height;
      if (!width || !height) throw new Error("图片尺寸无效");
      return decodeQrSource(loaded.source, width, height, qrImageMaxEdge);
    } finally {
      loaded.close();
    }
  };

  // completeQrLogin 把识别到的文本提交给后端，成功后走与密码登录相同的会话初始化。
  const completeQrLogin = async (text) => {
    if (qrLoginInFlight) return;
    qrLoginInFlight = true;
    setQrControlsDisabled(true);
    if (loginMessage instanceof HTMLElement) loginMessage.textContent = "正在扫码登录…";
    try {
      await requestJSON("/api/auth/qr-login", { method: "POST", body: JSON.stringify({ text }) });
      const user = await requestJSON("/api/auth/me");
      setAuthenticated(user);
      setLoginPanel(false);
      if (loginMessage instanceof HTMLElement) loginMessage.textContent = "";
      await Promise.all([loadCompanionState(), loadPartners()]);
    } catch (error) {
      if (loginMessage instanceof HTMLElement) loginMessage.textContent = error.message;
    } finally {
      qrLoginInFlight = false;
      setQrControlsDisabled(false);
    }
  };

  // requestUnbind 只调用统一后端入口；30 天未登录时由服务端要求第四次确认。
  const requestUnbind = async (bindingID) => {
    const endpoint = `/api/companion/bindings/${bindingID}/unbind-request`;
    let result = await requestJSON(endpoint, {
      method: "POST",
      body: JSON.stringify({ confirm_inactive: false }),
    });
    if (result.action !== "inactive_confirmation_required") return result;
    if (!window.confirm(`${result.message}，是否确认？`)) {
      return { action: "cancelled", message: "已取消直接解绑" };
    }
    await flushClicks();
    result = await requestJSON(endpoint, {
      method: "POST",
      body: JSON.stringify({ confirm_inactive: true }),
    });
    return result;
  };

  button.addEventListener("click", createFloatingMessage);
  button.addEventListener("pointerdown", () => button.classList.add("is-pressed"));
  for (const eventName of ["pointerup", "pointercancel", "pointerleave"]) button.addEventListener(eventName, () => button.classList.remove("is-pressed"));

  loginButton?.addEventListener("click", () => setLoginPanel(true));
  closeLoginButton?.addEventListener("click", () => { stopQrScan(); setLoginPanel(false); });
  closeCompanionButton?.addEventListener("click", () => setPanel(companionPanel, false));
  closeInboxButton?.addEventListener("click", () => setPanel(inboxPanel, false));
  closeNoteButton?.addEventListener("click", () => setPanel(notePanel, false));
  closeDetailsButton?.addEventListener("click", () => setPanel(detailsPanel, false));

  companionButton?.addEventListener("click", () => {
    if (!currentUser) {
      window.alert("该功能需登录后使用");
      setLoginPanel(true);
      return;
    }
    setPanel(companionPanel, true, "#companion-username");
  });

  inboxButton?.addEventListener("click", () => {
    setPanel(inboxPanel, true);
    if (inboxMessage instanceof HTMLElement) inboxMessage.textContent = "";
    void loadInbox().catch((error) => {
      if (!error.sessionEnded && inboxMessage instanceof HTMLElement) inboxMessage.textContent = error.message;
    });
  });
  noteButton?.addEventListener("click", () => {
    if (partnerNote instanceof HTMLInputElement) partnerNote.value = currentBinding.partner_note || "";
    setPanel(notePanel, true, "#partner-note");
  });
  detailsButton?.addEventListener("click", () => { detailPage = 1; setPanel(detailsPanel, true); void loadDetails(); });

  directionSwitch?.addEventListener("click", () => {
    statsDirection = statsDirection === "mine" ? "theirs" : "mine";
    void loadStats();
  });

  loginForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    const data = new FormData(loginForm);
    if (loginMessage instanceof HTMLElement) loginMessage.textContent = "正在登录…";
    try {
      await requestJSON("/api/auth/login", { method: "POST", body: JSON.stringify({ username: data.get("username"), password: data.get("password") }) });
      loginForm.reset();
      const user = await requestJSON("/api/auth/me");
      setAuthenticated(user);
      setLoginPanel(false);
      if (loginMessage instanceof HTMLElement) loginMessage.textContent = "";
      await Promise.all([loadCompanionState(), loadPartners()]);
    } catch (error) { if (loginMessage instanceof HTMLElement) loginMessage.textContent = error.message; }
  });

  qrScanButton?.addEventListener("click", () => {
    stopQrScan();
    if (loginMessage instanceof HTMLElement) loginMessage.textContent = "正在打开摄像头…";
    void startQrCameraScan();
  });
  qrPickButton?.addEventListener("click", () => qrFileInput?.click());
  qrFileInput?.addEventListener("change", async () => {
    const file = qrFileInput.files?.[0];
    if (!file || !(qrCanvas instanceof HTMLCanvasElement)) return;
    stopQrScan();
    setQrControlsDisabled(true);
    if (loginMessage instanceof HTMLElement) loginMessage.textContent = "正在识别相册图片…";
    try {
      const text = await decodeQrFile(file);
      if (text) {
        await completeQrLogin(text);
      } else if (loginMessage instanceof HTMLElement) {
        loginMessage.textContent = "图片中未识别到有效二维码，请尝试更清晰或更近的照片";
      }
    } catch (error) {
      if (loginMessage instanceof HTMLElement) loginMessage.textContent = error.message || "图片识别失败";
    } finally {
      qrFileInput.value = "";
      if (!qrLoginInFlight) setQrControlsDisabled(false);
    }
  });

  companionForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    const data = new FormData(companionForm);
    if (companionMessage instanceof HTMLElement) companionMessage.textContent = "正在发送…";
    try {
      await requestJSON("/api/companion/invitations", { method: "POST", body: JSON.stringify({ username: data.get("username"), note: data.get("note") }) });
      companionForm.reset();
      if (companionMessage instanceof HTMLElement) companionMessage.textContent = "绑定邀请已发送，可在收件箱查看";
      await loadInbox();
    } catch (error) { if (companionMessage instanceof HTMLElement) companionMessage.textContent = error.message; }
  });

  noteForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (!currentBinding.bound) return;
    const data = new FormData(noteForm);
    try {
      await requestJSON(`/api/companion/bindings/${currentBinding.binding_id}/note`, { method: "PATCH", body: JSON.stringify({ note: data.get("note") }) });
      if (noteMessage instanceof HTMLElement) noteMessage.textContent = "备注已修改";
      await refreshCompanionViews();
    } catch (error) { if (noteMessage instanceof HTMLElement) noteMessage.textContent = error.message; }
  });

  inboxList?.addEventListener("click", async (event) => {
    if (!(event.target instanceof Element)) return;
    const actionButton = event.target.closest("button[data-action]");
    if (!(actionButton instanceof HTMLButtonElement)) return;
    const endpoints = {
      "accept-binding": "accept",
      "accept-unbind": "unbind-accept",
      "cancel-unbind": "unbind-cancel",
      "reject-unbind": "unbind-reject",
    };
    const action = actionButton.dataset.action;
    if (action === "request-unbind" && !confirmUnbindRequest()) return;
    const endpoint = endpoints[action];
    if (action !== "request-unbind" && !endpoint) return;
    actionButton.disabled = true;
    if (inboxMessage instanceof HTMLElement) inboxMessage.textContent = "正在处理…";
    try {
      if (endpoint === "unbind-accept") await flushClicks();
      const result = action === "request-unbind"
        ? await requestUnbind(actionButton.dataset.id)
        : await requestJSON(`/api/companion/bindings/${actionButton.dataset.id}/${endpoint}`, { method: "POST" });
      if (inboxMessage instanceof HTMLElement) inboxMessage.textContent = result.message;
      if (result.action !== "cancelled") await refreshCompanionViews();
      else actionButton.disabled = false;
    } catch (error) {
      if (inboxMessage instanceof HTMLElement) inboxMessage.textContent = error.message;
      actionButton.disabled = false;
    }
  });

  partnerSelect?.addEventListener("change", () => { detailPage = 1; void loadDetails(); });
  previousPage?.addEventListener("click", () => { if (detailPage > 1) { detailPage -= 1; void loadDetails(); } });
  nextPage?.addEventListener("click", () => { if (detailPage < detailTotalPages) { detailPage += 1; void loadDetails(); } });

  logoutButton?.addEventListener("click", async () => {
    await flushClicks();
    try {
      await requestJSON("/api/auth/logout", { method: "POST" });
      setAuthenticated(null);
    } catch {
      if (authStatus instanceof HTMLElement) authStatus.textContent = "暂时无法退出登录，请稍后重试";
    }
  });

  document.addEventListener("visibilitychange", () => { if (document.visibilityState === "hidden") void flushClicks(true); });
  window.addEventListener("pagehide", () => void flushClicks(true));

  void requestJSON("/api/auth/me")
    .then(async (user) => { setAuthenticated(user); await Promise.all([loadCompanionState(), loadPartners()]); })
    .catch((error) => {
      if (!error.sessionEnded && authStatus instanceof HTMLElement) {
        authStatus.textContent = currentUser ? "暂时无法加载登录数据，请稍后刷新页面" : "暂时无法检查登录状态，请稍后刷新页面";
      }
    });
})();
