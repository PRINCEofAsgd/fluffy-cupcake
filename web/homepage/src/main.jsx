// 本文件负责 Homepage 的内容数据、资源加载门、滚动动效与全部 React 页面区块。
import React, { useEffect, useMemo, useRef, useState } from 'react';
import * as ReactDOMClient from 'react-dom/client';
import Lenis from 'lenis';
import { FaEnvelope, FaGithub, FaWeixin } from 'react-icons/fa6';
import { SiNeteasecloudmusic, SiTiktok } from 'react-icons/si';
import 'lenis/dist/lenis.css';
import './styles.css';
import { getScrollTuning } from './scrollTuning.js';

// Homepage 的公开资源使用独立路径，避免与 /yanlili 的资源命名冲突。
const homepageBase = '/homepage';
const placeholderImage = `${homepageBase}/images/profile-placeholder.svg`;
const avatarImage = `${homepageBase}/images/loki-yin-avatar.png`;
const screenBackgroundImage = `${homepageBase}/images/matcha-cake-screen-wide.jpg`;
const vionIcon = `${homepageBase}/images/experience/vion-tech.svg`;
const vionWordmark = `${homepageBase}/images/experience/vionvision-wordmark.svg`;
const sealosLogo = `${homepageBase}/images/sealos.svg`;
const mockMemberIcons = [
  { src: `${homepageBase}/images/github-mark.svg`, alt: 'GitHub' },
  { src: `${homepageBase}/images/sealos.svg`, alt: 'Sealos' },
  { src: `${homepageBase}/images/vion-tech.png`, alt: 'Vion Tech' },
];

// 技术栈 Logo 使用统一紧凑托盘，并保留初版提案的轻微倾角和不规则环绕布局。
const techStackIcons = [
  { name: 'Go', file: 'go.svg', x: 8, y: 23, size: 50, tilt: -2.4, duration: 3.8, delay: -1.2 },
  { name: 'Prometheus', file: 'prometheus.svg', x: 25, y: 10, size: 50, tilt: 1.8, duration: 4, delay: -2.9 },
  { name: 'Loki', file: 'loki.svg', x: 48, y: 5, size: 50, tilt: -1.2, duration: 5, delay: -4.4 },
  { name: 'Docker', file: 'docker.svg', x: 84, y: 8, size: 50, tilt: 2.6, duration: 5.1, delay: -4.1 },
  { name: 'C++', file: 'cpp.svg', x: 95, y: 25, size: 50, tilt: -2, duration: 4.4, delay: -2.6 },
  { name: 'MySQL', file: 'mysql.svg', x: 34, y: 30, size: 50, tilt: 2.2, duration: 4.9, delay: -3.3 },
  { name: 'Redis', file: 'redis.svg', x: 91, y: 31, size: 50, tilt: -1.6, duration: 3.5, delay: -0.7 },
  { name: 'Kafka', file: 'kafka.svg', x: 7, y: 43, size: 50, tilt: -2.8, duration: 4.2, delay: -2 },
  { name: 'Kubernetes', file: 'kubernetes.svg', x: 68, y: 20, size: 50, tilt: 2.4, duration: 3.6, delay: -1.8 },
  { name: 'Nginx', file: 'nginx.svg', x: 78, y: 35, size: 50, tilt: 1.2, duration: 4.7, delay: -3.6 },
].map((icon) => ({
  ...icon,
  src: `${homepageBase}/images/tech-stack/${icon.file}`,
}));

const projects = [
  {
    slug: 'fluffy-cupcake',
    title: 'Fluffy Cupcake',
    caption: '为伴侣互动与纪念设计的轻量 Web 服务，提供登录、陪伴绑定、双向想念记录与个人 Homepage。',
    href: '/yanlili',
    stack: ['Go', 'Gin', 'MySQL', 'React'],
    highlights: ['100+ 日活', '双向陪伴关系', '分钟桶记录'],
    preview: ['互动输入', '服务端聚合', '关系记录'],
  },
  {
    slug: 'reactor-web',
    title: 'Reactor Web',
    caption: '面向 Linux 的轻量高性能网络服务框架，以 Reactor + Epoll ET + 非阻塞 IO 组织多 Reactor、线程池与连接生命周期。',
    href: 'https://github.com/PRINCEofAsgd/-Reactor-Web-',
    external: true,
    stack: ['C++11', 'Reactor', 'Epoll ET', 'HTTP / RPC'],
    highlights: ['150k 并发请求', '20k 业务请求', '0.03% 错误率'],
    preview: ['事件接入', 'Reactor 调度', '业务处理'],
  },
  {
    slug: 'my-dock',
    title: 'My Dock',
    caption: '团队使用的 macOS / iOS 双端效率工具，可在局域网发现 Mac，并通过四位验证码从移动端便捷打开桌面应用。',
    href: 'https://github.com/PRINCEofAsgd/-My-Dock-',
    external: true,
    stack: ['Swift', 'macOS', 'iOS', '局域网发现'],
    highlights: ['双端协同', '验证码配对', '应用快捷启动'],
    preview: ['设备发现', '身份配对', '远程启动'],
  },
  {
    slug: 'card-game',
    title: 'Card Game',
    caption: '基于 Cocos2d-x 3.17 开发的跨平台纸牌游戏，覆盖桌面与移动端构建及核心卡牌交互。',
    href: 'https://github.com/PRINCEofAsgd/-Card-Game-',
    external: true,
    stack: ['C++', 'Cocos2d-x 3.17', '跨平台'],
    highlights: ['卡牌交互', '游戏状态管理', '多端构建'],
    preview: ['规则输入', '回合状态', '跨端呈现'],
  },
];

const profileRows = [
  ['姓名', '殷子佑'],
  ['技术 ID', 'LokiYin'],
  ['学校', '中国矿业大学(北京) 计算机科学与技术 · 本科大四在读'],
  ['当前工作地', '北京 · 中国'],
  ['职业方向', '后端研发 · AI Infra · 游戏服务端'],
  ['技术方向', 'Go · C++ · 分布式架构 · 云原生'],
];

