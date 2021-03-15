# CoreDNS LocalStar

A plugin for CoreDNS server to help with local development.

## Use Case

Let's say `example.org` is your internal network domain, `host1.example.org` is
a developer machine with running single-node Kuberneters cluster, and you want
to give people in network an easy access to the Kubernetes services with good
hostnames, like `grafana.monitoring.host1.dev.example.org`. This can be done
without any extra tool, but when stuff is dynamic and developer machine come
and go, it is not an easy to manage.

This plugin can help you! With small Corefile like:

```
dev.example.org {
  localstar {
    to_zone example.org
  }
}
```

what you only need is to delegate `dev.example.org` zone to custom CoreDNS
instance with the plugin and you're all set.

When request `A grafana.monitoring.host1.dev.example.org` come to the LocalStar,
it is converted to `A host1.example.org` sub-request, responsible DNS server
is asked, and response to the original request is returned using data from
sub-request's response.
