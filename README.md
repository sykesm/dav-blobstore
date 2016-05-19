## Simple dav blob store for bosh

There are times when you want to share a bosh release but you really don't
want to deal with the expense of setting up an s3 or swift blob store up on
the internet. What you really want is a very simple blob store that provides
basic read access to a small community of users and authenticated write access
for yourself.

Okay - even if you don't have that problem, I did. And when I did have the
problem, I didn't want to deal with setting up the ruby "simple" blob store
with SSL and I discovered that the bosh cli doesn't actually work with
compliant WebDAV servers so, here we are.

### Building the server

Simply use `go get` to get the server and install it to `${GOPATH}/bin`:

```
go get github.com/sykesm/dav-blobstore
```

### Configuring the server

In order to use the server, you need to create a JSON configuration file and
point the server to it.

```json
{
    "blobs_path": "/path/to/blobs/root",
    "public_read": true,
    "cert_file": "/path/to/ssl/certificate",
    "key_file": "/path/to/server/key",
    "users": {
        "user": "password"
    }
}
```

`blobs_path` is the only required field and it points to the root directory of
the blob store.

`public_read` indicates whether or not `GET` and `HEAD` requests require
authentication. If omitted or `false`, you will want to define authorized
users in the `users` field.

`cert_file` and `key_file` are used to enable TLS. When both of these fields
are set, the server will only support https. Enabling TLS is recommended.

`users` is a map of key value pairs representing authorized users and their
passwords. Basic authentication is always used for operations other than `GET`
and `HEAD` -- regardless of the value of `public_read`.

### Running the server

Once the configuration file is ready, you simply launch the server with the
`-configFile` flag.  If you wish to control which address the server listens
on, you can set `-listenAddress`.

```
${GOPATH}/bin/dav-blobstore -listenAddress :14000 -configFile /user/local/etc/config.json
```

### Configuring bosh

In your bosh release, you'll need to point to your blob store in
`config/final.yaml`. You'll likely need to set `ssl_no_verify` since, as a
simple server, you're probably using a self-signed certificate...

``` yaml
---
final_name: my-release
blobstore:
  provider: dav
  options:
    endpoint: https://blobs.example.com:14000
    ssl_no_verify: true
```

For those that need to upload blobs to the store, you'll also need to generate
a `config/private.yml` that contains credentials for the store.

```yaml
---
blobstore:
  dav:
    user: user
    password: password
```

