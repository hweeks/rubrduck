import * as vscode from "vscode";
import { RubrDuckAPI, Message } from "../api/client";

export async function handleExplainCode(
  api: RubrDuckAPI,
  range?: vscode.Range
): Promise<void> {
  const editor = vscode.window.activeTextEditor;
  if (!editor) {
    vscode.window.showWarningMessage("No active editor found");
    return;
  }

  const selection = range
    ? new vscode.Selection(range.start, range.end)
    : editor.selection;
  const selectedText = editor.document.getText(selection);

  if (!selectedText.trim()) {
    vscode.window.showWarningMessage("Please select some code to explain");
    return;
  }

  const language = editor.document.languageId;
  const fileName = editor.document.fileName;

  try {
    const messages: Message[] = [
      {
        role: "system",
        content:
          "You are a helpful coding assistant. Explain the given code clearly and concisely.",
      },
      {
        role: "user",
        content: `Please explain this ${language} code from ${fileName}:\n\n\`\`\`${language}\n${selectedText}\n\`\`\``,
      },
    ];

    const response = await api.sendChat({ messages });

    // Show explanation in a new document
    const doc = await vscode.workspace.openTextDocument({
      content: `# Code Explanation\n\n## Selected Code:\n\`\`\`${language}\n${selectedText}\n\`\`\`\n\n## Explanation:\n${response.message.content}`,
      language: "markdown",
    });

    await vscode.window.showTextDocument(doc, {
      viewColumn: vscode.ViewColumn.Beside,
    });
  } catch (error) {
    vscode.window.showErrorMessage(`Failed to explain code: ${error}`);
  }
}

export async function handleFixCode(
  api: RubrDuckAPI,
  range?: vscode.Range
): Promise<void> {
  const editor = vscode.window.activeTextEditor;
  if (!editor) {
    vscode.window.showWarningMessage("No active editor found");
    return;
  }

  const selection = range
    ? new vscode.Selection(range.start, range.end)
    : editor.selection;
  const selectedText = editor.document.getText(selection);

  if (!selectedText.trim()) {
    vscode.window.showWarningMessage("Please select some code to fix");
    return;
  }

  const language = editor.document.languageId;
  const fileName = editor.document.fileName;

  try {
    const messages: Message[] = [
      {
        role: "system",
        content:
          "You are a helpful coding assistant. Fix any issues in the given code and provide the corrected version.",
      },
      {
        role: "user",
        content: `Please fix any issues in this ${language} code from ${fileName} and provide the corrected version:\n\n\`\`\`${language}\n${selectedText}\n\`\`\``,
      },
    ];

    const response = await api.sendChat({ messages });

    // Extract code from response (look for code blocks)
    const codeBlockRegex = new RegExp(
      `\`\`\`${language}\\s*\\n([\\s\\S]*?)\\n\`\`\``,
      "i"
    );
    const match = response.message.content.match(codeBlockRegex);

    if (match && match[1]) {
      const fixedCode = match[1].trim();

      // Ask user if they want to apply the fix
      const action = await vscode.window.showInformationMessage(
        "Code fix generated. Would you like to apply it?",
        "Apply Fix",
        "Show Diff",
        "Cancel"
      );

      if (action === "Apply Fix") {
        await editor.edit((editBuilder) => {
          editBuilder.replace(selection, fixedCode);
        });
        vscode.window.showInformationMessage("Code fix applied successfully");
      } else if (action === "Show Diff") {
        // Show the original and fixed code side by side
        const originalDoc = await vscode.workspace.openTextDocument({
          content: selectedText,
          language: language,
        });
        const fixedDoc = await vscode.workspace.openTextDocument({
          content: fixedCode,
          language: language,
        });

        await vscode.commands.executeCommand(
          "vscode.diff",
          originalDoc.uri,
          fixedDoc.uri,
          "Original ↔ Fixed"
        );
      }
    } else {
      // Show full response if no code block found
      const doc = await vscode.workspace.openTextDocument({
        content: `# Code Fix Suggestion\n\n## Original Code:\n\`\`\`${language}\n${selectedText}\n\`\`\`\n\n## Suggestion:\n${response.message.content}`,
        language: "markdown",
      });

      await vscode.window.showTextDocument(doc, {
        viewColumn: vscode.ViewColumn.Beside,
      });
    }
  } catch (error) {
    vscode.window.showErrorMessage(`Failed to fix code: ${error}`);
  }
}

