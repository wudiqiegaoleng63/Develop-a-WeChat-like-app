import { createRouter, createWebHistory } from 'vue-router'
import Login from '../views/access/Login.vue'
import Register from '../views/access/Register.vue'
import SmsLogin from '../views/access/SmsLogin.vue'
import ContactChat from '../views/chat/ContactChat.vue'
import ContactList from '../views/chat/ContactList.vue'
import SessionList from '../views/chat/SessionList.vue'
import OwnInfo from '../views/chat/OwnInfo.vue'
import Manager from '../views/manager/Manager.vue'

const routes = [
  {
    path: '/',
    redirect: '/login'
  },
  {
    path: '/login',
    name: 'Login',
    component: Login
  },
  {
    path: '/register',
    name: 'Register',
    component: Register
  },
  {
    path: '/smslogin',
    name: 'SmsLogin',
    component: SmsLogin
  },
  {
    path: '/chat/:id',
    name: 'ContactChat',
    component: ContactChat,
    meta: { requiresAuth: true }
  },
  {
    path: '/contactlist',
    name: 'ContactList',
    component: ContactList,
    meta: { requiresAuth: true }
  },
  {
    path: '/sessionlist',
    name: 'SessionList',
    component: SessionList,
    meta: { requiresAuth: true }
  },
  {
    path: '/owninfo',
    name: 'OwnInfo',
    component: OwnInfo,
    meta: { requiresAuth: true }
  },
  {
    path: '/manager',
    name: 'Manager',
    component: Manager,
    meta: { requiresAuth: true, requiresAdmin: true }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 导航守卫
router.beforeEach((to, from, next) => {
  const userInfoStr = sessionStorage.getItem('userInfo')
  const userInfo = userInfoStr ? JSON.parse(userInfoStr) : null

  if (to.meta.requiresAuth) {
    if (!userInfo || !userInfo.uuid) {
      next('/login')
    } else if (to.meta.requiresAdmin && userInfo.uuid.substring(0, 4) !== '1000') {
      // 非靓号用户不能进入管理页面
      next('/sessionlist')
    } else {
      next()
    }
  } else {
    next()
  }
})

export default router