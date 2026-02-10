import * as vscode from 'vscode';
import {
  type Executable,
  LanguageClient,
  type LanguageClientOptions,
  type ServerOptions,
} from 'vscode-languageclient/node';

import { type ResolvedBinary } from '../binary/findBinary';

export interface TallyLanguageClientInit {
  output: vscode.OutputChannel;
  server: ResolvedBinary;
}

export class TallyLanguageClient {
  private readonly client: LanguageClient;
  private readonly server: ResolvedBinary;

  public constructor(init: TallyLanguageClientInit) {
    this.server = init.server;

    const executable: Executable = {
      command: init.server.command,
      args: init.server.args,
    };

    const serverOptions: ServerOptions = executable;

    const clientOptions: LanguageClientOptions = {
      outputChannel: init.output,
      documentSelector: [
        { language: 'dockerfile', scheme: 'file' },
        { language: 'dockerfile', scheme: 'untitled' },
      ],
      // VS Code supports the LSP 3.17 pull diagnostics model. Disable push
      // diagnostics to avoid duplicate diagnostics when both are enabled.
      initializationOptions: {
        disablePushDiagnostics: true,
      },
    };

    this.client = new LanguageClient('tally', 'Tally', serverOptions, clientOptions);
  }

  public serverKey(): string {
    return this.server.key;
  }

  public async start(): Promise<void> {
    await this.client.start();
  }

  public async stop(): Promise<void> {
    await this.client.stop();
  }

  public async sendConfiguration(settings: unknown): Promise<void> {
    await this.client.sendNotification('workspace/didChangeConfiguration', { settings });
  }
}