const internshipCards = [
  {
    eyebrow: '实习经历',
    company: '文安智能',
    period: '2026.04 — 2026.08',
    role: 'Go 后端研发',
    text: '参与分布式 AIoT 设备云管理平台与 2.5D 重建参数服务开发。',
    focus: '负责服务拆分、设备云端纳管、异步任务链路、数据访问与可观测性建设。',
    pipeline: ['设备接入', '异步任务', '云端纳管'],
    deliverables: [
      {
        title: '服务架构与设备纳管',
        code: 'SERVICE',
        text: '基于 go-zero 拆分 Device、Sync、Task 与 Auth 服务，通过 gRPC、事务和锁机制保障跨服务一致性。',
      },
      {
        title: '异步任务可靠链路',
        code: 'ASYNC TASK',
        text: '以 Outbox、Kafka 与 Worker 组织设备注册、状态同步和算法任务，覆盖重试、超时、取消、死信与状态追踪。',
      },
      {
        title: '数据与可观测性',
        code: 'OBSERVABILITY',
        text: '完善缓存、SQL 索引与读写链路，并通过 Kubernetes、Prometheus、Loki 和 Grafana 建立故障定位能力。',
      },
    ],
    brandLogo: vionWordmark,
    brandIcon: vionIcon,
    copyBrand: true,
    variant: 'minimax',
  },
  {
    eyebrow: '开源经历',
    company: '环界云计算',
    period: '2026.07 — 至今',
    title: 'Sealos 开源项目 Contributor',
    text: '参与基于 Kubernetes 的云原生操作系统 Sealos，负责问题发现与复现、bug 定位与修复。',
    focus: '完善应用删除时的关联资源清理与 Service Bind 新旧模型兼容链路。',
    pipeline: ['问题复现', '资源清理', '模型兼容'],
    deliverables: [
      {
        title: '问题复现与根因定位',
        code: 'DIAGNOSIS',
        text: '在真实环境复现 Network Service 残留与持续计费问题，沿调用链定位子资源引用缺失。',
      },
      {
        title: '资源清理链路',
        code: 'LIFECYCLE',
        text: '补全资源引用与清理绑定关系，验证关联网络资源能够随应用生命周期正确回收。',
      },
      {
        title: 'Service Bind 模型兼容',
        code: 'COMPAT',
        text: '兼容新旧资源模型下的 Service Bind 关系，持续跟进主线资源模型演进。',
      },
    ],
    brandLogo: sealosLogo,
    variant: 'sealos',
  },
];

const hobbyCards = [
  {
    eyebrow: '爱好 01',
    title: '魔方',
    text: '专注盲拧项目，四阶盲拧 WCA 全国前 30。',
    visualTitle: '4×4 BLIND',
    visualText: 'WCA 全国前 30',
    note: '持续练习记忆编码、复盘与稳定执行。',
    image: placeholderImage,
  },
  {
    eyebrow: '爱好 02',
    title: '音乐',
    text: '2024.08—2025.10 从事歌手与主播工作，负责虚拟、音乐、游戏类直播；曾作为月音浪 60w+ 主播，稳定主持 2k—1w+ 同时在线直播间。',
    visualTitle: 'MUSIC',
    visualText: '歌手 · 主播 · 直播运营',
    note: '查看网易云音乐主页',
    noteHref: 'https://music.163.com/#/artist?id=95001856',
    image: placeholderImage,
  },
  {
    eyebrow: '爱好 03',
    title: '体育',
    text: '400m 短跑国家二级运动员。',
    visualTitle: '400M',
    visualText: '国家二级运动员',
    note: '保持速度训练、体能与赛场节奏。',
    image: placeholderImage,
  },
  {
    eyebrow: '爱好 04',
    title: '游戏',
    text: '王者荣耀国 50 打野，Minecraft 小型生存服服主。',
    visualTitle: 'GAME',
    visualText: '竞技 · 生存服运营',
    note: '兼顾竞技决策、团队协作与社区维护。',
    image: placeholderImage,
  },
];

const socials = [
  {
    name: 'GitHub',
    href: 'https://github.com/PRINCEofAsgd',
    Icon: FaGithub,
    tone: 'github',
    external: true,
    orbit: { angle: 318, track: 'outer', duration: '42s', direction: 'normal', size: '64px', mobileSize: '46px' },
  },
  {
    name: '邮箱：lokiyin4819@foxmail.com',
    href: 'mailto:lokiyin4819@foxmail.com',
    Icon: FaEnvelope,
    tone: 'email',
    orbit: { angle: 42, track: 'outer', duration: '36s', direction: 'normal', size: '66px', mobileSize: '46px' },
  },
  {
    name: '微信：Yyg1061',
    href: '#contact-details',
    Icon: FaWeixin,
    tone: 'wechat',
    orbit: { angle: 142, track: 'middle', duration: '27s', direction: 'reverse', size: '58px', mobileSize: '42px' },
  },
  {
    name: '抖音',
    href: 'https://www.douyin.com/user/MS4wLjABAAAAqMBHTKsqMfKY5EldZgzCmvx6l3x-4sawM42QNPFn03LAB2oveG0wDDgPJNnDZNTa?from_tab_name=main',
    Icon: SiTiktok,
    tone: 'douyin',
    external: true,
    orbit: { angle: 248, track: 'inner', duration: '18s', direction: 'normal', size: '54px', mobileSize: '40px' },
  },
  {
    name: '网易云音乐',
    href: 'https://music.163.com/#/artist?id=95001856',
    Icon: SiNeteasecloudmusic,
    tone: 'netease',
    external: true,
    orbit: { angle: 196, track: 'outer', duration: '48s', direction: 'reverse', size: '58px', mobileSize: '42px' },
  },
];

const resumeAnchors = [
  { label: '信息', href: '#info' },
  { label: '工作经历', href: '#internship' },
  { label: '项目经历', href: '#projects' },
  { label: '爱好', href: '#hobbies' },
  { label: '社媒', href: '#socials' },
];

const entryAssetUrls = Array.from(new Set([
  avatarImage,
  screenBackgroundImage,
  vionIcon,
  vionWordmark,
  sealosLogo,
  ...mockMemberIcons.map((icon) => icon.src),
  ...techStackIcons.map((icon) => icon.src),
  `${homepageBase}/images/fora-hero-far.png`,
  `${homepageBase}/images/fora-hero-mid.png`,
  `${homepageBase}/images/fora-hero-foreground.png`,
]));

const deferredAssetUrls = Array.from(new Set([
  placeholderImage,
  ...internshipCards.flatMap((card) => [card.brandLogo, card.brandIcon].filter(Boolean)),
  ...hobbyCards.map((card) => card.image),
]));

// 所有页面区块使用的视觉资源都进入加载门，使进度表示整页而非仅首屏。
const pageAssetUrls = Array.from(new Set([
  ...entryAssetUrls,
  ...deferredAssetUrls,
]));

// clamp 把动态数值限制在视觉动效允许的范围内。
function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}

// delay 为资源加载完成后的短暂视觉交接提供异步等待。
function delay(duration) {
  return new Promise((resolve) => {
    window.setTimeout(resolve, duration);
  });
}

// loadImageAsset 加载并解码单张图片；失败也会结束任务，避免加载门永久阻塞。
function loadImageAsset(src) {
  return new Promise((resolve) => {
    const image = new Image();
    // done 等待浏览器解码结束后才把图片计入完成进度。
    const done = () => {
      if (image.decode) {
        image.decode().then(resolve, resolve);
        return;
      }

      resolve();
    };

    image.decoding = 'async';
    image.onload = done;
    image.onerror = resolve;
    image.src = src;

    if (image.complete) {
      done();
    }
  });
}

