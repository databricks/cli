const http = require('node:http');

// Standard library only: acceptance tests run with the network disabled, so the
// app cannot install anything from the NPM registry.

// The CLI sets PORT alongside DATABRICKS_APP_PORT. Unset, listen() would pick a random port
// and the run would fail as a proxy timeout instead.
const port = process.env.PORT;
if (!port) throw new Error('PORT is not set');

const server = http.createServer((req, res) => {
  if (req.url === '/shutdown') {
    // close() waits for open connections, so the response must end this one or the proxy holds it.
    res.setHeader('Connection', 'close');
    res.end();
    server.close();
    return;
  }

  res.writeHead(200, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify({ message: 'Hello From App' }) + '\n');
});

server.listen(port, '127.0.0.1', () => {
  console.log(`Server is running on port ${port}`);
});
