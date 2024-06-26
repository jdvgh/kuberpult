/*This file is part of kuberpult.

Kuberpult is free software: you can redistribute it and/or modify
it under the terms of the Expat(MIT) License as published by
the Free Software Foundation.

Kuberpult is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
MIT License for more details.

You should have received a copy of the MIT License
along with kuberpult. If not, see <https://directory.fsf.org/wiki/License:Expat>.

Copyright freiheit.com*/
import { useRef, cloneElement } from 'react';
import classNames from 'classnames';
import * as React from 'react';

export const Button = (props: {
    id?: string;
    disabled?: boolean;
    className?: string;
    label?: string;
    icon?: JSX.Element;
    onClick?: (e: React.MouseEvent<HTMLButtonElement, MouseEvent>) => void;
    testId?: string;
    highlightEffect: boolean;
}): JSX.Element => {
    const control = useRef<HTMLButtonElement>(null);
    const { id, highlightEffect, disabled, className, label, icon, onClick, testId } = props;

    return (
        <button
            id={id}
            disabled={disabled}
            className={classNames('mdc-button', className, {
                highlight: highlightEffect,
            })}
            onClick={onClick}
            ref={control}
            aria-label={label || ''}
            data-testid={testId}>
            <div className="mdc-button__ripple" />
            {icon &&
                cloneElement(icon, {
                    key: 'icon',
                })}
            {!!label && (
                <span key="label" className="mdc-button__label">
                    {label}
                </span>
            )}
        </button>
    );
};