// preloadPageResources 并发加载整页图片与字体，并持续报告真实任务进度。
async function preloadPageResources(onProgress) {
  const assets = pageAssetUrls;
  let completed = 0;
  const totalTasks = assets.length + 1;
  // updateProgress 将每个已完成资源折算为 0—1 之间的进度。
  const updateProgress = () => {
    completed += 1;
    onProgress(Math.min(1, completed / totalTasks));
  };
  const imageLoads = assets.map((src) => loadImageAsset(src).finally(updateProgress));
  const fontReady = (document.fonts?.ready ?? Promise.resolve())
    .catch(() => undefined)
    .finally(updateProgress);

  await Promise.all([
    ...imageLoads,
    fontReady,
  ]);
  onProgress(1);
}

// useResourceGate 在全部页面资源就绪前锁定滚动并保持加载页面可见。
function useResourceGate() {
  const [state, setState] = useState({ progress: 0, ready: false });

  useEffect(() => {
    let cancelled = false;
    const startedAt = performance.now();
    document.getElementById('boot-loader')?.remove();

    const complete = async () => {
      await preloadPageResources((progress) => {
        if (!cancelled) {
          setState((current) => ({
            ...current,
            progress,
          }));
        }
      });

      // 仅在全部页面资源完成后保留一段短暂的视觉交接。
      const remainingDuration = Math.max(0, 280 - (performance.now() - startedAt));
      if (remainingDuration > 0) {
        await delay(remainingDuration);
      }

      if (!cancelled) {
        setState({ progress: 1, ready: true });
      }
    };

    complete();

    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    document.documentElement.classList.toggle('is-resource-loading', !state.ready);
    document.body.classList.toggle('is-resource-loading', !state.ready);

    return () => {
      document.documentElement.classList.remove('is-resource-loading');
      document.body.classList.remove('is-resource-loading');
    };
  }, [state.ready]);

  return state;
}

