import { App as BaseApp, type AppParams } from "../../generated/apps/index.js";
import path from "path";
import { stringify } from "yaml";
import fs from "fs";
import { Bundle } from "../core/bundle.js";
import { Workspace } from "../core/workspace.js";

interface AppExtraParams {
  env?: { [key: string]: string };
  command?: string[];
}

export class App extends BaseApp {
  constructor(name: string, bundle: Bundle, params: AppParams & AppExtraParams) {
    const appYmlConfig: { env?: Array<{ name: string; value: string }>; command?: string[] } = {};

    const env : Record<string, string> = params.env || {};
    if (bundle.mode) {
        env["DEPLOY_MODE"] = bundle.mode;
    }

    const command = params.command || [];
    appYmlConfig.env = Object.entries(env).map(([name, value]) => ({
      name,
      value: value as string,
    }));
    appYmlConfig.command = command;

    if (typeof params.source_code_path === "string") {
      fs.writeFileSync(path.join(params.source_code_path, "app.yml"), stringify(appYmlConfig));
    }

    delete params.env;
    delete params.command;

    if (bundle.mode === "development") {
      params.name = `dev-${Workspace.currentUser.domainFriendlyName}-${params.name}`;
    }
    super(name, params);
  }
}
