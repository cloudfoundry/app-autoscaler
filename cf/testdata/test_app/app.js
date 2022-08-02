const express = require('express')
const app = express()

app.get('/health', function (req, res) {
  res.status(200).json({ status: 'OK' })
})

app.get('/', function (req, res) {
  res.status(200).send('dummy application root')
})

app.listen(process.env.PORT || 8080, function () {
  console.log('dummy application started')
})
