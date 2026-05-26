<script setup lang="ts">
import { ElConfigProvider } from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
</script>

<template>
  <el-config-provider :locale="zhCn">
    <router-view />
  </el-config-provider>
</template>

<style>
:root {
  --theme-primary: #6ea96a;
  --theme-primary-deep: #517a4c;
  --theme-primary-soft: #dff1df;
  --theme-primary-mist: #eef8f0;
  --theme-accent: #8bcf7d;
  --theme-bg: #f3faf2;
  --theme-surface: rgba(255, 255, 255, 0.96);
  --theme-surface-strong: rgba(255, 255, 255, 0.985);
  --theme-menu-bg: rgba(247, 251, 245, 0.98);
  --theme-menu-hover: rgba(226, 240, 223, 0.95);
  --theme-menu-active: rgba(220, 239, 217, 0.98);
  --theme-text: #1f3224;
  --theme-text-secondary: #5f6f5f;
  --theme-border: #dbe9db;
  --theme-input-bg: rgba(245, 250, 243, 0.92);
  --theme-input-border: #cde1cb;
  --theme-input-focus: rgba(110, 169, 106, 0.18);
  --theme-shadow: 0 18px 48px rgba(81, 122, 76, 0.1);
  --theme-shadow-soft: 0 10px 26px rgba(81, 122, 76, 0.08);
  --theme-shadow-subtle: 0 8px 20px rgba(81, 122, 76, 0.06);
  --el-color-primary: var(--theme-primary);
  --el-color-primary-dark-2: var(--theme-primary-deep);
  --el-color-primary-light-3: #95c48c;
  --el-color-primary-light-5: #cfe8cf;
  --el-color-primary-light-7: #e8f5e7;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body, #app {
  height: 100%;
  font-family: 'Helvetica Neue', Helvetica, 'PingFang SC', 'Hiragino Sans GB',
    'Microsoft YaHei', '微软雅黑', Arial, sans-serif;
  background: linear-gradient(180deg, #f8fcf7 0%, #eef8ee 100%);
  color: var(--theme-text);
}

::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

::-webkit-scrollbar-thumb {
  background: #c8d8c5;
  border-radius: 3px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

.el-card {
  background: var(--theme-surface) !important;
  border: 1px solid var(--theme-border) !important;
  border-radius: 18px !important;
  box-shadow: var(--theme-shadow-soft) !important;
  backdrop-filter: blur(10px);
  transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease;
}

.el-card:hover {
  box-shadow: var(--theme-shadow) !important;
  transform: translateY(-1px);
}

.el-card.is-hover-shadow,
.el-card.is-always-shadow {
  box-shadow: var(--theme-shadow-soft) !important;
}

.el-card__header {
  border-bottom-color: var(--theme-border) !important;
}

.el-dialog,
.el-message-box,
.el-notification,
.el-message {
  background: var(--theme-surface-strong) !important;
  border: 1px solid var(--theme-border) !important;
  box-shadow: var(--theme-shadow) !important;
}

.el-dialog {
  border-radius: 18px !important;
}

.el-message-box,
.el-notification,
.el-message {
  border-radius: 16px !important;
}

.el-dialog__header,
.el-message-box__header,
.el-notification__title {
  border-bottom-color: var(--theme-border) !important;
}

.el-dialog__body,
.el-message-box__content,
.el-notification__content,
.el-message {
  color: var(--theme-text-secondary);
}

.el-dropdown-menu {
  background: var(--theme-surface-strong) !important;
  border: 1px solid var(--theme-border) !important;
  border-radius: 16px !important;
  box-shadow: var(--theme-shadow) !important;
  padding: 6px !important;
}

.el-dropdown-menu__item {
  border-radius: 12px !important;
  color: var(--theme-text) !important;
  font-weight: 500;
}

.el-dropdown-menu__item:hover {
  background: var(--theme-menu-hover) !important;
  color: var(--theme-primary-deep) !important;
}

.el-tooltip__popper {
  background: var(--theme-surface-strong) !important;
  border: 1px solid var(--theme-border) !important;
  border-radius: 12px !important;
  box-shadow: var(--theme-shadow-soft) !important;
  color: var(--theme-text) !important;
}

.el-button--primary {
  background: linear-gradient(135deg, var(--theme-primary) 0%, var(--theme-primary-deep) 100%) !important;
  border-color: transparent !important;
  color: #fff !important;
  box-shadow: 0 10px 24px rgba(110, 169, 106, 0.2) !important;
}

.el-button--primary:hover,
.el-button--primary:focus {
  filter: brightness(1.03);
}

.el-button--default,
.el-button--default:hover,
.el-button--default:focus {
  background: rgba(255, 255, 255, 0.96) !important;
  border-color: var(--theme-input-border) !important;
  color: var(--theme-text) !important;
}

.el-button--default:hover,
.el-button--default:focus {
  background: var(--theme-primary-soft) !important;
}

.el-input__wrapper,
.el-select__wrapper,
.el-textarea__inner {
  background: var(--theme-input-bg) !important;
  border: 1px solid var(--theme-input-border) !important;
  border-radius: 12px !important;
  box-shadow: none !important;
}

.el-input__wrapper.is-focus,
.el-select__wrapper.is-focused,
.el-textarea__inner:focus,
.el-textarea__inner:focus-visible {
  border-color: var(--theme-primary) !important;
  box-shadow: 0 0 0 3px var(--theme-input-focus) !important;
}

.el-textarea__inner {
  padding: 12px 14px;
}

.el-input__inner,
.el-textarea__inner,
.el-select__placeholder,
.el-select-dropdown__item {
  color: var(--theme-text);
}

.el-select-dropdown,
.el-select-dropdown__list,
.el-popper {
  border-radius: 14px !important;
  border: 1px solid var(--theme-border) !important;
  box-shadow: var(--theme-shadow) !important;
  background: rgba(255, 255, 255, 0.98) !important;
}

.el-table {
  --el-table-border-color: var(--theme-border) !important;
  --el-table-header-bg-color: #f7fbf6 !important;
  --el-table-row-hover-bg-color: #f3faf3 !important;
  --el-table-bg-color: transparent !important;
  --el-table-tr-bg-color: transparent !important;
  border-radius: 14px !important;
  overflow: hidden;
}

.el-table th {
  background: linear-gradient(180deg, #f7fbf6 0%, #eef8ee 100%) !important;
  color: var(--theme-text) !important;
  font-weight: 700 !important;
  border-bottom: 1px solid var(--theme-border) !important;
}

.el-table td {
  border-bottom: 1px solid rgba(219, 233, 219, 0.75) !important;
  color: var(--theme-text-secondary) !important;
}

.el-table--striped .el-table__body tr.el-table__row--striped td {
  background: rgba(244, 250, 243, 0.7) !important;
}

.el-table tr:hover > td {
  background: #f4faf4 !important;
}

.el-pagination {
  --el-pagination-hover-color: var(--theme-primary-deep);
  --el-pagination-active-color: var(--theme-primary);
  --el-pagination-border-color: var(--theme-border);
  --el-pagination-bg-color: rgba(248, 252, 247, 0.95);
  --el-pagination-hover-bg-color: rgba(239, 248, 239, 0.98);
}

.el-pagination.is-background .el-pager li,
.el-pagination.is-background .btn-prev,
.el-pagination.is-background .btn-next {
  background: rgba(248, 252, 247, 0.95) !important;
  border: 1px solid var(--theme-border) !important;
  border-radius: 10px !important;
  color: var(--theme-text-secondary) !important;
  box-shadow: var(--theme-shadow-subtle) !important;
}

.el-pagination.is-background .el-pager li:hover,
.el-pagination.is-background .btn-prev:hover,
.el-pagination.is-background .btn-next:hover {
  background: rgba(239, 248, 239, 0.98) !important;
  color: var(--theme-primary-deep) !important;
}

.el-pagination.is-background .el-pager li.is-active {
  background: linear-gradient(135deg, var(--theme-primary) 0%, var(--theme-primary-deep) 100%) !important;
  border-color: transparent !important;
  color: #fff !important;
}

.el-tag {
  border-radius: 999px !important;
  font-weight: 600;
  border: 1px solid transparent !important;
  background: var(--theme-primary-soft) !important;
  color: var(--theme-primary-deep) !important;
}

.el-tag--success {
  background: var(--theme-primary-soft) !important;
  color: var(--theme-primary-deep) !important;
  border-color: #cde1cb !important;
}

.el-tag--info {
  background: #eef7f0 !important;
  color: #4f6a52 !important;
  border-color: #d7e8d3 !important;
}

.el-tag--warning {
  background: #f1f8ee !important;
  color: #5e7b56 !important;
  border-color: #d9e9d4 !important;
}

.el-tag--danger {
  background: #f5efe8 !important;
  color: #8a5d46 !important;
  border-color: #e6d1c1 !important;
}

.el-alert {
  border-radius: 14px !important;
  border: 1px solid var(--theme-border) !important;
  background: rgba(244, 250, 243, 0.94) !important;
  color: var(--theme-text) !important;
  box-shadow: var(--theme-shadow-subtle) !important;
}

.el-alert--success {
  background: rgba(239, 248, 239, 0.95) !important;
}

.el-alert--warning {
  background: rgba(246, 250, 241, 0.95) !important;
}

.el-alert--error {
  background: rgba(251, 245, 241, 0.95) !important;
}

.el-menu {
  border-right: none !important;
}

.el-menu-item {
  border-radius: 12px !important;
  margin: 4px 10px !important;
  color: #4f6a52 !important;
  transition: background-color 0.2s ease, color 0.2s ease, transform 0.2s ease;
}

.el-menu-item:hover,
.el-menu-item:focus {
  background: var(--theme-menu-hover) !important;
  color: var(--theme-primary-deep) !important;
}

.el-menu-item.is-active {
  background: var(--theme-menu-active) !important;
  color: var(--theme-primary-deep) !important;
  font-weight: 700;
}

.el-form-item__label {
  color: var(--theme-text-secondary);
  font-weight: 600;
}

.el-switch.is-checked .el-switch__core {
  background-color: var(--theme-primary) !important;
  border-color: var(--theme-primary) !important;
}

.el-checkbox__input.is-checked .el-checkbox__inner,
.el-radio__input.is-checked .el-radio__inner {
  background-color: var(--theme-primary) !important;
  border-color: var(--theme-primary) !important;
}
</style>
