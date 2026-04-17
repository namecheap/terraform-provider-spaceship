# Changelog

## [0.1.0](https://github.com/namecheap/terraform-provider-spaceship/compare/v0.0.20...v0.1.0) (2026-04-17)


### Features

* add validators and tests for alias record ([#61](https://github.com/namecheap/terraform-provider-spaceship/issues/61)) ([fbbd85c](https://github.com/namecheap/terraform-provider-spaceship/commit/fbbd85c5a5cb42b6b1173b946d824052f6bf6967))


### Bug Fixes

* add validation for a record ([#50](https://github.com/namecheap/terraform-provider-spaceship/issues/50)) ([aced141](https://github.com/namecheap/terraform-provider-spaceship/commit/aced141d509db52a4d3d42c0ee6f070c9350db08))
* add validation for aaaa record ([#58](https://github.com/namecheap/terraform-provider-spaceship/issues/58)) ([3ec57e4](https://github.com/namecheap/terraform-provider-spaceship/commit/3ec57e44d1d5b5ceba68f8ea111df7b5093b4e91))
* address code quality findings from AI scan ([#57](https://github.com/namecheap/terraform-provider-spaceship/issues/57)) ([83eeda2](https://github.com/namecheap/terraform-provider-spaceship/commit/83eeda2f1c0e4a570101d0c5e88c67bb6226b85e))
* **ci:** use default GITHUB_TOKEN for PR title check ([#52](https://github.com/namecheap/terraform-provider-spaceship/issues/52)) ([6317463](https://github.com/namecheap/terraform-provider-spaceship/commit/6317463d1d6ab686d25fd3448b4bf6d2f4b22d2c))

## [0.0.20](https://github.com/namecheap/terraform-provider-spaceship/compare/v0.0.19...v0.0.20) (2026-04-08)


### Bug Fixes

* add warning for using force param ([#45](https://github.com/namecheap/terraform-provider-spaceship/issues/45)) ([5269d12](https://github.com/namecheap/terraform-provider-spaceship/commit/5269d120b847cda361c77700f80fb9bd033baaa8))

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
