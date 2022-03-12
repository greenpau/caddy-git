# caddy-git

<a href="https://github.com/greenpau/caddy-git/actions/" target="_blank"><img src="https://github.com/greenpau/caddy-git/workflows/build/badge.svg?branch=main"></a>
<a href="https://pkg.go.dev/github.com/greenpau/caddy-git" target="_blank"><img src="https://img.shields.io/badge/godoc-reference-blue.svg"></a>
<a href="https://caddy.community" target="_blank"><img src="https://img.shields.io/badge/community-forum-ff69b4.svg"></a>
<a href="https://caddyserver.com/docs/modules/git" target="_blank"><img src="https://img.shields.io/badge/caddydocs-git-green.svg"></a>

Git Plugin for [Caddy v2](https://github.com/caddyserver/caddy).

Inspired by [this comment](https://github.com/vrongmeal/caddygit/pull/5#issuecomment-1010440830).

Please ask questions either here or via LinkedIn. I am happy to help you! @greenpau

Please see other plugins:
* [caddy-security](https://github.com/greenpau/caddy-security)
* [caddy-trace](https://github.com/greenpau/caddy-trace)
* [caddy-systemd](https://github.com/greenpau/caddy-systemd)

<!-- begin-markdown-toc -->
## Table of Contents

* [Overview](#overview)
* [Getting Started](#getting-started)

<!-- end-markdown-toc -->

## Overview

The `caddy-git` allows updating a directory backed by a git repo.

## Getting Started

Configuration examples:
* [Public repo over HTTPS](./assets/config/Caddyfile)
* [Private or public repo over SSH with key-based authentication](./assets/config/ssh/Caddyfile)
* [Repo with Webhooks](./assets/config/webhook/Caddyfile)
* [Repo with post pull execution scripts](./assets/config/post_cmd_exec/Caddyfile)
* [Routeless config](./assets/config/routeless/Caddyfile)

For example, the following configuration sets up a definition for `authp.github.io`
repo. The request to `authp.myfiosgateway.com/update/authp.github.io` trigger
`git pull` of the `authp.github.io` repository.

```
{
  git {
    repo authp.github.io {
      base_dir /tmp
      url https://github.com/authp/authp.github.io.git
      branch gh-pages
      post pull exec {
        name Pager
        command /usr/bin/echo
        args "pulled authp.github.io repo"
      }
    }
  }
}

authp.myfiosgateway.com {
  route /version* {
    respond * "1.0.0" 200
  }
  route /update/authp.github.io {
    git update repo authp.github.io
  }
  route {
    file_server {
      root /tmp/authp.github.io
    }
  }
}
```

The cloning of the repository happens on startup. Additionally, the cloning
happens when `/update/authp.github.io` is being hit.

```
curl https://authp.myfiosgateway.com/update/authp.github.io
```
