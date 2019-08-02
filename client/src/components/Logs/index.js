import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import { saveAs } from 'file-saver/FileSaver';
import escapeRegExp from 'lodash/escapeRegExp';
import endsWith from 'lodash/endsWith';
import { Trans, withNamespaces } from 'react-i18next';
import { HashLink as Link } from 'react-router-hash-link';

import { formatTime, getClientName } from '../../helpers/helpers';
import { getTrackerData } from '../../helpers/trackers/trackers';

import Controls from './Controls';
import PageTitle from '../ui/PageTitle';
import Card from '../ui/Card';
import PopoverFiltered from '../ui/PopoverFilter';
import Popover from '../ui/Popover';
import Loading from '../ui/Loading';
import './Logs.css';

const DOWNLOAD_LOG_FILENAME = 'dns-logs.txt';

class Logs extends Component {
    componentDidMount() {
        this.getLogs();
        this.props.getFilteringStatus();
        this.props.getClients();
    }

    componentDidUpdate(prevProps) {
        // get logs when queryLog becomes enabled
        if (this.props.dashboard.queryLogEnabled && !prevProps.dashboard.queryLogEnabled) {
            this.getLogs();
        }
    }

    getLogs = () => {
        // get logs on initialization if queryLogIsEnabled
        if (this.props.dashboard.queryLogEnabled) {
            this.props.getLogs();
        }
    };

    clearLogs = () => {
        // eslint-disable-next-line no-alert
        if (window.confirm(this.props.t('query_log_confirm_clear'))) {
            this.props.clearLogs();
        }
    }

    renderTooltip = (isFiltered, rule, filter) =>
        (rule && isFiltered ? <PopoverFiltered rule={rule} filter={filter} /> : '');

    toggleBlocking = (type, domain) => {
        const { userRules } = this.props.filtering;
        const { t } = this.props;
        const lineEnding = !endsWith(userRules, '\n') ? '\n' : '';
        const baseRule = `||${domain}^$important`;
        const baseUnblocking = `@@${baseRule}`;
        const blockingRule = type === 'block' ? baseUnblocking : baseRule;
        const unblockingRule = type === 'block' ? baseRule : baseUnblocking;
        const preparedBlockingRule = new RegExp(`(^|\n)${escapeRegExp(blockingRule)}($|\n)`);
        const preparedUnblockingRule = new RegExp(`(^|\n)${escapeRegExp(unblockingRule)}($|\n)`);

        if (userRules.match(preparedBlockingRule)) {
            this.props.setRules(userRules.replace(`${blockingRule}`, ''));
            this.props.addSuccessToast(`${t('rule_removed_from_custom_filtering_toast')}: ${blockingRule}`);
        } else if (!userRules.match(preparedUnblockingRule)) {
            this.props.setRules(`${userRules}${lineEnding}${unblockingRule}\n`);
            this.props.addSuccessToast(`${t('rule_added_to_custom_filtering_toast')}: ${unblockingRule}`);
        }

        this.props.getFilteringStatus();
    };

    renderBlockingButton(isFiltered, domain) {
        const buttonClass = isFiltered ? 'btn-outline-secondary' : 'btn-outline-danger';
        const buttonText = isFiltered ? 'unblock_btn' : 'block_btn';
        const buttonType = isFiltered ? 'unblock' : 'block';

        return (
            <div className="logs__action">
                <button
                    type="button"
                    className={`btn btn-sm ${buttonClass}`}
                    onClick={() => this.toggleBlocking(buttonType, domain)}
                    disabled={this.props.filtering.processingRules}
                >
                    <Trans>{buttonText}</Trans>
                </button>
            </div>
        );
    }

