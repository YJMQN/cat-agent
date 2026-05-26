<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const isCollapse = ref(false)

const menuItems = [
  { path: '/chat', icon: 'Promotion', title: '对话测试' },
  { path: '/vendors', icon: 'Setting', title: '模型厂商' },
  { path: '/dashboard', icon: 'DataBoard', title: '数据概览' },
  { path: '/agents', icon: 'Robot', title: 'Agent管理' },
  { path: '/tools', icon: 'SetUp', title: '工具管理' },
  { path: '/sessions', icon: 'ChatDotRound', title: '对话监控' },
  { path: '/memories', icon: 'Coin', title: '记忆管理' },
  { path: '/users', icon: 'User', title: '用户管理' },
]

const activeMenu = computed(() => route.path)
const pageTitle = computed(() => {
  const item = menuItems.find(m => m.path === route.path)
  return item?.title || 'Eino Agent'
})

function handleLogout() {
  userStore.logout()
  router.push('/login')
}
</script>

<template>
  <el-container class="layout-container">
    <!-- 侧边栏 -->
    <el-aside :width="isCollapse ? '64px' : '220px'" class="aside">
      <div class="logo" @click="router.push('/dashboard')">
        <el-icon :size="28" color="#6ea96a"><Cpu /></el-icon>
        <span v-show="!isCollapse" class="logo-text">Eino Agent</span>
      </div>

      <el-menu
        :default-active="activeMenu"
        :collapse="isCollapse"
        background-color="transparent"
        text-color="#4f6a52"
        active-text-color="#2f5b36"
        router
        class="side-menu"
      >
        <el-menu-item v-for="item in menuItems" :key="item.path" :index="item.path">
          <el-icon><component :is="item.icon" /></el-icon>
          <template #title>{{ item.title }}</template>
        </el-menu-item>
      </el-menu>

      <div class="collapse-btn" @click="isCollapse = !isCollapse">
        <el-icon><Fold v-if="!isCollapse" /><Expand v-else /></el-icon>
      </div>
    </el-aside>

    <!-- 主内容区 -->
    <el-container>
      <el-header class="header">
        <div class="header-left">
          <h2>{{ pageTitle }}</h2>
        </div>
        <div class="header-right">
          <el-dropdown trigger="click">
            <span class="user-dropdown">
              <el-icon><UserFilled /></el-icon>
              <span>{{ userStore.user?.username || '管理员' }}</span>
              <el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="handleLogout">
                  <el-icon><SwitchButton /></el-icon>
                  退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <el-main class="main">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <keep-alive>
              <component :is="Component" />
            </keep-alive>
          </transition>
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<style scoped>
.layout-container {
  height: 100vh;
}

.aside {
  background: linear-gradient(180deg, #f8fcf7 0%, #f2f8f0 100%);
  border-right: 1px solid #dbe9db;
  transition: width 0.3s;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  cursor: pointer;
  border-bottom: 1px solid #dbe9db;
  flex-shrink: 0;
  background: rgba(255, 255, 255, 0.6);
}

.logo-text {
  color: #2f5b36;
  font-size: 18px;
  font-weight: 700;
  white-space: nowrap;
}

.side-menu {
  flex: 1;
  border-right: none;
  overflow-y: auto;
  padding: 12px 0;
  background: transparent !important;
}

.side-menu :deep(.el-menu-item) {
  margin: 4px 10px;
  border-radius: 12px;
}

.side-menu :deep(.el-menu-item:hover) {
  background: rgba(226, 240, 223, 0.95) !important;
}

.side-menu :deep(.el-menu-item.is-active) {
  background: rgba(220, 239, 217, 0.98) !important;
  color: #2f5b36 !important;
}

.collapse-btn {
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #5f6f5f;
  cursor: pointer;
  border-top: 1px solid #dbe9db;
  flex-shrink: 0;
}

.collapse-btn:hover {
  color: #2f5b36;
}

.header {
  background: rgba(255, 255, 255, 0.95);
  border-bottom: 1px solid #dbe9db;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
}

.header-left h2 {
  font-size: 18px;
  font-weight: 500;
  color: #1f3224;
}

.user-dropdown {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  color: #5f6f5f;
  font-size: 14px;
}

.user-dropdown:hover {
  color: #517a4c;
}

.main {
  background: linear-gradient(180deg, #f8fcf7 0%, #eef8ee 100%);
  padding: 20px;
  overflow-y: auto;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