export async function handleGenerateCode(api: RubrDuckAPI): Promise<void> {
  const editor = vscode.window.activeTextEditor;
  if (!editor) {
    vscode.window.showWarningMessage("No active editor found");
    return;
  }

  const language = editor.document.languageId;

  // Get user input for what to generate
  const prompt = await vscode.window.showInputBox({
    prompt: "What code would you like to generate?",
    placeHolder: 'e.g., "a function to sort an array of objects by name"',
  });

  if (!prompt) {
    return;
  }

  try {
    const messages: Message[] = [
      {
        role: "system",
        content: `You are a helpful coding assistant. Generate ${language} code based on the user's request. Provide clean, well-commented code.`,
      },
      {
        role: "user",
        content: `Generate ${language} code for: ${prompt}`,
      },
    ];

    const response = await api.sendChat({ messages });

    // Extract code from response
    const codeBlockRegex = new RegExp(
      `\`\`\`${language}\\s*\\n([\\s\\S]*?)\\n\`\`\``,
      "i"
    );
    const match = response.message.content.match(codeBlockRegex);

    if (match && match[1]) {
      const generatedCode = match[1].trim();

      // Insert at cursor position
      const position = editor.selection.active;
      await editor.edit((editBuilder) => {
        editBuilder.insert(position, generatedCode);
      });

      vscode.window.showInformationMessage(
        "Code generated and inserted successfully"
      );
    } else {
      // Show full response if no clear code block
      const doc = await vscode.workspace.openTextDocument({
        content: `# Generated Code\n\n**Request:** ${prompt}\n\n**Response:**\n${response.message.content}`,
        language: "markdown",
      });

      await vscode.window.showTextDocument(doc, {
        viewColumn: vscode.ViewColumn.Beside,
      });
    }
  } catch (error) {
    vscode.window.showErrorMessage(`Failed to generate code: ${error}`);
  }
}

export async function handleGenerateTests(api: RubrDuckAPI): Promise<void> {
  const editor = vscode.window.activeTextEditor;
  if (!editor) {
    vscode.window.showWarningMessage("No active editor found");
    return;
  }

  const selection = editor.selection;
  const selectedText = editor.document.getText(selection);

  if (!selectedText.trim()) {
    vscode.window.showWarningMessage(
      "Please select some code to generate tests for"
    );
    return;
  }

  const language = editor.document.languageId;
  const fileName = editor.document.fileName;

  try {
    const messages: Message[] = [
      {
        role: "system",
        content:
          "You are a helpful coding assistant. Generate comprehensive unit tests for the given code using appropriate testing frameworks.",
      },
      {
        role: "user",
        content: `Generate unit tests for this ${language} code from ${fileName}:\n\n\`\`\`${language}\n${selectedText}\n\`\`\``,
      },
    ];

    const response = await api.sendChat({ messages });

    // Create a new test file
    const testFileName = getTestFileName(fileName, language);
    const doc = await vscode.workspace.openTextDocument({
      content: response.message.content,
      language: language,
    });

    await vscode.window.showTextDocument(doc, {
      viewColumn: vscode.ViewColumn.Beside,
    });

    // Suggest saving with test file name
    vscode.window
      .showInformationMessage(
        `Tests generated! Consider saving as: ${testFileName}`,
        "Save As"
      )
      .then((action) => {
        if (action === "Save As") {
          vscode.commands.executeCommand("workbench.action.files.saveAs");
        }
      });
  } catch (error) {
    vscode.window.showErrorMessage(`Failed to generate tests: ${error}`);
  }
}

