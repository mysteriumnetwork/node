#!/bin/bash
### BEGIN INIT INFO
# Provides:          mysterium-node
# Required-Start:    $all
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: Start the Mysterium VPN node process
### END INIT INFO

# If you modify this, please make sure to also edit systemd.service

OS_DIR_BIN="/usr/bin"
OS_DIR_CONFIG="/etc/mysterium-node"
OS_DIR_DATA="/var/lib/mysterium-node"
OS_DIR_LOG="/var/log/mysterium-node"
OS_DIR_RUN="/var/run/mysterium-node"

# Process name (For display)
DAEMON_NAME="mysterium-node"
#Daemon name, where is the actual executable
DAEMON_BIN="$OS_DIR_BIN/mysterium_server"
# User and group
DAEMON_USER="mysterium-node"
DAEMON_GROUP="mysterium-node"
# PID file for the daemon
DAEMON_PIDFILE="$OS_DIR_RUN/daemon.pid"
# Logging
DAEMON_STDOUT="$OS_DIR_LOG/daemon.log"
DAEMON_STDERR="$OS_DIR_LOG/error.log"
# Command-line options that can be set in /etc/default/mysterium-node.
# These will override any config file values.
DAEMON_DEFAULT="/etc/default/mysterium-node"

# Check for sudo or root privileges before continuing
if [ "$UID" != "0" ]; then
    echo "You must be root to run this script"
    exit 1
fi

# If the daemon is not there, then exit.
if [ ! -x $DAEMON_BIN ]; then
    echo "Executable $DAEMON_BIN does not exist!"
    exit 5
fi

# Create directory for pid file
PIDDIR=`dirname $DAEMON_PIDFILE`
if [ ! -d "$PIDDIR" ]; then
    mkdir -p $PIDDIR
    chown $DAEMON_USER:$DAEMON_GROUP $PIDDIR
fi


# Create directory for logs
LOGDIR=`dirname $DAEMON_STDOUT`
if [ ! -d "$LOGDIR" ]; then
    mkdir -p $LOGDIR
    chown -R -L $DAEMON_USER:$DAEMON_GROUP $LOGDIR
fi
LOGDIR=`dirname $DAEMON_STDERR`
if [ ! -d "$LOGDIR" ]; then
    mkdir -p $LOGDIR
    chown -R -L $DAEMON_USER:$DAEMON_GROUP $LOGDIR
fi

if [ -r /lib/lsb/init-functions ]; then
    source /lib/lsb/init-functions
fi

# Override init script variables with DEFAULT values
if [ -r $DAEMON_DEFAULT ]; then
    source $DAEMON_DEFAULT
fi

function log_failure_msg() {
    echo "$@" "[ FAILED ]"
}

function log_success_msg() {
    echo "$@" "[ OK ]"
}

function start() {
    # Check that the PID file exists, and check the actual status of process
    if [ -f $DAEMON_PIDFILE ]; then
        PID="$(cat $DAEMON_PIDFILE)"
        if kill -0 "$PID" &>/dev/null; then
            # Process is already up
            log_success_msg "$DAEMON_NAME process is already running"
            return 0
        fi
    else
        su -s /bin/sh -c "touch $DAEMON_PIDFILE" $DAEMON_USER &>/dev/null
        if [ $? -ne 0 ]; then
            log_failure_msg "$DAEMON_PIDFILE not writable, check permissions"
            exit 5
        fi
    fi

    # Launch process
    echo "Starting $DAEMON_NAME..."
    start-stop-daemon \
        --start \
        --quiet \
        --background \
        --no-close \
        --make-pidfile \
        --pidfile $DAEMON_PIDFILE \
        --user $DAEMON_USER \
        --group $DAEMON_GROUP \
        --exec $DAEMON_BIN \
        -- \
        --config-dir=$OS_DIR_CONFIG \
        --data-dir=$OS_DIR_DATA \
        --runtime-dir=$OS_DIR_RUN \
        $(DAEMON_OPTS) \
        >>$DAEMON_STDOUT \
        2>>$DAEMON_STDERR

    # Sleep to verify process is still up
    sleep 1
    if [ -f $DAEMON_PIDFILE ]; then
        # PIDFILE exists
        if kill -0 $(cat $DAEMON_PIDFILE) &>/dev/null; then
            # PID up, service running
            log_success_msg "$DAEMON_NAME process was started"
            return 0
        fi
    fi
    log_failure_msg "$DAEMON_NAME process was unable to start"
    exit 1
}

function stop() {
    # Stop the daemon.
    if [ -f $DAEMON_PIDFILE ]; then
        local PID="$(cat $DAEMON_PIDFILE)"
        if kill -0 $PID &>/dev/null; then
            echo "Stopping $DAEMON_NAME..."
            # Process still up, send SIGTERM and remove PIDFILE
            kill -s TERM $PID &>/dev/null && rm -f "$DAEMON_PIDFILE" &>/dev/null
            n=0
            while true; do
                # Enter loop to ensure process is stopped
                kill -0 $PID &>/dev/null
                if [ "$?" != "0" ]; then
                    # Process stopped, break from loop
                    log_success_msg "$DAEMON_NAME process was stopped"

                    # Kill open Openvpn, until signal handling will be implemented
                    killall openvpn

                    return 0
                fi

                # Process still up after signal, sleep and wait
                sleep 1
                n=$(expr $n + 1)
                if [ $n -eq 30 ]; then
                    # After 30 seconds, send SIGKILL
                    echo "Timeout exceeded, sending SIGKILL..."
                    kill -s KILL $PID &>/dev/null

                    # Kill open Openvpn, until signal handling will be implemented
                    killall openvpn
                elif [ $? -eq 40 ]; then
                    # After 40 seconds, error out
                    log_failure_msg "could not stop $DAEMON_NAME process"
                    exit 1
                fi
            done
        fi
    fi
    log_success_msg "$DAEMON_NAME process already stopped"
}

function restart() {
    # Restart the daemon.
    stop
    start
}

function status() {
    # Check the status of the process.
    if [ -f $DAEMON_PIDFILE ]; then
        PID="$(cat $DAEMON_PIDFILE)"
        if kill -0 $PID &>/dev/null; then
            log_success_msg "$DAEMON_NAME process is running"
            exit 0
        fi
    fi
    log_failure_msg "$DAEMON_NAME process is not running"
    exit 1
}

case $1 in
    start)
        start
        ;;

    stop)
        stop
        ;;

    restart)
        restart
        ;;

    status)
        status
        ;;

    *)
        # For invalid arguments, print the usage message.
        echo "Usage: $0 {start|stop|restart|status}"
        exit 2
        ;;
esac
