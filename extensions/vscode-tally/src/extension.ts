import * as vscode from 'vscode';

import { findTallyBinary } from './binary/findBinary';
import { ConfigService } from './config/configService';
import { TallyLanguageClient } from './lsp/client';

let client: TallyLanguageClient | undefined;
let starting: Promise<void> | undefined;

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  const output = vscode.window.createOutputChannel('Tally');
  const configService = new ConfigService();
  context.subscriptions.push(output, configService);

  async function startOrRestart(reason: string): Promise<void> {
    if (starting) {
      await starting;
      return;
    }
    starting = (async () => {
      const cfg = configService.snapshot();
      if (!cfg.global.enable) {
        await client?.stop();
        client = undefined;
        return;
      }

      const binarySettings = configService.binarySettings();
      const resolved = await findTallyBinary({
        extensionContext: context,
        isTrusted: vscode.workspace.isTrusted,
        settings: binarySettings,
        workspaceFolders: vscode.workspace.workspaceFolders ?? [],
      });

      const settingsEnvelope = configService.lspSettings();

      if (client && client.serverKey() === resolved.key) {
        await client.sendConfiguration(settingsEnvelope);
        return;
      }

      await client?.stop();
      client = new TallyLanguageClient({
        output,
        server: resolved,
      });

      try {
        await client.start();
        await client.sendConfiguration(settingsEnvelope);
      } catch (err) {
        await client.stop();
        client = undefined;

        const msg = err instanceof Error ? err.message : String(err);
        output.appendLine(`[tally] failed to start language server (${reason}): ${msg}`);
        void vscode.window.showErrorMessage(`Tally: failed to start language server: ${msg}`);
      }
    })().finally(() => {
      starting = undefined;
    });
    await starting;
  }

  context.subscriptions.push(
    vscode.commands.registerCommand('tally.restartServer', async () => {
      await startOrRestart('manual restart');
    }),

    vscode.commands.registerCommand('tally.applyAllFixes', async () => {
      const editor = vscode.window.activeTextEditor;
      if (!editor || !client) {
        return;
      }
      const uri = editor.document.uri.toString();
      const edit = await client.executeApplyAllFixes(uri);
      if (edit) {
        await vscode.workspace.applyEdit(edit);
      }
    }),

    vscode.commands.registerCommand('tally.configureDefaultFormatterForDockerfile', async () => {
      const config = vscode.workspace.getConfiguration(undefined, { languageId: 'dockerfile' });
      await config.update(
        'editor.defaultFormatter',
        'wharflab.tally',
        vscode.ConfigurationTarget.Global,
        true,
      );
      await config.update(
        'editor.formatOnSave',
        true,
        vscode.ConfigurationTarget.Global,
        true,
      );
      void vscode.window.showInformationMessage(
        'Tally is now the default Dockerfile formatter with format-on-save enabled.',
      );
    }),
  );

  configService.onDidChange(async (change) => {
    if (change.requiresRestart) {
      await startOrRestart('settings change');
      return;
    }
    await client?.sendConfiguration(configService.lspSettings());
  });

  context.subscriptions.push(
    vscode.workspace.onDidGrantWorkspaceTrust(() => {
      void startOrRestart('workspace trusted');
    }),
  );

  await startOrRestart('activation');
}

export async function deactivate(): Promise<void> {
  await client?.stop();
  client = undefined;
}
