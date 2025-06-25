# Resolyte
service-gps

## Environment Variables

Create a `.env` file in the root directory of the project and add the following fields:

```ini
ENV=development
USE_HTTPS=false

GPS_HTTP_PORT=12003
GOVERNANCE_GRPC_ADDR=governance:9090

SECRET_KEY=my-secret-key # must be at least 32 bytes

# Mongo configuration
MONGO_HOST=mongo
MONGO_PORT=27017
MONGODB_INITDB_ROOT_USERNAME=admin
MONGODB_INITDB_ROOT_PASSWORD=mymongopassword

# AWS
AWS_ACCESS_KEY_ID=my-access-key-id
AWS_SECRET_ACCESS_KEY=my-secret-access-key
AWS_REGION=aws-region

CORS_ORIGINS=http://localhost:5000
```

## Github Token

Create a `.github_token` file at the project root

```bash
echo "ghp_xxxYourTokenHere" > .github_token
```

## HTTPS Setup

If `HTTPS=true` is set in your `.env` file, you must generate TLS certificates (`cert.pem` and `key.pem`) to enable secure communication.

To generate self-signed certificates:

```bash
# run from project root
make generate_cert
```

NOTE: self-signed certs won't be accepted by browsers unless you explicitly configure them to