function getTestFileName(originalFileName: string, language: string): string {
  const pathParts = originalFileName.split("/");
  const fileName = pathParts[pathParts.length - 1];
  const nameWithoutExt = fileName.substring(0, fileName.lastIndexOf("."));
  const ext = fileName.substring(fileName.lastIndexOf("."));

  switch (language) {
    case "javascript":
    case "typescript":
      return `${nameWithoutExt}.test${ext}`;
    case "python":
      return `test_${nameWithoutExt}.py`;
    case "go":
      return `${nameWithoutExt}_test.go`;
    case "java":
      return `${nameWithoutExt}Test.java`;
    case "csharp":
      return `${nameWithoutExt}Tests.cs`;
    default:
      return `${nameWithoutExt}_test${ext}`;
  }
}

export async function handleFixFile(
  api: RubrDuckAPI,
  uris?: vscode.Uri | vscode.Uri[]
): Promise<void> {
  const targets: vscode.Uri[] = [];
  if (!uris) {
    const editor = vscode.window.activeTextEditor;
    if (editor) {
      targets.push(editor.document.uri);
    }
  } else if (Array.isArray(uris)) {
    targets.push(...uris);
  } else {
    targets.push(uris);
  }

  if (targets.length === 0) {
    vscode.window.showWarningMessage("No file selected");
    return;
  }

  for (const uri of targets) {
    const doc = await vscode.workspace.openTextDocument(uri);
    const text = doc.getText();
    const language = doc.languageId;
    const fileName = doc.fileName;

    const messages: Message[] = [
      {
        role: "system",
        content:
          "You are a helpful coding assistant. Fix any issues in the given code and provide the corrected version.",
      },
      {
        role: "user",
        content: `Please fix any issues in this ${language} code from ${fileName} and provide the corrected version:\n\n\`\`\`${language}\n${text}\n\`\`\``,
      },
    ];

    try {
      const response = await api.sendChat({ messages });
      const regex = new RegExp(`\`\`\`${language}\\s*\\n([\\s\\S]*?)\\n\`\`\``, "i");
      const match = response.message.content.match(regex);
      const fixedCode = match && match[1] ? match[1].trim() : response.message.content;

      const action = await vscode.window.showInformationMessage(
        `Apply fix to ${fileName}?`,
        "Apply",
        "Show Diff",
        "Skip"
      );

      if (action === "Apply") {
        const edit = new vscode.WorkspaceEdit();
        const fullRange = new vscode.Range(
          doc.positionAt(0),
          doc.positionAt(text.length)
        );
        edit.replace(uri, fullRange, fixedCode);
        await vscode.workspace.applyEdit(edit);
        await doc.save();
      } else if (action === "Show Diff") {
        const originalDoc = await vscode.workspace.openTextDocument({
          content: text,
          language,
        });
        const fixedDoc = await vscode.workspace.openTextDocument({
          content: fixedCode,
          language,
        });
        await vscode.commands.executeCommand(
          "vscode.diff",
          originalDoc.uri,
          fixedDoc.uri,
          `${fileName} ↔ Fixed`
        );
      }
    } catch (error) {
      vscode.window.showErrorMessage(`Failed to fix ${fileName}: ${error}`);
    }
  }
}

export async function handleCustomCommand(
  api: RubrDuckAPI,
  prompt: string
): Promise<void> {
  const editor = vscode.window.activeTextEditor;
  if (!editor) {
    vscode.window.showWarningMessage("No active editor found");
    return;
  }

  const selection = editor.selection;
  const selectedText = editor.document.getText(selection);

  if (!selectedText.trim()) {
    vscode.window.showWarningMessage("Please select some code to use with the custom command");
    return;
  }

  const language = editor.document.languageId;
  const fileName = editor.document.fileName;

  const messages: Message[] = [
    { role: "system", content: `You are a helpful coding assistant. ${prompt}` },
    {
      role: "user",
      content: `${prompt} for this ${language} code from ${fileName}:\n\n\`\`\`${language}\n${selectedText}\n\`\`\``,
    },
  ];

  try {
    const response = await api.sendChat({ messages });
    const doc = await vscode.workspace.openTextDocument({
      content: response.message.content,
      language: "markdown",
    });
    await vscode.window.showTextDocument(doc, { viewColumn: vscode.ViewColumn.Beside });
  } catch (error) {
    vscode.window.showErrorMessage(`Failed to run custom command: ${error}`);
  }
}
