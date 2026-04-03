# Changelog

## [0.0.19](https://github.com/namecheap/terraform-provider-spaceship/compare/v0.0.18...v0.0.19) (2026-04-03)


### Bug Fixes

* make records optional+computed and add initial fetching acceptance tests ([#42](https://github.com/namecheap/terraform-provider-spaceship/issues/42)) ([f5c81d9](https://github.com/namecheap/terraform-provider-spaceship/commit/f5c81d996bcf83ff1947688fc537dc1a17b0034f))

## [0.0.18](https://github.com/namecheap/terraform-provider-spaceship/compare/v0.0.17...v0.0.18) (2026-04-02)


### Bug Fixes

* correct doc comments for DNS client functions ([18e1d06](https://github.com/namecheap/terraform-provider-spaceship/commit/18e1d06b3dca21c34c87f707748fec42894bef0d))
* filter DNS records to only manage custom group records ([3a874f8](https://github.com/namecheap/terraform-provider-spaceship/commit/3a874f8e00f32240fabe88566fd13112acb9f132)), closes [#21](https://github.com/namecheap/terraform-provider-spaceship/issues/21)
* filter out only custom dns records ([#36](https://github.com/namecheap/terraform-provider-spaceship/issues/36)) ([46435fa](https://github.com/namecheap/terraform-provider-spaceship/commit/46435fa4df3b85ff8697c9cb8596c1b6e97f0935))
* lowercase SvcParams in HTTPS/SVCB record signature ([74c8e82](https://github.com/namecheap/terraform-provider-spaceship/commit/74c8e8286c525ce8d80ba31187cd010ac1cc88a9))
* use case-insensitive comparison for CAA record value in diff ([a33a27e](https://github.com/namecheap/terraform-provider-spaceship/commit/a33a27ec2e2fcfee761d1691f33e3efdf3408bab))

## [0.0.17](https://github.com/namecheap/terraform-provider-spaceship/compare/v0.0.16...v0.0.17) (2026-04-01)


### Bug Fixes

* trigger release with latest updates ([#32](https://github.com/namecheap/terraform-provider-spaceship/issues/32)) ([bf34a00](https://github.com/namecheap/terraform-provider-spaceship/commit/bf34a00e3a2c70f78859a6c6c3488b9384b83eea))

## [0.0.16](https://github.com/namecheap/terraform-provider-spaceship/compare/v0.0.15...v0.0.16) (2026-03-28)


### Bug Fixes

* handle eventual consistency for auto_renew and nameservers updates ([#23](https://github.com/namecheap/terraform-provider-spaceship/issues/23)) ([fa3afcc](https://github.com/namecheap/terraform-provider-spaceship/commit/fa3afccae64cc064b7326d5cdc217ce9267d89a3))

## [0.0.15](https://github.com/namecheap/terraform-provider-spaceship/compare/v0.0.14...v0.0.15) (2026-03-21)


### Bug Fixes

* make tests green ([d59d9c6](https://github.com/namecheap/terraform-provider-spaceship/commit/d59d9c68ef901ee2a7e3c4c2799fd70e119b46f1))
* upgrade GitHub Actions to Node.js 24 versions ([aa21be4](https://github.com/namecheap/terraform-provider-spaceship/commit/aa21be42541188024845cb43c799eea591cc1fcd))
* upgrade golangci-lint to v2.11 for action v9 compatibility ([71e6c51](https://github.com/namecheap/terraform-provider-spaceship/commit/71e6c5133632399991b35c0feb249b92be771ac6))

## [0.0.14](https://github.com/namecheap/terraform-provider-spaceship/compare/v0.0.13...v0.0.14) (2026-01-28)


### Bug Fixes

* create ci workflow with automatic docs generation ([9c01eaf](https://github.com/namecheap/terraform-provider-spaceship/commit/9c01eafee1034f3388b4ae8fbb2ca7e638b9d0d9))
