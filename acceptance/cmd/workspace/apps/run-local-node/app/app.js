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

    // Ensure all file handles are closed before exiting
    // This is particularly important on Windows to prevent file locking issues
    process.stdin.destroy();
    process.stdout.destroy();
    process.stderr.destroy();

    // Force exit after a short delay to ensure cleanup
    setTimeout(() => {
      process.exit(0);
    }, 100);
  });
});

server.listen(port, hostname, () => {
  console.log(`Server running at http://${hostname}:${port}/`);
});

// Handle process termination signals to ensure clean shutdown
process.on('SIGTERM', () => {
  console.log('Received SIGTERM, shutting down gracefully...');
  server.close(() => {
    process.exit(0);
  });
});

process.on('SIGINT', () => {
  console.log('Received SIGINT, shutting down gracefully...');
  server.close(() => {
    process.exit(0);
  });
});
