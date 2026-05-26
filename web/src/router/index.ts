import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/pages/Login.vue'),
    meta: { title: '登录', public: true },
  },
  {
    path: '/',
    component: () => import('@/layouts/MainLayout.vue'),
    redirect: '/chat',
    children: [
      {
        path: 'chat',
        name: 'Chat',
        component: () => import('@/pages/Chat.vue'),
        meta: { title: '对话测试', icon: 'Promotion' },
      },
      {
        path: 'vendors',
        name: 'Vendors',
        component: () => import('@/pages/Vendors.vue'),
        meta: { title: '模型厂商', icon: 'Setting' },
      },
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/pages/Dashboard.vue'),
        meta: { title: '数据概览', icon: 'DataBoard' },
      },
      {
        path: 'agents',
        name: 'Agents',
        component: () => import('@/pages/Agents.vue'),
        meta: { title: 'Agent管理', icon: 'Robot' },
      },
      {
        path: 'tools',
        name: 'Tools',
        component: () => import('@/pages/Tools.vue'),
        meta: { title: '工具管理', icon: 'SetUp' },
      },
      {
        path: 'sessions',
        name: 'Sessions',
        component: () => import('@/pages/Sessions.vue'),
        meta: { title: '对话监控', icon: 'ChatDotRound' },
      },
      {
        path: 'memories',
        name: 'Memories',
        component: () => import('@/pages/Memories.vue'),
        meta: { title: '记忆管理', icon: 'Coin' },
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('@/pages/Users.vue'),
        meta: { title: '用户管理', icon: 'User' },
      },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// 路由守卫
router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('token')
  if (!to.meta.public && !token) {
    next('/login')
  } else if (to.path === '/login' && token) {
    next('/chat')
  } else {
    document.title = `${to.meta.title || 'Eino Agent'} - 管理系统`
    next()
  }
})

export default router
