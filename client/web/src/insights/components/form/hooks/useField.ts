import { ChangeEvent, FocusEventHandler, Ref, useEffect, useRef, useState } from 'react'
import { noop } from 'rxjs'

import { FieldErrorState, FormAPI, Validator } from './useForm'
import { getEventValue } from './utils/get-event-value';
import { AsyncValidator, useAsyncValidation } from './utils/use-async-validation';

export interface Validators<FieldValue> {
    sync?: Validator<FieldValue>,
    async?: AsyncValidator<FieldValue>
}

/**
 * Public API for input element. Contains all handlers and props for
 * native input element and expose meta state of input like touched,
 * validState and etc.
 * */
export interface useFieldAPI<FieldValue> {
    /**
     * Props and handles which should be passed to input component in order
     * to track change of value, set controlled value and set errors by
     * Constraint validation API.
     * */
    input: {
        ref: Ref<HTMLInputElement & HTMLFieldSetElement>
        name: string
        value: FieldValue
        onChange: (event: ChangeEvent<HTMLInputElement> | FieldValue) => void
        onBlur: FocusEventHandler<HTMLInputElement>
    }
    /**
     * Meta state of form field - like touched, valid state and last
     * native validity state.
     * */
    meta: FieldErrorState
}

/**
 * React hook to manage validation of a single form input field.
 * `useInputValidation` helps with coordinating the constraint validation API
 * and custom synchronous and asynchronous validators.
 *
 * Should be used with useForm hook to connect field and form component's states.
 * */
export function useField<FormValues, FieldValueKey extends keyof FormAPI<FormValues>['initialValues']>(
    name: FieldValueKey,
    formApi: FormAPI<FormValues>,
    validators?: Validators<FormValues[FieldValueKey]>,
): useFieldAPI<FormValues[FieldValueKey]> {
    const { setFieldState, initialValues, submitted, touched: formTouched } = formApi

    const { sync = noop, async } = validators ?? {};

    const inputReference = useRef<HTMLInputElement & HTMLFieldSetElement>(null)
    const [value, setValue] = useState<FormValues[FieldValueKey]>(initialValues[name])
    const [state, setState] = useState<FieldErrorState>({
        touched: false,
        validState: 'NOT_VALIDATED',
        error: '',
        validity: null,
    })

    const { start: startAsyncValidation, cancel: cancelAsyncValidation } = useAsyncValidation({
        inputReference,
        asyncValidator: async,
        onValidationChange: state => setState(previousState => ({ ...previousState, ...state}))
    })

    // Use useRef for form api handler in order to avoid unnecessary
    // calls if API handler has been changed.
    const setFieldStateReference = useRef<FormAPI<FormValues>['setFieldState']>(setFieldState)
    setFieldStateReference.current = setFieldState

    useEffect(() => {
        const inputElement = inputReference.current

        // Clear custom validity from the last validation call.
        inputElement?.setCustomValidity?.('')

        const nativeAttributeValidation = inputElement?.checkValidity?.() ?? true
        const validity = inputElement?.validity ?? null

        // If we got error from native attr validation (required, pattern, type)
        // we still run validator in order to get some custom error message for
        // standard validation error if validator doesn't provide message we fallback
        // on standard validationMessage string [1] (ex. Please fill in input.)
        const nativeErrorMessage = inputElement?.validationMessage ?? ''
        const customValidation = sync(value, validity)

        if (customValidation || !nativeAttributeValidation) {
            // We have to cancel async validation from previous call
            // if we got sync validation native or custom.
            cancelAsyncValidation()

            // [1] Custom error message or fallback on native error message
            const validationMessage = customValidation || nativeErrorMessage

            inputElement?.setCustomValidity?.(validationMessage)

            return setState(state => ({
                ...state,
                validState: 'INVALID',
                error: validationMessage,
                validity,
            }))
        }

        if (async) {
            startAsyncValidation({ value, validity })
        }

        return setState(state => ({
            ...state,
            validState: 'VALID' as const,
            error: '',
            validity,
        }))
    }, [value, sync, startAsyncValidation, async, cancelAsyncValidation])

    // Sync field state with state on form level - useForm hook will used this state to run
    // onSubmit handler and track validation state to prevent onSubmit run when async
    // validation is going.
    useEffect(
        () => setFieldStateReference.current(name, { ...state, value }),
        [name, state, value]
    )

    return {
        input: {
            name: name.toString(),
            ref: inputReference,
            value,
            onBlur: () => setState(state => ({ ...state, touched: true })),
            onChange: (event: ChangeEvent<HTMLInputElement> | FormValues[FieldValueKey]) => {
                setValue(getEventValue(event))
            },
        },
        meta: {
            ...state,
            touched: state.touched || submitted || formTouched,
        },
    }
}