// useScrollEffects 统一管理平滑滚动、视差、弹簧惯性和区块显现状态。
function useScrollEffects() {
  useEffect(() => {
    const root = document.documentElement;
    const rootVars = new Map();
    const mobileQuery = window.matchMedia('(max-width: 768px), (pointer: coarse)');
    let scrollTuning = getScrollTuning(mobileQuery.matches);
    // handleResize 在视口尺寸变化时重新计算所有滚动目标。
    const handleResize = () => update();
    let previousScrollY = window.scrollY;
    let previousScrollTime = performance.now();
    let scrollVelocity = 0;
    const motion = {
      raf: 0,
      lastFrame: performance.now(),
      current: {
        backY: 0,
        frontY: 0,
        forestY: 0,
        forestScale: 1,
        heroCopyY: 0,
        heroDeviceY: 0,
        heroDeviceOpacity: 1,
        heroUIOpacity: 1,
        heroLightX: 0,
        heroLightY: 0,
      },
      target: {
        backY: 0,
        frontY: 0,
        forestY: 0,
        forestScale: 1,
        heroCopyY: 0,
        heroDeviceY: 0,
        heroDeviceOpacity: 1,
        heroUIOpacity: 1,
        heroLightX: 0,
        heroLightY: 0,
      },
      velocity: {
        backY: 0,
        frontY: 0,
        forestY: 0,
        forestScale: 0,
        heroCopyY: 0,
        heroDeviceY: 0,
        heroDeviceOpacity: 0,
        heroUIOpacity: 0,
        heroLightX: 0,
        heroLightY: 0,
      },
    };
    // setRootVar 仅在值变化时写入 CSS 变量，减少重复布局计算。
    const setRootVar = (name, value) => {
      if (rootVars.get(name) === value) {
        return;
      }

      rootVars.set(name, value);
      root.style.setProperty(name, value);
    };
    // setMotionTarget 更新单个视差属性的弹簧目标值。
    const setMotionTarget = (name, value) => {
      motion.target[name] = value;
    };
    // stepSpring 按刚度和阻尼推进一帧弹簧运动。
    const stepSpring = (name, stiffness, damping, deltaTime) => {
      const distance = motion.target[name] - motion.current[name];
      const acceleration =
        distance * stiffness * scrollTuning.spring.stiffnessScale -
        motion.velocity[name] * damping * scrollTuning.spring.dampingScale;
      motion.velocity[name] += acceleration * deltaTime;
      motion.current[name] += motion.velocity[name] * deltaTime;

      if (Math.abs(distance) < 0.001 && Math.abs(motion.velocity[name]) < 0.001) {
        motion.current[name] = motion.target[name];
        motion.velocity[name] = 0;
        return false;
      }

      return true;
    };
    // renderMotion 把当前弹簧状态同步到页面 CSS 变量。
    const renderMotion = () => {
      const now = performance.now();
      const deltaTime = Math.min(0.034, Math.max(0.001, (now - motion.lastFrame) / 1000));
      motion.lastFrame = now;

      const active = [
        stepSpring('backY', 36, 12, deltaTime),
        stepSpring('frontY', 44, 13, deltaTime),
        stepSpring('forestY', 52, 14, deltaTime),
        stepSpring('forestScale', 46, 13, deltaTime),
        stepSpring('heroCopyY', 42, 13, deltaTime),
        stepSpring('heroDeviceY', 38, 12, deltaTime),
        stepSpring('heroDeviceOpacity', 48, 14, deltaTime),
        stepSpring('heroUIOpacity', 48, 14, deltaTime),
        stepSpring('heroLightX', 34, 12, deltaTime),
        stepSpring('heroLightY', 34, 12, deltaTime),
      ].some(Boolean);

      setRootVar('--hill-back-y', `${Math.round(motion.current.backY)}px`);
      setRootVar('--hill-front-y', `${Math.round(motion.current.frontY)}px`);
      setRootVar('--forest-y', `${Math.round(motion.current.forestY)}px`);
      setRootVar('--forest-scale', motion.current.forestScale.toFixed(3));
      setRootVar('--hero-copy-y', `${Math.round(motion.current.heroCopyY)}px`);
      setRootVar('--hero-device-y', `${Math.round(motion.current.heroDeviceY)}px`);
      setRootVar('--hero-device-opacity', motion.current.heroDeviceOpacity.toFixed(3));
      setRootVar('--hero-ui-opacity', motion.current.heroUIOpacity.toFixed(3));
      setRootVar('--hero-light-x', `${Math.round(motion.current.heroLightX)}px`);
      setRootVar('--hero-light-y', `${Math.round(motion.current.heroLightY)}px`);

      if (active) {
        motion.raf = requestAnimationFrame(renderMotion);
        return;
      }

      motion.raf = 0;
    };
    // ensureMotion 确保动效循环只启动一次。
    const ensureMotion = () => {
      if (!motion.raf) {
        motion.lastFrame = performance.now();
        motion.raf = requestAnimationFrame(renderMotion);
      }
    };
    const revealTargets = [
      '.hero-device',
      '.split-heading',
      '.feature-stage',
      '.feature-caption',
      '.profile-grid div',
      '.social-strip a',
      '.final-copy',
    ].join(',');

    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            entry.target.classList.add('is-visible');
          }
        });
      },
      {
        rootMargin: '0px 0px 8% 0px',
        threshold: 0.08,
      },
    );

    const observedTargets = new WeakSet();
    // observeRevealTargets 监听进入视口的内容并触发显现动画。
    const observeRevealTargets = (scope = document) => {
      if (scope.matches?.(revealTargets) && !observedTargets.has(scope)) {
        observedTargets.add(scope);
        observer.observe(scope);
        if (scope.getBoundingClientRect().top < window.innerHeight * 1.12) {
          scope.classList.add('is-visible');
        }
      }

      scope.querySelectorAll?.(revealTargets).forEach((element) => {
        if (!observedTargets.has(element)) {
          observedTargets.add(element);
          observer.observe(element);
          if (element.getBoundingClientRect().top < window.innerHeight * 1.12) {
            element.classList.add('is-visible');
          }
        }
      });
    };

    observeRevealTargets();

    const mutationObserver = new MutationObserver((mutations) => {
      mutations.forEach((mutation) => {
        mutation.addedNodes.forEach((node) => {
          if (node.nodeType === Node.ELEMENT_NODE) {
            observeRevealTargets(node);
          }
        });
      });
    });

    mutationObserver.observe(document.body, {
      childList: true,
      subtree: true,
    });

    const lenis = new Lenis({
      autoRaf: true,
      autoToggle: true,
      anchors: true,
      lerp: scrollTuning.lenis.lerp,
      syncTouch: scrollTuning.lenis.syncTouch,
      syncTouchLerp: scrollTuning.lenis.syncTouchLerp,
      touchInertiaExponent: scrollTuning.lenis.touchInertiaExponent,
      smoothWheel: true,
      wheelMultiplier: scrollTuning.lenis.wheelMultiplier,
      touchMultiplier: scrollTuning.lenis.touchMultiplier,
      overscroll: true,
    });

    window.lenis = lenis;
    setRootVar('--hero-copy-y', '0px');
    setRootVar('--hero-device-y', '0px');
    setRootVar('--hero-device-opacity', '1');
    setRootVar('--hero-ui-opacity', '1');
    setRootVar('--forest-scale', '1');

    const stackGroups = [...document.querySelectorAll('.get-stack:not(.is-flow)')].map((stack) => ({
      stack,
      cards: [...stack.querySelectorAll('.get-card')],
    }));

    // update 根据当前滚动位置和速度计算整页动效目标。
    const update = (nextScroll) => {
      scrollTuning = getScrollTuning(mobileQuery.matches);
      Object.assign(lenis.options, scrollTuning.lenis);
      const scrollY = nextScroll ?? lenis.animatedScroll ?? window.scrollY;
      const now = performance.now();
      const elapsed = Math.max(16, now - previousScrollTime);
      const instantVelocity = ((scrollY - previousScrollY) / elapsed) * 16.67;
      const viewportHeight = window.innerHeight;
      const viewportWidth = window.innerWidth;
      const heroTravel = Math.min(900, viewportHeight * 1.04);
      const heroMotionProgress = clamp(scrollY / heroTravel, 0, 1);
      const gravityProgress = Math.pow(heroMotionProgress, 1.54);
      const settleProgress = 1 - Math.pow(1 - heroMotionProgress, 2.2);
      // 保留景深感，但避免首屏在滚动时产生被过度拉远的感觉。
      const forestLiftTarget = viewportWidth <= 560 ? -34 : viewportWidth <= 980 ? -54 : -76;
      const frontHillLiftTarget = viewportWidth <= 560 ? -18 : -34;
      const backHillLiftTarget = viewportWidth <= 560 ? -6 : -14;
      const copyLiftTarget = viewportWidth <= 560 ? -38 : viewportWidth <= 980 ? -50 : -62;
      const deviceLiftTarget = viewportWidth <= 560 ? -9 : -18;
      scrollVelocity =
        scrollVelocity * scrollTuning.inertia.velocityRetain +
        instantVelocity * scrollTuning.inertia.velocityInject;
      previousScrollY = scrollY;
      previousScrollTime = now;
      const inertialPull = clamp(
        scrollVelocity * scrollTuning.inertia.pullMultiplier,
        scrollTuning.inertia.pullMin,
        scrollTuning.inertia.pullMax,
      );

      document.body.classList.toggle('is-scrolled', scrollY > 16);
      setMotionTarget('backY', gravityProgress * backHillLiftTarget + inertialPull * 0.08);
      setMotionTarget('frontY', gravityProgress * frontHillLiftTarget + inertialPull * 0.24);
      setMotionTarget('forestY', gravityProgress * forestLiftTarget + inertialPull);
      setMotionTarget(
        'forestScale',
        1 +
          settleProgress * 0.018 +
          clamp(
            Math.abs(scrollVelocity) * scrollTuning.inertia.forestScaleVelocity,
            0,
            scrollTuning.inertia.forestScaleMax,
          ),
      );
      setMotionTarget('heroCopyY', settleProgress * copyLiftTarget + inertialPull * 0.16);
      setMotionTarget('heroDeviceY', gravityProgress * deviceLiftTarget + inertialPull * 0.18);
      setMotionTarget('heroDeviceOpacity', 1 - Math.min(heroMotionProgress * 0.04, 0.04));
      setMotionTarget('heroUIOpacity', 1 - Math.min(heroMotionProgress * 0.1, 0.1));
      setMotionTarget('heroLightX', (heroMotionProgress - 0.5) * 12);
      setMotionTarget('heroLightY', heroMotionProgress * -12);
      setRootVar('--final-glow-y', `${Math.round(scrollY * -0.018)}px`);
      ensureMotion();

      if (!stackGroups.length) {
        return;
      }

      stackGroups.forEach(({ stack, cards }) => {
        const stackTop = stack.offsetTop - viewportHeight * 1.1;
        const stackBottom = stack.offsetTop + stack.offsetHeight + viewportHeight;
        if (scrollY < stackTop || scrollY > stackBottom) {
          cards.forEach((card) => card.classList.remove('is-edge-slice'));
          return;
        }

        cards.forEach((card, index) => {
          const rect = card.getBoundingClientRect();
          const nextRect = cards[index + 1]?.getBoundingClientRect();
          const coverProgress = nextRect ? clamp((rect.bottom - nextRect.top) / rect.height, 0, 1) : 0;
          const centerOffset = (rect.top + rect.height / 2 - viewportHeight / 2) / viewportHeight;
          const scale = 1 - Math.min(coverProgress * 0.058 + Math.abs(centerOffset) * 0.006, 0.07);
          const imageScale = 1.045 - Math.min(coverProgress * 0.035 + Math.abs(centerOffset) * 0.016, 0.055);
          const layerOpacity = 1 - Math.min(coverProgress * 0.18, 0.18);
          const isEdgeSlice = rect.bottom < 24 || rect.top > viewportHeight - 24;
          card.style.setProperty('--card-scale', scale.toFixed(3));
          card.style.setProperty('--card-image-scale', imageScale.toFixed(3));
          card.style.setProperty('--stack-opacity', layerOpacity.toFixed(3));
          card.classList.toggle('is-edge-slice', isEdgeSlice);
        });
      });
    };

    update();
    lenis.on('scroll', ({ scroll }) => update(scroll));
    window.addEventListener('resize', handleResize);

    return () => {
      observer.disconnect();
      mutationObserver.disconnect();
      if (window.lenis === lenis) {
        delete window.lenis;
      }
      if (motion.raf) {
        cancelAnimationFrame(motion.raf);
      }
      lenis.destroy();
      window.removeEventListener('resize', handleResize);
    };
  }, []);
}

