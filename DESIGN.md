# Design System

## Overview

OpenHook 管理控制台采用现代 SaaS 工具风格，以浅色为默认主题。设计语言追求"精密仪器"的感觉：干净的排版、克制的色彩、清晰的信息层级。界面以内容为中心，装饰元素最小化。

## Theme

默认浅色主题。场景：开发者在明亮的办公室或居家环境中使用，白天为主，偶尔夜间值班。浅色减少长时间使用的视觉疲劳，同时保持专业感。

深色主题作为可选切换，通过 CSS 变量实现。

## Color Palette

使用 OKLCH 色彩空间，确保在不同亮度下色彩感知一致。

### 主色板

| Token | Light | Dark | Usage |
|-------|-------|------|-------|
| `--bg-primary` | `oklch(99% 0.005 260)` | `oklch(18% 0.02 260)` | 页面背景 |
| `--bg-secondary` | `oklch(97% 0.01 260)` | `oklch(22% 0.02 260)` | 卡片、面板背景 |
| `--bg-tertiary` | `oklch(92% 0.015 260)` | `oklch(28% 0.03 260)` | Hover 状态、选中状态 |
| `--border-subtle` | `oklch(88% 0.01 260)` | `oklch(35% 0.03 260)` | 分割线、表单边框 |
| `--border-default` | `oklch(75% 0.02 260)` | `oklch(45% 0.04 260)` | 输入框边框、卡片边框 |
| `--text-primary` | `oklch(25% 0.02 260)` | `oklch(95% 0.01 260)` | 主标题、正文 |
| `--text-secondary` | `oklch(50% 0.02 260)` | `oklch(70% 0.02 260)` | 次要文字、描述 |
| `--text-tertiary` | `oklch(65% 0.02 260)` | `oklch(55% 0.02 260)` | 占位符、禁用状态 |

### 语义色

| Token | Light | Dark | Usage |
|-------|-------|------|-------|
| `--accent` | `oklch(55% 0.18 250)` | `oklch(65% 0.16 250)` | 主按钮、链接、活跃状态 |
| `--accent-hover` | `oklch(48% 0.2 250)` | `oklch(72% 0.18 250)` | Accent hover |
| `--success` | `oklch(55% 0.15 145)` | `oklch(65% 0.13 145)` | 成功、在线、已投递 |
| `--warning` | `oklch(65% 0.14 85)` | `oklch(75% 0.12 85)` | 警告、需要注意 |
| `--error` | `oklch(55% 0.18 25)` | `oklch(65% 0.16 25)` | 错误、失败、删除 |
| `--info` | `oklch(55% 0.12 230)` | `oklch(65% 0.1 230)` | 信息提示 |

### 状态色透明度变体

- `--success-bg`: `oklch(55% 0.15 145 / 0.08)` — 成功状态背景
- `--error-bg`: `oklch(55% 0.18 25 / 0.08)` — 错误状态背景
- `--warning-bg`: `oklch(65% 0.14 85 / 0.08)` — 警告状态背景

## Typography

### Font Stack

- **界面字体**：`Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif`
- **等宽字体**：`"JetBrains Mono", "Fira Code", "SF Mono", Consolas, monospace` — 用于模板内容、代码、日志

### 字号层级

| Token | Size | Weight | Line Height | Letter Spacing | Usage |
|-------|------|--------|-------------|----------------|-------|
| `text-xs` | 12px | 400 | 16px | 0.01em | 标签、辅助文字 |
| `text-sm` | 13px | 400 | 20px | 0 | 正文、表格内容 |
| `text-base` | 14px | 400 | 22px | -0.01em | 默认正文 |
| `text-lg` | 16px | 500 | 24px | -0.02em | 小标题、 emphasized |
| `text-xl` | 20px | 600 | 28px | -0.02em | 页面标题 |
| `text-2xl` | 24px | 600 | 32px | -0.03em | 大标题 |

