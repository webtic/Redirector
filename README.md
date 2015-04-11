
Redirector
==========

An experiment in Go to write a HTTP Redirect service.

Sometimes you don't want a complete Apache or NGINX to just do redirects :)

It started as a simple process which read a file and did redirects, just to get a feeling for the Go language.

As things went smooth HTTP-wise I also added a web-interface just for fun.

As it is now its only suitable for a single trusted user and the process should run as root because it binds to port 80.

The config can either contains relative paths or a hostname, a whitespace and the url to redirect to.

Eg:

```
/some/url     http://google.com/
my.host.com/  http://yandex.com/

```

To make it really useful it would need a user-database and a way to store rules per user.

Using html/template instead of a string replace would probably be better too.
