# Changelog

## [0.19.3](https://github.com/SchoolyB/EZ/compare/v0.19.2...v0.19.3) (2025-12-11)


### Bug Fixes

* remove [@string](https://github.com/string) alias and add module suggestion hints ([#502](https://github.com/SchoolyB/EZ/issues/502)) ([#503](https://github.com/SchoolyB/EZ/issues/503)) ([53122a1](https://github.com/SchoolyB/EZ/commit/53122a1350104efae0bed292167851a34e04d4a6))

## [0.19.2](https://github.com/SchoolyB/EZ/compare/v0.19.1...v0.19.2) (2025-12-11)


### Bug Fixes

* show update notification immediately on all commands ([#499](https://github.com/SchoolyB/EZ/issues/499)) ([c436b2f](https://github.com/SchoolyB/EZ/commit/c436b2fe91d723317f54c5021499f45f94934ac7)), closes [#498](https://github.com/SchoolyB/EZ/issues/498)

## [0.19.1](https://github.com/SchoolyB/EZ/compare/v0.19.0...v0.19.1) (2025-12-11)


### Bug Fixes

* add path validation to archive extraction functions ([#487](https://github.com/SchoolyB/EZ/issues/487)) ([#495](https://github.com/SchoolyB/EZ/issues/495)) ([974250a](https://github.com/SchoolyB/EZ/commit/974250a0f45c8cbee29039c8b5e8ed002283f736))

## [0.19.0](https://github.com/SchoolyB/EZ/compare/v0.18.1...v0.19.0) (2025-12-11)


### Features

* add when/is switch statements ([#179](https://github.com/SchoolyB/EZ/issues/179)) ([#491](https://github.com/SchoolyB/EZ/issues/491)) ([4f0c068](https://github.com/SchoolyB/EZ/commit/4f0c0689c8efcf3b2dbf8bd08764fb5b0fa510eb))

## [0.18.1](https://github.com/SchoolyB/EZ/compare/v0.18.0...v0.18.1) (2025-12-11)


### Bug Fixes

* correct error codes for immutable variable/map errors ([#488](https://github.com/SchoolyB/EZ/issues/488)) ([727ce1b](https://github.com/SchoolyB/EZ/commit/727ce1bce576f0628aef67c1e7dd500a4c24e7d1))

## [0.18.0](https://github.com/SchoolyB/EZ/compare/v0.17.2...v0.18.0) (2025-12-10)


### Features

* add default parameter values for functions ([#312](https://github.com/SchoolyB/EZ/issues/312)) ([#485](https://github.com/SchoolyB/EZ/issues/485)) ([933ca6f](https://github.com/SchoolyB/EZ/commit/933ca6fe278654cfea9a1172e4a3eb0419f05f58))

## [0.17.2](https://github.com/SchoolyB/EZ/compare/v0.17.1...v0.17.2) (2025-12-10)


### Bug Fixes

* auto-elevate to sudo when update needs root permissions ([#483](https://github.com/SchoolyB/EZ/issues/483)) ([8cfa5c3](https://github.com/SchoolyB/EZ/commit/8cfa5c3d2814d8ec2fa7c2593ab405829e80bdb9))

## [0.17.1](https://github.com/SchoolyB/EZ/compare/v0.17.0...v0.17.1) (2025-12-10)


### Bug Fixes

* handle .tar.gz/.zip archives in ez update ([#481](https://github.com/SchoolyB/EZ/issues/481)) ([b7d4314](https://github.com/SchoolyB/EZ/commit/b7d43143472a1436037fa79e63efb396e4e3126f))

## [0.17.0](https://github.com/SchoolyB/EZ/compare/v0.16.15...v0.17.0) (2025-12-10)


### Features

* add update checker and `ez update` command ([#478](https://github.com/SchoolyB/EZ/issues/478)) ([#479](https://github.com/SchoolyB/EZ/issues/479)) ([a33c92b](https://github.com/SchoolyB/EZ/commit/a33c92be20dc111bf0d8a1529b7a06b7f3b7522b))

## [0.16.15](https://github.com/SchoolyB/EZ/compare/v0.16.14...v0.16.15) (2025-12-10)


### Bug Fixes

* show correct file location for multi-file module errors ([#466](https://github.com/SchoolyB/EZ/issues/466)) ([#476](https://github.com/SchoolyB/EZ/issues/476)) ([b187e1a](https://github.com/SchoolyB/EZ/commit/b187e1a6b8e98347acc541775eb27d6d59c98869))

## [0.16.14](https://github.com/SchoolyB/EZ/compare/v0.16.13...v0.16.14) (2025-12-10)


### Bug Fixes

* show file location in module not found error ([#461](https://github.com/SchoolyB/EZ/issues/461)) ([#474](https://github.com/SchoolyB/EZ/issues/474)) ([a6eec23](https://github.com/SchoolyB/EZ/commit/a6eec2353111ce07adfca2733e93d58afe352c3d))

## [0.16.13](https://github.com/SchoolyB/EZ/compare/v0.16.12...v0.16.13) (2025-12-10)


### Bug Fixes

* suppress module name warning for files in matching directories ([#464](https://github.com/SchoolyB/EZ/issues/464)) ([#472](https://github.com/SchoolyB/EZ/issues/472)) ([6be1062](https://github.com/SchoolyB/EZ/commit/6be106280f0cd375a9a93dac8197b6ab4fe34c2e))

## [0.16.12](https://github.com/SchoolyB/EZ/compare/v0.16.11...v0.16.12) (2025-12-10)


### Bug Fixes

* handle module-prefixed types in type compatibility checks ([#463](https://github.com/SchoolyB/EZ/issues/463)) ([#469](https://github.com/SchoolyB/EZ/issues/469)) ([3c597de](https://github.com/SchoolyB/EZ/commit/3c597dec48fafc0a2cdb9c2fadee796f04bb73ac))

## [0.16.11](https://github.com/SchoolyB/EZ/compare/v0.16.10...v0.16.11) (2025-12-10)


### Bug Fixes

* resolve uppercase constants being parsed as struct literals ([#462](https://github.com/SchoolyB/EZ/issues/462)) ([a406de3](https://github.com/SchoolyB/EZ/commit/a406de3d496cf89c4f405182d5392b16405c6fc4))

## [0.16.10](https://github.com/SchoolyB/EZ/compare/v0.16.9...v0.16.10) (2025-12-09)


### Bug Fixes

* bug fixes batch - parser, stdlib, interpreter, and enum map keys ([#454](https://github.com/SchoolyB/EZ/issues/454)) ([f90e10f](https://github.com/SchoolyB/EZ/commit/f90e10fd02b3946ac37341956b453db744198017)), closes [#452](https://github.com/SchoolyB/EZ/issues/452)

## [0.16.9](https://github.com/SchoolyB/EZ/compare/v0.16.8...v0.16.9) (2025-12-08)


### Bug Fixes

* implement arbitrary precision integers for i128/u128 support ([#437](https://github.com/SchoolyB/EZ/issues/437)) ([#440](https://github.com/SchoolyB/EZ/issues/440)) ([af4cdbf](https://github.com/SchoolyB/EZ/commit/af4cdbfd42edcd87d5982032d06cc913cc01e75f))

## [0.16.8](https://github.com/SchoolyB/EZ/compare/v0.16.7...v0.16.8) (2025-12-08)


### Bug Fixes

* integration test overhaul and bug fixes ([#435](https://github.com/SchoolyB/EZ/issues/435)) ([a4ad4e8](https://github.com/SchoolyB/EZ/commit/a4ad4e8549490a5cf03b8d54bd9b48511a3af264))

## [0.16.7](https://github.com/SchoolyB/EZ/compare/v0.16.6...v0.16.7) (2025-12-08)


### Bug Fixes

* detect integer overflow in negation and division ([#424](https://github.com/SchoolyB/EZ/issues/424)) ([e83bdfe](https://github.com/SchoolyB/EZ/commit/e83bdfef81ab73cad1d2b5d9062b1a8be071f2d8))

## [0.16.6](https://github.com/SchoolyB/EZ/compare/v0.16.5...v0.16.6) (2025-12-08)


### Bug Fixes

* stdlib functions preserve input type ([#422](https://github.com/SchoolyB/EZ/issues/422)) ([1e91801](https://github.com/SchoolyB/EZ/commit/1e91801ff35219dbdee45e87ae6a4af4be9453eb)), closes [#404](https://github.com/SchoolyB/EZ/issues/404)

## [0.16.5](https://github.com/SchoolyB/EZ/compare/v0.16.4...v0.16.5) (2025-12-08)


### Bug Fixes

* float division by zero returns INF per IEEE 754 ([#420](https://github.com/SchoolyB/EZ/issues/420)) ([744cf75](https://github.com/SchoolyB/EZ/commit/744cf75b2f9188103cfd4e4734900b364c3a4781)), closes [#402](https://github.com/SchoolyB/EZ/issues/402)

## [0.16.4](https://github.com/SchoolyB/EZ/compare/v0.16.3...v0.16.4) (2025-12-08)


### Bug Fixes

* allow byte to int conversion with int() ([#418](https://github.com/SchoolyB/EZ/issues/418)) ([33dd463](https://github.com/SchoolyB/EZ/commit/33dd463d19ed16fe36f03fa3b9aba641c56edffb)), closes [#403](https://github.com/SchoolyB/EZ/issues/403)

## [0.16.3](https://github.com/SchoolyB/EZ/compare/v0.16.2...v0.16.3) (2025-12-08)


### Bug Fixes

* reject nil assignment to non-nullable types ([#416](https://github.com/SchoolyB/EZ/issues/416)) ([40e82e3](https://github.com/SchoolyB/EZ/commit/40e82e3f7addc2410ffa0a3ec062c4f5184a34f0)), closes [#407](https://github.com/SchoolyB/EZ/issues/407)

## [0.16.2](https://github.com/SchoolyB/EZ/compare/v0.16.1...v0.16.2) (2025-12-08)


### Bug Fixes

* detect mixed-type enums at compile time ([#414](https://github.com/SchoolyB/EZ/issues/414)) ([e1b8c63](https://github.com/SchoolyB/EZ/commit/e1b8c63fd7c841d8d3aea2ad48b0d2d31483e6ca)), closes [#410](https://github.com/SchoolyB/EZ/issues/410)

## [0.16.1](https://github.com/SchoolyB/EZ/compare/v0.16.0...v0.16.1) (2025-12-08)


### Bug Fixes

* make array functions modify in-place and fix remove() API ([#412](https://github.com/SchoolyB/EZ/issues/412)) ([f6890b8](https://github.com/SchoolyB/EZ/commit/f6890b82553549702b3d52186843ca7c11e2860d))

## [0.16.0](https://github.com/SchoolyB/EZ/compare/v0.15.2...v0.16.0) (2025-12-07)


### Features

* add line editor for REPL with arrow key navigation and history ([#400](https://github.com/SchoolyB/EZ/issues/400)) ([bce7343](https://github.com/SchoolyB/EZ/commit/bce7343118564ec2f8b7b2e79b2eec3cd0b49711))

## [0.15.2](https://github.com/SchoolyB/EZ/compare/v0.15.1...v0.15.2) (2025-12-07)


### Bug Fixes

* resolve stdlib bugs for arrays, strings, and math modules ([#398](https://github.com/SchoolyB/EZ/issues/398)) ([b278ce6](https://github.com/SchoolyB/EZ/commit/b278ce6744cba9c3e8bf22237038f8cce398bda8))

## [0.15.1](https://github.com/SchoolyB/EZ/compare/v0.15.0...v0.15.1) (2025-12-07)


### Bug Fixes

* resolve several interpreter bugs ([5c083d3](https://github.com/SchoolyB/EZ/commit/5c083d33e3d0f18cef30049f3f57e184ce131d78))

## [0.15.0](https://github.com/SchoolyB/EZ/compare/v0.14.10...v0.15.0) (2025-12-07)


### Features

* replace [@ignore](https://github.com/ignore) with _ (underscore) blank identifier ([#376](https://github.com/SchoolyB/EZ/issues/376)) ([66111be](https://github.com/SchoolyB/EZ/commit/66111beaf80ca72dab092a9b0e62b1fa5e4da58a))

## [0.14.10](https://github.com/SchoolyB/EZ/compare/v0.14.9...v0.14.10) (2025-12-07)


### Bug Fixes

* resolve multiple bugs in stdlib, lexer, and parser ([#374](https://github.com/SchoolyB/EZ/issues/374)) ([c4be966](https://github.com/SchoolyB/EZ/commit/c4be9660ed9ced0f6a812d5a98ef5eea9389a813))

## [0.14.9](https://github.com/SchoolyB/EZ/compare/v0.14.8...v0.14.9) (2025-12-05)


### Bug Fixes

* **interpreter:** nested mutable parameter forwarding now works correctly ([#367](https://github.com/SchoolyB/EZ/issues/367)) ([5762939](https://github.com/SchoolyB/EZ/commit/57629390fb7bd4a9c53b55d9a259f853841de901)), closes [#338](https://github.com/SchoolyB/EZ/issues/338)

## [0.14.8](https://github.com/SchoolyB/EZ/compare/v0.14.7...v0.14.8) (2025-12-05)


### Bug Fixes

* **typechecker:** type errors now report correct source location ([#362](https://github.com/SchoolyB/EZ/issues/362)) ([13cdb4e](https://github.com/SchoolyB/EZ/commit/13cdb4ecd40f282c01749bf0f0ff65a22b6ea011))

## [0.14.7](https://github.com/SchoolyB/EZ/compare/v0.14.6...v0.14.7) (2025-12-05)


### Bug Fixes

* **parser:** handle minimum int64 literal (-9223372036854775808) ([#360](https://github.com/SchoolyB/EZ/issues/360)) ([8c75b43](https://github.com/SchoolyB/EZ/commit/8c75b434ff40f3e11066edbeb13a563e03a43b76))

## [0.14.6](https://github.com/SchoolyB/EZ/compare/v0.14.5...v0.14.6) (2025-12-05)


### Bug Fixes

* **parser:** enum member access in if condition no longer causes parser error ([#358](https://github.com/SchoolyB/EZ/issues/358)) ([6e0c5de](https://github.com/SchoolyB/EZ/commit/6e0c5de28bc3ba4e287bd1bbbb92ac5abd126fc8))

## [0.14.5](https://github.com/SchoolyB/EZ/compare/v0.14.4...v0.14.5) (2025-12-05)


### Bug Fixes

* **evaluator:** isTruthy now checks Boolean.Value instead of pointer identity ([#354](https://github.com/SchoolyB/EZ/issues/354)) ([18edd78](https://github.com/SchoolyB/EZ/commit/18edd78036eb4c57637303283729496f2d3c03b1))

## [0.14.4](https://github.com/SchoolyB/EZ/compare/v0.14.3...v0.14.4) (2025-12-05)


### Bug Fixes

* **ci:** auto-merge release PRs even when a release is created in same run ([eabee09](https://github.com/SchoolyB/EZ/commit/eabee09ea4800cc6618924636b5ed1511b2f25e8))
* **stdlib:** arrays.insert() and arrays.remove() now modify array in-place ([3ad569a](https://github.com/SchoolyB/EZ/commit/3ad569a9a0e3b3b39664335551d7d0e2cfbf89e2))
* **stdlib:** arrays.insert() and arrays.remove() now modify array in-place ([f26797d](https://github.com/SchoolyB/EZ/commit/f26797ddcd2a2be3ba23f06384b3b1e3ceb4330a))

## [0.14.3](https://github.com/SchoolyB/EZ/compare/v0.14.2...v0.14.3) (2025-12-05)


### Bug Fixes

* **evaluator:** implement short-circuit evaluation for && and || ([a9e4fe1](https://github.com/SchoolyB/EZ/commit/a9e4fe13877775af1a2f7859cc218484a31909b2))
* **evaluator:** implement short-circuit evaluation for && and || ([2f6f0a6](https://github.com/SchoolyB/EZ/commit/2f6f0a6bdc10484e58ade9b54c2a45ca4acd3a25))

## [0.14.2](https://github.com/SchoolyB/EZ/compare/v0.14.1...v0.14.2) (2025-12-05)


### Bug Fixes

* **typechecker:** handle array and map types in struct fields ([0455540](https://github.com/SchoolyB/EZ/commit/04555409b90e3bcc90825f2915348eecc84e6b40))
* **typechecker:** handle array and map types in struct fields ([0d1cd53](https://github.com/SchoolyB/EZ/commit/0d1cd53cd6d738c969c4beccfad56ba49f18ddd1))

## [0.14.1](https://github.com/SchoolyB/EZ/compare/v0.14.0...v0.14.1) (2025-12-04)


### Bug Fixes

* **interpreter:** add recursion depth limit to prevent stack overflow ([ccda7b7](https://github.com/SchoolyB/EZ/commit/ccda7b779bc8534b00f3e6a5f8396b4a7c574231)), closes [#327](https://github.com/SchoolyB/EZ/issues/327)
* **lexer:** preserve line numbers in string interpolation expressions ([5a236b7](https://github.com/SchoolyB/EZ/commit/5a236b72949fb37a54a4acdcb65cbe8069978f07)), closes [#323](https://github.com/SchoolyB/EZ/issues/323)
* **parser:** block reserved names as function parameter names ([1a52eb0](https://github.com/SchoolyB/EZ/commit/1a52eb00148ccf97b2e80684183d541335a12985))
* **parser:** disallow import statements inside blocks and after declarations ([19118f6](https://github.com/SchoolyB/EZ/commit/19118f6eddcecd4632c9f1d893fe78ab90251d51)), closes [#324](https://github.com/SchoolyB/EZ/issues/324)
* **parser:** prevent nil pointer crash when using 'const' as identifier ([e2177b3](https://github.com/SchoolyB/EZ/commit/e2177b389aed3acd022e8214387146609d1b6e17)), closes [#317](https://github.com/SchoolyB/EZ/issues/317)
* **parser:** use helpful error when keywords used as identifiers ([614d2bd](https://github.com/SchoolyB/EZ/commit/614d2bda79524de9348734bbaa42c9be2ca86f0b))
* **parser:** use helpful error when keywords used as identifiers ([5ffb6e3](https://github.com/SchoolyB/EZ/commit/5ffb6e393445ae14bf806bbaa1f3baa39174746a)), closes [#326](https://github.com/SchoolyB/EZ/issues/326)
* **parser:** use specific error codes for struct/enum reserved names ([c86b903](https://github.com/SchoolyB/EZ/commit/c86b9038ca1e85e46e77b93d90a8a1d56236ec4a))
* **parser:** validate enum value names against reserved keywords ([1a3f9b1](https://github.com/SchoolyB/EZ/commit/1a3f9b18df2107bd2806fdd9b525461ffd3f2c10)), closes [#322](https://github.com/SchoolyB/EZ/issues/322)
* **parser:** validate struct field names against reserved keywords ([a077535](https://github.com/SchoolyB/EZ/commit/a077535e905202e19dc34fd696f02e4bc51f59d9)), closes [#321](https://github.com/SchoolyB/EZ/issues/321)
* **typechecker:** block user-defined types/functions as parameter names ([2bea2dc](https://github.com/SchoolyB/EZ/commit/2bea2dc6377f246adb34601f1b0e0ab058b4868a))
* **typechecker:** enforce static typing for type-inferred variables ([e1a4437](https://github.com/SchoolyB/EZ/commit/e1a443768b3f0afce1aab733f868fa5ff83fedb7)), closes [#329](https://github.com/SchoolyB/EZ/issues/329)
* **typechecker:** make missing return statement an error instead of warning ([1338860](https://github.com/SchoolyB/EZ/commit/1338860d0ebabe392f05f3b311dc2d6981960a69)), closes [#318](https://github.com/SchoolyB/EZ/issues/318)
* **typechecker:** use correct error code for missing main function ([92b3463](https://github.com/SchoolyB/EZ/commit/92b3463ad86ecb87b809a23ae6ee3cefe66495c3)), closes [#328](https://github.com/SchoolyB/EZ/issues/328)
* **typechecker:** validate member access on non-struct types ([df0daee](https://github.com/SchoolyB/EZ/commit/df0daee12fe43410abce26e41b0811adaeac3936)), closes [#313](https://github.com/SchoolyB/EZ/issues/313)

## [0.14.0](https://github.com/SchoolyB/EZ/compare/v0.13.0...v0.14.0) (2025-12-04)


### Features

* **stdlib:** add [@random](https://github.com/random) module with comprehensive random functions ([0f2f981](https://github.com/SchoolyB/EZ/commit/0f2f981b0ef78b90e653190100d521f7b91d6a3f)), closes [#309](https://github.com/SchoolyB/EZ/issues/309)
* **stdlib:** add command execution functions to [@os](https://github.com/os) ([b8b2e04](https://github.com/SchoolyB/EZ/commit/b8b2e04475d8a2c31b751fa8a2cbe673d574deb2))
* **stdlib:** add filesystem utilities to [@io](https://github.com/io) ([bf0e035](https://github.com/SchoolyB/EZ/commit/bf0e0359f1c01236e73f706e497329e5506284c5)), closes [#287](https://github.com/SchoolyB/EZ/issues/287)
* **stdlib:** add math.log_base() for arbitrary base logarithms ([7b304f6](https://github.com/SchoolyB/EZ/commit/7b304f6fed3fd54247daaa74288a4f7eb22ef404))
* **stdlib:** add new string utility functions to [@strings](https://github.com/strings) ([7cce6fd](https://github.com/SchoolyB/EZ/commit/7cce6fd96551c8ebdc91c055a607ed439e35e7cf)), closes [#285](https://github.com/SchoolyB/EZ/issues/285)
* **stdlib:** expand standard library with new modules and functions ([401203e](https://github.com/SchoolyB/EZ/commit/401203eb730037dfc3579b810b18097d77617fc7))

## [0.13.0](https://github.com/SchoolyB/EZ/compare/v0.12.0...v0.13.0) (2025-12-04)


### Features

* add error() constructor for user-defined errors ([95a85fe](https://github.com/SchoolyB/EZ/commit/95a85fed06360177d671cb18dedd1bf55e465be2))
* add error() constructor for user-defined errors ([#292](https://github.com/SchoolyB/EZ/issues/292)) ([4f2bc5e](https://github.com/SchoolyB/EZ/commit/4f2bc5eed24b63d31d9b3bc76c44035260a74434))

## [0.12.0](https://github.com/SchoolyB/EZ/compare/v0.11.3...v0.12.0) (2025-12-04)


### Features

* allow type inference for const declarations ([279afc8](https://github.com/SchoolyB/EZ/commit/279afc80d91d072a3b35aa85372bd3ea9ae6c893))
* allow type inference for const declarations ([#302](https://github.com/SchoolyB/EZ/issues/302)) ([ff94be9](https://github.com/SchoolyB/EZ/commit/ff94be94f855368dc08f67efe0cee3d398a0b8df))

## [0.11.3](https://github.com/SchoolyB/EZ/compare/v0.11.2...v0.11.3) (2025-12-04)


### Bug Fixes

* struct mutability now respects temp/const declaration ([e91d7d9](https://github.com/SchoolyB/EZ/commit/e91d7d998ced22e2faaea8d4fcdd56d27d73d75c))
* struct mutability now respects temp/const declaration ([#298](https://github.com/SchoolyB/EZ/issues/298)) ([66be5c0](https://github.com/SchoolyB/EZ/commit/66be5c04da3c65a11abcf9bf62ddffbd2cd22cc8))

## [0.11.2](https://github.com/SchoolyB/EZ/compare/v0.11.1...v0.11.2) (2025-12-04)


### Bug Fixes

* CI REPL test failing on macOS ([42cb93c](https://github.com/SchoolyB/EZ/commit/42cb93cccd995d927606e05a26fdf02e145afba2))
* CI REPL test failing on macOS ([#299](https://github.com/SchoolyB/EZ/issues/299)) ([8b14f75](https://github.com/SchoolyB/EZ/commit/8b14f754baf770a2bca091fa9748fa342d8b7de7))

## [0.11.1](https://github.com/SchoolyB/EZ/compare/v0.11.0...v0.11.1) (2025-12-03)


### Bug Fixes

* allow optional parentheses in for loops ([#293](https://github.com/SchoolyB/EZ/issues/293)) ([d86b71c](https://github.com/SchoolyB/EZ/commit/d86b71cb48281f221f19bf0aa40e98e3f67ac9c2))
* allow optional parentheses in for loops ([#293](https://github.com/SchoolyB/EZ/issues/293)) ([16c968d](https://github.com/SchoolyB/EZ/commit/16c968d5b03fc08de82a4589207f8852b0ad8841))

## [0.11.0](https://github.com/SchoolyB/EZ/compare/v0.10.0...v0.11.0) (2025-12-03)


### Features

* implement char() type conversion function ([d7362bf](https://github.com/SchoolyB/EZ/commit/d7362bf42bb6539a9592e95dcf8710b3965d9aad))
* implement char() type conversion function ([#100](https://github.com/SchoolyB/EZ/issues/100)) ([e23c6d3](https://github.com/SchoolyB/EZ/commit/e23c6d35a67488fc42c4355f98136b9223e32e61))

## [0.10.0](https://github.com/SchoolyB/EZ/compare/v0.9.0...v0.10.0) (2025-12-03)


### Features

* remove arrays.copy() and maps.copy() in favor of copy() builtin ([#269](https://github.com/SchoolyB/EZ/issues/269)) ([aa2de1d](https://github.com/SchoolyB/EZ/commit/aa2de1dbe8304eaea2950aa60c2dd8d02e537acd))

## [0.9.0](https://github.com/SchoolyB/EZ/compare/v0.8.3...v0.9.0) (2025-12-03)


### Features

* add `copy()` builtin for explicit value semantics ([051a752](https://github.com/SchoolyB/EZ/commit/051a7529922b92768cb26ecf5bfc08cb4b04b3bd))
* add copy() builtin for explicit value semantics ([#265](https://github.com/SchoolyB/EZ/issues/265)) ([4681cdd](https://github.com/SchoolyB/EZ/commit/4681cddcd4009f7157fddfe53df9be013bcde503))

## [0.8.3](https://github.com/SchoolyB/EZ/compare/v0.8.2...v0.8.3) (2025-12-03)


### Bug Fixes

* fixed-size array indexing returns correct element type ([4c7211f](https://github.com/SchoolyB/EZ/commit/4c7211fe3edfb4ef4f3188cb9c5b65a12bb7a368))
* fixed-size array indexing returns correct element type ([#267](https://github.com/SchoolyB/EZ/issues/267)) ([92eeb28](https://github.com/SchoolyB/EZ/commit/92eeb2809e33d9a39eed3ad6f00cd6e43fa72634))
* Handle comma in array type when extracting element type in inferIndexType. ([92eeb28](https://github.com/SchoolyB/EZ/commit/92eeb2809e33d9a39eed3ad6f00cd6e43fa72634))

## [0.8.2](https://github.com/SchoolyB/EZ/compare/v0.8.1...v0.8.2) (2025-12-03)


### Bug Fixes

* prevent modification of const struct fields ([20074af](https://github.com/SchoolyB/EZ/commit/20074af830d6828088817465a6e5b1410e39cca2))
* prevent modification of const struct fields ([#266](https://github.com/SchoolyB/EZ/issues/266)) ([17b033a](https://github.com/SchoolyB/EZ/commit/17b033a466f6591c211b64bc86be051c96201fd5))

## [0.8.1](https://github.com/SchoolyB/EZ/compare/v0.8.0...v0.8.1) (2025-12-03)


### Bug Fixes

* remove deprecated lowercase math constants ([9b7b238](https://github.com/SchoolyB/EZ/commit/9b7b2384fe189c6b4f1d1dd941ac8b5d2f7ed155))
* remove deprecated lowercase math constants ([#260](https://github.com/SchoolyB/EZ/issues/260)) ([7e182fd](https://github.com/SchoolyB/EZ/commit/7e182fda260c9bd49a274939ac1656b1d3cafc41))

## [0.8.0](https://github.com/SchoolyB/EZ/compare/v0.7.1...v0.8.0) (2025-12-03)


### Features

* add mutable parameters with & syntax ([7fdeb24](https://github.com/SchoolyB/EZ/commit/7fdeb24241a643a050beaeceb8cd1b9f600be9ca))
* add mutable parameters with & syntax ([#268](https://github.com/SchoolyB/EZ/issues/268)) ([5075f6f](https://github.com/SchoolyB/EZ/commit/5075f6fbdc888dbf3cf3dbded0dda7928f884b32))

## [0.7.1](https://github.com/SchoolyB/EZ/compare/v0.7.0...v0.7.1) (2025-12-02)


### Bug Fixes

* **ci:** sync release-please manifest to v0.7.0 ([d908c55](https://github.com/SchoolyB/EZ/commit/d908c55a0aee63d8e9644bd03fbb73aeed901fb4))
* **ci:** sync release-please manifest to v0.7.0 ([dd05c6e](https://github.com/SchoolyB/EZ/commit/dd05c6e13a6cb68479b800e972a40e87fa460229))

## [0.7.0](https://github.com/SchoolyB/EZ/compare/v0.6.1...v0.7.0) (2025-12-02)


### Features

* **io:** add byte I/O and atomic writes ([ab37220](https://github.com/SchoolyB/EZ/commit/ab37220ea5343df91b40856fe565858fa73149cf))
* **io:** add file handles, constants, and convenience functions ([870df20](https://github.com/SchoolyB/EZ/commit/870df203ac6cac574a25b49b6bac71168abcf2d4))
* **io:** byte operations, file handles & enhancements ([7832642](https://github.com/SchoolyB/EZ/commit/783264228085fe2467954fdb8f0b140578777c16))


### Bug Fixes

* **io:** handle close errors on writable file handles ([aebdd96](https://github.com/SchoolyB/EZ/commit/aebdd9679296b87152c229aae5a0cc05ede728ef))

## [0.6.1](https://github.com/SchoolyB/EZ/compare/v0.6.0...v0.6.1) (2025-12-02)


### Bug Fixes

* **io:** add path security validation ([3f82018](https://github.com/SchoolyB/EZ/commit/3f82018747ce98752772faa1304e117009185c48))
* **io:** add path security validation ([a520f30](https://github.com/SchoolyB/EZ/commit/a520f30c9f4f49d9bb3cd07b3953272dd22277d6)), closes [#254](https://github.com/SchoolyB/EZ/issues/254)

## [0.6.0](https://github.com/SchoolyB/EZ/compare/v0.5.0...v0.6.0) (2025-12-02)


### Features

* **lang:** add hex/binary literals and byte warnings ([172028a](https://github.com/SchoolyB/EZ/commit/172028a98ff60b27111b9c87548e599e797e1b35))
* **lang:** implement byte and [byte] data types ([d1f98be](https://github.com/SchoolyB/EZ/commit/d1f98be06acefca64562411f67366232235b40c5))
* **lang:** implement byte and [byte] data types ([4499882](https://github.com/SchoolyB/EZ/commit/4499882917a030a02e842df37ab05c9f563878a8)), closes [#248](https://github.com/SchoolyB/EZ/issues/248)
* **stdlib:** implement [@bytes](https://github.com/bytes) module for binary data operations ([48a05b6](https://github.com/SchoolyB/EZ/commit/48a05b6d3d63e416694abeac198e0008566893a5))
* **stdlib:** implement [@io](https://github.com/io) module for file system operations ([99115a1](https://github.com/SchoolyB/EZ/commit/99115a15f661f200450c7a34a90dc08ffe1d4acd)), closes [#243](https://github.com/SchoolyB/EZ/issues/243)
* **stdlib:** implement [@os](https://github.com/os) module ([f2feac2](https://github.com/SchoolyB/EZ/commit/f2feac2761e028bd1d2178b795e75c7b996231ba))
* **stdlib:** implement [@os](https://github.com/os) module for operating system operations ([7595236](https://github.com/SchoolyB/EZ/commit/75952363d82098c2faad05058985227a7254f348))
* **stdlib:** implement `[@bytes](https://github.com/bytes)` module for binary data operations ([5533a97](https://github.com/SchoolyB/EZ/commit/5533a972a16ad14168cd5187e653d95914324867))
* **stdlib:** implement `[@io](https://github.com/io)` module for file system operations ([4b195d4](https://github.com/SchoolyB/EZ/commit/4b195d45596dea2e2431c4b5c481484cedf27646))


### Bug Fixes

* **stdlib:** resolve CodeQL warnings ([0a7b971](https://github.com/SchoolyB/EZ/commit/0a7b9714fbbb46ebfc23d5b0ae1cc0ce1cba265b))

## [0.5.0](https://github.com/SchoolyB/EZ/compare/v0.4.0...v0.5.0) (2025-12-02)


### Features

* rename std.print to std.printf and fix escape sequences ([d5a03a0](https://github.com/SchoolyB/EZ/commit/d5a03a0f819a566c0d47a2616a11c19e2660838b))
* rename std.print to std.printf and fix escape sequences ([2a17192](https://github.com/SchoolyB/EZ/commit/2a171922b963ff2a37bbc92b0c3cdf50c5c9d6fc)), closes [#237](https://github.com/SchoolyB/EZ/issues/237)


### Bug Fixes

* correct CodeQL workflow order and permissions ([d93affa](https://github.com/SchoolyB/EZ/commit/d93affa9ba06d493044359f7eb48295bcab1d84d))
* correct CodeQL workflow order and permissions ([#221](https://github.com/SchoolyB/EZ/issues/221)) ([29a4c21](https://github.com/SchoolyB/EZ/commit/29a4c21c67055de62938b22bd815b89ba68c4ec3))

## [0.4.0](https://github.com/SchoolyB/EZ/compare/v0.3.0...v0.4.0) (2025-12-01)


### Features

* **stdlib:** add arrays.chunk function ([7bb900b](https://github.com/SchoolyB/EZ/commit/7bb900b4ed0bfe00eebc6e48e138748eaba6b406))
* **stdlib:** add arrays.chunk function ([3964f73](https://github.com/SchoolyB/EZ/commit/3964f7369daa2cb7c344ad1d948be07cb3722256)), closes [#228](https://github.com/SchoolyB/EZ/issues/228)
* **stdlib:** add math constants and infinity check ([b9c4a59](https://github.com/SchoolyB/EZ/commit/b9c4a59dc6b008d4d70bf18f7aa2b094bddc0f3c))
* **stdlib:** add math constants and infinity check ([bdf7c2a](https://github.com/SchoolyB/EZ/commit/bdf7c2a09c3874faef2a389e3a491703710c8585))
* **stdlib:** expand strings module with 12 new functions ([9add23b](https://github.com/SchoolyB/EZ/commit/9add23b10c8bb1dd8e2eca02bee242d412e88c22))
* **stdlib:** expand strings module with 12 new functions ([be16cfe](https://github.com/SchoolyB/EZ/commit/be16cfe857056ea423a723c49dc341002f15ad7e)), closes [#228](https://github.com/SchoolyB/EZ/issues/228)
* **stdlib:** expand time module with arithmetic and utilities ([c7f2be1](https://github.com/SchoolyB/EZ/commit/c7f2be1dc1e8c43fb20d4d4a404cb50f5e2f7cc8))
* **stdlib:** expand time module with arithmetic and utilities ([f7edf78](https://github.com/SchoolyB/EZ/commit/f7edf78f21e7e4a98516d72dc5f46f225a79de73))


### Bug Fixes

* **errors:** standardize error codes across codebase ([fdf91d2](https://github.com/SchoolyB/EZ/commit/fdf91d27a4ace86d15f6037b31c47de159886331))
* **errors:** standardize error codes across codebase ([8cf3bfd](https://github.com/SchoolyB/EZ/commit/8cf3bfd752b016468130c09cb8dd1f98cff23eeb)), closes [#233](https://github.com/SchoolyB/EZ/issues/233)

## [0.3.0](https://github.com/SchoolyB/EZ/compare/v0.2.2...v0.3.0) (2025-12-01)


### Features

* **stdlib:** add strings.repeat() function ([e13972a](https://github.com/SchoolyB/EZ/commit/e13972ad8cc2ec02bba52e88bf0be781dd914c55))
* **stdlib:** add strings.repeat() function ([c50253e](https://github.com/SchoolyB/EZ/commit/c50253e0a56af1c7da6e0349ba8f8d084327934b)), closes [#198](https://github.com/SchoolyB/EZ/issues/198)


### Bug Fixes

* **interpreter:** add helpful suggestions for common function mistakes ([3c56d14](https://github.com/SchoolyB/EZ/commit/3c56d142c59c052d87968740145c468223d1c8b3))
* **interpreter:** add helpful suggestions for common function mistakes ([7a64a64](https://github.com/SchoolyB/EZ/commit/7a64a64aa7f3a207c0c1f9004fc5f03e1fa0eaf4)), closes [#199](https://github.com/SchoolyB/EZ/issues/199)
* **interpreter:** allow empty {} literal for map types ([6f22b9f](https://github.com/SchoolyB/EZ/commit/6f22b9f27ba50e24fc679fa2314651880c7494eb))
* **interpreter:** allow empty {} literal for map types ([f59819e](https://github.com/SchoolyB/EZ/commit/f59819e4ecb35e5640739c5e0394d5cb3449453e)), closes [#194](https://github.com/SchoolyB/EZ/issues/194)
* **interpreter:** auto-detect descending range when start &gt; end ([e21d282](https://github.com/SchoolyB/EZ/commit/e21d2820cac91164fc2095102f3feee18a2de31b))
* **interpreter:** auto-detect descending range when start &gt; end ([057df6d](https://github.com/SchoolyB/EZ/commit/057df6dd4f6ce083dfb617e0804f17e588ab3267)), closes [#197](https://github.com/SchoolyB/EZ/issues/197)
* **lexer:** handle nested quotes in string interpolation ([8506973](https://github.com/SchoolyB/EZ/commit/8506973bb7c7e2ee14cd8b72ebb8071bf311f90a))
* **lexer:** handle nested quotes in string interpolation ([adebd2c](https://github.com/SchoolyB/EZ/commit/adebd2cd25dd2868e88fcc458e5ee0b6d8c2a115)), closes [#193](https://github.com/SchoolyB/EZ/issues/193)
* **loader:** properly format parse errors from imported modules ([30c13fd](https://github.com/SchoolyB/EZ/commit/30c13fd83890a89002a65b7b31f009ea6f239774))
* **loader:** properly format parse errors from imported modules ([bd39c18](https://github.com/SchoolyB/EZ/commit/bd39c18f48c5536a653ed79ed20d407a743f3fcf)), closes [#203](https://github.com/SchoolyB/EZ/issues/203)
* **stdlib:** add arrays.index() function ([868223e](https://github.com/SchoolyB/EZ/commit/868223e676d41b4bb1f884e4cd491a89172e7a35))
* **stdlib:** add arrays.index() function ([da25e94](https://github.com/SchoolyB/EZ/commit/da25e9455e561bb848b149b6be757dcfeb480c52)), closes [#200](https://github.com/SchoolyB/EZ/issues/200)
* **stdlib:** add strings.slice() function ([faeecaa](https://github.com/SchoolyB/EZ/commit/faeecaac9f9ae43bdacba65974dea3e2bb2689b8))
* **stdlib:** add strings.slice() function ([5fd5d9e](https://github.com/SchoolyB/EZ/commit/5fd5d9ee0af2c4fa5339d7f698cb00cee3069225)), closes [#201](https://github.com/SchoolyB/EZ/issues/201)
* **stdlib:** fix time.format() argument order and format conversion ([470c0c7](https://github.com/SchoolyB/EZ/commit/470c0c7cd900631d726a8e8685b8cd69c6ef89e8))
* **stdlib:** fix time.format() argument order and format conversion ([a62b2cb](https://github.com/SchoolyB/EZ/commit/a62b2cb7f5bb03364f18e5d82b232d83e2af1875)), closes [#195](https://github.com/SchoolyB/EZ/issues/195)

## [0.2.2](https://github.com/SchoolyB/EZ/compare/v0.2.1...v0.2.2) (2025-12-01)


### Bug Fixes

* **stdlib:** add uppercase math constants, deprecate lowercase ([f81bfcf](https://github.com/SchoolyB/EZ/commit/f81bfcfc34787337808dbc8e0553658cc2c9c5d0))
* **stdlib:** add uppercase math constants, deprecate lowercase ([4fab99d](https://github.com/SchoolyB/EZ/commit/4fab99dde3e2c2254fea11411f76e08fdb2a6a1f)), closes [#196](https://github.com/SchoolyB/EZ/issues/196)

## [0.2.1](https://github.com/SchoolyB/EZ/compare/v0.2.0...v0.2.1) (2025-12-01)


### Bug Fixes

* **stdlib:** strings.join no longer includes quotes around elements ([1e8c55a](https://github.com/SchoolyB/EZ/commit/1e8c55a99d1fbbd61bbaf7b4d2b851f6dc538220))
* **stdlib:** strings.join no longer includes quotes around elements ([1a4b531](https://github.com/SchoolyB/EZ/commit/1a4b531fb689e04a9767efaba6e6f034010d086f)), closes [#205](https://github.com/SchoolyB/EZ/issues/205)

## [0.2.0](https://github.com/SchoolyB/EZ/compare/v0.1.0...v0.2.0) (2025-12-01)


### Features

* add error tests and simplify README ([8fe0851](https://github.com/SchoolyB/EZ/commit/8fe0851f05f5ee967e3b925554312bd6aa9f093b))
* restructure EZ tests with pass/fail counting ([665588a](https://github.com/SchoolyB/EZ/commit/665588a786cf301de40f13caa95d7988f5968045))
* restructure EZ tests with pass/fail counting ([34893a0](https://github.com/SchoolyB/EZ/commit/34893a0da8e2b4a21bb6efb6a659e7f418446138))

## 0.1.0 (2025-12-01)


### Features

* add [@suppress](https://github.com/suppress)() attribute functionality ([6386409](https://github.com/SchoolyB/EZ/commit/6386409d9d9e009f64218406ce8b2a05a5da368b))
* add automatic versioning with release-please ([ebc27d7](https://github.com/SchoolyB/EZ/commit/ebc27d78562ac54506885557e309a6e31e96937c))
* add basic warning system ([990ffe9](https://github.com/SchoolyB/EZ/commit/990ffe9f37035e0dd8fdbf7337205f044093bb61))
* add installation and distribution system ([230ab5f](https://github.com/SchoolyB/EZ/commit/230ab5f3731524be839c60a4ed41cee8e492fb4c))
* add installation and distribution system ([9839456](https://github.com/SchoolyB/EZ/commit/983945650c2ec8c7d194d7d2795b860a132bc268))
* add project-wide build support to ez build ([ccf189e](https://github.com/SchoolyB/EZ/commit/ccf189e9a3ac82c067fb576844ac85d8beffeb98))
* add support for multi return values/update Parser, AST, Evaluator ([6801f89](https://github.com/SchoolyB/EZ/commit/6801f89a33c34f2ed3f36ece5e0d8de6deaa5d5f))
* add support for multiple inline imports ([3a040e6](https://github.com/SchoolyB/EZ/commit/3a040e6ee45d8263c265584d0e4ba99aed5e31ab))
* add support for multiple inline imports ([eb7d411](https://github.com/SchoolyB/EZ/commit/eb7d411c6c048b68df64fd78baea3d61f061ecb7)), closes [#72](https://github.com/SchoolyB/EZ/issues/72)
* Complete module system implementation ([a9734bb](https://github.com/SchoolyB/EZ/commit/a9734bb7d314e9dfeeb65487b31d7da94de94449))
* enhance CI with comprehensive tests and REPL validation ([222ff12](https://github.com/SchoolyB/EZ/commit/222ff1259bc6145c473e2eb69cb52585429af5da))
* enhance CI with comprehensive tests and REPL validation ([bfaf414](https://github.com/SchoolyB/EZ/commit/bfaf414162a5a3aae6814228feeb65bf0bdba772))
* implement full enum support with attributes ([98c90ce](https://github.com/SchoolyB/EZ/commit/98c90cefff3a80c3499091d8db322b45d1966f38))
* implement full enum support with attributes ([a8f6e06](https://github.com/SchoolyB/EZ/commit/a8f6e065d9a3d65a2a43042b718bc9f1c57bbd9a)), closes [#9](https://github.com/SchoolyB/EZ/issues/9)
* implement interactive REPL mode ([fcca9f8](https://github.com/SchoolyB/EZ/commit/fcca9f8a049db91f91dcb2b161979a43ab2f2438))
* implement interactive REPL mode ([fae00d9](https://github.com/SchoolyB/EZ/commit/fae00d9dcef0f4abb3298eac206d9024593d427e)), closes [#45](https://github.com/SchoolyB/EZ/issues/45)
* implement type sharing for function parameters ([1991480](https://github.com/SchoolyB/EZ/commit/19914808a5a3dbcba799063b1cdb6b08dd25dd91))
* implement type sharing for function parameters ([5fc89e5](https://github.com/SchoolyB/EZ/commit/5fc89e5a73b339ee8afdf48325a30a03b4a29e0f)), closes [#10](https://github.com/SchoolyB/EZ/issues/10)
* improve release artifacts with archives and checksums ([5d217c5](https://github.com/SchoolyB/EZ/commit/5d217c5e388176ff5a167470727c68bdb231acaf))
* improve release artifacts with archives and checksums ([f51725d](https://github.com/SchoolyB/EZ/commit/f51725dde475760259c284c27d2f16bf9e26c36f))
* intergrate new error handling system into parser, evaluator, & lexer ([6613db9](https://github.com/SchoolyB/EZ/commit/6613db9cee3d0072565115bc5adf4f4a718522fa))


### Bug Fixes

* add error E3007 for missing main() function ([b5f6a90](https://github.com/SchoolyB/EZ/commit/b5f6a90d9f5975db1449d83c331ce11604314a50))
* add error E3007 for missing main() function ([fb995eb](https://github.com/SchoolyB/EZ/commit/fb995eba05f15261547ea9a8a492f247293ba496))
* add support for array and string index assignment ([e46004b](https://github.com/SchoolyB/EZ/commit/e46004b624f0fd1bf007add4cb90f8a00cf3ee15))
* add support for array and string index assignment (issue [#4](https://github.com/SchoolyB/EZ/issues/4), [#12](https://github.com/SchoolyB/EZ/issues/12)) ([f92d63e](https://github.com/SchoolyB/EZ/commit/f92d63e965efd74a033226c16e999b37ee774852))
* add warning W2003 for missing return statement ([09f73d0](https://github.com/SchoolyB/EZ/commit/09f73d0df3fa0ed0faf4d36c5b15dd6ac8e9472d))
* add warning W2003 for missing return statement ([0ca1056](https://github.com/SchoolyB/EZ/commit/0ca105620ef8c81a9294509b6829b0a5d83ceca3))
* allow otherwise/or keywords after return/break/continue in if blocks ([2523722](https://github.com/SchoolyB/EZ/commit/25237225c58040544fff511b208fdc46440cfed0))
* allow otherwise/or keywords after return/break/continue in if blocks ([0719e54](https://github.com/SchoolyB/EZ/commit/0719e5442ec762ee252192aeb816ef33d5362bc7)), closes [#14](https://github.com/SchoolyB/EZ/issues/14)
* allow variable shadowing in nested blocks (parser only) ([27babab](https://github.com/SchoolyB/EZ/commit/27babab6c5062eb1c535c9299117d997a5b983ee))
* allow variable shadowing in nested blocks (parser) ([7f8b40d](https://github.com/SchoolyB/EZ/commit/7f8b40d1627c56df94c747eae06baf0b7f5d39d2))
* Boolean negation and comparison for stdlib returns ([1f274ff](https://github.com/SchoolyB/EZ/commit/1f274ffb14d40ee92e4b8d4f64dd5a384f8bda32)), closes [#146](https://github.com/SchoolyB/EZ/issues/146)
* const declarations now support qualified type names (module.Type) ([250c944](https://github.com/SchoolyB/EZ/commit/250c94431ce1fc310d565e6b20cbe67b3ef75340)), closes [#157](https://github.com/SchoolyB/EZ/issues/157)
* const variable reassignment bug ([44f41d0](https://github.com/SchoolyB/EZ/commit/44f41d046fee129cc2fd353e1955ab020d7f8a66))
* deduplicate modules in using scope to prevent ambiguity ([34fc8c7](https://github.com/SchoolyB/EZ/commit/34fc8c7aff1ef8dd3ee787b5e2698d357ca78df1))
* deduplicate modules in using scope to prevent ambiguity ([b48200e](https://github.com/SchoolyB/EZ/commit/b48200ed0b8e52320b09fd0d5791d4526dcb2ef3))
* directory modules now properly share namespace across files ([ec0d84b](https://github.com/SchoolyB/EZ/commit/ec0d84b2897694490cf9c9f4d21f49c50868f0c6))
* Display strings with quotes to distinguish from numbers (issue [#99](https://github.com/SchoolyB/EZ/issues/99)) ([6fa2f77](https://github.com/SchoolyB/EZ/commit/6fa2f7704ed98b5d990565c1fb86e943f2ef41d4))
* duplicate struct member names/add: loop depth tracking ([fbcbb90](https://github.com/SchoolyB/EZ/commit/fbcbb908afa5ff0cc1fdf70fbf75e7f763b55826))
* enable logical operators (&&, ||, !) in if conditions ([907f2c1](https://github.com/SchoolyB/EZ/commit/907f2c1c3324acc3bce5e346bdb6b603373a40db))
* enable logical operators (&&, ||, !) in if conditions (issue [#8](https://github.com/SchoolyB/EZ/issues/8), [#17](https://github.com/SchoolyB/EZ/issues/17)) ([ebf46da](https://github.com/SchoolyB/EZ/commit/ebf46da46ca4f1edf6c4077e9ec3a883f70337c6))
* error bug, module importing error bugs ([e11c8de](https://github.com/SchoolyB/EZ/commit/e11c8deb4bf896f5e116f7025c234c7249329249))
* extend for_each loop to support string iteration ([89cd5f1](https://github.com/SchoolyB/EZ/commit/89cd5f1015deb8eb12cbfde5b2cfe5fbcb9cc0a7))
* extend for_each loop to support string iteration ([f8e1ae7](https://github.com/SchoolyB/EZ/commit/f8e1ae76d49afb71194552622a50cdcee3bac732)), closes [#7](https://github.com/SchoolyB/EZ/issues/7)
* Float values ending in .0 now display decimal point ([5eaa329](https://github.com/SchoolyB/EZ/commit/5eaa329043951382f656087cbac854ea3dd75754))
* function parameter parsing bug ([dc49ff8](https://github.com/SchoolyB/EZ/commit/dc49ff808264e9e41d31e183c8edc17b4438c49c))
* implement char comparison operators ([50f1420](https://github.com/SchoolyB/EZ/commit/50f1420b38f6256d1030828d20d692abed601143))
* implement char comparison operators ([2d68f4e](https://github.com/SchoolyB/EZ/commit/2d68f4e860e31cfffe557c8830850756737217e5))
* initialize dynamic arrays to empty array instead of NIL ([87a5f3d](https://github.com/SchoolyB/EZ/commit/87a5f3d0c95de0f32b507606e7ee53831fe5fbbb))
* initialize dynamic arrays to empty array instead of NIL ([68e767f](https://github.com/SchoolyB/EZ/commit/68e767f4f2747a1b35d206c571e07a51e585783b))
* int() builtin now supports enum value conversion ([857e9dc](https://github.com/SchoolyB/EZ/commit/857e9dc7d0da94251c19792679e5ed8c8d6e4a85)), closes [#154](https://github.com/SchoolyB/EZ/issues/154)
* invalid import statments not throwing errors ([3f8ea40](https://github.com/SchoolyB/EZ/commit/3f8ea40a04309dd3143e970f40f4c70730eb61d5))
* len() builtin now supports maps ([8550f56](https://github.com/SchoolyB/EZ/commit/8550f56655a7f8fc88f5219a7ed7ce0a902c3301)), closes [#147](https://github.com/SchoolyB/EZ/issues/147)
* lexer bug when handling @ in imports and attributes ([75786d6](https://github.com/SchoolyB/EZ/commit/75786d6f6557280d2f20789fb1187a1815bffef2))
* logic that allowed nested function declarations ([e2a60e9](https://github.com/SchoolyB/EZ/commit/e2a60e9e89b2c3d00a30d2899367efe9c578f60e))
* make arrays.push() and arrays.pop() modify arrays in-place (issu… ([34f2b35](https://github.com/SchoolyB/EZ/commit/34f2b35f461ac2d55dbbcd351d4f044205329e2a))
* make arrays.push() and arrays.pop() modify arrays in-place (issue [#1](https://github.com/SchoolyB/EZ/issues/1), [#18](https://github.com/SchoolyB/EZ/issues/18)) ([c751492](https://github.com/SchoolyB/EZ/commit/c7514928a12534e92b290e1343917f2c5a6d76c0))
* Make range() end value exclusive (issue [#112](https://github.com/SchoolyB/EZ/issues/112)) ([430abd5](https://github.com/SchoolyB/EZ/commit/430abd532cda2bd20da7c452366e367e292138bf))
* modulo by zero bug/ arg count mismatch bug ([630a716](https://github.com/SchoolyB/EZ/commit/630a7164a736b979d6f8b6254ac2a1d30bc8d962))
* parser no longer misidentifies uppercase enum members as struct literals ([1a08edf](https://github.com/SchoolyB/EZ/commit/1a08edf284e5abd5d40864043b46a6e6f388912d))
* parser no longer misidentifies uppercase enum members as struct literals ([fe481c6](https://github.com/SchoolyB/EZ/commit/fe481c6ca9a5ade43b61e53cbc999e8b8d44501a)), closes [#162](https://github.com/SchoolyB/EZ/issues/162)
* preserve array mutability when passed to functions ([4a71667](https://github.com/SchoolyB/EZ/commit/4a716678a855b8b23a67cc94cde664825af549f3))
* preserve array mutability when passed to functions ([1ef204f](https://github.com/SchoolyB/EZ/commit/1ef204f5f7d7ce6a6a40a9a73a620cc17f137a3d)), closes [#155](https://github.com/SchoolyB/EZ/issues/155)
* preserve enum type information with EnumValue wrapper ([a5c9e5f](https://github.com/SchoolyB/EZ/commit/a5c9e5fb97505a5312745afae42ce14b62735cfe))
* preserve enum type information with EnumValue wrapper ([5b4890b](https://github.com/SchoolyB/EZ/commit/5b4890b996426c9f84bf0ff2f8eda5194a2a658b))
* Prevent const variable mutation through arrays and index assignment (issue [#98](https://github.com/SchoolyB/EZ/issues/98)) ([9fac9b6](https://github.com/SchoolyB/EZ/commit/9fac9b669fbf59e01d66490ba332ae3a10fb6b35))
* Prevent using reserved keywords/types as identifiers ([fb88466](https://github.com/SchoolyB/EZ/commit/fb88466ff9ac89e260978eba195f1df57e10652e))
* provide default values for uninitialized primitive types ([0584478](https://github.com/SchoolyB/EZ/commit/058447852b16b1e1852f139bed36a72f75372380))
* provide default values for uninitialized primitive types ([550a331](https://github.com/SchoolyB/EZ/commit/550a331e18521171f203035c189a87e7f3e15e43))
* range() now supports 1, 2, or 3 arguments ([cb27fef](https://github.com/SchoolyB/EZ/commit/cb27fef976ffc3ad88a6025d7bc7dcd7cb7b407d))
* resolve golangci-lint errors ([8316faf](https://github.com/SchoolyB/EZ/commit/8316faf23a41c7c7d43b8dbeb98f6f64c2c4a9b2))
* resolve module system bugs for type resolution, arrays, enums, and using directive ([045bfca](https://github.com/SchoolyB/EZ/commit/045bfcadb9ac09876b7d2199811b52428bf48b82))
* resolve types from modules via using directive ([26801df](https://github.com/SchoolyB/EZ/commit/26801df8cd5d60f027da797708be097820eb178a))
* resolve types from modules via using directive ([69c433d](https://github.com/SchoolyB/EZ/commit/69c433d43524f75eadcafddf7a9c51cc43c18975)), closes [#153](https://github.com/SchoolyB/EZ/issues/153)
* Return statements now correctly parse struct/array/map literals ([f91bf8c](https://github.com/SchoolyB/EZ/commit/f91bf8c35336d67904b168ddb060f5b320b43c02)), closes [#142](https://github.com/SchoolyB/EZ/issues/142)
* struct evaluation bug & updatec core struct logic ([7660298](https://github.com/SchoolyB/EZ/commit/76602980c97ddd1a99d7dd7867714bc01f8df8bc))
* support array types in function return declarations ([5ef0207](https://github.com/SchoolyB/EZ/commit/5ef0207d327f0a69c9ee71f0c76e0d7db2910ee9))
* support array types in function return declarations ([3c42e1c](https://github.com/SchoolyB/EZ/commit/3c42e1c71778c0615fc7a8a58638823ae3dadd64))
* support enum type annotation syntax and prevent nil pointer crash ([66ef481](https://github.com/SchoolyB/EZ/commit/66ef481c7ac325cd2da737b4888edebfe426ad52))
* support enum type annotation syntax and prevent nil pointer crash ([464ed95](https://github.com/SchoolyB/EZ/commit/464ed956ede4c4cf91ee4d5f7d608fd430548ce4)), closes [#149](https://github.com/SchoolyB/EZ/issues/149)
* support file-level using directive in type resolution ([843f912](https://github.com/SchoolyB/EZ/commit/843f9127eb9aa067b702acf2aaa8149354981bde))
* support fixed-size array declarations ([8313847](https://github.com/SchoolyB/EZ/commit/83138475dd85c9e97c1ab2dd7a0f2441128f5d52))
* support fixed-size array declarations ([92edd88](https://github.com/SchoolyB/EZ/commit/92edd882e7acc66f8ff788217148c6693cb5254e))
* Support underscore syntax in int() and float() conversions (issue [#103](https://github.com/SchoolyB/EZ/issues/103)) ([58352ce](https://github.com/SchoolyB/EZ/commit/58352cebb38e6053926a5938ce704eddc0909fc5))
* track imports by alias to support aliased using statements ([421d0d5](https://github.com/SchoolyB/EZ/commit/421d0d515acf0044c35320e4765d1a8974ac0a15))
* track imports by alias to support aliased using statements ([7396473](https://github.com/SchoolyB/EZ/commit/7396473a98432bfde408c61bd5217c9496ea6cf4))
* Typed tuple unpacking now parses correctly ([8a5e599](https://github.com/SchoolyB/EZ/commit/8a5e5993e171cdbe1647d8e92c4d4922a58e3724)), closes [#145](https://github.com/SchoolyB/EZ/issues/145)
* uncaptured return value bug ([4cbc30b](https://github.com/SchoolyB/EZ/commit/4cbc30b9093cad96a34f942c925b7c30d08baa9c))
* unwrap enum values before comparison operations ([51338dd](https://github.com/SchoolyB/EZ/commit/51338dde2344ef8f19f555557153b3a0b7d077f6))
* unwrap enum values before comparison operations ([fca6931](https://github.com/SchoolyB/EZ/commit/fca6931c15ae9c066de268f4d1b2d034cdeed9ed)), closes [#163](https://github.com/SchoolyB/EZ/issues/163)
* use global stdin reader to fix input() buffering issues ([a18aa23](https://github.com/SchoolyB/EZ/commit/a18aa23fff18fda4c00b65d8d16ffd16e3f896cc))
* use global stdin reader to fix input() buffering issues ([cf20c81](https://github.com/SchoolyB/EZ/commit/cf20c81b02579c6cdc8a654b4487c3adcb78aad8))
* Validate enum type attributes to only allow primitives (issue [#119](https://github.com/SchoolyB/EZ/issues/119)) ([135058d](https://github.com/SchoolyB/EZ/commit/135058dfcfc1b01a59ff8bf5caef23c0f6254f7a))
* Validate module import before using statement ([5aabb28](https://github.com/SchoolyB/EZ/commit/5aabb2869270a325632dbadb41ba7b50941277d6))
* Validate unknown attributes and prevent parser hang ([3a41c6b](https://github.com/SchoolyB/EZ/commit/3a41c6b8e8ced0c802432e372e6b217084803603)), closes [#165](https://github.com/SchoolyB/EZ/issues/165)


### Miscellaneous Chores

* prepare initial release ([fefa831](https://github.com/SchoolyB/EZ/commit/fefa831dacb27e7931d6e25b0d38169a9eec861a))
