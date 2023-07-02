import * as vscode from 'vscode'

import { History, LightTextDocument } from '@sourcegraph/cody-shared/src/autocomplete'

export class VSCodeHistory implements vscode.Disposable, History {
    private window = 50

    // tracks history in chronological order (latest at the end of the array)
    private history: LightTextDocument[]

    private subscriptions: vscode.Disposable[] = []

    constructor(
        register: () => vscode.Disposable | null = () =>
            vscode.window.onDidChangeActiveTextEditor(event => {
                if (!event?.document.uri) {
                    return
                }
                this.addItem({
                    uri: event.document.uri.toString(),
                    languageId: event.document.languageId,
                })
            })
    ) {
        this.history = []
        if (register) {
            const disposable = register()
            if (disposable) {
                this.subscriptions.push(disposable)
            }
        }
    }

    public dispose(): void {
        vscode.Disposable.from(...this.subscriptions).dispose()
    }

    public addItem(newItem: LightTextDocument): void {
        const uri = vscode.Uri.parse(newItem.uri)
        if (uri.scheme === 'codegen') {
            return
        }
        const foundIndex = this.history.findIndex(item => item.uri === newItem.uri)
        if (foundIndex >= 0) {
            this.history = [...this.history.slice(0, foundIndex), ...this.history.slice(foundIndex + 1)]
        }
        this.history.push(newItem)
        if (this.history.length > this.window) {
            this.history.shift()
        }
    }

    /**
     * Returns the last n items of history in reverse chronological order (latest item at the front)
     */
    public lastN(n: number, languageId?: string, ignoreUris?: string[]): LightTextDocument[] {
        const ret: LightTextDocument[] = []
        const ignoreSet = new Set(ignoreUris || [])
        for (let i = this.history.length - 1; i >= 0; i--) {
            const item = this.history[i]
            if (ret.length > n) {
                break
            }
            if (ignoreSet.has(item.uri)) {
                continue
            }
            if (languageId && languageId !== item.languageId) {
                continue
            }
            ret.push(item)
        }
        return ret
    }
}