import * as vscode from "vscode";
import { RubrDuckAPI } from "./api/client";
import { ChatProvider } from "./views/chatProvider";
import {
  HistoryProvider,
  registerHistoryCommands,
} from "./views/historyProvider";
import {
  handleExplainCode,
  handleFixCode,
  handleGenerateCode,
  handleGenerateTests,
} from "./commands/codeActions";
import {
  handleFixFile,
  handleCustomCommand,
} from "./commands/codeActions";
import { RubrDuckCodeLensProvider } from "./codelensProvider";

let api: RubrDuckAPI;
let chatProvider: ChatProvider;
let historyProvider: HistoryProvider;

export function activate(context: vscode.ExtensionContext) {
  console.log("RubrDuck extension is being activated");

  // Initialize API client
  const config = vscode.workspace.getConfiguration("rubrduck");
  const serverUrl = config.get<string>("serverUrl", "http://localhost:8080");
  const authToken = config.get<string>("authToken", "");

  api = new RubrDuckAPI(serverUrl, authToken);

  // Initialize providers
  chatProvider = new ChatProvider(context.extensionUri, api);
  historyProvider = new HistoryProvider(api);

  // Register webview providers
  const chatWebviewProvider = vscode.window.registerWebviewViewProvider(
    "rubrduck.chat",
    chatProvider
  );
  const historyTreeDataProvider = vscode.window.registerTreeDataProvider(
    "rubrduck.history",
    historyProvider
  );

  // Register CodeLens provider if enabled
  const enableCodeLens = config.get<boolean>("enableCodeLens", true);
  let codeLensDisposable: vscode.Disposable | undefined;
  if (enableCodeLens) {
    codeLensDisposable = vscode.languages.registerCodeLensProvider(
      { scheme: "file" },
      new RubrDuckCodeLensProvider()
    );
  }

  // Register commands
  const openChatCommand = vscode.commands.registerCommand(
    "rubrduck.chat",
    () => {
      vscode.commands.executeCommand("rubrduck.chat.focus");
    }
  );

  const explainCommand = vscode.commands.registerCommand(
    "rubrduck.explain",
    (range?: vscode.Range) => {
      handleExplainCode(api, range);
    }
  );

  const fixCommand = vscode.commands.registerCommand("rubrduck.fix", () => {
    handleFixCode(api);
  });

  const fixFileCommand = vscode.commands.registerCommand(
    "rubrduck.fixFile",
    (uri?: vscode.Uri | vscode.Uri[]) => {
      handleFixFile(api, uri);
    }
  );

  const customCommand = vscode.commands.registerCommand(
    "rubrduck.custom",
    async () => {
      const cfg = vscode.workspace.getConfiguration("rubrduck");
      const commands = cfg.get<any[]>("customCommands", []);
      if (commands.length === 0) {
        vscode.window.showInformationMessage("No custom commands configured");
        return;
      }
      const pick = await vscode.window.showQuickPick(
        commands.map((c) => c.name),
        { placeHolder: "Select a custom command" }
      );
      const item = commands.find((c) => c.name === pick);
      if (item) {
        handleCustomCommand(api, item.prompt);
      }
    }
  );

  const generateCommand = vscode.commands.registerCommand(
    "rubrduck.generate",
    () => {
      handleGenerateCode(api);
    }
  );

  const testCommand = vscode.commands.registerCommand("rubrduck.test", () => {
    handleGenerateTests(api);
  });


  // Register history commands
  registerHistoryCommands(context, api, historyProvider);

  // Auto-start server if enabled
  const autoStart = config.get<boolean>("autoStart", true);
  if (autoStart) {
    checkServerConnection();
  }

  // Register configuration change listener
  const configChangeListener = vscode.workspace.onDidChangeConfiguration(
    (event) => {
      if (event.affectsConfiguration("rubrduck")) {
        const newConfig = vscode.workspace.getConfiguration("rubrduck");
        const newServerUrl = newConfig.get<string>(
          "serverUrl",
          "http://localhost:8080"
        );
        const newAuthToken = newConfig.get<string>("authToken", "");

        const enableLens = newConfig.get<boolean>("enableCodeLens", true);
        if (enableLens && !codeLensDisposable) {
          codeLensDisposable = vscode.languages.registerCodeLensProvider(
            { scheme: "file" },
            new RubrDuckCodeLensProvider()
          );
          context.subscriptions.push(codeLensDisposable);
        } else if (!enableLens && codeLensDisposable) {
          codeLensDisposable.dispose();
          codeLensDisposable = undefined;
        }

        api.updateConfig(newServerUrl, newAuthToken);
      }
    }
  );

  // Add all disposables to context
  context.subscriptions.push(
    chatWebviewProvider,
    historyTreeDataProvider,
    openChatCommand,
    explainCommand,
    fixCommand,
    fixFileCommand,
    customCommand,
    generateCommand,
    testCommand,
    configChangeListener,
    ...(codeLensDisposable ? [codeLensDisposable] : [])
  );

  console.log("RubrDuck extension activated successfully");
}

export function deactivate() {
  console.log("RubrDuck extension is being deactivated");
  if (api) {
    api.dispose();
  }
}

async function checkServerConnection() {
  try {
    await api.healthCheck();
    vscode.window.showInformationMessage("Connected to RubrDuck server");
  } catch (error) {
    vscode.window
      .showWarningMessage(
        "Could not connect to RubrDuck server. Please check your configuration.",
        "Open Settings"
      )
      .then((selection) => {
        if (selection === "Open Settings") {
          vscode.commands.executeCommand(
            "workbench.action.openSettings",
            "rubrduck"
          );
        }
      });
  }
}
