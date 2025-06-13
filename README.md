# Resolyte
service-gps

## Environment Variables

Create a `.env` file in the root directory of the project and add the following fields:

```ini
ENV=development
USE_HTTPS=false

GPS_HTTP_PORT=12001
GPS_GRPC_PORT=9090

SECRET_KEY=my-secret-key # must be at least 32 bytes

CORS_ORIGINS=http://localhost:5000
```

## HTTPS Setup

If `HTTPS=true` is set in your `.env` file, you must generate TLS certificates (`cert.pem` and `key.pem`) to enable secure communication.

To generate self-signed certificates:

```bash
# run from project root
make generate_cert
```

NOTE: self-signed certs won't be accepted by browsers unless you explicitly configure them to