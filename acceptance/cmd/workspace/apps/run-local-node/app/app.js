const { createServer } = require('node:http');
const hostname = '127.0.0.1';
const port = process.env.PORT || 8000;
const server = createServer((req, res) => {
  res.statusCode = 200;
  res.setHeader('Content-Type', 'text/plain');
  res.end('Hello From App');

  // Close the server and exit the process after sending response
  server.close(() => {
    console.log('Server closed');
    process.exit(0);
  });
});

server.listen(port, hostname, () => {
  console.log(`Server running at http://${hostname}:${port}/`);
});
