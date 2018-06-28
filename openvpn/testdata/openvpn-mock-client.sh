#!/usr/bin/env bash

echo "Starting fake openvpn process"

while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    --management)
    connAddr="$2"
    connPort="$3"
    shift # past value
    shift # past second value
    ;;
    *)    # unknown option
    # donothing
    ;;
esac
shift
done

echo "Using address $connAddr $connPort for management"

mng_output=$(mktemp -u)
mkfifo $mng_output
exec 3<>$mng_output

mng_input=$(mktemp -u)
mkfifo $mng_input
exec 4<>$mng_input

nc $connAddr $connPort <&3 >&4 &  #async process
nc_pid=$!

while read cmd; do
    case $cmd in
    SINGLELINE_CMD)
    echo "SUCCESS: ${cmd}_OK" >&3
    ;;
    MULTILINE_CMD)
    echo "SUCCESS: ${cmd}_OK" >&3
    echo "LINE1" >&3
    echo "LINE2" >&3
    echo "END" >&3
    ;;
    *)
    echo "ERROR: Unknown command $cmd" >&3
    ;;
    esac
done <&4 &

echo ">INFO:OpenVPN Management Interface Version 1 -- type 'help' for more info" >&3
echo ">STATE:1522855903,CONNECTING,,,,,," >&3
echo ">STATE:1522855903,WAIT,,,,,," >&3
echo ">STATE:1522855903,AUTH,,,,,," >&3
echo ">STATE:1522855904,GET_CONFIG,,,,,," >&3
echo ">STATE:1522855904,ASSIGN_IP,,10.8.0.133,,,," >&3
echo ">STATE:1522855905,CONNECTED,SUCCESS,10.8.0.133,$serverIp,$serverPort,," >&3

killed=0
trap 'killed=1' SIGTERM SIGINT

while (( killed != 1))
do
   echo ">BYTECOUNT:36987,32252" >&3
   sleep 1
done


echo ">STATE:1522855911,EXITING,SIGTERM,,,,," >&3
exec 3>&-
exec 4>&-
echo "Killing nc instance with pid: ${nc_pid}" && kill -s SIGTERM ${nc_pid}
sleep 0.3
exit 0
