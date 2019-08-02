import { connect } from 'react-redux';
import {
    getLogs,
    toggleLogStatus,
    downloadQueryLog,
    getFilteringStatus,
    setRules,
    addSuccessToast,
    getClients,
    clearLogs,
} from '../actions';
import Logs from '../components/Logs';

const mapStateToProps = (state) => {
    const { queryLogs, dashboard, filtering } = state;
    const props = { queryLogs, dashboard, filtering };
    return props;
};

const mapDispatchToProps = {
    getLogs,
    toggleLogStatus,
    downloadQueryLog,
    getFilteringStatus,
    setRules,
    addSuccessToast,
    getClients,
    clearLogs,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Logs);
