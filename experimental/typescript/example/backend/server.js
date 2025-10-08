import express from 'express';
import path from 'path';
import { fileURLToPath } from 'url';

const app = express();
const port = process.env.PORT || 8000;

const __dirname = path.dirname(fileURLToPath(import.meta.url));
app.use('/static', express.static(path.join(__dirname, 'static')));

// Serve chart page
app.get('/', (req, res) => {
  res.sendFile(path.join(__dirname, 'static/index.html'));
});

// Serve mock time-series data
app.get('/data', (req, res) => {
  const now = Date.now();
  const data = Array.from({ length: 12 }, (_, i) => ({
    date: new Date(now - i * 86400000).toISOString().slice(0, 10),
    sales: Math.floor(Math.random() * 1000) + 100,
  })).reverse();
  res.json(data);
});

app.listen(port, () => {
  console.log(`🚀 App running at http://localhost:${port}`);
});
