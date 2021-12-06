#!/bin/bash
#echo 101 adsl1 >> /etc/iproute2/rt_tables
#echo 102 adsl2 >> /etc/iproute2/rt_tables
createRule(){
    if [ $# -ne 2 ]; then
        echo "Missing parameters"
        exit
    fi
    DEV=$1
    TABLE=$2
    NET=$(ip route|grep $DEV|grep src|egrep -o [0-9]+\.[0-9]+\.[0-9]+\.[0-9]+\/[0-9]+)
    ROUTE=$(ip route |grep default|grep $DEV|egrep -o [0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)

    ip rule add from $NET table $TABLE
    ip route add $NET dev $DEV scope link table $TABLE
    ip route add default via $ROUTE dev $DEV table $TABLE
}

getAllDefault(){
    count=1
    for i in $(ip route|grep default|egrep -o "dev [a-Z0-9]+"|cut -d" " -f2);do
        createRule $i adsl$count
        count=$((count+1))
    done
}

getAllDefault