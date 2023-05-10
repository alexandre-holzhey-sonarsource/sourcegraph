import cookies from 'js-cookie'
import * as vscode from 'vscode'
import { Memento } from 'vscode'

import { SourcegraphGraphQLAPIClient } from '../sourcegraph-api/graphql'

function _getServerEndpointFromConfig(config: vscode.WorkspaceConfiguration): string {
    return config.get<string>('cody.serverEndpoint', '')
}

export const ANONYMOUS_USER_ID_KEY = 'sourcegraphAnonymousUid'

const config = vscode.workspace.getConfiguration()

let storage: Memento | undefined
if (typeof localStorage === 'undefined') {
    storage = vscode.workspace.getConfiguration().get<vscode.Memento>('cody.storage')
} else {
    storage = {
        get: (key: string) => cookies.get(key),
        update: (key: string, value: any) =>
            new Promise(resolve => {
                cookies.set(key, value)
                resolve()
            }),
        keys: () => Object.keys(cookies.get()),
    }
}

let anonymousUserID: string
if (storage) {
    anonymousUserID = storage.get(ANONYMOUS_USER_ID_KEY)
}
if (!anonymousUserID) {
    anonymousUserID = cookies.get(ANONYMOUS_USER_ID_KEY)
}

export class EventLogger {
    private serverEndpoint = _getServerEndpointFromConfig(config)
    private extensionDetails = { ide: 'VSCode', ideExtensionType: 'Cody' }

    private constructor(private gqlAPIClient: SourcegraphGraphQLAPIClient) {}

    public static create(gqlAPIClient: SourcegraphGraphQLAPIClient): EventLogger {
        return new EventLogger(gqlAPIClient)
    }

    /**
     * @param eventName The ID of the action executed.
     */
    public async log(eventName: string, eventProperties?: any, publicProperties?: any): Promise<void> {
        const anonymousUserID = cookies.get(ANONYMOUS_USER_ID_KEY) || storage.get(ANONYMOUS_USER_ID_KEY)

        // Don't log events if the UID has not yet been generated.
        if (!anonymousUserID) {
            return
        }
        const argument = {
            ...eventProperties,
            serverEndpoint: this.serverEndpoint,
            extensionDetails: this.extensionDetails,
        }
        const publicArgument = {
            ...publicProperties,
            serverEndpoint: this.serverEndpoint,
            extensionDetails: this.extensionDetails,
        }

        try {
            await this.gqlAPIClient.logEvent({
                event: eventName,
                userCookieID: anonymousUserID,
                source: 'IDEEXTENSION',
                url: '',
                argument: JSON.stringify(argument),
                publicArgument: JSON.stringify(publicArgument),
            })
        } catch (error) {
            console.log(error)
        }
    }
}
