const express = require('express');
const app = express();
const port = process.env.PORT || 8000;

// Root route
app.get('/', (req, res) => {
  res.json({
    message: 'Hello From App',
    timestamp: new Date().toISOString(),
    status: 'running'
  });
});

app.get('/shutdown', (req, res) => {
  console.log('Server closed')
  process.exit(0);
});

// Start the server
app.listen(port, () => {
  console.log(`Server is running on port ${port}`);
});

module.exports = app;
