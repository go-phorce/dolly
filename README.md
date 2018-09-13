# go-phorce/pkg

GO packages for building web apps

[![Build Status](https://travis-ci.org/go-phorce/pkg.svg?branch=master)](https://travis-ci.org/go-phorce/pkg)
[![Coverage Status](https://coveralls.io/repos/github/go-phorce/pkg/badge.svg?branch=master)](https://coveralls.io/github/go-phorce/pkg?branch=master)

## Contribution

Before openning VSCODE or running make, run once:
    ./vscode.sh

* `make all` complete build and test
* `make get` fetches the pinned dependencies from repos
* `make devtools` get the dev tools for local development in VSCODE
* `make test` run the tests
* `make testshort` runs the tests skipping the end-to-end tests and the code coverage reporting
* `make covtest` runs the tests with end-to-end and the code coverage reporting
* `make coverage` view the code coverage results from the last make test run.
* `make generate` runs go generate to update any code generated files
* `make fmt` runs go fmt on the project.
* `make lint` runs the go linter on the project.

run `make get` once, then run `make build` or `make test` as needed.

First run:

    make all

Tests:

    make test

Optionally run golang race detector with test targets by setting RACE flag:

    make test RACE=true

Review coverage report:

    make covtest coverage
