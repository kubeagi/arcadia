# graphql go server

## How to develop

**There can only be one Query structure in all schemas, and Mutation can have one or none.**
**Query and Mutation define the entry point to the service. The fields, functions, defined below these two structures are what the front-end uses to query.**

Now that there is a `datasource` definition in the code, we can add new structures and related functions as follows.

### Add schema

go to `graph/schema` dir.

```shell
cat << EOF > x.graphqls
type X {
  name: String
}
EOF
```

### Add a query function

The `graph/schema/datasource.graphqls` file defines the Query structure, so we need to edit this file to add a function.

```shell
type Query {
    ds(input: QueryDatasource!): [Datasource!]
    findX(input: String!): String!
}
```

### generate resolver

in the root dir of the project, update `gqlgen.yaml` file.

in the root dir of the project:

```shell
make gql-gen
```

Looking at `datasource.resolver.go`, I can see the addition of the function we defined:

```go
func (r *queryResolver) FindX(ctx context.Context, input string) (string, error) {
   panic(fmt.Errorf("not implemented: FindX - findX"))
}
```

The file `model/model_gen.go` has a new structure x that we defined.

All we have to do is just implement the `FindX` function. And the content of the function is up to you to play with.

## How to run

At the root dir of the project

1. Build graphql server

```shell
make build-graphql-server
```

2. Try graphql-server

```shell
$ ./bin/graphql-server -h
Usage of ./bin/graphql-server:
  -host string
        bind to the host, default is 0.0.0.0
  -port int
        service listening port (default 8081)
  -enable-playground
        whether to enable the graphql playground
  -enable-oidc
        whether to enable oidc authentication
  -client-id string
        oidc client id
  -client-secret string
        oidc client secret
  -master-url string
        k8s master url
  -issuer-url string
        oidc issuer url
  -kubeconfig string
        Paths to a kubeconfig. Only required if out-of-cluster.
```

3. Run graphql-server

> If you don't want to try playground, do not pass flag `-enable-plaground`

```shell
./bin/graphql-server -enable-playground --client-id=bff-client --client-secret=some-secret --master-url=https://k8s-adress --issuer-url=https://oidc-server
```

4. Try apis with plaground

Open http://localhost:8081/ in your browser!
