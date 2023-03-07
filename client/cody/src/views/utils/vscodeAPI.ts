export type MessageFromWebview = {
  command:
    | 'get-token'
    | 'set-token'
    | 'set-endpoint'
    | 'remove-token'
    | 'test-message'
    | 'on-startup'
    | 'executeRecipe'
    | 'submit'
    | 'reset'
    | 'settings'
    | 'initialized';
  value?: string;
  text?: string;
  recipeID?: string;
  accessToken?: string;
  serverURL?: string;
};

declare const acquireVsCodeApi: Function;

interface VSCodeApi {
  getState: () => any;
  setState: (newState: any) => any;
  postMessage: (message: any) => void;
}

class VSCodeWrapper {
  private readonly vscodeApi: VSCodeApi = acquireVsCodeApi();

  public postMessage(message: MessageFromWebview): void {
    this.vscodeApi.postMessage(message);
  }

  public onMessage(callback: (message: any) => void): () => void {
    window.addEventListener('message', callback);
    return () => window.removeEventListener('message', callback);
  }
}

export const vscodeAPI: VSCodeWrapper = new VSCodeWrapper();
