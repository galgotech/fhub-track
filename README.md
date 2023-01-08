# fhub-track

Fhub-track allows you to fork a project with only the code you need, change for your necessity, and keep track of the changes from the base repository.

## Example
The project [gotools](https://go.googlesource.com/tools) has an internal tool for diff that is not a public package to use. Using the fhub-track, we can fork only the diff tool and update the code when it has a new release. The new project with the public diff tool [galgotech/gotools](https://github.com/galgotech/gotools).

## License

Fermions is distributed under [AGPL-3.0-only](LICENSE). 
