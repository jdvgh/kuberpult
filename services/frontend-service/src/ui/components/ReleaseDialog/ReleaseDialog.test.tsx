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

Copyright 2023 freiheit.com*/
import { EnvironmentListItem, ReleaseDialog, ReleaseDialogProps } from './ReleaseDialog';
import { fireEvent, render } from '@testing-library/react';
import { UpdateOverview, updateReleaseDialog, UpdateSidebar } from '../../utils/store';
import { Environment, Priority, Release } from '../../../api/api';
import { Spy } from 'spy4js';
import { SideBar } from '../SideBar/SideBar';

const mock_getFormattedReleaseDate = Spy.mockModule('../ReleaseCard/ReleaseCard', 'getFormattedReleaseDate');

describe('Release Dialog', () => {
    interface dataT {
        name: string;
        props: ReleaseDialogProps;
        rels: Release[];
        envs: Environment[];
        expect_message: boolean;
        expect_queues: number;
        data_length: number;
        teamName: string;
    }
    const data: dataT[] = [
        {
            name: 'normal release',
            props: {
                app: 'test1',
                version: 2,
                release: {
                    version: 2,
                    sourceMessage: 'test1',
                    sourceAuthor: 'test',
                    sourceCommitId: 'commit',
                    createdAt: new Date(2002),
                    undeployVersion: false,
                    prNumber: '#1337',
                },
            },
            rels: [],
            envs: [
                {
                    name: 'prod',
                    locks: { envLock: { message: 'envLock', lockId: 'ui-envlock' } },
                    applications: {
                        test1: {
                            name: 'test1',
                            version: 2,
                            locks: { applock: { message: 'appLock', lockId: 'ui-applock' } },
                            queuedVersion: 0,
                            undeployVersion: false,
                        },
                    },
                    distanceToUpstream: 0,
                    priority: Priority.UPSTREAM,
                },
            ],
            expect_message: true,
            expect_queues: 0,
            data_length: 1,
            teamName: '',
        },
        {
            name: 'two envs release',
            props: {
                app: 'test1',
                version: 2,
                release: {
                    version: 2,
                    sourceMessage: 'test1',
                    sourceAuthor: 'test',
                    sourceCommitId: 'commit',
                    createdAt: new Date(2002),
                    undeployVersion: false,
                    prNumber: '#1337',
                },
            },
            envs: [
                {
                    name: 'prod',
                    locks: { envLock: { message: 'envLock', lockId: 'ui-envlock' } },
                    applications: {
                        test1: {
                            name: 'test1',
                            version: 2,
                            locks: { applock: { message: 'appLock', lockId: 'ui-applock' } },
                            queuedVersion: 0,
                            undeployVersion: false,
                        },
                    },
                    distanceToUpstream: 0,
                    priority: Priority.UPSTREAM,
                },
                {
                    name: 'dev',
                    locks: { envLock: { message: 'envLock', lockId: 'ui-envlock' } },
                    applications: {
                        test1: {
                            name: 'test1',
                            version: 3,
                            locks: { applock: { message: 'appLock', lockId: 'ui-applock' } },
                            queuedVersion: 666,
                            undeployVersion: false,
                        },
                    },
                    distanceToUpstream: 0,
                    priority: Priority.UPSTREAM,
                },
            ],
            rels: [
                {
                    sourceCommitId: 'cafe',
                    sourceMessage: 'the other commit message 2',
                    version: 2,
                    undeployVersion: false,
                    prNumber: 'PR123',
                    sourceAuthor: 'nobody',
                },
                {
                    sourceCommitId: 'cafe',
                    sourceMessage: 'the other commit message 3',
                    version: 3,
                    undeployVersion: false,
                    prNumber: 'PR123',
                    sourceAuthor: 'nobody',
                },
            ],

            expect_message: true,
            expect_queues: 1,
            data_length: 3,
            teamName: 'test me team',
        },
        {
            name: 'no release',
            props: {
                app: 'test1',
                version: -1,
                release: {} as Release,
            },
            rels: [],
            envs: [],
            expect_message: false,
            expect_queues: 0,
            data_length: 0,
            teamName: '',
        },
    ];

    const setTheStore = (testcase: dataT) =>
        UpdateOverview.set({
            applications: { [testcase.props.app]: { releases: testcase.rels, team: testcase.teamName } },
            environments: testcase.envs,
            environmentGroups: [
                {
                    environmentGroupName: 'dev',
                    environments: testcase.envs,
                    distanceToUpstream: 2,
                },
            ],
        } as any);

    describe.each(data)(`Renders a Release Dialog`, (testcase) => {
        it(testcase.name, () => {
            // when
            setTheStore(testcase);
            updateReleaseDialog(testcase.props.app, testcase.props.version);
            render(<ReleaseDialog {...testcase.props} />);
            if (testcase.expect_message) {
                expect(document.querySelector('.release-dialog-message')?.textContent).toContain(
                    testcase.props.release.sourceMessage
                );
            } else {
                expect(document.querySelector('.release-dialog-message') === undefined);
            }
            expect(document.querySelectorAll('.env-card-data')).toHaveLength(testcase.data_length);
            expect(document.querySelectorAll('.env-card-data-queue')).toHaveLength(testcase.expect_queues);
        });
    });

    describe.each(data)(`Renders the environment cards`, (testcase) => {
        it(testcase.name, () => {
            // when
            setTheStore(testcase);
            updateReleaseDialog(testcase.props.app, testcase.props.version);
            render(<ReleaseDialog {...testcase.props} />);
            expect(document.querySelector('.release-env-list')?.children).toHaveLength(testcase.envs.length);
        });
    });

    describe.each(data)(`Renders the environment locks`, (testcase) => {
        it(testcase.name, () => {
            // given
            mock_getFormattedReleaseDate.getFormattedReleaseDate.returns(<div>some formatted date</div>);
            // when
            setTheStore(testcase);
            updateReleaseDialog(testcase.props.app, testcase.props.version);
            render(<ReleaseDialog {...testcase.props} />);
            expect(document.body).toMatchSnapshot();
            expect(document.querySelectorAll('.release-env-group-list')).toHaveLength(1);

            testcase.envs.forEach((env) => {
                expect(document.querySelector('.env-locks')?.children).toHaveLength(Object.values(env.locks).length);
            });
        });
    });

    describe.each(data)(`Renders the queuedVersion`, (testcase) => {
        it(testcase.name, () => {
            // when
            setTheStore(testcase);
            updateReleaseDialog(testcase.props.app, testcase.props.version);
            render(<ReleaseDialog {...testcase.props} />);
            expect(document.querySelectorAll('.env-card-data-queue')).toHaveLength(testcase.expect_queues);
        });
    });

    describe(`Test automatic cart opening`, () => {
        const testcase = data[0];
        it('Test using direct call to open function', () => {
            UpdateSidebar.set({ shown: false });
            UpdateSidebar.set({ shown: true });
            expect(UpdateSidebar.get().shown).toBeTruthy();
        });
        it('Test using deploy button click simulation', () => {
            UpdateSidebar.set({ shown: false });
            setTheStore(testcase);

            render(
                <EnvironmentListItem
                    env={testcase.envs[0]}
                    app={testcase.props.app}
                    queuedVersion={0}
                    release={{ ...testcase.props.release, version: 3 }}
                />
            );
            const result = document.querySelector('.env-card-deploy-btn')!;
            fireEvent.click(result);
            expect(UpdateSidebar.get().shown).toBeTruthy();
        });
        it('Test using add lock button click simulation', () => {
            UpdateSidebar.set({ shown: false });
            setTheStore(testcase);

            render(<ReleaseDialog {...testcase.props} />);
            render(
                <EnvironmentListItem
                    env={testcase.envs[0]}
                    app={testcase.props.app}
                    queuedVersion={0}
                    release={testcase.props.release}
                />
            );
            render(<SideBar toggleSidebar={Spy()} />);
            const result = document.querySelector('.env-card-add-lock-btn')!;
            fireEvent.click(result);
            expect(UpdateSidebar.get().shown).toBeTruthy();
        });
    });
});
