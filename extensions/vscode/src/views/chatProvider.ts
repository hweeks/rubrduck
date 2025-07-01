import * as vscode from "vscode";
import { RubrDuckAPI, Message, StreamChunk } from "../api/client";

export class ChatProvider implements vscode.WebviewViewProvider {
  public static readonly viewType = "rubrduck.chat";
  private _view?: vscode.WebviewView;
  private messages: Message[] = [];

  constructor(
    private readonly _extensionUri: vscode.Uri,
    private readonly api: RubrDuckAPI
  ) {}

  public resolveWebviewView(
    webviewView: vscode.WebviewView,
    context: vscode.WebviewViewResolveContext,
    _token: vscode.CancellationToken
  ) {
    this._view = webviewView;

    webviewView.webview.options = {
      enableScripts: true,
      localResourceRoots: [this._extensionUri],
    };

    webviewView.webview.html = this._getHtmlForWebview(webviewView.webview);

    // Handle messages from the webview
    webviewView.webview.onDidReceiveMessage(
      (message) => {
        switch (message.type) {
          case "sendMessage":
            this.handleSendMessage(message.content);
            break;
          case "clearChat":
            this.handleClearChat();
            break;
          case "insertCode":
            this.handleInsertCode(message.content);
            break;
        }
      },
      undefined,
      []
    );
  }

  private async handleSendMessage(content: string) {
    if (!content.trim()) {
      return;
    }

    // Add user message to conversation
    const userMessage: Message = { role: "user", content: content.trim() };
    this.messages.push(userMessage);

    // Update webview with user message
    this._view?.webview.postMessage({
      type: "addMessage",
      message: userMessage,
    });

    // Show typing indicator
    this._view?.webview.postMessage({
      type: "showTyping",
    });

    try {
      // Get current editor context if available
      const editor = vscode.window.activeTextEditor;
      const contextMessages = [...this.messages];

      if (editor && editor.selection && !editor.selection.isEmpty) {
        const selectedText = editor.document.getText(editor.selection);
        const language = editor.document.languageId;
        const fileName = editor.document.fileName;

        // Add context about selected code
        const contextMessage: Message = {
          role: "system",
          content: `The user has selected the following ${language} code from ${fileName}:\n\n\`\`\`${language}\n${selectedText}\n\`\`\`\n\nPlease consider this context when responding.`,
        };
        contextMessages.unshift(contextMessage);
      }

      // Send chat request (using regular API instead of streaming for now)
      const response = await this.api.sendChat({
        messages: contextMessages,
      });

      // Hide typing indicator
      this._view?.webview.postMessage({
        type: "hideTyping",
      });

      if (response.message.content) {
        const responseMessage: Message = {
          role: "assistant",
          content: response.message.content,
        };
        this.messages.push(responseMessage);

        this._view?.webview.postMessage({
          type: "addMessage",
          message: responseMessage,
        });
      }
    } catch (error) {
      this._view?.webview.postMessage({
        type: "hideTyping",
      });

      this._view?.webview.postMessage({
        type: "showError",
        message: `Error: ${error}`,
      });
    }
  }

  private handleClearChat() {
    this.messages = [];
    this._view?.webview.postMessage({
      type: "clearMessages",
    });
  }

  private async handleInsertCode(code: string) {
    const editor = vscode.window.activeTextEditor;
    if (!editor) {
      vscode.window.showWarningMessage("No active editor found");
      return;
    }

    const position = editor.selection.active;
    await editor.edit((editBuilder) => {
      editBuilder.insert(position, code);
    });

    vscode.window.showInformationMessage("Code inserted successfully");
  }

  public addContextFromEditor() {
    const editor = vscode.window.activeTextEditor;
    if (!editor) {
      return;
    }

    const selection = editor.selection;
    const selectedText = editor.document.getText(selection);

    if (selectedText.trim()) {
      const language = editor.document.languageId;
      const message = `Here's the selected ${language} code:\n\n\`\`\`${language}\n${selectedText}\n\`\`\``;

      this._view?.webview.postMessage({
        type: "addContext",
        content: message,
      });
    }
  }

