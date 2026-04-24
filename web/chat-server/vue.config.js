const { defineConfig } = require('@vue/cli-service')
const fs = require('fs')
const path = require('path')
module.exports = defineConfig({
  transpileDependencies: true,
  lintOnSave: false,
  // HTTP开发模式（本地开发使用）
  devServer: {
    host: '0.0.0.0',
    port: 8080,
  }
  // HTTPS生产模式（云服务器部署使用）
  // devServer: {
  //   host: '0.0.0.0',
  //   https: {
  //     // Ubuntu22.04云服务器部署
  //     cert: fs.readFileSync(path.join("/etc/ssl/certs/server.crt")),
  //     key: fs.readFileSync(path.join("/etc/ssl/private/server.key")),
  //   },
  //   port: 443,
  // }
})