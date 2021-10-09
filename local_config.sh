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



#other machine
ip rule show
ip route show table adsl2

sudo ip rule add from 192.168.248.0/24 table adsl2
sudo ip route add 192.168.248.0/24 dev wlp3s0 table adsl2
sudo ip route add default via 192.168.248.128 dev wlp3s0 table adsl2

sudo ip rule add from 192.168.96.0/24 table adsl1
sudo ip route add 192.168.96.0/24 dev wlp3s0 table adsl1
sudo ip route add default via 192.168.96.128 dev wlp3s0 table adsl1


[69 0 0 52 239 4 64 0 64 6 206 112 10 9 0 2 179 60 192 7]
2021/10/08 23:14:42 IP packet to InChan : 
[69 0 0 84 137 109 64 0 64 1 157 39 10 9 0 2 10 9 0 1]
2021/10/08 23:14:42 IP packet to InChan : 
[69 0 0 52 223 159 64 0 64 6 112 127 10 9 0 2 35 166 188 244]
2021/10/08 23:14:42 IP packet to Iface : 
[69 0 0 84 168 233 0 0 64 1 189 171 10 9 0 1 10 9 0 2]
2021/10/08 23:14:43 IP packet to InChan : 
[69 0 0 84 138 38 64 0 64 1 156 110 10 9 0 2 10 9 0 1]
2021/10/08 23:14:43 IP packet to Iface : 
[69 0 0 84 169 8 0 0 64 1 189 140 10 9 0 1 10 9 0 2]
2021/10/08 23:14:44 IP packet to InChan : 
[69 0 0 84 138 244 64 0 64 1 155 160 10 9 0 2 10 9 0 1]
2021/10/08 23:14:44 IP packet to Iface : 
[69 0 0 84 169 64 0 0 64 1 189 84 10 9 0 1 10 9 0 2]
2021/10/08 23:14:44 IP packet to InChan : 
[69 0 0 52 247 249 64 0 64 6 217 171 10 9 0 2 44 239 50 37]
2021/10/08 23:14:45 IP packet to InChan : 
[69 0 0 84 139 189 64 0 64 1 154 215 10 9 0 2 10 9 0 1]