```
# Hyphora

Derived from Hyphae (/ˈhʌɪfə/) -- the branching filaments that make up the mycelium of a fungus

### About
Something I built to help me remember stuff across my servers in my homelab 🌟
```
<p align="center">
  <img src="https://github.com/user-attachments/assets/8bdb3805-2fa5-423a-a338-e3644a5ed750" width="500"/>
</p>


I explained about it in detail [here !!](https://medium.com/@ams_132/building-hyphora-a-distributed-kv-store-with-a-bitcask-soul-3f44c4ff9ba8)

## Setup

```
git clone https://github.com/AMS003010/Hyphora.git
go build -o hyphora-node cmd/hyphora-node/main.go
```

<br/>

Start hyphora in all nodes

`Node 1`
```
./hyphora-node data1 <ip-address-of-node1>:9001 node1 8081
```

`Node 2`
```
./hyphora-node data2 <ip-address-of-node2>:9002 node2 8082
```

`Node 3`
```
./hyphora-node data3 <ip-address-of-node3>:9003 node3 8083
```

Add `Node 2` and `Node 3` as peers

```
curl "http://<ip-address-of-node1>:8081/addpeer?id=node2&addr=<ip-address-of-node2>:9002"
curl "http://<ip-address-of-node1>:8081/addpeer?id=node3&addr=<ip-address-of-node3>:9003"
```

You now have a distributed key-value store ready !!

<br/>

### Store a key-value

Always make write queries to the leader (here node1 is the leader)

```
curl --location 'http://<ip-address-of-node1>:<port-of-node1>/put' \
--header 'Content-Type: application/json' \
--data '{
    "key": "favourite_quote",
    "value": "It does not do to dwell on dreams and forget to live. – Albus Dumbledore, Harry Potter and the Philosopher’s Stone"
}'
```

### Get a value to a key

You can make a read query from any node

```
curl --location 'http://<ip-address-of-node2>:<port-of-node2>/get?key=hp1'
```

### Delete a key

Always make write queries to the leader (here node1 is the leader)

```
curl --location 'http://<ip-address-of-node1>:<port-of-node1>/del' \
--header 'Content-Type: application/json' \
--data '{
    "key": "pls6"
}'
```

### Replicate a local file

You can replicate a file from any node to other nodes as long as the node initiating the replication has the file locally. Any type of file will work.

```
curl --location 'http://<ip-address-of-node>:<port-of-node>/replicate' \
--header 'Content-Type: application/json' \
--data '{
    "path": "<absolute-path-of-file>"
}'
```

If `windows` an example of the path would be `D:\\Stuff\\Pics\\pic.jpeg`.
If `linux` an example of the path would be `/home/pi/Pics/pic.jpeg`.

### Fetch the replicated file

```
curl http://<ip-address-of-node>:<port-of-node>/download?key=secret.png --output pic.jpeg
```
