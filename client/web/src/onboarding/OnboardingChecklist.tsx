import { type FC, type PropsWithChildren, useCallback, useState, useEffect } from 'react'

import type { ApolloQueryResult } from '@apollo/client'
import { mdiAlertCircle, mdiChevronDown, mdiCheckCircle, mdiCheckCircleOutline } from '@mdi/js'
// eslint-disable-next-line id-length
import cx from 'classnames'
import * as jsonc from 'jsonc-parser'
import { useNavigate } from 'react-router-dom'
import type { SiteConfigResult } from 'src/graphql-operations'
import { useDebounce } from 'use-debounce'

import { useIsLightTheme } from '@sourcegraph/shared/src/theme'
import {
    Alert,
    Button,
    Form,
    H2,
    H3,
    H4,
    Input,
    Label,
    LoadingSpinner,
    Modal,
    Text,
    Popover,
    PopoverContent,
    PopoverTrigger,
    Position,
    Icon,
    Link,
} from '@sourcegraph/wildcard'

import { BrandLogo } from '../components/branding/BrandLogo'

import { content } from './OnboardingChecklist.content'
import {
    useOnboardingChecklistQuery,
    useUpdateLicenseKey,
    type OnboardingChecklistItem,
} from './useOnboardingChecklist'

import styles from './OnboardingChecklist.module.scss'

export const OnboardingChecklist: FC = (): JSX.Element => {
    const { data, loading, error, refetch } = useOnboardingChecklistQuery()
    const [isDropdownOpen, setIsDropdownOpen] = useState(false)
    const toggleDropdownOpen = useCallback(() => setIsDropdownOpen(isOpen => !isOpen), [])

    if (loading || !data) {
        return <LoadingSpinner />
    }
    if (error) {
        return <></>
    }

    const { licenseKey, checklistItem, config, id } = data

    const isComplete = Object.values(checklistItem).every((value: boolean) => value)
    if (isComplete) {
        return <></>
    }

    return (
        <>
            <Popover isOpen={isDropdownOpen} onOpenChange={toggleDropdownOpen}>
                <PopoverTrigger type="button" className={styles.button}>
                    <Icon aria-hidden={true} size="md" svgPath={mdiAlertCircle} />
                    <span>Setup</span>
                    <Icon aria-hidden={true} svgPath={mdiChevronDown} />
                </PopoverTrigger>
                <PopoverContent className={styles.container} position={Position.bottom}>
                    <OnboardingChecklistList>
                        {content.map(({ id, isComplete, title, description, link }) => (
                            <OnboardingChecklistItem
                                key={id}
                                isComplete={checklistItem[id as keyof OnboardingChecklistItem] || isComplete}
                                title={title}
                                description={description}
                                link={link}
                            />
                        ))}
                    </OnboardingChecklistList>
                </PopoverContent>
            </Popover>
            {!licenseKey && <LicenseKeyModal licenseKey={licenseKey} config={config} id={id} refetch={refetch} />}
        </>
    )
}

const OnboardingChecklistList: FC<PropsWithChildren<{}>> = ({ children }): JSX.Element => (
    <ul className={styles.list}>{children}</ul>
)

interface OnboardingChecklistItemProps {
    isComplete: boolean
    title: string
    description: string
    link: string
}

const OnboardingChecklistItem: FC<OnboardingChecklistItemProps> = ({
    isComplete,
    title,
    description,
    link,
}): JSX.Element => (
    <li className={styles.item}>
        <div className={styles.wrapper}>
            <Icon
                aria-hidden={true}
                svgPath={isComplete ? mdiCheckCircle : mdiCheckCircleOutline}
                className={cx({ [styles.checked]: isComplete })}
            />
            <div className={styles.content}>
                <H4>{title}</H4>
                {!isComplete && (
                    <>
                        <Text>{description}</Text>
                        <Link to={link} target="_blank" rel="noopener">
                            Configure now
                        </Link>{' '}
                    </>
                )}
            </div>
        </div>
    </li>
)

interface LicenseKeyModalProps {
    id: number
    config: string
    licenseKey: string
    refetch: () => Promise<ApolloQueryResult<SiteConfigResult>>
}