// Logo 渲染固定导航中的头像与个人名称。
function Logo() {
  return (
    <a className="logo" href="#hero" aria-label="回到首页">
      <span>
        <img src={avatarImage} alt="Loki Yin 头像" />
      </span>
      <strong>Loki Yin.</strong>
    </a>
  );
}

// Nav 提供桌面和移动端共用的页内分区导航。
function Nav() {
  return (
    <nav className="top-nav" aria-label="主导航">
      <Logo />
      <div className="nav-center">
        {resumeAnchors.map((item) => (
          <a href={item.href} key={item.href}>{item.label}</a>
        ))}
      </div>
      <div className="nav-actions">
        <a href="https://github.com/PRINCEofAsgd" rel="noreferrer" target="_blank">GitHub</a>
        <a href="#socials">联系我</a>
      </div>
    </nav>
  );
}

// MockColorField 使用低分辨率 Canvas 绘制首屏设备卡片中的环境色场。
function MockColorField() {
  const canvasRef = useRef(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) {
      return undefined;
    }

    const context = canvas.getContext('2d', { alpha: true });
    let width = 0;
    let height = 0;
    let animationFrame = 0;
    let lastPaintTime = Number.NEGATIVE_INFINITY;
    const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
    const colorStops = [
      { x: 0.66, y: 0.22, xAmp: 0.08, yAmp: 0.06, radius: 0.56, hue: 170, sat: 42, light: 70, alpha: 0.5, speed: 0.00024, phase: 0 },
      { x: 0.78, y: 0.62, xAmp: 0.08, yAmp: 0.08, radius: 0.5, hue: 192, sat: 38, light: 46, alpha: 0.4, speed: 0.0002, phase: 1.8 },
      { x: 0.44, y: 0.3, xAmp: 0.07, yAmp: 0.05, radius: 0.48, hue: 36, sat: 32, light: 76, alpha: 0.28, speed: 0.00018, phase: 3.2 },
      { x: 0.36, y: 0.72, xAmp: 0.06, yAmp: 0.07, radius: 0.54, hue: 158, sat: 28, light: 28, alpha: 0.34, speed: 0.00016, phase: 4.6 },
    ];

    // hsla 根据动画相位生成平滑变化的色彩值。
    const hsla = (stop, phase, alphaFactor = 1) => {
      const hue = (stop.hue + Math.sin(phase * 0.7) * 5 + Math.cos(phase * 0.38) * 3 + 360) % 360;
      const light = Math.max(26, Math.min(84, stop.light + Math.sin(phase * 0.52) * 3));
      return `hsla(${hue}, ${stop.sat}%, ${light}%, ${stop.alpha * alphaFactor})`;
    };

    // paint 绘制一帧多层渐变、暗部和高光。
    const paint = (time = 0) => {
      if (!width || !height) {
        return;
      }

      context.clearRect(0, 0, width, height);
      const colorPhase = time * 0.00008;
      const base = context.createLinearGradient(0, 0, width, height);
      base.addColorStop(0, `hsl(${166 + Math.sin(colorPhase) * 3}, 30%, 88%)`);
      base.addColorStop(0.44, `hsl(${180 + Math.cos(colorPhase * 1.18) * 4}, 32%, 76%)`);
      base.addColorStop(1, `hsl(${194 + Math.sin(colorPhase * 0.92) * 5}, 34%, 52%)`);
      context.fillStyle = base;
      context.fillRect(0, 0, width, height);
      context.globalCompositeOperation = 'source-over';
      context.filter = 'blur(18px)';

      colorStops.forEach((stop) => {
        const phase = time * stop.speed + stop.phase;
        const x = width * (stop.x + Math.sin(phase) * stop.xAmp + Math.sin(phase * 0.34 + stop.phase) * 0.02);
        const y = height * (stop.y + Math.cos(phase * 0.82) * stop.yAmp + Math.cos(phase * 0.41) * 0.02);
        const radius = Math.max(width, height) * (stop.radius + Math.sin(phase * 0.54) * 0.04);
        const gradient = context.createRadialGradient(x, y, 0, x, y, radius);
        gradient.addColorStop(0, hsla(stop, phase, 1));
        gradient.addColorStop(0.42, hsla(stop, phase + 0.8, 0.62));
        gradient.addColorStop(1, hsla(stop, phase + 1.2, 0));
        context.fillStyle = gradient;
        context.fillRect(0, 0, width, height);
      });

      context.globalCompositeOperation = 'multiply';
      context.filter = 'blur(22px)';
      const darkPhase = time * 0.00018;
      const darkX = width * (0.24 + Math.sin(darkPhase) * 0.06);
      const darkY = height * (0.5 + Math.cos(darkPhase * 0.82) * 0.05);
      const dark = context.createRadialGradient(darkX, darkY, 0, darkX, darkY, Math.max(width, height) * 0.64);
      dark.addColorStop(0, 'rgba(20, 45, 43, 0.34)');
      dark.addColorStop(0.54, 'rgba(20, 45, 43, 0.14)');
      dark.addColorStop(1, 'rgba(20, 45, 43, 0)');
      context.fillStyle = dark;
      context.fillRect(0, 0, width, height);

      context.globalCompositeOperation = 'source-over';
      context.filter = 'none';
      const sheenPhase = time * 0.0002;
      const sheen = context.createLinearGradient(0, height * (0.2 + Math.sin(sheenPhase) * 0.03), width, height * (0.8 + Math.cos(sheenPhase) * 0.03));
      sheen.addColorStop(0, 'rgba(255, 255, 255, 0.24)');
      sheen.addColorStop(0.42, 'rgba(255, 255, 255, 0.06)');
      sheen.addColorStop(0.76, 'rgba(105, 173, 169, 0.1)');
      sheen.addColorStop(1, 'rgba(0, 0, 0, 0.06)');
      context.fillStyle = sheen;
      context.fillRect(0, 0, width, height);

      const highlight = context.createLinearGradient(0, 0, 0, height);
      highlight.addColorStop(0, 'rgba(255, 255, 255, 0.18)');
      highlight.addColorStop(0.28, 'rgba(255, 255, 255, 0.04)');
      highlight.addColorStop(0.54, 'rgba(255, 255, 255, 0)');
      context.fillStyle = highlight;
      context.fillRect(0, 0, width, height);

      const shadow = context.createLinearGradient(0, 0, 0, height);
      shadow.addColorStop(0, 'rgba(0, 0, 0, 0)');
      shadow.addColorStop(0.44, 'rgba(0, 0, 0, 0)');
      shadow.addColorStop(0.72, 'rgba(0, 0, 0, 0.12)');
      shadow.addColorStop(1, 'rgba(0, 0, 0, 0.38)');
      context.fillStyle = shadow;
      context.fillRect(0, 0, width, height);
    };

    // animate 把环境色场限制为约 30 FPS，降低高刷新率设备的渐变绘制开销。
    const animate = (time) => {
      if (time - lastPaintTime >= 1000 / 30) {
        paint(time);
        lastPaintTime = time;
      }
      animationFrame = window.requestAnimationFrame(animate);
    };

    // startAnimation 根据“减少动态效果”偏好选择单帧或持续动画。
    const startAnimation = () => {
      window.cancelAnimationFrame(animationFrame);
      if (reduceMotion.matches) {
        paint(window.performance.now());
        return;
      }
      lastPaintTime = Number.NEGATIVE_INFINITY;
      animationFrame = window.requestAnimationFrame(animate);
    };

    // resize 以一半显示分辨率调整画布，兼顾清晰度和性能。
    const resize = () => {
      const rect = canvas.getBoundingClientRect();
      width = Math.max(280, Math.round(rect.width / 2));
      height = Math.max(220, Math.round(rect.height / 2));
      canvas.width = width;
      canvas.height = height;
      paint(window.performance.now());
    };

    resize();
    startAnimation();

    const resizeObserver = new ResizeObserver(resize);
    resizeObserver.observe(canvas);
    reduceMotion.addEventListener('change', startAnimation);

    return () => {
      window.cancelAnimationFrame(animationFrame);
      resizeObserver.disconnect();
      reduceMotion.removeEventListener('change', startAnimation);
    };
  }, []);

  return <canvas className="mock-color-field" ref={canvasRef} aria-hidden="true" />;
}

