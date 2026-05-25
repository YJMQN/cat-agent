import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { User } from '@/types'

export const useUserStore = defineStore('user', () => {
  const user = ref<User | null>(null)
  const token = ref(localStorage.getItem('token') || '')

  function setUser(u: User, t: string) {
    user.value = u
    token.value = t
    localStorage.setItem('token', t)
  }

  function logout() {
    user.value = null
    token.value = ''
    localStorage.removeItem('token')
  }

  function isLoggedIn() {
    return !!token.value
  }

  return { user, token, setUser, logout, isLoggedIn }
})