  private _getHtmlForWebview(webview: vscode.Webview) {
    return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>RubrDuck Chat</title>
    <style>
        body {
            font-family: var(--vscode-font-family);
            font-size: var(--vscode-font-size);
            color: var(--vscode-foreground);
            background-color: var(--vscode-editor-background);
            margin: 0;
            padding: 10px;
            height: 100vh;
            display: flex;
            flex-direction: column;
        }
        
        .chat-container {
            flex: 1;
            overflow-y: auto;
            margin-bottom: 10px;
            padding: 5px;
        }
        
        .message {
            margin-bottom: 15px;
            padding: 10px;
            border-radius: 8px;
            max-width: 100%;
            word-wrap: break-word;
        }
        
        .user-message {
            background-color: var(--vscode-input-background);
            border: 1px solid var(--vscode-input-border);
            margin-left: 20px;
        }
        
        .assistant-message {
            background-color: var(--vscode-editor-background);
            border: 1px solid var(--vscode-panel-border);
            margin-right: 20px;
        }
        
        .message-header {
            font-weight: bold;
            margin-bottom: 5px;
            font-size: 0.9em;
            opacity: 0.8;
        }
        
        .message-content {
            line-height: 1.4;
        }
        
        .message-content pre {
            background-color: var(--vscode-textCodeBlock-background);
            padding: 10px;
            border-radius: 4px;
            overflow-x: auto;
            margin: 10px 0;
        }
        
        .message-content code {
            background-color: var(--vscode-textCodeBlock-background);
            padding: 2px 4px;
            border-radius: 3px;
            font-family: var(--vscode-editor-font-family);
        }
        
        .input-container {
            display: flex;
            flex-direction: column;
            gap: 10px;
        }
        
        .input-row {
            display: flex;
            gap: 5px;
        }
        
        #messageInput {
            flex: 1;
            padding: 10px;
            background-color: var(--vscode-input-background);
            color: var(--vscode-input-foreground);
            border: 1px solid var(--vscode-input-border);
            border-radius: 4px;
            resize: vertical;
            min-height: 60px;
            font-family: var(--vscode-font-family);
        }
        
        button {
            padding: 8px 16px;
            background-color: var(--vscode-button-background);
            color: var(--vscode-button-foreground);
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-family: var(--vscode-font-family);
        }
        
        button:hover {
            background-color: var(--vscode-button-hoverBackground);
        }
        
        button:disabled {
            opacity: 0.5;
            cursor: not-allowed;
        }
        
        .button-row {
            display: flex;
            gap: 5px;
            justify-content: space-between;
        }
        
        .typing-indicator {
            display: none;
            padding: 10px;
            color: var(--vscode-descriptionForeground);
            font-style: italic;
        }
        
        .error-message {
            background-color: var(--vscode-inputValidation-errorBackground);
            color: var(--vscode-inputValidation-errorForeground);
            border: 1px solid var(--vscode-inputValidation-errorBorder);
            padding: 10px;
            border-radius: 4px;
            margin: 10px 0;
        }
        
        .code-block {
            position: relative;
        }
        
        .code-block button {
            position: absolute;
            top: 5px;
            right: 5px;
            padding: 4px 8px;
            font-size: 0.8em;
        }
    </style>
</head>
<body>
    <div class="chat-container" id="chatContainer">
        <div class="message assistant-message">
            <div class="message-header">RubrDuck</div>
            <div class="message-content">Hello! I'm RubrDuck, your AI coding assistant. How can I help you today?</div>
        </div>
    </div>
    
    <div class="typing-indicator" id="typingIndicator">
        RubrDuck is typing...
    </div>
    
    <div class="input-container">
        <div class="input-row">
            <textarea id="messageInput" placeholder="Ask me anything about your code..."></textarea>
        </div>
        <div class="button-row">
            <div>
                <button id="sendButton">Send</button>
                <button id="clearButton">Clear</button>
            </div>
            <button id="addContextButton">Add Selection</button>
        </div>
    </div>

    <script>
        const vscode = acquireVsCodeApi();
        const chatContainer = document.getElementById('chatContainer');
        const messageInput = document.getElementById('messageInput');
        const sendButton = document.getElementById('sendButton');
        const clearButton = document.getElementById('clearButton');
        const addContextButton = document.getElementById('addContextButton');
        const typingIndicator = document.getElementById('typingIndicator');
        let currentResponseElement = null;

        // Handle send button click
        sendButton.addEventListener('click', sendMessage);
        clearButton.addEventListener('click', clearChat);
        addContextButton.addEventListener('click', addContext);

