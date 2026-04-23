<template>
  <div class="register-wrap">
    <div
      class="register-window"
      :style="{
        boxShadow: `var(${'--el-box-shadow-dark'})`,
      }"
    >
      <h2 class="register-item">注册</h2>
      <el-form
        ref="formRef"
        :model="registerData"
        label-width="70px"
        class="demo-dynamic"
      >
        <el-form-item
          prop="nickname"
          label="昵称"
          :rules="[
            {
              required: true,
              message: '此项为必填项',
              trigger: 'blur',
            },
            {
              min: 3,
              max: 10,
              message: '昵称长度在 3 到 10 个字符',
              trigger: 'blur',
            },
          ]"
        >
          <el-input v-model="registerData.nickname" />
        </el-form-item>
        <el-form-item
          prop="telephone"
          label="账号"
          :rules="[
            {
              required: true,
              message: '此项为必填项',
              trigger: 'blur',
            },
          ]"
        >
          <el-input v-model="registerData.telephone" />
        </el-form-item>
        <el-form-item
          prop="password"
          label="密码"
          :rules="[
            {
              required: true,
              message: '此项为必填项',
              trigger: 'blur',
            },
          ]"
        >
          <el-input type="password" v-model="registerData.password" />
        </el-form-item>
        <el-form-item
          prop="sms_code"
          label="验证码"
          :rules="[
            {
              required: true,
              message: '此项为必填项',
              trigger: 'blur',
            },
          ]"
        >
          <el-input v-model="registerData.sms_code" style="max-width: 200px">
            <template #append>
              <el-button
                @click="sendSmsCode"
                style="background-color: rgb(229, 132, 132); color: #ffffff"
                >点击发送</el-button
              >
            </template>
          </el-input>
        </el-form-item>
      </el-form>
      <div class="register-button-container">
        <el-button type="primary" class="register-btn" @click="handleRegister"
          >注册</el-button
        >
      </div>
      <div class="go-login-button-container">
        <button class="go-sms-login-btn" @click="handleSmsLogin">
          验证码登录
        </button>
        <button class="go-password-login-btn" @click="handleLogin">
          密码登录
        </button>
      </div>
    </div>
  </div>
</template>

<script>
import { reactive, toRefs } from "vue";
import axios from "axios";
import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import { useStore } from "vuex";
export default {
  name: "Register",
  setup() {
    const data = reactive({
      registerData: {
        telephone: "",
        password: "",
        nickname: "",
        sms_code: "",
      },
    });
    const router = useRouter();
    const store = useStore();
    const handleRegister = async () => {
      try {
        if (
          !data.registerData.nickname ||
          !data.registerData.telephone ||
          !data.registerData.password ||
          !data.registerData.sms_code
        ) {
          ElMessage.error("请填写完整注册信息。");
          return;
        }
        if (
          data.registerData.nickname.length < 3 ||
          data.registerData.nickname.length > 10
        ) {
          ElMessage.error("昵称长度在 3 到 10 个字符。");
          return;
        }
        if (!checkTelephoneValid()) {
          ElMessage.error("请输入有效的手机号码。");
          return;
        }
        const response = await axios.post(
          store.state.backendUrl + "/register",
          data.registerData
        ); // 发送POST请求
        if (response.data.code == 200) {
          ElMessage.success(response.data.message);
          console.log(response.data.message);
          // 查看avatar前缀有没有http
          if (!response.data.data.avatar.startsWith("http")) {
            response.data.data.avatar =
              store.state.backendUrl + response.data.data.avatar;
          }
          store.commit("setUserInfo", response.data.data);
          // 准备创建websocket连接
          const wsUrl =
            store.state.wsUrl + "/wss?client_id=" + response.data.data.uuid;
          console.log(wsUrl);
          store.state.socket = new WebSocket(wsUrl);
          store.state.socket.onopen = () => {
            console.log("WebSocket连接已打开");
          };
          store.state.socket.onmessage = (message) => {
            console.log("收到消息：", message.data);
          };
          store.state.socket.onclose = () => {
            console.log("WebSocket连接已关闭");
          };
          store.state.socket.onerror = () => {
            console.log("WebSocket连接发生错误");
          };
          router.push("/chat/sessionlist");
        } else {
          ElMessage.error(response.data.message);
          console.log(response.data.message);
        }
      } catch (error) {
        ElMessage.error(error);
        console.log(error);
      }
    };
    const checkTelephoneValid = () => {
      const regex = /^1[3456789]\d{9}$/;
      return regex.test(data.registerData.telephone);
    };

    const handleLogin = () => {
      router.push("/login");
    };

    const handleSmsLogin = () => {
      router.push("/smsLogin");
    };

    const sendSmsCode = async () => {
      if (
        !data.registerData.telephone ||
        !data.registerData.nickname ||
        !data.registerData.password
      ) {
        ElMessage.error("请填写完整注册信息。");
        return;
      }
      if (!checkTelephoneValid()) {
        ElMessage.error("请输入有效的手机号码。");
        return;
      }
      const req = {
        telephone: data.registerData.telephone,
      };
      const rsp = await axios.post(
        store.state.backendUrl + "/user/sendSmsCode",
        req
      );
      console.log(rsp);
      if (rsp.data.code == 200) {
        ElMessage.success(rsp.data.message);
      } else if (rsp.data.code == 400) {
        ElMessage.warning(rsp.data.message);
      } else {
        ElMessage.error(rsp.data.message);
      }
    };

    return {
      ...toRefs(data),
      router,
      handleRegister,
      handleLogin,
      handleSmsLogin,
      sendSmsCode,
    };
  },
};
</script>

<style>
.register-wrap {
  height: 100vh;
  background-image: url("@/assets/img/chat_server_background.jpg");
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
}

.register-window {
  background-color: rgb(255, 255, 255, 0.7);
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  padding: 30px 50px;
  border-radius: 20px;
}

.register-item {
  text-align: center;
  margin-bottom: 20px;
  color: #494949;
}

.register-button-container {
  display: flex;
  justify-content: center; /* 水平居中 */
  margin-top: 20px; /* 可选，根据需要调整按钮与输入框之间的间距 */
  width: 100%;
}

.register-btn,
.register-btn:hover {
  background-color: rgb(229, 132, 132);
  border: none;
  color: #ffffff;
  font-weight: bold;
}

.el-alert {
  margin-top: 20px;
}

.go-login-button-container {
  display: flex;
  flex-direction: row-reverse;
  margin-top: 10px;
}

.go-sms-login-btn,
.go-password-login-btn {
  background-color: rgba(255, 255, 255, 0);
  border: none;
  cursor: pointer;
  color: #d65b54;
  font-weight: bold;
  text-decoration: underline;
  text-underline-offset: 0.2em;
  margin-left: 10px;
}
</style>