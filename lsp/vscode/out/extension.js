"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.activate = activate;
exports.deactivate = deactivate;
const vscode = require("vscode");
const node_1 = require("vscode-languageclient/node");
let client;
function activate(context) {
    const config = vscode.workspace.getConfiguration("origami");
    const binaryPath = config.get("lsp.path", "origami");
    const serverOptions = {
        run: {
            command: binaryPath,
            args: ["lsp"],
            transport: node_1.TransportKind.stdio,
        },
        debug: {
            command: binaryPath,
            args: ["lsp", "--verbose"],
            transport: node_1.TransportKind.stdio,
        },
    };
    const clientOptions = {
        documentSelector: [
            { scheme: "file", language: "origami-circuit" },
            { scheme: "file", language: "yaml", pattern: "**/circuits/**/*.yaml" },
            { scheme: "file", language: "yaml", pattern: "**/circuits/**/*.yml" },
        ],
        synchronize: {
            configurationSection: "origami",
        },
    };
    client = new node_1.LanguageClient("origami-lsp", "Origami Circuit LSP", serverOptions, clientOptions);
    client.start();
    context.subscriptions.push({
        dispose: () => {
            if (client) {
                client.stop();
            }
        },
    });
    const statusBar = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
    statusBar.text = "$(symbol-misc) Origami LSP";
    statusBar.tooltip = "Origami Circuit Language Server";
    statusBar.show();
    context.subscriptions.push(statusBar);
}
function deactivate() {
    if (!client) {
        return undefined;
    }
    return client.stop();
}
//# sourceMappingURL=extension.js.map