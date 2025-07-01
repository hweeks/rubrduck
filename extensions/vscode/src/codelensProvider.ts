import * as vscode from "vscode";

export class RubrDuckCodeLensProvider implements vscode.CodeLensProvider {
  private regex = /^\s*(function|def|func|class|public|private|protected|async\s+function)\b/;

  provideCodeLenses(document: vscode.TextDocument): vscode.CodeLens[] {
    const lenses: vscode.CodeLens[] = [];
    for (let i = 0; i < document.lineCount; i++) {
      const line = document.lineAt(i);
      if (this.regex.test(line.text)) {
        const range = new vscode.Range(i, 0, i, 0);
        lenses.push(
          new vscode.CodeLens(range, {
            title: "RubrDuck: Explain",
            command: "rubrduck.explain",
            arguments: [range],
          }),
          new vscode.CodeLens(range, {
            title: "RubrDuck: Fix",
            command: "rubrduck.fix",
            arguments: [range],
          })
        );
      }
    }
    return lenses;
  }
}
