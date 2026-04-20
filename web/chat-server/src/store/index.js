import { createStore } from 'vuex'

export default createStore({
  state: {
    backendUrl: 'https://127.0.0.1:8000',
    wsUrl: 'wss://127.0.0.1:8000',
    userInfo: {},
    socket: null
  },
  mutations: {
    setUserInfo(state, userInfo) {
      state.userInfo = userInfo
      // 同步保存到sessionStorage
      sessionStorage.setItem('userInfo', JSON.stringify(userInfo))
    },
    cleanUserInfo(state) {
      state.userInfo = {}
      sessionStorage.removeItem('userInfo')
    },
    setSocket(state, socket) {
      state.socket = socket
    }
  },
  actions: {
    logout({ commit }) {
      commit('cleanUserInfo')
      if (state.socket) {
        state.socket.close()
      }
    }
  },
  getters: {
    isLogin: state => {
      return state.userInfo && state.userInfo.uuid
    }
  }
})