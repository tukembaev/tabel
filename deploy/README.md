# Deploy without Docker

Copy Linux binaries built by `./scripts/build-linux.sh`, then on the server:

```bash
sudo ./scripts/install-server.sh
sudo nano /etc/tabel/tabel.env
# migrate with goose
sudo systemctl start tabel-api tabel-worker
```

Units:

- [systemd/tabel-api.service](systemd/tabel-api.service)
- [systemd/tabel-worker.service](systemd/tabel-worker.service)

Process listens on `HTTP_ADDR` on the host network. No container networking.
