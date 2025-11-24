#!/usr/bin/env node
const path = require("node:path");
const fs = require("node:fs");

async function startViteServer() {
  const vitePath = safeViteResolve();

  if (!vitePath) {
    console.log(
      "\n❌ Vite needs to be installed in the current directory. Run `npm install vite`.\n"
    );
    process.exit(1);
  }

  const { createServer, loadConfigFromFile, mergeConfig } = require(vitePath);

  /**
   * This script is controlled by us, and shouldn't be called directly by the user.
   * We know the order of the arguments is always:
   * 1. appUrl
   * 2. port
   *
   * We can safely access the arguments by index.
   */
  const clientPath = path.join(process.cwd(), "client");
  const appUrl = process.argv[2] || "";
  const port = parseInt(process.argv[3] || 5173);

  if (!fs.existsSync(clientPath)) {
    console.error("client folder doesn't exist.");
    process.exit(1);
  }

  if (!appUrl) {
    console.error("App URL is required");
    process.exit(1);
  }

  try {
    const domain = new URL(appUrl);

    const loadedConfig = await loadConfigFromFile(
      {
        mode: "development",
        command: "serve",
      },
      undefined,
      clientPath
    );
    const userConfig = loadedConfig?.config ?? {};
    const coreConfig = {
      configFile: false,
      root: clientPath,
      server: {
        open: `${domain.origin}?dev=true`,
        port: port,
        hmr: {
          overlay: true,
          path: `/dev-hmr`,
        },
        middlewareMode: false,
      },
      plugins: [queriesHMRPlugin()],
    };
    const mergedConfigs = mergeConfig(userConfig, coreConfig);
    const server = await createServer(mergedConfigs);

    await server.listen();

    console.log(`\n✅ Vite dev server started successfully!`);
    console.log(`\nPress Ctrl+C to stop the server\n`);

    const shutdown = async () => {
      await server.close();
      process.exit(0);
    };

    process.on("SIGINT", shutdown);
    process.on("SIGTERM", shutdown);
  } catch (error) {
    console.error(`❌ Failed to start Vite server:`, error.message);
    if (error.stack) {
      console.error(error.stack);
    }
    process.exit(1);
  }
}

function safeViteResolve() {
  try {
    const vitePath = require.resolve("vite", { paths: [process.cwd()] });

    return vitePath;
  } catch (error) {
    return null;
  }
}

// Start the server
startViteServer().catch((error) => {
  console.error("Fatal error:", error);
  process.exit(1);
});

/*
 * development only, watches for changes in the queries directory and sends HMR updates to the client.
 */
function queriesHMRPlugin(options = {}) {
  const { queriesPath = path.resolve(process.cwd(), "config/queries") } =
    options;
  let isServe = false;
  let serverRunning = false;

  return {
    name: "queries-hmr",
    async buildStart() {
      if (!isServe) return;
      if (serverRunning) {
        return;
      }
      serverRunning = true;
    },
    configResolved(config) {
      isServe = config.command === "serve";
    },
    configureServer(server) {
      if (!isServe) return;
      if (!server.config.mode || server.config.mode === "development") {
        // 1. check if queries directory exists
        if (fs.existsSync(queriesPath)) {
          // 2. add the queries directory to the watcher
          server.watcher.add(queriesPath);

          const handleFileChange = (file) => {
            if (file.includes("config/queries") && file.endsWith(".sql")) {
              const fileName = path.basename(file);
              const queryKey = fileName.replace(/\.(sql)$/, "");

              console.log("🔄 Query updated:", queryKey, fileName);

              server.ws.send({
                type: "custom",
                event: "query-update",
                data: {
                  key: queryKey,
                  timestamp: Date.now(),
                },
              });
            }
          };

          server.watcher.on("change", handleFileChange);
        }

        process.on("SIGINT", () => {
          console.log("🛑 SIGINT received — cleaning up before exit...");
          serverRunning = false;
          process.exit(0);
        });
      }
    },
  };
}
