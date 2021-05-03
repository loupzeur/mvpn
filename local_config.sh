#echo 101 adsl1 >> /etc/iproute2/rt_tables
#echo 102 adsl2 >> /etc/iproute2/rt_tables

ip route add 192.168.1.0/24 dev wlp0s20u1 scope link table adsl1
ip route add default via 192.168.1.1 dev wlp0s20u1 table adsl1

ip route add 192.168.2.0/24 dev enp2s0 scope link table adsl2
ip route add default via 192.168.2.1 dev enp2s0 table adsl2

ip rule add from 192.168.1.0/24 table adsl1
ip rule add from 192.168.2.0/24 table adsl2

ip route add 192.168.42.0/24 dev enp0s20u2 scope link table adsl3
ip route add default via 192.168.42.47 dev enp0s20u2 table adsl3
ip rule add from 192.168.42.0/24 table adsl3
