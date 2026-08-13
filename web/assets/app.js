// 页面交互：每次点击按钮都创建一条向上漂浮并渐隐的提示文字。
(() => {
  "use strict";

  const button = document.querySelector("#miss-button");
  const floatLayer = document.querySelector("#float-layer");
  const liveMessage = document.querySelector("#live-message");
  const messageText = button instanceof HTMLButtonElement ? button.dataset.message : "";
  let clickCount = 0;

  if (!(button instanceof HTMLButtonElement) || !(floatLayer instanceof HTMLElement) || !messageText) {
    return;
  }

  // createFloatingMessage 创建独立动画元素，使连续点击也能逐条显示。
  const createFloatingMessage = () => {
    clickCount += 1;
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
      liveMessage.textContent = `${messageText}，共 ${clickCount} 次`;
    }
  };

  button.addEventListener("click", createFloatingMessage);
  button.addEventListener("pointerdown", () => button.classList.add("is-pressed"));
  for (const eventName of ["pointerup", "pointercancel", "pointerleave"]) {
    button.addEventListener(eventName, () => button.classList.remove("is-pressed"));
  }
})();
