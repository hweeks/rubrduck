import * as vscode from "vscode";
import { RubrDuckAPI, Conversation } from "../api/client";

export class HistoryProvider implements vscode.TreeDataProvider<HistoryItem> {
  private _onDidChangeTreeData: vscode.EventEmitter<
    HistoryItem | undefined | null | void
  > = new vscode.EventEmitter<HistoryItem | undefined | null | void>();
  readonly onDidChangeTreeData: vscode.Event<
    HistoryItem | undefined | null | void
  > = this._onDidChangeTreeData.event;

  private conversations: Conversation[] = [];

  constructor(private api: RubrDuckAPI) {
    this.refresh();
  }

  refresh(): void {
    this.loadHistory();
    this._onDidChangeTreeData.fire();
  }

  getTreeItem(element: HistoryItem): vscode.TreeItem {
    return element;
  }

  getChildren(element?: HistoryItem): Thenable<HistoryItem[]> {
    if (!element) {
      // Root level - return conversations
      return Promise.resolve(
        this.conversations.map((conv) => new ConversationItem(conv))
      );
    } else if (element instanceof ConversationItem) {
      // Conversation level - return messages
      return Promise.resolve(
        element.conversation.messages?.map(
          (msg, index) => new MessageItem(msg, index, element.conversation.id)
        ) || []
      );
    } else {
      return Promise.resolve([]);
    }
  }

  private async loadHistory() {
    try {
      const history = await this.api.getHistory(1, 50);
      this.conversations = history.conversations;
    } catch (error) {
      console.error("Failed to load conversation history:", error);
      this.conversations = [];
    }
  }
}

abstract class HistoryItem extends vscode.TreeItem {
  constructor(
    public readonly label: string,
    public readonly collapsibleState: vscode.TreeItemCollapsibleState
  ) {
    super(label, collapsibleState);
  }
}

class ConversationItem extends HistoryItem {
  constructor(public readonly conversation: Conversation) {
    super(
      conversation.title,
      conversation.messages && conversation.messages.length > 0
        ? vscode.TreeItemCollapsibleState.Collapsed
        : vscode.TreeItemCollapsibleState.None
    );

    this.tooltip = `Created: ${new Date(
      conversation.created
    ).toLocaleString()}`;
    this.description = `${conversation.messages?.length || 0} messages`;
    this.contextValue = "conversation";

    // Add command to open conversation
    this.command = {
      command: "rubrduck.openConversation",
      title: "Open Conversation",
      arguments: [conversation.id],
    };

    // Set icon
    this.iconPath = new vscode.ThemeIcon("comment-discussion");
  }
}

class MessageItem extends HistoryItem {
  constructor(
    public readonly message: { role: string; content: string },
    public readonly index: number,
    public readonly conversationId: string
  ) {
    const preview =
      message.content.substring(0, 50) +
      (message.content.length > 50 ? "..." : "");
    super(preview, vscode.TreeItemCollapsibleState.None);

    this.tooltip = message.content;
    this.description = message.role;
    this.contextValue = "message";

    // Set icon based on role
    this.iconPath = new vscode.ThemeIcon(
      message.role === "user" ? "person" : "robot"
    );

    // Add command to show message content
    this.command = {
      command: "rubrduck.showMessage",
      title: "Show Message",
      arguments: [message],
    };
  }
}

// Register additional commands for history management
export function registerHistoryCommands(
  context: vscode.ExtensionContext,
  api: RubrDuckAPI,
  historyProvider: HistoryProvider
) {
  const refreshCommand = vscode.commands.registerCommand(
    "rubrduck.refreshHistory",
    () => {
      historyProvider.refresh();
    }
  );

  const openConversationCommand = vscode.commands.registerCommand(
    "rubrduck.openConversation",
    async (conversationId: string) => {
      try {
        const conversation = await api.getConversation(conversationId);
        if (conversation) {
          // Create a document showing the conversation
          let content = `# Conversation: ${conversation.title}\n\n`;
          content += `**Created:** ${new Date(
            conversation.created
          ).toLocaleString()}\n`;
          content += `**Updated:** ${new Date(
            conversation.updated
          ).toLocaleString()}\n\n`;
          content += "---\n\n";

          if (conversation.messages) {
            for (const message of conversation.messages) {
              content += `## ${
                message.role === "user" ? "You" : "RubrDuck"
              }\n\n`;
              content += `${message.content}\n\n`;
              content += "---\n\n";
            }
          }

          const doc = await vscode.workspace.openTextDocument({
            content,
            language: "markdown",
          });

          await vscode.window.showTextDocument(doc);
        } else {
          vscode.window.showErrorMessage("Conversation not found");
        }
      } catch (error) {
        vscode.window.showErrorMessage(`Failed to open conversation: ${error}`);
      }
    }
  );

  const showMessageCommand = vscode.commands.registerCommand(
    "rubrduck.showMessage",
    async (message: { role: string; content: string }) => {
      const doc = await vscode.workspace.openTextDocument({
        content: `# Message from ${
          message.role === "user" ? "You" : "RubrDuck"
        }\n\n${message.content}`,
        language: "markdown",
      });

      await vscode.window.showTextDocument(doc, { preview: true });
    }
  );

  context.subscriptions.push(
    refreshCommand,
    openConversationCommand,
    showMessageCommand
  );
}
