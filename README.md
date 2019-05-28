## SFTP Server with Google Cloud Storage backend

Runs an isolated, sandboxed SFTP server that only interacts with virtual backing storage on Google Cloud Storage (GCS).


Set the following environment variables, e.g.

```
SFTP_USERNAME=user123
SFTP_PASSWORD=kl5dfqpw3NXCZX0
SFTP_PORT=2022
SFTP_SERVER_KEY_PATH=/id_rsa
GCS_CREDENTIALS_FILE=/credentials.json
GCS_BUCKET=my-sftp-bucket
```

GET, PUT, STAT, LIST & MKDIR are currently the only methods implemented.