    renderLogs(logs) {
        const { t, dashboard } = this.props;
        const columns = [
            {
                Header: t('time_table_header'),
                accessor: 'time',
                maxWidth: 110,
                filterable: false,
                Cell: ({ value }) => (
                    <div className="logs__row">
                        <span className="logs__text" title={value}>
                            {formatTime(value)}
                        </span>
                    </div>
                ),
            },
            {
                Header: t('domain_name_table_header'),
                accessor: 'domain',
                Cell: (row) => {
                    const response = row.value;
                    const trackerData = getTrackerData(response);

                    return (
                        <div className="logs__row" title={response}>
                            <div className="logs__text">{response}</div>
                            {trackerData && <Popover data={trackerData} />}
                        </div>
                    );
                },
            },
            {
                Header: t('type_table_header'),
                accessor: 'type',
                maxWidth: 60,
            },
            {
                Header: t('response_table_header'),
                accessor: 'response',
                Cell: (row) => {
                    const responses = row.value;
                    const { reason } = row.original;
                    const isFiltered = row ? reason.indexOf('Filtered') === 0 : false;
                    const parsedFilteredReason = reason.replace('Filtered', 'Filtered by ');
                    const rule = row && row.original && row.original.rule;
                    const { filterId } = row.original;
                    const { filters } = this.props.filtering;
                    const isRewrite = reason && reason === 'Rewrite';
                    let filterName = '';

                    if (reason === 'FilteredBlackList' || reason === 'NotFilteredWhiteList') {
                        if (filterId === 0) {
                            filterName = t('custom_filter_rules');
                        } else {
                            const filterItem = Object.keys(filters)
                                .filter(key => filters[key].id === filterId);

                            if (
                                typeof filterItem !== 'undefined' &&
                                typeof filters[filterItem] !== 'undefined'
                            ) {
                                filterName = filters[filterItem].name;
                            }

                            if (!filterName) {
                                filterName = t('unknown_filter', { filterId });
                            }
                        }
                    }

                    if (isFiltered) {
                        return (
                            <div className="logs__row">
                                <span className="logs__text" title={parsedFilteredReason}>
                                    {parsedFilteredReason}
                                </span>
                                {this.renderTooltip(isFiltered, rule, filterName)}
                            </div>
                        );
                    }

                    if (responses.length > 0) {
                        const liNodes = responses.map((response, index) => (
                            <li key={index} title={response}>
                                {response}
                            </li>
                        ));
                        const isRenderTooltip = reason === 'NotFilteredWhiteList';

                        return (
                            <div className={`logs__row ${isRewrite && 'logs__row--column'}`}>
                                {isRewrite && <strong><Trans>rewrite_applied</Trans></strong>}
                                <ul className="list-unstyled">{liNodes}</ul>
                                {this.renderTooltip(isRenderTooltip, rule, filterName)}
                            </div>
                        );
                    }
                    return (
                        <div className={`logs__row ${isRewrite && 'logs__row--column'}`}>
                            {isRewrite && <strong><Trans>rewrite_applied</Trans></strong>}
                            <span><Trans>empty_response_status</Trans></span>
                            {this.renderTooltip(isFiltered, rule, filterName)}
                        </div>
                    );
                },
                filterMethod: (filter, row) => {
                    if (filter.value === 'filtered') {
                        return (
                            // eslint-disable-next-line no-underscore-dangle
                            row._original.reason.indexOf('Filtered') === 0 || row._original.reason === 'NotFilteredWhiteList'
                        );
                    }
                    return true;
                },
                Filter: ({ filter, onChange }) => (
                    <select
                        onChange={event => onChange(event.target.value)}
                        className="form-control"
                        value={filter ? filter.value : 'all'}
                    >
                        <option value="all">{t('show_all_filter_type')}</option>
                        <option value="filtered">{t('show_filtered_type')}</option>
                    </select>
                ),
            },
            {
                Header: t('client_table_header'),
                accessor: 'client',
                maxWidth: 250,
                Cell: (row) => {
                    const { reason } = row.original;
                    const isFiltered = row ? reason.indexOf('Filtered') === 0 : false;
                    const isRewrite = reason && reason === 'Rewrite';
                    const clientName =
                        getClientName(dashboard.clients, row.value) ||
                        getClientName(dashboard.autoClients, row.value);
                    let client;

                    if (clientName) {
                        client = (
                            <span>
                                {clientName} <small>({row.value})</small>
                            </span>
                        );
                    } else {
                        client = row.value;
                    }

                    if (isRewrite) {
                        return (
                            <Fragment>
                                <div className="logs__row">
                                    {client}
                                </div>
                                <div className="logs__action">
                                    <Link to="/dns#rewrites" className="btn btn-sm btn-outline-primary">
                                        <Trans>configure</Trans>
                                    </Link>
                                </div>
                            </Fragment>
                        );
                    }

                    return (
                        <Fragment>
                            <div className="logs__row">{client}</div>
                            {this.renderBlockingButton(isFiltered, row.original.domain)}
                        </Fragment>
                    );
                },
            },
        ];

        if (logs) {
            return (
                <ReactTable
                    data={logs}
                    className="logs__table"
                    filterable
                    columns={columns}
                    showPagination={true}
                    defaultPageSize={50}
                    minRows={5}
                    previousText={t('previous_btn')}
                    nextText={t('next_btn')}
                    loadingText={t('loading_table_status')}
                    pageText={t('page_table_footer_text')}
                    ofText={t('of_table_footer_text')}
                    rowsText={t('rows_table_footer_text')}
                    noDataText={t('no_logs_found')}
                    defaultFilterMethod={(filter, row) => {
                        const id = filter.pivotId || filter.id;
                        return row[id] !== undefined
                            ? String(row[id]).indexOf(filter.value) !== -1
                            : true;
                    }}
                    defaultSorted={[
                        {
                            id: 'time',
                            desc: true,
                        },
                    ]}
                    getTrProps={(_state, rowInfo) => {
                        // highlight filtered requests
                        if (!rowInfo) {
                            return {};
                        }

                        if (rowInfo.original.reason.indexOf('Filtered') === 0) {
                            return {
                                className: 'red',
                            };
                        } else if (rowInfo.original.reason === 'NotFilteredWhiteList') {
                            return {
                                className: 'green',
                            };
                        } else if (rowInfo.original.reason === 'Rewrite') {
                            return {
                                className: 'blue',
                            };
                        }

                        return {
                            className: '',
                        };
                    }}
                />
            );
        }

        return false;
    }

