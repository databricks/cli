import { Variable } from "./variable.js";

export class Workspace {
    static readonly host = new Variable<string>("workspace.host");
    static readonly currentUser = {
        domainFriendlyName: new Variable<string>("workspace.current_user.domain_friendly_name"),
        userName: new Variable<string>("workspace.current_user.user_name"),
        shortName: new Variable<string>("workspace.current_user.short_name"),
    };
    static readonly filePath = new Variable<string>("workspace.file_path");
    static readonly rootPath = new Variable<string>("workspace.root_path");
}

