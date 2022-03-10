# tfpgen

An experimental OpenAPI → Terraform Provider generator that does not yet function. The goal is to allow developers to incrementally generate and maintain their own simple [Terraform Provider](https://www.terraform.io/registry/providers) using an [OpenAPI 3 specification](https://en.wikipedia.org/wiki/OpenAPI_Specification).

- [x] Examine an OpenAPI spec, identify RESTful resource groups `tfpgen examine spec.yaml`
- [x] Generate a config file for each discovered resource/data source `tfpgen init spec.yaml`
- [ ] Using a combination of the spec and config, generate the provider `tfpgen generate`
  - [ ] Generate [Terraform plugin framework](https://github.com/hashicorp/terraform-plugin-framework) code for each resource/datasource
  - [ ] Generate http client code and caller code for each resource/datasource
  - [x] Generate Terraform framework provider code to describe resource schema
  - [ ] Generate acceptance tests

## Other Solutions

- [terraform-provider-openapi](https://github.com/dikhan/terraform-provider-openapi)
A single provider that configures itself _at runtime_ given an OpenAPI specification. This is incredibly cool. However, in the author's opinion, given that the underlying state AND the specification can drift away from the stored Terraform state, operating this provider can give unpredictable results.

- [terraform-provider-restapi](https://github.com/Mastercard/terraform-provider-restapi)
Manage a RESTful resource as a terraform resource. Another neat approach, but it requires that you define the API as Terraform config. The state of the object is completely dependent on the raw response of the endpoint.

## Usage

`tfpgen [--version] [--help] <command> [<args>]`

### Available commands are:

| command  | description                                                                                                     |
|----------|-----------------------------------------------------------------------------------------------------------------|
| examine  | Examine an openapi 3 specification and display a list of possible data sources and resources                    |
| init     | Create an initial configuration based on a spec. This file is meant to be edited in order to compose a provider |
| generate | Generate Terraform provider using the configuration                                                             |

## Try it:

`go run main.go examine examples/openapi3/petstore.yaml`