    handleDownloadButton = async () => {
        const data = await this.props.downloadQueryLog();
        const jsonStr = JSON.stringify(data);
        const dataBlob = new Blob([jsonStr], { type: 'text/plain;charset=utf-8' });
        saveAs(dataBlob, DOWNLOAD_LOG_FILENAME);
    };

    render() {
        const { queryLogs, dashboard, t } = this.props;
        const { queryLogEnabled } = dashboard;
        return (
            <Fragment>
                <PageTitle title={t('query_log')} subtitle={t('last_dns_queries')}>
                    <div className="page-title__actions">
                        <Controls
                            queryLogEnabled={queryLogEnabled}
                            logStatusProcessing={dashboard.logStatusProcessing}
                            toggleLogStatus={this.props.toggleLogStatus}
                            handleDownloadButton={this.handleDownloadButton}
                            getLogs={this.getLogs}
                            clearLogs={this.clearLogs}
                        />
                    </div>
                </PageTitle>
                {queryLogEnabled && queryLogs.getLogsProcessing && <Loading />}
                {queryLogEnabled && !queryLogs.getLogsProcessing ?
                        <Card>{this.renderLogs(queryLogs.logs)}</Card> : ''}
            </Fragment>
        );
    }
}

Logs.propTypes = {
    getLogs: PropTypes.func,
    queryLogs: PropTypes.object,
    dashboard: PropTypes.object,
    toggleLogStatus: PropTypes.func,
    downloadQueryLog: PropTypes.func,
    getFilteringStatus: PropTypes.func,
    filtering: PropTypes.object,
    userRules: PropTypes.string,
    setRules: PropTypes.func,
    addSuccessToast: PropTypes.func,
    processingRules: PropTypes.bool,
    logStatusProcessing: PropTypes.bool,
    t: PropTypes.func,
    getClients: PropTypes.func.isRequired,
    clearLogs: PropTypes.func.isRequired,
};

export default withNamespaces()(Logs);