const DEFAULT_FORMAT_OPTIONS = {
    formattingOptions: {
        eol: '\n',
        insertSpaces: true,
        tabSize: 2,
    },
}
const LicenseKeyModal: FC<LicenseKeyModalProps> = ({ licenseKey, config, id, refetch }): JSX.Element => {
    const navigate = useNavigate()
    const [isOpen, setIsOpen] = useState(true)
    const [isValid, setIsValid] = useState(false)
    const [licenseKeyInput, setLicenseKeyInput] = useState(licenseKey)
    const [debouncedValue] = useDebounce(licenseKeyInput, 500)

    const isOnboarding = localStorage.getItem('isOnboarding') === null

    const onDismiss = useCallback(() => {
        localStorage.setItem('isOnboarding', 'false')
        setIsOpen(false)
    }, [setIsOpen])

    const [updateLicenseKey, { error, loading }] = useUpdateLicenseKey()

    useEffect(() => {
        const setLicenseKey = async (): Promise<void> => {
            const editFns = [
                (contents: string) => jsonc.modify(contents, ['licenseKey'], debouncedValue, DEFAULT_FORMAT_OPTIONS),
            ]
            let input = config
            for (const editFunc of editFns) {
                input = jsonc.applyEdits(config, editFunc(config))
            }

            await updateLicenseKey({ variables: { lastID: id, input } })
            setIsValid(true)
        }
        if (debouncedValue.length > 10) {
            setLicenseKey().catch(() => setIsValid(false))
        }
    }, [config, id, updateLicenseKey, debouncedValue, refetch])

    const onSubmit = useCallback(
        async (event?: React.FormEvent<HTMLFormElement>) => {
            const submit = async (): Promise<void> => {
                event?.preventDefault()
                setIsOpen(false)
                localStorage.setItem('isOnboarding', 'false')
                navigate('site-admin/configuration')
                await refetch()
            }
            await submit()
        },
        [navigate, refetch]
    )

    return (
        <>
            {(isOpen || isOnboarding || licenseKey.length === 0) && (
                <Modal
                    className={styles.modal}
                    containerClassName={styles.modalContainer}
                    onDismiss={() => setIsOpen(false)}
                    aria-labelledby="license-key"
                >
                    <H3 className="m-0 pb-4">Upgrade your license</H3>
                    <Text className="m-0 pb-3">Enter your license key to start your enterprise set up:</Text>
                    {error && <Alert variant="danger">License key not recognized. Please try again.</Alert>}
                    <Form onSubmit={onSubmit}>
                        <Label htmlFor="license-key">License key</Label>
                        <Input
                            type="text"
                            name="license-key"
                            value={licenseKeyInput}
                            className="pb-4"
                            onChange={({ target: { value } }) => setLicenseKeyInput(value)}
                        />
                        <LicenseKey isValid={isValid} />
                        <div className="d-flex justify-content-end">
                            <Button
                                className="mr-2"
                                onClick={onDismiss}
                                outline={true}
                                variant="secondary"
                                disabled={isValid}
                            >
                                Skip for now
                            </Button>
                            <Button className={styles.submit} type="submit" disabled={!isValid} variant="primary">
                                Upgrade and start set up
                                {loading && (
                                    <div data-testid="action-item-spinner">
                                        <LoadingSpinner inline={false} />
                                    </div>
                                )}
                            </Button>
                        </div>
                    </Form>
                </Modal>
            )}
        </>
    )
}

interface LicenseKeyProps {
    isValid: boolean
}
const LicenseKey: FC<LicenseKeyProps> = ({ isValid }): JSX.Element => {
    const isLightTheme = useIsLightTheme()
    return (
        <div className={cx(styles.keyWrapper, 'p-4 mb-4')}>
            <div className={styles.keyContainer}>
                <div className={styles.logoContainer}>
                    {isValid ? (
                        <BrandLogo isLightTheme={isLightTheme} variant="symbol" />
                    ) : (
                        <div className={styles.emptyLogo} />
                    )}
                </div>
                <div>
                    {isValid ? (
                        <>
                            <Text className="mb-1">License</Text>
                            <H2 className="mb-1">Sourcegraph Enterprise (dev-only)</H2>
                            <Text className="mb-0">1000-user license, valid until 2029-01-20 (5 years remaining)</Text>
                        </>
                    ) : (
                        <>
                            <Text className="mb-1">Current License</Text>
                            <H2 className="mb-1">Free</H2>
                            <Text className="mb-0">1-user license, valid indefinitely</Text>
                        </>
                    )}
                </div>
            </div>
        </div>
    )
}