### 代码/日志字体

| Token | Size | Weight | Line Height | Usage |
|-------|------|--------|-------------|-------|
| `code-sm` | 12px | 400 | 18px | 日志内容、JSON |
| `code-base` | 13px | 400 | 20px | 模板编辑器 |

## Spacing

基于 4px 网格：

| Token | Value |
|-------|-------|
| `space-1` | 4px |
| `space-2` | 8px |
| `space-3` | 12px |
| `space-4` | 16px |
| `space-5` | 20px |
| `space-6` | 24px |
| `space-8` | 32px |
| `space-10` | 40px |
| `space-12` | 48px |

### 布局间距

- 页面内边距：`space-6` (24px)
- 卡片内边距：`space-4` (16px)
- 表单字段间距：`space-4` (16px)
- 列表项间距：`space-2` (8px)
- 紧凑表格行高：44px

## Components

### Button

**Primary**
- Background: `--accent`
- Text: white
- Padding: 8px 16px
- Border-radius: 6px
- Font: text-sm, weight 500
- Hover: `--accent-hover`, transform scale(1.01)
- Active: scale(0.98)
- Transition: 150ms ease-out

**Secondary**
- Background: `--bg-secondary`
- Border: 1px solid `--border-default`
- Text: `--text-primary`
- Hover: `--bg-tertiary`

**Danger**
- Background: transparent
- Border: 1px solid `--error`
- Text: `--error`
- Hover: `--error-bg`

**Ghost**
- Background: transparent
- Text: `--text-secondary`
- Hover: `--bg-tertiary`

### Input

- Background: `--bg-primary`
- Border: 1px solid `--border-default`
- Border-radius: 6px
- Padding: 8px 12px
- Font: text-base
- Focus: border `--accent`, ring 2px `--accent` at 20% opacity
- Placeholder: `--text-tertiary`
- Transition: border-color 150ms, box-shadow 150ms

**Textarea (模板编辑器)**
- Font: code-base (JetBrains Mono)
- Line-height: 22px
- Min-height: 120px
- Tab-size: 2

### Card

- Background: `--bg-secondary`
- Border: 1px solid `--border-subtle`
- Border-radius: 8px
- Padding: space-4
- No box-shadow (flat design)
- Hover: border-color `--border-default`

### Table

- Header: text-xs, weight 500, uppercase, `--text-tertiary`, background `--bg-secondary`
- Row height: 44px
- Row border-bottom: 1px solid `--border-subtle`
- Row hover: `--bg-tertiary`
- Selected row: `--accent` background at 5% opacity

### Sidebar Navigation

- Width: 240px
- Background: `--bg-secondary`
- Border-right: 1px solid `--border-subtle`
- Nav item: padding 8px 12px, border-radius 6px
- Nav item hover: `--bg-tertiary`
- Nav item active: `--accent` at 8% opacity, text `--accent`
- Icon + label 间距: space-3

### Status Badge

| Status | Style |
|--------|-------|
| 成功/已投递 | `--success` 文字 + `--success-bg` 背景，圆角 4px，padding 2px 8px |
| 失败/错误 | `--error` 文字 + `--error-bg` 背景 |
| 警告 | `--warning` 文字 + `--warning-bg` 背景 |
| 处理中 | `--info` 文字 + 旋转的加载图标 |

### Toast / Notification

- Position: top-right
- Background: `--bg-secondary`
- Border: 1px solid `--border-subtle`
- Border-radius: 8px
- Padding: 12px 16px
- Shadow: 0 4px 12px rgba(0,0,0,0.08)
- Auto-dismiss: 4s
- Slide-in from right: 200ms ease-out-quart

## Layout

### App Shell

