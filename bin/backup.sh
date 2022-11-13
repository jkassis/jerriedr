mkdir /var/multi/single/local-server-0/restore
rm /var/multi/single/local-server-0/restore/* /var/multi/single/local-server-0/backup/*
curl --output - -d '{ "UUID": "9db4caec-a449-4082-a1c3-ac82b4d25444", "Fn": "/v1/Backup", "Body": {} }' -H 'Content-Type: application/json' http://localhost:10000/raft/leader/write
cp -Rf /var/multi/single/local-server-0/backup/* /var/multi/single/local-server-0/restore
curl --output - -d '{ "UUID": "9db4caec-a449-4082-a1c3-ac82b4d25444", "Fn": "/v1/Restore/Dockie", "Body": {} }' -H 'Content-Type: application/json' http://localhost:10000/raft/leader/write