// ResumeMockup 渲染首屏中的个人资料产品化预览卡。
function ResumeMockup({ compact = false }) {
  return (
    <div className={`product-mockup ${compact ? 'compact' : ''}`}>
      <img className="mock-screen-background" src={screenBackgroundImage} alt="阳光窗边的咖啡与抹茶蛋糕" />
      <aside className="mock-sidebar">
        <div className="mock-pill" />
        <div className="mock-search" />
        {resumeAnchors.map((item) => (
          <a
            aria-label={`跳转到${item.label}`}
            className="mock-nav-row"
            href={item.href}
            key={item.href}
            rel={item.external ? 'noreferrer' : undefined}
            target={item.external ? '_blank' : undefined}
          >
            <span />
            <b>{item.label}</b>
          </a>
        ))}
        <div className="mock-course-title">信息</div>
        <div className="mock-line long" />
        <div className="mock-line" />
        <div className="mock-line short" />
      </aside>
      <section className="mock-panel">
        <div className="mock-community">
          <div className="mock-avatar">
            <img src={avatarImage} alt="Loki Yin 头像" />
          </div>
          <h3>Loki Yin</h3>
          <p>Profile Page</p>
          <div className="mock-members" aria-label="个人信息">
            {mockMemberIcons.map((icon) => (
              <span key={icon.alt}>
                <img src={icon.src} alt={icon.alt} />
              </span>
            ))}
            <strong>CS 大四 · 中国矿业大学(北京) · 北京</strong>
          </div>
          <a className="mock-button" href="#socials">查看联系方式</a>
          <small>
            211 · CS → Go / C++ → Backend / AI Infra
          </small>
        </div>
      </section>
      <ul className="mock-tech-stack" aria-label="技术栈">
        {techStackIcons.map((icon) => (
          <li
            className="mock-tech-logo"
            key={icon.name}
            style={{
              '--logo-x': `${icon.x}%`,
              '--logo-y': `${icon.y}%`,
              '--logo-size': `${icon.size}px`,
              '--logo-mobile-size': `${Math.round(icon.size * 0.7)}px`,
              '--logo-tilt': `${icon.tilt}deg`,
              '--logo-duration': `${icon.duration}s`,
              '--logo-delay': `${icon.delay}s`,
            }}
            title={icon.name}
          >
            <img src={icon.src} alt={icon.name} />
          </li>
        ))}
      </ul>
    </div>
  );
}

// Hero 渲染 Homepage 首屏、主要操作和森林视差图层。
function Hero() {
  return (
    <section id="hero" className="hero">
      <div className="hero-bg hills-back" />
      <div className="hero-copy reveal">
        <h1>
          <span>Loki Yin&apos;s Profile</span>
        </h1>
        <p>
          <span>欢迎来到 Loki 的主页，这里记录了我的信息与经历，希望你能更好地了解我。</span>
        </p>
        <div className="hero-actions">
          <a className="cta" href="/yanlili">给伴侣的点滴纪念</a>
          <a className="cta secondary" href="#socials">可向我申请内测账号</a>
        </div>
      </div>
      <div className="hero-device reveal">
        <ResumeMockup />
      </div>
      <div className="hero-bg forest-foreground" />
      <div className="hero-bg hills-front" />
    </section>
  );
}

