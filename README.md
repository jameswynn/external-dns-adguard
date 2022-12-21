# external-dns-adguard

This project aims to be a mostly compatible implementation of external-dns
but targeting AdGuard Home as the DNS provider. It is similar to and, in fact,
inspired by the [python version](https://github.com/NatiSayada/external-dns-adguard).

## Features

* Annotation-based configuration allows custom hostnames per entity
* Persistent database to more accurately detect changes
* Annotation defaults to be compatible with external-dns (external-dns.alpha.kubernetes.io/hostname),
  but can be configured to anything
* Small memory footprint ~10MB

## Configuration

The following environment variables can be configured:

| Variables        | Default                          | Description                                                                                                   |
|------------------|----------------------------------|---------------------------------------------------------------------------------------------------------------|
| MODE             | DEV                              | If `DEV` then it will use the default kubeconfig file; if `PROD` then it will use the cluster service account |
| DATABASE_FILE    | rules.db                         | The full path of the database file                                                                            |
| ADGUARD_URL      | localhost:8080                   | The URL of the AdGuard Home instance                                                                          |
| ADGUARD_SCHEME   | http                             | The protocol to use - either `http` or `https`                                                                |
| ADGUARD_USERNAME |                                  | The AdGuard Home username                                                                                     |
| ADGUARD_PASSWORD |                                  | The AdGuard Home password                                                                                     |
| ADGUARD_LOGGING  | false                            | Controls whether verbose feedback of the AdGuard client is enabled                                            |
| ANNOTATION       | external-dns.alpha.kubernetes.io | The annotation to pull the hostnames from                                                                     |

## Deployment

The application can be deployed with the sample deployment in [/deployment/deployment.yaml].

**NOTE:** The sample deployment manifest does not account for storage of the database, so a PVC should be created,
or it will lose track of the mappings. This is not catastrophic, but it will prevent it from detecting changes to
the hostname annotations, and it will leave behind stale entries in AdGuard.

## Examples

Configuring a service to point `sample.example.com` to `192.168.123`:

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    external-dns.alpha.kubernetes.io/hostname: sample.example.com
  name: sample
spec:
  loadBalancerIP: 192.168.1.123
  ports:
    - nodePort: 32700
      port: 9000
      protocol: UDP
      targetPort: 19132
  selector:
    app.kubernetes.io/instance: sample
    app.kubernetes.io/name: sample
  sessionAffinity: None
  type: LoadBalancer
```

Configuring an ingress to point `sample.example.com` at the ingress
controller's default IP:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    external-dns.alpha.kubernetes.io/hostname: sample.example.com
  name: sample
spec:
  ingressClassName: nginx
  rules:
    - host: sample.example.com
      http:
        paths:
          - backend:
              service:
                name: sample
                port:
                  name: http
            path: /
            pathType: Prefix
```

## License

This code is released under the Apache License 2.0.