```
┌─────────────────────────────────────────────┐
│  Sidebar (240px)    │  Main Content Area     │
│  - Logo             │  ┌──────────────────┐  │
│  - Navigation       │  │ Header           │  │
│    - Templates      │  ├──────────────────┤  │
│    - Routes         │  │                  │  │
│    - Middlewares    │  │  Content         │  │
│    - Tokens         │  │                  │  │
│    - Deliveries     │  │                  │  │
│    - Filters        │  │                  │  │
│    - Dedup Rules    │  │                  │  │
│  ─────────────────  │  └──────────────────┘  │
│  - Settings         │                        │
└─────────────────────────────────────────────┘
```

### Template Editor Layout (Split View)

```
┌──────────────────────────────┬──────────────────┐
│  编辑区 (60%)                 │  预览区 (40%)     │
│  ┌────────────────────────┐  │  ┌────────────┐  │
│  │ Template Name          │  │  │ Rendered   │  │
│  ├────────────────────────┤  │  │ Preview    │  │
│  │ Content (textarea)     │  │  │            │  │
│  │                        │  │  │            │  │
│  │ # {{data.title}}       │  │  │ # Alert    │  │
│  │ - severity: ...        │  │  │ - sev...   │  │
│  │                        │  │  │            │  │
│  ├────────────────────────┤  │  └────────────┘  │
│  │ Simulation Data (JSON) │  │  ┌────────────┐  │
│  │ { ... }                │  │  │ Envelope   │  │
│  └────────────────────────┘  │  │ JSON       │  │
│                              │  └────────────┘  │
└──────────────────────────────┴──────────────────┘
```

## Elevation

不使用传统的 box-shadow 层级系统。采用扁平设计，通过 border 和 background 区分层级：

- 页面背景: `--bg-primary`
- 面板/卡片: `--bg-secondary` + `--border-subtle`
- 浮层/下拉: `--bg-secondary` + `--border-default` + `0 2px 8px rgba(0,0,0,0.04)`

## Motion

### 原则

- 所有动画使用 `ease-out-quart`：`cubic-bezier(0.25, 1, 0.5, 1)`
- 动画时长：快速反馈 150ms，内容过渡 200ms，页面切换 300ms
- 绝不动画化 layout 属性（width, height, top, left）
- 尊重 `prefers-reduced-motion: reduce`

### 具体效果

| 场景 | 动画 | 时长 |
|------|------|------|
| Button hover | background-color, scale(1.01) | 150ms |
| Button active | scale(0.98) | 100ms |
| 页面切换 | opacity 0→1 + translateY(4px→0) | 200ms |
| Toast 出现 | translateX(100%→0) + opacity | 200ms ease-out-quart |
| 模态框出现 | opacity 0→1 + scale(0.97→1) | 200ms |
| 列表项加载 | stagger opacity 0→1, 间隔 30ms | 150ms each |
| Sidebar 折叠 | width 变化（使用 CSS grid/flex 过渡） | 250ms |

## Icons

使用 Lucide Icons（`lucide-svelte`）。图标尺寸规范：

- 导航图标: 18px, stroke-width 1.5
- 按钮图标: 16px, stroke-width 2
- 表格内图标: 14px, stroke-width 2
- 状态指示: 16px, 与文字同色

## Responsive

### 断点

| Name | Width | Behavior |
|------|-------|----------|
| Mobile | < 768px | Sidebar 变为汉堡菜单，堆叠布局 |
| Tablet | 768px - 1024px | Sidebar 可折叠，表格横向滚动 |
| Desktop | > 1024px | 完整布局，分栏编辑器 |

### 移动端适配

- 模板编辑器的分栏变为垂直堆叠（编辑在上，预览在下）
- 表格转为卡片列表视图
- 底部固定操作栏（保存/取消）

## Z-Index Scale

| Layer | Z-Index | Elements |
|-------|---------|----------|
| Base | 0 | 正常内容 |
| Sticky | 10 | Sticky header |
| Dropdown | 50 | Select dropdown, popover |
| Modal | 100 | Dialog overlay |
| Toast | 200 | Notification stack |