// About 展示个人简介，并按视口中心高亮当前段落。
function About() {
  const sectionRef = useRef(null);
  const [activeLine, setActiveLine] = useState(0);
  const lines = [
    '4³ ❖ Blind National Top 30\n3³ ❖ Blind sub60\n3³ ❖ sub8',
    'HOK Jungle National Top 50\nNational Level II Athlete · 400m',
    '211 · CS | Go · Backend | AI Infra',
  ];

  useEffect(() => {
    // updateActiveLine 选择最接近视口中心的简介段落。
    const updateActiveLine = () => {
      const section = sectionRef.current;
      if (!section) {
        return;
      }

      const viewportCenter = window.innerHeight * 0.52;
      const paragraphs = [...section.querySelectorAll('.intro-lines p')];
      const nextIndex = paragraphs.reduce((closestIndex, paragraph, index) => {
        const rect = paragraph.getBoundingClientRect();
        const paragraphCenter = rect.top + rect.height / 2;
        const currentRect = paragraphs[closestIndex].getBoundingClientRect();
        const currentCenter = currentRect.top + currentRect.height / 2;

        return Math.abs(paragraphCenter - viewportCenter) <
          Math.abs(currentCenter - viewportCenter)
          ? index
          : closestIndex;
      }, 0);

      setActiveLine((current) => (current === nextIndex ? current : nextIndex));
    };

    updateActiveLine();
    window.addEventListener('scroll', updateActiveLine, { passive: true });
    window.addEventListener('resize', updateActiveLine);

    return () => {
      window.removeEventListener('scroll', updateActiveLine);
      window.removeEventListener('resize', updateActiveLine);
    };
  }, []);

  return (
    <section id="info" className="about-section" ref={sectionRef}>
      <div className="section-inner narrow">
        <span className="pill muted">信息</span>
        <div className="intro-lines">
          {lines.map((line, index) => (
            <p
              className={index === activeLine ? 'active' : ''}
              key={line}
            >
              {line}
            </p>
          ))}
        </div>
        <div className="profile-grid">
          {profileRows.map(([label, value]) => (
            <div key={label}>
              <span>{label}</span>
              <strong>{value}</strong>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

// Projects 提供可用键盘和按钮切换的代表项目标签页。
function Projects() {
  const [activeIndex, setActiveIndex] = useState(0);
  const active = useMemo(() => projects[activeIndex], [activeIndex]);
  const activeTabId = `project-tab-${active.slug}`;
  const activePanelId = `project-panel-${active.slug}`;
  // selectProject 循环选择项目，允许首尾连续切换。
  const selectProject = (index) => setActiveIndex((index + projects.length) % projects.length);
  // handleProjectKeyDown 支持左右方向键切换项目标签。
  const handleProjectKeyDown = (event, index) => {
    if (event.key === 'ArrowRight') {
      event.preventDefault();
      selectProject(index + 1);
    }

    if (event.key === 'ArrowLeft') {
      event.preventDefault();
      selectProject(index - 1);
    }
  };

  return (
    <section id="projects" className="features-section">
      <div className="section-inner">
        <div className="split-heading">
          <div>
            <span className="pill muted">项目经历</span>
            <h2>{'我的项目 < - > 也是我的小心愿'}</h2>
          </div>
          <p>
            四个项目分别覆盖高性能网络、Web 服务、Apple 双端协同与跨平台游戏开发。
          </p>
        </div>

        <div className="feature-tabs" role="tablist" aria-label="项目分类">
          {projects.map((project, index) => (
            <button
              aria-controls={`project-panel-${project.slug}`}
              aria-selected={index === activeIndex}
              className={index === activeIndex ? 'active' : ''}
              id={`project-tab-${project.slug}`}
              key={project.slug}
              onClick={() => selectProject(index)}
              onKeyDown={(event) => handleProjectKeyDown(event, index)}
              role="tab"
              tabIndex={index === activeIndex ? 0 : -1}
              type="button"
            >
              {project.title}
            </button>
          ))}
        </div>

        <div
          aria-labelledby={activeTabId}
          className="feature-stage"
          id={activePanelId}
          key={active.slug}
          role="tabpanel"
        >
          <div className="feature-app">
            <aside>
              <span className="feature-kicker">{active.title}</span>
              <h3>{active.title}</h3>
              <p>{active.caption}</p>
              <div className="feature-stack" aria-label={`${active.title} 技术栈`}>
                {active.stack.map((item) => (
                  <span key={item}>{item}</span>
                ))}
              </div>
              <a
                href={active.href}
                rel={active.external ? 'noreferrer' : undefined}
                target={active.external ? '_blank' : undefined}
              >
                查看项目
              </a>
            </aside>
            <main>
              <div className="feature-preview">
                <div className="feature-preview-head">
                  <span>{active.title}</span>
                  <em>{active.stack[0]}</em>
                </div>
                <div className="feature-preview-flow" aria-label={`${active.title} 项目链路`}>
                  {active.preview.map((item, index) => (
                    <React.Fragment key={item}>
                      <span>{item}</span>
                      {index < active.preview.length - 1 && <i aria-hidden="true" />}
                    </React.Fragment>
                  ))}
                </div>
                <div className="feature-preview-grid" aria-hidden="true">
                  {active.highlights.map((item) => (
                    <span key={item}>{item}</span>
                  ))}
                </div>
              </div>
              <div className="feature-highlights">
                {active.highlights.map((item) => (
                  <span key={item}>{item}</span>
                ))}
              </div>
            </main>
          </div>
        </div>

        <div className="feature-caption">
          <button
            aria-label="上一个项目"
            type="button"
            onClick={() => selectProject(activeIndex - 1)}
          >
            ←
          </button>
          <p>{active.caption}</p>
          <button
            aria-label="下一个项目"
            type="button"
            onClick={() => selectProject(activeIndex + 1)}
          >
            →
          </button>
        </div>
      </div>
    </section>
  );
}

// VionInternshipVisual 复用原作者的控制台结构，在右栏展示完整实习职责。
function VionInternshipVisual({ card }) {
  return (
    <div className="get-visual minimax-visual">
      <div className="visual-toolbar">
        <span />
        <span />
        <span />
        <span className="visual-tag">Go 后端</span>
      </div>
      <div className="minimax-console vion-console">
        <div className="minimax-console-head vion-console-head">
          <span>AIOT CLOUD / GO BACKEND</span>
          <i><b />INTERNSHIP</i>
        </div>

        <div className="minimax-flow vion-flow" aria-label="文安智能实习工作链路">
          {card.pipeline.map((item, index) => (
            <React.Fragment key={item}>
              <span>
                <small>0{index + 1}</small>
                {item}
              </span>
              {index < card.pipeline.length - 1 && <i aria-hidden="true">→</i>}
            </React.Fragment>
          ))}
        </div>

        <div className="minimax-milestones vion-capabilities" aria-label="文安智能实习职责">
          {card.deliverables.map((item, index) => (
            <article key={item.title}>
              <header>
                <span><small>0{index + 1}</small>{item.title}</span>
                <time>{item.code}</time>
              </header>
              <p>{item.text}</p>
            </article>
          ))}
        </div>
      </div>
    </div>
  );
}

// SealosOpenSourceVisual 复用原作者第二段经历的流程和职责控制台结构。
function SealosOpenSourceVisual({ card }) {
  return (
    <div className="get-visual sealos-visual">
      <div className="visual-toolbar">
        <span />
        <span />
        <span />
        <span className="visual-tag">开源协作</span>
      </div>
      <div className="minimax-console sealos-console">
        <div className="minimax-console-head sealos-console-head">
          <span className="sealos-console-logo">
            <img src={card.brandLogo} alt="Sealos Logo" />
            <b>Sealos</b>
          </span>
          <span>ISSUE DETECTION / BUG FIX</span>
          <i><b />OPEN SOURCE</i>
        </div>

        <div className="minimax-flow sealos-flow" aria-label="Sealos 资源生命周期链路">
          {card.pipeline.map((item, index) => (
            <React.Fragment key={item}>
              <span>
                <small>0{index + 1}</small>
                {item}
              </span>
              {index < card.pipeline.length - 1 && <i aria-hidden="true">→</i>}
            </React.Fragment>
          ))}
        </div>

        <div className="minimax-milestones sealos-capabilities" aria-label="Sealos 开源职责">
          {card.deliverables.map((item, index) => (
            <article key={item.title}>
              <header>
                <span><small>0{index + 1}</small>{item.title}</span>
                <time>{item.code}</time>
              </header>
              <p>{item.text}</p>
            </article>
          ))}
        </div>
      </div>
    </div>
  );
}

// StorySection 复用同一叙事卡片结构展示经历与兴趣。
function StorySection({ id, eyebrow, title, mutedTitle, text, cards, stacked = true }) {
  return (
    <section id={id} className="what-section">
      <div className="section-inner">
        <div className="split-heading">
          <div>
            <span className="pill muted">{eyebrow}</span>
            <h2>
              {title}
              <em>{mutedTitle}</em>
            </h2>
          </div>
          <p>{text}</p>
        </div>

        <div className={`get-stack${stacked ? '' : ' is-flow'}`}>
          {cards.map((card, index) => (
            <article
              className={`get-card card-${index + 1}${card.variant ? ` ${card.variant}-card` : ''}`}
              key={card.company ?? card.title}
              style={stacked ? {
                '--stack-top': `${86 + index * 18}px`,
                '--stack-z': index + 1,
              } : undefined}
            >
              <div className="get-copy">
                <span>{card.eyebrow}</span>
                {card.company && (
                  <div className="experience-meta">
                    <b>{card.company}</b>
                    <time>{card.period}</time>
                  </div>
                )}
                {card.copyBrand && (
                  <div className="experience-copy-brand">
                    <span className="minimax-logo vion-logo">
                      <img className="vion-logo-icon" src={card.brandIcon} alt="Vion Tech Icon" />
                      <img className="vion-logo-wordmark" src={card.brandLogo} alt="VionVision" />
                    </span>
                    <strong>{card.role}</strong>
                  </div>
                )}
                {card.title && <h3>{card.title}</h3>}
                {card.text && <p>{card.text}</p>}
                {card.focus && (
                  <div className="experience-focus">
                    <b>核心职责</b>
                    <span>{card.focus}</span>
                  </div>
                )}
                {card.note && (card.noteHref ? (
                  <a className="story-note-link" href={card.noteHref} rel="noreferrer" target="_blank">
                    {card.note}
                  </a>
                ) : (
                  <strong>{card.note}</strong>
                ))}
              </div>
              {card.variant === 'minimax' ? (
                <VionInternshipVisual card={card} />
              ) : card.variant === 'sealos' ? (
                <SealosOpenSourceVisual card={card} />
              ) : (
                <div
                  className="get-visual"
                  style={{
                    '--visual-image': `url(${card.image})`,
                    '--visual-position': card.imagePosition ?? 'center',
                  }}
                >
                  <div className="visual-toolbar">
                    <span />
                    <span />
                    <span />
                    <span className="visual-tag">记录</span>
                  </div>
                  <div className="visual-card">
                    <div />
                    <h4>{card.visualTitle ?? 'EXPERIENCE'}</h4>
                    <p>{card.visualText ?? '内容待填写'}</p>
                    <span className="visual-note">重点记录</span>
                  </div>
                </div>
              )}
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}

// Internship 组装工作与开源贡献经历区块。
function Internship() {
  return (
    <StorySection
      id="internship"
      eyebrow="工作经历"
      title="用真实工程经历，铺叙出成长道路。"
      mutedTitle=""
      text="从企业后端研发到云原生开源协作"
      cards={internshipCards}
      stacked={false}
    />
  );
}

// Hobbies 组装四项兴趣爱好与公开音乐主页入口。
function Hobbies() {
  return (
    <StorySection
      id="hobbies"
      eyebrow="爱好"
      title="兴趣爱好构成生活纹理。"
      mutedTitle=" 让个人页不止是一份简历。"
      text="先记录四项长期兴趣与代表经历，配图和更完整的故事将在后续补充。"
      cards={hobbyCards}
    />
  );
}

// Socials 展示公开联系信息与社媒轨道入口。
function Socials() {
  return (
    <section id="socials" className="faq-section" aria-labelledby="socials-title">
      <div className="section-inner">
        <h2 id="socials-title" className="section-visually-hidden">社媒与联系</h2>

        <div className="social-section-heading">
          <span className="social-section-label pill muted">社媒</span>
          <span className="social-hint">GitHub、邮箱、微信、抖音与网易云音乐</span>
        </div>

        <div id="contact-details" className="final-copy">
          <h2>沟通与协作，从这里开始。</h2>
          <p>
            邮箱：lokiyin4819@foxmail.com
            <br />
            微信：Yyg1061
          </p>
          <a className="cta" href="mailto:lokiyin4819@foxmail.com">发送邮件</a>
        </div>

        <div className="social-directory">
          <div className="social-strip" aria-label="社媒账号">
            <span className="social-track social-track-outer" aria-hidden="true" />
            <span className="social-track social-track-middle" aria-hidden="true" />
            <span className="social-track social-track-inner" aria-hidden="true" />
            <div className="social-orbit">
              {socials.map((item) => (
                <a
                  aria-label={item.name}
                  className={`social-${item.tone}`}
                  href={item.href}
                  key={item.name}
                  rel={item.external ? 'noreferrer' : undefined}
                  style={{
                    '--social-angle': `${item.orbit.angle}deg`,
                    '--social-radius': `var(--social-radius-${item.orbit.track})`,
                    '--social-duration': item.orbit.duration,
                    '--social-direction': item.orbit.direction,
                    '--social-icon-size': item.orbit.size,
                    '--social-icon-size-mobile': item.orbit.mobileSize,
                  }}
                  target={item.external ? '_blank' : undefined}
                >
                  <span className="social-orbit-card">
                    <span aria-hidden="true">
                      <item.Icon />
                    </span>
                  </span>
                </a>
              ))}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

// StartupLoader 在全部图片与字体就绪前显示进度和参考来源说明。
function StartupLoader({ progress, ready }) {
  return (
    <div className={ready ? 'startup-loader is-done' : 'startup-loader'} role="status" aria-live="polite" aria-label="页面资源加载中">
      <div className="startup-loader-inner">
        <span>Loki Yin&apos;s Profile</span>
        <div className="startup-progress" aria-hidden="true">
          <i style={{ transform: `scaleX(${progress})` }} />
        </div>
        <strong>{Math.round(progress * 100)}%</strong>
        <small className="startup-attribution">页面参考 github.com/LeoninCS/Homepage 设计。</small>
      </div>
    </div>
  );
}

// App 组合 Homepage 的资源门、导航与全部叙事区块。
function App() {
  const { progress, ready } = useResourceGate();
  useScrollEffects();

  return (
    <>
      <StartupLoader progress={progress} ready={ready} />
      <Nav />
      <main className={ready ? 'site-shell is-ready' : 'site-shell'}>
        <Hero />
        <About />
        <Internship />
        <Projects />
        <Hobbies />
        <Socials />
      </main>
    </>
  );
}

ReactDOMClient.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