        // Handle Enter key (Shift+Enter for new line)
        messageInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                sendMessage();
            }
        });

        function sendMessage() {
            const content = messageInput.value.trim();
            if (!content) return;

            vscode.postMessage({
                type: 'sendMessage',
                content: content
            });

            messageInput.value = '';
        }

        function clearChat() {
            vscode.postMessage({
                type: 'clearChat'
            });
        }

        function addContext() {
            vscode.postMessage({
                type: 'addContext'
            });
        }

        function insertCode(code) {
            vscode.postMessage({
                type: 'insertCode',
                content: code
            });
        }

        function addMessage(message) {
            const messageDiv = document.createElement('div');
            messageDiv.className = \`message \${message.role}-message\`;
            
            const headerDiv = document.createElement('div');
            headerDiv.className = 'message-header';
            headerDiv.textContent = message.role === 'user' ? 'You' : 'RubrDuck';
            
            const contentDiv = document.createElement('div');
            contentDiv.className = 'message-content';
            contentDiv.innerHTML = formatMessage(message.content);
            
            messageDiv.appendChild(headerDiv);
            messageDiv.appendChild(contentDiv);
            chatContainer.appendChild(messageDiv);
            
            // Add insert buttons to code blocks
            addInsertButtons(messageDiv);
            
            scrollToBottom();
        }

        function formatMessage(content) {
            // Convert markdown-style code blocks to HTML
            content = content.replace(/\`\`\`(\w+)?\n?([\s\S]*?)\`\`\`/g, (match, lang, code) => {
                return \`<div class="code-block"><pre><code>\${escapeHtml(code.trim())}</code></pre><button onclick="insertCode('\${escapeHtml(code.trim()).replace(/'/g, "\\\\'")}')">Insert</button></div>\`;
            });
            
            // Convert inline code
            content = content.replace(/\`([^\`]+)\`/g, '<code>$1</code>');
            
            // Convert line breaks
            content = content.replace(/\n/g, '<br>');
            
            return content;
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        function addInsertButtons(messageElement) {
            const codeBlocks = messageElement.querySelectorAll('.code-block button');
            codeBlocks.forEach(button => {
                button.addEventListener('click', (e) => {
                    e.preventDefault();
                    const code = e.target.getAttribute('onclick').match(/insertCode\('(.*)'\)/)[1];
                    insertCode(code.replace(/\\\\'/g, "'"));
                });
            });
        }

        function scrollToBottom() {
            chatContainer.scrollTop = chatContainer.scrollHeight;
        }

        // Handle messages from extension
        window.addEventListener('message', event => {
            const message = event.data;
            
            switch (message.type) {
                case 'addMessage':
                    addMessage(message.message);
                    break;
                case 'showTyping':
                    typingIndicator.style.display = 'block';
                    currentResponseElement = null;
                    scrollToBottom();
                    break;
                case 'updateResponse':
                    if (!currentResponseElement) {
                        currentResponseElement = document.createElement('div');
                        currentResponseElement.className = 'message assistant-message';
                        currentResponseElement.innerHTML = \`
                            <div class="message-header">RubrDuck</div>
                            <div class="message-content"></div>
                        \`;
                        chatContainer.appendChild(currentResponseElement);
                    }
                    
                    const contentDiv = currentResponseElement.querySelector('.message-content');
                    contentDiv.innerHTML = formatMessage(message.content);
                    addInsertButtons(currentResponseElement);
                    scrollToBottom();
                    break;
                case 'hideTyping':
                    typingIndicator.style.display = 'none';
                    break;
                case 'finalizeMessage':
                    currentResponseElement = null;
                    break;
                case 'clearMessages':
                    chatContainer.innerHTML = \`
                        <div class="message assistant-message">
                            <div class="message-header">RubrDuck</div>
                            <div class="message-content">Hello! I'm RubrDuck, your AI coding assistant. How can I help you today?</div>
                        </div>
                    \`;
                    break;
                case 'showError':
                    const errorDiv = document.createElement('div');
                    errorDiv.className = 'error-message';
                    errorDiv.textContent = message.message;
                    chatContainer.appendChild(errorDiv);
                    scrollToBottom();
                    break;
                case 'addContext':
                    messageInput.value += messageInput.value ? '\n\n' + message.content : message.content;
                    messageInput.focus();
                    break;
            }
        });
    </script>
</body>
</html>`;
  }
}
