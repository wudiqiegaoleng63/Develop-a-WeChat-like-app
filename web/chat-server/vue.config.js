const { defineConfig } = require('@vue/cli-service')
const fs = require('fs')
const path = require('path')

module.exports = defineConfig({
  transpileDependencies: true,
  lintOnSave: false,
  devServer: {
    host: '0.0.0.0',
    https: {
      cert: fs.readFileSync(path.join(__dirname, 'src/assets/cert/127.0.0.1+2.pem')),
      key: fs.readFileSync(path.join(__dirname, 'src/assets/cert/127.0.0.1+2-key.pem'))
    },
    port: 443
  }
})